package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// Handlers contains web page handlers
type Handlers struct {
	printerSvc printer.Service
	printQueue *print.Queue
	version    string
}

// NewHandlers creates new web handlers
func NewHandlers(printerSvc printer.Service, printQueue *print.Queue, version string) *Handlers {
	return &Handlers{
		printerSvc: printerSvc,
		printQueue: printQueue,
		version:    version,
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
	Version      string
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
	ExportedBy         string               `json:"exported_by"`
	ExportedAt         time.Time            `json:"exported_at"`
	Platform           string               `json:"platform"`
	ServiceVersion     string               `json:"service_version"`
	Job                *printer.PrintJob    `json:"job,omitempty"`
	Result             *printer.JobResult   `json:"result,omitempty"`
	PrinterStatus      *printer.PrinterInfo `json:"printer_status,omitempty"`
	PrinterStatusError string               `json:"printer_status_error,omitempty"`
	PrinterDiagnostics *PrinterDiagnostics  `json:"printer_diagnostics,omitempty"`
	PayloadBytes       int                  `json:"payload_bytes"`
	PayloadBase64Bytes int                  `json:"payload_base64_bytes"`
	PayloadSHA256      string               `json:"payload_sha256,omitempty"`
	ProcessingDuration string               `json:"processing_duration"`
	QueueWaitMS        int64                `json:"queue_wait_ms,omitempty"`
	PrintRunMS         int64                `json:"print_run_ms,omitempty"`
	TotalElapsedMS     int64                `json:"total_elapsed_ms,omitempty"`
}

// PrinterDiagnostics captures raw OS-level printer details for troubleshooting.
type PrinterDiagnostics struct {
	CollectedAt time.Time            `json:"collected_at"`
	Platform    string               `json:"platform"`
	PrinterName string               `json:"printer_name"`
	CUPSName    string               `json:"cups_name,omitempty"`
	Commands    []CommandDiagnostics `json:"commands,omitempty"`
}

// CommandDiagnostics stores one command execution result.
type CommandDiagnostics struct {
	Command    string    `json:"command"`
	Args       []string  `json:"args,omitempty"`
	RanAt      time.Time `json:"ran_at"`
	DurationMS int64     `json:"duration_ms"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout,omitempty"`
	Stderr     string    `json:"stderr,omitempty"`
	Error      string    `json:"error,omitempty"`
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
		Version:      h.version,
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
		Version:  h.version,
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
		Title:   "Jobs",
		Active:  "jobs",
		Version: h.version,
		Jobs:    jobs,
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

	var (
		printerDiag  *PrinterDiagnostics
		queueWaitMS  int64
		printRunMS   int64
		totalElapsed int64
	)
	if data.Entry.Result != nil {
		printerDiag = h.collectPrinterDiagnostics(data.Entry.Result)
		queueWaitMS = durationMS(data.Entry.Result.CreatedAt, data.Entry.Result.StartedAt)
		printRunMS = durationMS(data.Entry.Result.StartedAt, data.Entry.Result.CompletedAt)
		totalElapsed = durationMS(data.Entry.Result.CreatedAt, data.Entry.Result.CompletedAt)
	}

	exportPayload := JobDiagnosticsExport{
		ExportedBy:         "OpenPrintHub",
		ExportedAt:         time.Now(),
		Platform:           runtime.GOOS,
		ServiceVersion:     h.version,
		Job:                data.Entry.Job,
		Result:             data.Entry.Result,
		PrinterStatus:      data.PrinterStatus,
		PrinterStatusError: data.PrinterStatusError,
		PrinterDiagnostics: printerDiag,
		PayloadBytes:       data.PayloadBytes,
		PayloadBase64Bytes: data.PayloadBase64Bytes,
		PayloadSHA256:      data.PayloadSHA256,
		ProcessingDuration: data.ProcessingDuration,
		QueueWaitMS:        queueWaitMS,
		PrintRunMS:         printRunMS,
		TotalElapsedMS:     totalElapsed,
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

func (h *Handlers) collectPrinterDiagnostics(result *printer.JobResult) *PrinterDiagnostics {
	if result == nil || result.PrinterName == "" {
		return nil
	}

	diag := &PrinterDiagnostics{
		CollectedAt: time.Now(),
		Platform:    runtime.GOOS,
		PrinterName: result.PrinterName,
	}

	switch runtime.GOOS {
	case "darwin", "linux":
		cupsName := strings.ReplaceAll(result.PrinterName, " ", "_")
		diag.CUPSName = cupsName
		diag.Commands = []CommandDiagnostics{
			runDiagnosticCommand("lpstat", "-p", cupsName, "-l"),
			runDiagnosticCommand("lpstat", "-v", cupsName),
			runDiagnosticCommand("lpstat", "-W", "all", "-o", cupsName),
			runDiagnosticCommand("lpoptions", "-p", cupsName, "-l"),
			runDiagnosticCommand("lpstat", "-t"),
		}
	case "windows":
		escaped := strings.ReplaceAll(result.PrinterName, "'", "''")
		diag.Commands = []CommandDiagnostics{
			runDiagnosticCommand("powershell.exe", "-NoProfile", "-Command", fmt.Sprintf("Get-Printer -Name '%s' | Format-List *", escaped)),
			runDiagnosticCommand("powershell.exe", "-NoProfile", "-Command", fmt.Sprintf("Get-PrintJob -PrinterName '%s' | Select-Object -First 20 | Format-List *", escaped)),
		}
	default:
		diag.Commands = []CommandDiagnostics{
			{
				Command:  "diagnostics",
				RanAt:    time.Now(),
				ExitCode: -1,
				Error:    "platform diagnostics not implemented",
			},
		}
	}

	return diag
}

func runDiagnosticCommand(command string, args ...string) CommandDiagnostics {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		exitCode = -1
		var exitErr *exec.ExitError
		if ok := errors.As(err, &exitErr); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return CommandDiagnostics{
		Command:    command,
		Args:       args,
		RanAt:      start,
		DurationMS: time.Since(start).Milliseconds(),
		ExitCode:   exitCode,
		Stdout:     truncateForExport(stdout.String(), 32*1024),
		Stderr:     truncateForExport(stderr.String(), 32*1024),
		Error:      errMsg,
	}
}

func truncateForExport(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "\n...[truncated]"
}

func durationMS(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}
