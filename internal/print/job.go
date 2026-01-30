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
	mu      sync.RWMutex
	jobs    map[string]*JobEntry
	history []*JobEntry
	maxHistory int
}

// JobEntry represents a stored print job with its result
type JobEntry struct {
	Job    *printer.PrintJob   `json:"job"`
	Result *printer.JobResult  `json:"result"`
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

	// Create job
	job := &printer.PrintJob{
		ID:          jobID,
		PrinterName: req.Printer,
		Type:        printer.PrintJobType(req.Type),
		Data:        data,
		DataBase64:  req.Data,
		Settings:    req.Settings,
		CreatedAt:   time.Now(),
	}

	// Apply default settings
	if job.Settings.Copies <= 0 {
		job.Settings.Copies = 1
	}
	if job.Settings.Orientation == "" {
		job.Settings.Orientation = "portrait"
	}

	// Store job
	s.mu.Lock()
	s.jobs[jobID] = &JobEntry{
		Job: job,
		Result: &printer.JobResult{
			JobID:       jobID,
			Status:      printer.JobStatusQueued,
			PrinterName: req.Printer,
		},
	}
	s.mu.Unlock()

	return job, nil
}

// GetJob returns a job by ID
func (s *JobStore) GetJob(jobID string) (*JobEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.jobs[jobID]
	return entry, ok
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
	case printer.JobStatusCompleted, printer.JobStatusFailed, printer.JobStatusCancelled:
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

	// Return most recent first
	result := make([]*JobEntry, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.history[len(s.history)-1-i]
	}
	return result
}

// GetAllJobs returns all active jobs
func (s *JobStore) GetAllJobs() []*JobEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*JobEntry, 0, len(s.jobs))
	for _, entry := range s.jobs {
		result = append(result, entry)
	}
	return result
}

// PrintRequest represents an incoming print request
type PrintRequest struct {
	Printer  string                `json:"printer" binding:"required"`
	Type     string                `json:"type" binding:"required"`
	Data     string                `json:"data" binding:"required"`
	Settings printer.PrintSettings `json:"settings"`
}

// PrintResponse represents the response to a print request
type PrintResponse struct {
	JobID   string             `json:"job_id"`
	Status  printer.JobStatus  `json:"status"`
	Message string             `json:"message,omitempty"`
}
