package print

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// StatusCallback is called when a job status changes
type StatusCallback func(jobID string, status printer.JobStatus, message string)

// Queue manages the print job queue
type Queue struct {
	printerSvc printer.Service
	jobStore   *JobStore
	jobs       chan *printer.PrintJob
	callbacks  []StatusCallback
	callbackMu sync.RWMutex
	wg         sync.WaitGroup
	stopCh     chan struct{}
	stopped    bool
	mu         sync.Mutex
}

// NewQueue creates a new print queue
func NewQueue(printerSvc printer.Service) *Queue {
	q := &Queue{
		printerSvc: printerSvc,
		jobStore:   NewJobStore(100),
		jobs:       make(chan *printer.PrintJob, 100),
		callbacks:  make([]StatusCallback, 0),
		stopCh:     make(chan struct{}),
	}

	// Start worker
	q.wg.Add(1)
	go q.worker()

	return q
}

// Submit adds a new print job to the queue
func (q *Queue) Submit(req *PrintRequest) (*PrintResponse, error) {
	job, err := q.jobStore.CreateJob(req)
	if err != nil {
		return nil, err
	}

	// Add to queue
	select {
	case q.jobs <- job:
		q.notifyStatus(job.ID, printer.JobStatusQueued, "Job queued")
	default:
		q.jobStore.UpdateJobStatus(job.ID, printer.JobStatusFailed, "Queue full", nil)
		return &PrintResponse{
			JobID:   job.ID,
			Status:  printer.JobStatusFailed,
			Message: "Print queue is full",
		}, nil
	}

	return &PrintResponse{
		JobID:  job.ID,
		Status: printer.JobStatusQueued,
	}, nil
}

// GetJob returns a job by ID
func (q *Queue) GetJob(jobID string) (*JobEntry, bool) {
	return q.jobStore.GetJob(jobID)
}

// GetJobs returns all jobs
func (q *Queue) GetJobs() []*JobEntry {
	return q.jobStore.GetAllJobs()
}

// GetHistory returns completed jobs
func (q *Queue) GetHistory(limit int) []*JobEntry {
	return q.jobStore.GetHistory(limit)
}

// CancelJob cancels a job by ID
func (q *Queue) CancelJob(jobID string) (*printer.JobResult, error) {
	result, err := q.jobStore.CancelJob(jobID)
	if err != nil {
		return nil, err
	}
	q.notifyStatus(jobID, printer.JobStatusCanceled, result.Message)
	return result, nil
}

// SubmitBatch adds multiple print jobs to the queue
func (q *Queue) SubmitBatch(req *BatchPrintRequest) (*BatchPrintResponse, error) {
	batchID := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	resp := &BatchPrintResponse{
		BatchID: batchID,
		Jobs:    make([]PrintResponse, 0, len(req.Jobs)),
		Total:   len(req.Jobs),
	}

	for i, jobItem := range req.Jobs {
		// Create print request from batch item
		printReq := &PrintRequest{
			Printer:  req.Printer,
			Type:     jobItem.Type,
			Data:     jobItem.Data,
			Name:     jobItem.Name,
			Settings: jobItem.Settings,
		}

		// Generate default name if not provided
		if printReq.Name == "" {
			printReq.Name = fmt.Sprintf("%s-%s-%d", batchID, jobItem.Type, i+1)
		}

		// Submit job
		jobResp, err := q.Submit(printReq)
		if err != nil {
			resp.Jobs = append(resp.Jobs, PrintResponse{
				Status:  printer.JobStatusFailed,
				Message: err.Error(),
			})
			resp.Failed++
		} else {
			resp.Jobs = append(resp.Jobs, *jobResp)
			if jobResp.Status == printer.JobStatusQueued {
				resp.Queued++
			} else {
				resp.Failed++
			}
		}
	}

	return resp, nil
}

// OnStatusChange registers a callback for job status changes
func (q *Queue) OnStatusChange(callback StatusCallback) {
	q.callbackMu.Lock()
	defer q.callbackMu.Unlock()
	q.callbacks = append(q.callbacks, callback)
}

// notifyStatus notifies all registered callbacks of a status change
func (q *Queue) notifyStatus(jobID string, status printer.JobStatus, message string) {
	q.callbackMu.RLock()
	callbacks := q.callbacks
	q.callbackMu.RUnlock()

	for _, cb := range callbacks {
		go cb(jobID, status, message)
	}
}

// worker processes jobs from the queue
func (q *Queue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopCh:
			return
		case job := <-q.jobs:
			q.processJob(job)
		}
	}
}

// processJob processes a single print job
func (q *Queue) processJob(job *printer.PrintJob) {
	// Update status to processing
	q.jobStore.UpdateJobStatus(job.ID, printer.JobStatusProcessing, "Processing job", nil)
	q.notifyStatus(job.ID, printer.JobStatusProcessing, "Processing job")

	// Small delay to allow status update to propagate
	time.Sleep(100 * time.Millisecond)

	// Update status to printing
	q.jobStore.UpdateJobStatus(job.ID, printer.JobStatusPrinting, "Sending to printer", nil)
	q.notifyStatus(job.ID, printer.JobStatusPrinting, "Sending to printer")

	// Send to printer
	var err error
	if job.Type == printer.JobTypeRaw {
		err = q.printerSvc.PrintRaw(job.PrinterName, job.Data)
	} else {
		err = q.printerSvc.Print(job)
	}

	if err != nil {
		log.Printf("Print job %s failed: %v", job.ID, err)
		q.jobStore.UpdateJobStatus(job.ID, printer.JobStatusFailed, "Print failed", err)
		q.notifyStatus(job.ID, printer.JobStatusFailed, err.Error())
		return
	}

	// Success
	q.jobStore.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Print completed", nil)
	q.notifyStatus(job.ID, printer.JobStatusCompleted, "Print completed")
	log.Printf("Print job %s completed successfully", job.ID)
}

// Stop stops the queue worker
func (q *Queue) Stop() {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		return
	}
	q.stopped = true
	q.mu.Unlock()

	close(q.stopCh)
	q.wg.Wait()
}

// QueueStats returns queue statistics
type QueueStats struct {
	QueuedJobs    int `json:"queued_jobs"`
	CompletedJobs int `json:"completed_jobs"`
	FailedJobs    int `json:"failed_jobs"`
}

// Stats returns current queue statistics
func (q *Queue) Stats() QueueStats {
	jobs := q.jobStore.GetAllJobs()
	history := q.jobStore.GetHistory(0)

	stats := QueueStats{}
	for _, entry := range jobs {
		switch entry.Result.Status {
		case printer.JobStatusQueued, printer.JobStatusProcessing, printer.JobStatusPrinting:
			stats.QueuedJobs++
		}
	}

	for _, entry := range history {
		switch entry.Result.Status {
		case printer.JobStatusCompleted:
			stats.CompletedJobs++
		case printer.JobStatusFailed:
			stats.FailedJobs++
		}
	}

	return stats
}
