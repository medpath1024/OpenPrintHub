package web

import (
	"net/http"

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

// StatsPartial handles GET /partials/stats (HTMX partial)
func (h *Handlers) StatsPartial(c *gin.Context) {
	stats := h.printQueue.Stats()

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(c.Writer, "stats_partial.html", stats); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}
