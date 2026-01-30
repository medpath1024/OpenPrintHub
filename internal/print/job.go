package print

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// JobStore manages print job storage and retrieval
type JobStore struct {
	mu         sync.RWMutex
	jobs       map[string]*JobEntry
	history    []*JobEntry
	maxHistory int
}

// JobEntry represents a stored print job with its result
type JobEntry struct {
	Job    *printer.PrintJob  `json:"job"`
	Result *printer.JobResult `json:"result"`
}

// NewJobStore creates a new job store
func NewJobStore(maxHistory int) *JobStore {
	if maxHistory <= 0 {
		maxHistory = 100
	}
	return &JobStore{
		jobs:       make(map[string]*JobEntry),
		history:    make([]*JobEntry, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// CreateJob creates a new print job from a request
func (s *JobStore) CreateJob(req *PrintRequest) (*printer.PrintJob, error) {
	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}

	// Generate job ID
	jobID := uuid.New().String()
	now := time.Now()

	// Generate default name if not provided
	jobName := req.Name
	if jobName == "" {
		jobName = fmt.Sprintf("%s-%s", req.Type, now.Format("150405"))
	}

	// Create job
	job := &printer.PrintJob{
		ID:          jobID,
		Name:        jobName,
		PrinterName: req.Printer,
		Type:        printer.PrintJobType(req.Type),
		Data:        data,
		DataBase64:  req.Data,
		Settings:    req.Settings,
		CreatedAt:   now,
	}

	// Apply default settings
	if job.Settings.Copies <= 0 {
		job.Settings.Copies = 1
	}
	if job.Settings.Orientation == "" {
		job.Settings.Orientation = "portrait"
	}
	if job.Settings.DPI <= 0 && job.Type == printer.JobTypeImage {
		job.Settings.DPI = 300
	}

	// Store job
	s.mu.Lock()
	s.jobs[jobID] = &JobEntry{
		Job: job,
		Result: &printer.JobResult{
			JobID:       jobID,
			Name:        jobName,
			Status:      printer.JobStatusQueued,
			PrinterName: req.Printer,
			CreatedAt:   now,
		},
	}
	s.mu.Unlock()

	return job, nil
}

// CancelJob cancels a job if it's still in a cancellable state
func (s *JobStore) CancelJob(jobID string) (*printer.JobResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Check if job can be canceled
	switch entry.Result.Status {
	case printer.JobStatusQueued:
		// Can cancel queued jobs
		entry.Result.Status = printer.JobStatusCanceled
		entry.Result.Message = "Canceled by user"
		entry.Result.CompletedAt = time.Now()
		s.addToHistory(entry)
		// Return a copy to avoid race conditions
		result := *entry.Result
		return &result, nil
	case printer.JobStatusProcessing, printer.JobStatusPrinting:
		// Job is already being processed, mark for cancellation
		entry.Result.Status = printer.JobStatusCanceled
		entry.Result.Message = "Cancellation requested"
		entry.Result.CompletedAt = time.Now()
		s.addToHistory(entry)
		// Return a copy to avoid race conditions
		result := *entry.Result
		return &result, nil
	case printer.JobStatusCompleted, printer.JobStatusFailed, printer.JobStatusCanceled:
		return nil, fmt.Errorf("job already in terminal state: %s", entry.Result.Status)
	default:
		return nil, fmt.Errorf("unknown job status: %s", entry.Result.Status)
	}
}

// GetJob returns a job by ID
func (s *JobStore) GetJob(jobID string) (*JobEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.jobs[jobID]
	if !ok {
		return nil, false
	}
	// Return deep copy to avoid race conditions
	resultCopy := *entry.Result
	return &JobEntry{
		Job:    entry.Job,
		Result: &resultCopy,
	}, true
}

// UpdateJobStatus updates the status of a job
func (s *JobStore) UpdateJobStatus(jobID string, status printer.JobStatus, message string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.jobs[jobID]
	if !ok {
		return
	}

	entry.Result.Status = status
	entry.Result.Message = message
	if err != nil {
		entry.Result.Error = err.Error()
	}

	switch status {
	case printer.JobStatusProcessing, printer.JobStatusPrinting:
		if entry.Result.StartedAt.IsZero() {
			entry.Result.StartedAt = time.Now()
		}
	case printer.JobStatusCompleted, printer.JobStatusFailed, printer.JobStatusCanceled:
		entry.Result.CompletedAt = time.Now()
		// Add to history
		s.addToHistory(entry)
	}
}

// addToHistory adds a completed job to history (caller must hold lock)
func (s *JobStore) addToHistory(entry *JobEntry) {
	s.history = append(s.history, entry)
	// Trim history if needed
	if len(s.history) > s.maxHistory {
		s.history = s.history[len(s.history)-s.maxHistory:]
	}
}

// GetHistory returns the job history
func (s *JobStore) GetHistory(limit int) []*JobEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	// Return most recent first with deep copies to avoid race conditions
	result := make([]*JobEntry, limit)
	for i := 0; i < limit; i++ {
		entry := s.history[len(s.history)-1-i]
		resultCopy := *entry.Result
		result[i] = &JobEntry{
			Job:    entry.Job,
			Result: &resultCopy,
		}
	}
	return result
}

// GetAllJobs returns all active jobs
func (s *JobStore) GetAllJobs() []*JobEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*JobEntry, 0, len(s.jobs))
	for _, entry := range s.jobs {
		// Return deep copy to avoid race conditions
		resultCopy := *entry.Result
		result = append(result, &JobEntry{
			Job:    entry.Job,
			Result: &resultCopy,
		})
	}
	return result
}

// PrintRequest represents an incoming print request
type PrintRequest struct {
	Printer  string                `json:"printer" binding:"required"`
	Type     string                `json:"type" binding:"required"`
	Data     string                `json:"data" binding:"required"`
	Name     string                `json:"name,omitempty"` // Job name for identification
	Settings printer.PrintSettings `json:"settings"`
}

// BatchPrintRequest represents a batch print request
type BatchPrintRequest struct {
	Printer string              `json:"printer" binding:"required"`
	Jobs    []BatchPrintJobItem `json:"jobs" binding:"required,min=1,max=100"`
}

// BatchPrintJobItem represents a single job in a batch request
type BatchPrintJobItem struct {
	Type     string                `json:"type" binding:"required"`
	Data     string                `json:"data" binding:"required"`
	Name     string                `json:"name,omitempty"`
	Settings printer.PrintSettings `json:"settings"`
}

// BatchPrintResponse represents the response to a batch print request
type BatchPrintResponse struct {
	BatchID string          `json:"batch_id"`
	Jobs    []PrintResponse `json:"jobs"`
	Total   int             `json:"total"`
	Queued  int             `json:"queued"`
	Failed  int             `json:"failed"`
}

// PrintResponse represents the response to a print request
type PrintResponse struct {
	JobID   string            `json:"job_id"`
	Status  printer.JobStatus `json:"status"`
	Message string            `json:"message,omitempty"`
}
