package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// Version information (set at build time)
var (
	Version   = "0.1.0"
	BuildTime = ""
	GitCommit = ""
	startTime = time.Now()
)

// Handlers contains all API handlers
type Handlers struct {
	printerSvc printer.Service
	printQueue *print.Queue
}

// NewHandlers creates new API handlers
func NewHandlers(printerSvc printer.Service, printQueue *print.Queue) *Handlers {
	return &Handlers{
		printerSvc: printerSvc,
		printQueue: printQueue,
	}
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": Version,
	})
}

// GetInfo handles GET /v1/info - returns detailed service information
func (h *Handlers) GetInfo(c *gin.Context) {
	printers, _ := h.printerSvc.List()
	stats := h.printQueue.Stats()

	c.JSON(http.StatusOK, gin.H{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"platform":   runtime.GOOS,
		"arch":       runtime.GOARCH,
		"go_version": runtime.Version(),
		"uptime":     int64(time.Since(startTime).Seconds()),
		"printers":   len(printers),
		"stats":      stats,
		"downloads": gin.H{
			"darwin":  "https://github.com/medpath1024/OpenPrintHub/releases/latest/download/oph-darwin-amd64",
			"windows": "https://github.com/medpath1024/OpenPrintHub/releases/latest/download/oph-windows-amd64.exe",
			"linux":   "https://github.com/medpath1024/OpenPrintHub/releases/latest/download/oph-linux-amd64",
		},
	})
}

// ListPrinters handles GET /v1/printers
func (h *Handlers) ListPrinters(c *gin.Context) {
	printers, err := h.printerSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, printers)
}

// GetPrinterStatus handles GET /v1/printers/:id/status
func (h *Handlers) GetPrinterStatus(c *gin.Context) {
	printerID := c.Param("id")

	status, err := h.printerSvc.Status(printerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// SubmitPrintJob handles POST /v1/print
func (h *Handlers) SubmitPrintJob(c *gin.Context) {
	var req print.PrintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate print type
	switch req.Type {
	case "pdf", "raw", "image":
		// Valid types
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid print type. Supported: pdf, raw, image",
		})
		return
	}

	// Submit to queue
	resp, err := h.printQueue.Submit(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// GetJobStatus handles GET /v1/jobs/:id
func (h *Handlers) GetJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	entry, ok := h.printQueue.GetJob(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, entry.Result)
}

// CancelJob handles POST /v1/jobs/:id/cancel
func (h *Handlers) CancelJob(c *gin.Context) {
	jobID := c.Param("id")

	result, err := h.printQueue.CancelJob(jobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SubmitBatchPrintJob handles POST /v1/print/batch
func (h *Handlers) SubmitBatchPrintJob(c *gin.Context) {
	var req print.BatchPrintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate each job in the batch
	for i, job := range req.Jobs {
		switch job.Type {
		case "pdf", "raw", "image":
			// Valid types
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "Invalid print type. Supported: pdf, raw, image",
				"job_index": i,
			})
			return
		}
	}

	// Submit batch
	resp, err := h.printQueue.SubmitBatch(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// ListJobs handles GET /v1/jobs
func (h *Handlers) ListJobs(c *gin.Context) {
	// Check if history is requested
	historyStr := c.DefaultQuery("history", "false")
	if historyStr == "true" {
		history := h.printQueue.GetHistory(50)
		results := make([]*printer.JobResult, len(history))
		for i, entry := range history {
			results[i] = entry.Result
		}
		c.JSON(http.StatusOK, results)
		return
	}

	// Return active jobs
	jobs := h.printQueue.GetJobs()
	results := make([]*printer.JobResult, len(jobs))
	for i, entry := range jobs {
		results[i] = entry.Result
	}
	c.JSON(http.StatusOK, results)
}

// GetStats handles GET /v1/stats
func (h *Handlers) GetStats(c *gin.Context) {
	stats := h.printQueue.Stats()
	c.JSON(http.StatusOK, stats)
}

// GetDefaultPrinter handles GET /v1/printers/default
func (h *Handlers) GetDefaultPrinter(c *gin.Context) {
	name, err := h.printerSvc.GetDefault()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No default printer set",
		})
		return
	}

	status, err := h.printerSvc.Status(name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"name": name,
		})
		return
	}

	c.JSON(http.StatusOK, status)
}
