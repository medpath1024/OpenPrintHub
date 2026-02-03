package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// Handlers contains web page handlers
type Handlers struct {
	printerSvc printer.Service
	printQueue *print.Queue
}

// NewHandlers creates new web handlers
func NewHandlers(printerSvc printer.Service, printQueue *print.Queue) *Handlers {
	return &Handlers{
		printerSvc: printerSvc,
		printQueue: printQueue,
	}
}

// PrinterStats holds printer status counts
type PrinterStats struct {
	Total   int
	Ready   int
	Busy    int
	Offline int
	Error   int
}

// PageData holds common data for all pages
type PageData struct {
	Title        string
	Active       string
	Printers     []printer.PrinterInfo
	PrinterStats PrinterStats
	Jobs         []*print.JobEntry
	Stats        print.QueueStats
}

// JobInfoData holds detailed job information for troubleshooting view
type JobInfoData struct {
	Entry              *print.JobEntry
	PayloadBytes       int
	PayloadBase64Bytes int
	PayloadSHA256      string
	ProcessingDuration string
	PrinterStatus      *printer.PrinterInfo
	PrinterStatusError string
	Error              string
}

// JobDiagnosticsExport is a serializable diagnostics payload for download
type JobDiagnosticsExport struct {
	ExportedAt         time.Time            `json:"exported_at"`
	Job                *printer.PrintJob    `json:"job,omitempty"`
	Result             *printer.JobResult   `json:"result,omitempty"`
	PrinterStatus      *printer.PrinterInfo `json:"printer_status,omitempty"`
	PrinterStatusError string               `json:"printer_status_error,omitempty"`
	PayloadBytes       int                  `json:"payload_bytes"`
	PayloadBase64Bytes int                  `json:"payload_base64_bytes"`
	PayloadSHA256      string               `json:"payload_sha256,omitempty"`
	ProcessingDuration string               `json:"processing_duration"`
}

// Index handles GET /
func (h *Handlers) Index(c *gin.Context) {
	printers, _ := h.printerSvc.List()
	stats := h.printQueue.Stats()

	// Calculate printer status counts
	printerStats := PrinterStats{Total: len(printers)}
	for _, p := range printers {
		switch p.Status {
		case "Ready":
			printerStats.Ready++
		case "Busy":
			printerStats.Busy++
		case "Offline":
			printerStats.Offline++
		case "Error":
			printerStats.Error++
		}
	}

	data := PageData{
		Title:        "Dashboard",
		Active:       "dashboard",
		Printers:     printers,
		PrinterStats: printerStats,
		Stats:        stats,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "index.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// Printers handles GET /printers
func (h *Handlers) Printers(c *gin.Context) {
	printers, _ := h.printerSvc.List()

	data := PageData{
		Title:    "Printers",
		Active:   "printers",
		Printers: printers,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "printers.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// Jobs handles GET /jobs
func (h *Handlers) Jobs(c *gin.Context) {
	jobs := h.printQueue.GetHistory(50)

	data := PageData{
		Title:  "Jobs",
		Active: "jobs",
		Jobs:   jobs,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "jobs.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// PrintersPartial handles GET /partials/printers (HTMX partial)
func (h *Handlers) PrintersPartial(c *gin.Context) {
	printers, _ := h.printerSvc.List()

	data := PageData{
		Printers: printers,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "printers_partial.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// JobsPartial handles GET /partials/jobs (HTMX partial)
func (h *Handlers) JobsPartial(c *gin.Context) {
	jobs := h.printQueue.GetHistory(50)

	data := PageData{
		Jobs: jobs,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "jobs_partial.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// JobInfoPartial handles GET /partials/jobs/:id/info (HTMX partial)
func (h *Handlers) JobInfoPartial(c *gin.Context) {
	jobID := c.Param("id")
	data, ok := h.collectJobInfo(jobID)
	if !ok {
		c.Status(http.StatusNotFound)
		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := Templates.ExecuteTemplate(c.Writer, "job_info_partial.html", JobInfoData{
			Error: fmt.Sprintf("Job %s not found", jobID),
		}); err != nil {
			c.String(http.StatusInternalServerError, "Template error: %v", err)
		}
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "job_info_partial.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// ExportJobInfo handles GET /partials/jobs/:id/info/export (JSON download)
func (h *Handlers) ExportJobInfo(c *gin.Context) {
	jobID := c.Param("id")
	data, ok := h.collectJobInfo(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("job %s not found", jobID),
		})
		return
	}

	exportPayload := JobDiagnosticsExport{
		ExportedAt:         time.Now(),
		Job:                data.Entry.Job,
		Result:             data.Entry.Result,
		PrinterStatus:      data.PrinterStatus,
		PrinterStatusError: data.PrinterStatusError,
		PayloadBytes:       data.PayloadBytes,
		PayloadBase64Bytes: data.PayloadBase64Bytes,
		PayloadSHA256:      data.PayloadSHA256,
		ProcessingDuration: data.ProcessingDuration,
	}

	filenameID := jobID
	if len(filenameID) > 8 {
		filenameID = filenameID[:8]
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"job-%s-diagnostics.json\"", filenameID))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, exportPayload)
}

// StatsPartial handles GET /partials/stats (HTMX partial)
func (h *Handlers) StatsPartial(c *gin.Context) {
	stats := h.printQueue.Stats()

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "stats_partial.html", stats); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

func processingDuration(result *printer.JobResult) string {
	if result == nil || result.StartedAt.IsZero() {
		return "-"
	}

	end := result.CompletedAt
	if end.IsZero() {
		end = time.Now()
	}
	if end.Before(result.StartedAt) {
		return "-"
	}

	return end.Sub(result.StartedAt).Round(time.Millisecond).String()
}

func (h *Handlers) collectJobInfo(jobID string) (JobInfoData, bool) {
	entry, ok := h.printQueue.GetJob(jobID)
	if !ok {
		return JobInfoData{}, false
	}

	var (
		printerStatus      *printer.PrinterInfo
		printerStatusError string
	)
	if entry.Result != nil && entry.Result.PrinterName != "" {
		status, err := h.printerSvc.Status(entry.Result.PrinterName)
		if err != nil {
			printerStatusError = err.Error()
		} else {
			printerStatus = status
		}
	}

	payloadHash := ""
	if entry.Job != nil && len(entry.Job.Data) > 0 {
		sum := sha256.Sum256(entry.Job.Data)
		payloadHash = hex.EncodeToString(sum[:])
	}

	data := JobInfoData{
		Entry:              entry,
		PayloadSHA256:      payloadHash,
		ProcessingDuration: processingDuration(entry.Result),
		PrinterStatus:      printerStatus,
		PrinterStatusError: printerStatusError,
	}
	if entry.Job != nil {
		data.PayloadBytes = len(entry.Job.Data)
		data.PayloadBase64Bytes = len(entry.Job.DataBase64)
	}

	return data, true
}
