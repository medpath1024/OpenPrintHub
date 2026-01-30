package print

import (
	"encoding/base64"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// MockPrinterService is a mock implementation for testing
type MockPrinterService struct {
	mu            sync.RWMutex
	PrintError    error
	PrintRawError error
	PrintCalls    int32
	PrintRawCalls int32
	PrintDelay    time.Duration
}

func NewMockPrinterService() *MockPrinterService {
	return &MockPrinterService{}
}

func (m *MockPrinterService) List() ([]printer.PrinterInfo, error) {
	return []printer.PrinterInfo{
		{ID: "printer1", Name: "Test Printer", Status: printer.StatusReady, IsDefault: true},
	}, nil
}

func (m *MockPrinterService) GetDefault() (string, error) {
	return "printer1", nil
}

func (m *MockPrinterService) Status(printerName string) (*printer.PrinterInfo, error) {
	return &printer.PrinterInfo{
		ID:     printerName,
		Name:   printerName,
		Status: printer.StatusReady,
	}, nil
}

func (m *MockPrinterService) Print(job *printer.PrintJob) error {
	atomic.AddInt32(&m.PrintCalls, 1)
	if m.PrintDelay > 0 {
		time.Sleep(m.PrintDelay)
	}
	m.mu.RLock()
	err := m.PrintError
	m.mu.RUnlock()
	return err
}

func (m *MockPrinterService) PrintRaw(printerName string, data []byte) error {
	atomic.AddInt32(&m.PrintRawCalls, 1)
	if m.PrintDelay > 0 {
		time.Sleep(m.PrintDelay)
	}
	m.mu.RLock()
	err := m.PrintRawError
	m.mu.RUnlock()
	return err
}

func (m *MockPrinterService) SetPrintError(err error) {
	m.mu.Lock()
	m.PrintError = err
	m.mu.Unlock()
}

func (m *MockPrinterService) SetPrintRawError(err error) {
	m.mu.Lock()
	m.PrintRawError = err
	m.mu.Unlock()
}

func (m *MockPrinterService) GetPrintCalls() int {
	return int(atomic.LoadInt32(&m.PrintCalls))
}

func (m *MockPrinterService) GetPrintRawCalls() int {
	return int(atomic.LoadInt32(&m.PrintRawCalls))
}

func TestNewQueue(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	if q == nil {
		t.Fatal("NewQueue returned nil")
	}
	if q.printerSvc == nil {
		t.Error("printerSvc is nil")
	}
	if q.jobStore == nil {
		t.Error("jobStore is nil")
	}
	if q.jobs == nil {
		t.Error("jobs channel is nil")
	}
}

func TestQueue_Submit_Success(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	resp, err := q.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if resp.JobID == "" {
		t.Error("Response JobID is empty")
	}
	if resp.Status != printer.JobStatusQueued {
		t.Errorf("Expected status queued, got %s", resp.Status)
	}

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify print was called
	if mockSvc.GetPrintCalls() != 1 {
		t.Errorf("Expected 1 print call, got %d", mockSvc.GetPrintCalls())
	}
}

func TestQueue_Submit_RawType(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte{0x1B, 0x40, 0x48, 0x65, 0x6C, 0x6C, 0x6F} // ESC/POS data
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "raw",
		Data:    b64Data,
	}

	resp, err := q.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if resp.Status != printer.JobStatusQueued {
		t.Errorf("Expected status queued, got %s", resp.Status)
	}

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify PrintRaw was called for raw type
	if mockSvc.GetPrintRawCalls() != 1 {
		t.Errorf("Expected 1 PrintRaw call, got %d", mockSvc.GetPrintRawCalls())
	}
}

func TestQueue_Submit_InvalidBase64(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    "invalid-base64!!!",
	}

	_, err := q.Submit(req)
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestQueue_GetJob(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	resp, _ := q.Submit(req)

	// Get existing job
	entry, ok := q.GetJob(resp.JobID)
	if !ok {
		t.Error("Expected to find job")
	}
	if entry.Result.JobID != resp.JobID {
		t.Errorf("Expected JobID %s, got %s", resp.JobID, entry.Result.JobID)
	}

	// Get non-existent job
	_, ok = q.GetJob("non-existent")
	if ok {
		t.Error("Expected not to find non-existent job")
	}
}

func TestQueue_GetJobs(t *testing.T) {
	mockSvc := NewMockPrinterService()
	mockSvc.PrintDelay = 100 * time.Millisecond // Slow down processing
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Submit multiple jobs
	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "printer1",
			Type:    "pdf",
			Data:    b64Data,
		}
		q.Submit(req)
	}

	// Get all jobs
	jobs := q.GetJobs()
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestQueue_GetHistory(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Submit and wait for completion
	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "printer1",
			Type:    "pdf",
			Data:    b64Data,
		}
		q.Submit(req)
	}

	// Wait for jobs to complete
	time.Sleep(500 * time.Millisecond)

	history := q.GetHistory(10)
	if len(history) != 3 {
		t.Errorf("Expected 3 jobs in history, got %d", len(history))
	}

	// Verify all jobs are completed
	for _, entry := range history {
		if entry.Result.Status != printer.JobStatusCompleted {
			t.Errorf("Expected completed status, got %s", entry.Result.Status)
		}
	}
}

func TestQueue_PrintFailure(t *testing.T) {
	mockSvc := NewMockPrinterService()
	mockSvc.PrintError = errors.New("print failed")
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	resp, _ := q.Submit(req)

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	entry, _ := q.GetJob(resp.JobID)
	if entry.Result.Status != printer.JobStatusFailed {
		t.Errorf("Expected failed status, got %s", entry.Result.Status)
	}
}

func TestQueue_OnStatusChange(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	statusChanges := make([]printer.JobStatus, 0)
	var mu sync.Mutex

	q.OnStatusChange(func(jobID string, status printer.JobStatus, message string) {
		mu.Lock()
		statusChanges = append(statusChanges, status)
		mu.Unlock()
	})

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	q.Submit(req)

	// Wait for job to complete
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have received status changes: queued, processing, printing, completed
	if len(statusChanges) < 3 {
		t.Errorf("Expected at least 3 status changes, got %d", len(statusChanges))
	}

	// Verify status progression
	expectedStatuses := []printer.JobStatus{
		printer.JobStatusQueued,
		printer.JobStatusProcessing,
		printer.JobStatusPrinting,
		printer.JobStatusCompleted,
	}

	for i, expected := range expectedStatuses {
		if i < len(statusChanges) && statusChanges[i] != expected {
			t.Errorf("Status change %d: expected %s, got %s", i, expected, statusChanges[i])
		}
	}
}

func TestQueue_MultipleCallbacks(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	var count1, count2 int32

	q.OnStatusChange(func(jobID string, status printer.JobStatus, message string) {
		atomic.AddInt32(&count1, 1)
	})

	q.OnStatusChange(func(jobID string, status printer.JobStatus, message string) {
		atomic.AddInt32(&count2, 1)
	})

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	q.Submit(req)

	// Wait for job to complete
	time.Sleep(500 * time.Millisecond)

	// Both callbacks should have been called the same number of times
	if atomic.LoadInt32(&count1) != atomic.LoadInt32(&count2) {
		t.Errorf("Callback counts don't match: %d vs %d", count1, count2)
	}
}

func TestQueue_Stop(t *testing.T) {
	mockSvc := NewMockPrinterService()
	mockSvc.PrintDelay = 50 * time.Millisecond
	q := NewQueue(mockSvc)

	// Submit some jobs
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "printer1",
			Type:    "pdf",
			Data:    b64Data,
		}
		q.Submit(req)
	}

	// Stop should complete without hanging
	done := make(chan bool)
	go func() {
		q.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stop timed out")
	}

	// Second stop should be a no-op
	q.Stop()
}

func TestQueue_Stats(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	// Initially stats should be zero
	stats := q.Stats()
	if stats.QueuedJobs != 0 {
		t.Errorf("Expected 0 queued jobs initially, got %d", stats.QueuedJobs)
	}
	if stats.CompletedJobs != 0 {
		t.Errorf("Expected 0 completed jobs initially, got %d", stats.CompletedJobs)
	}
	if stats.FailedJobs != 0 {
		t.Errorf("Expected 0 failed jobs initially, got %d", stats.FailedJobs)
	}

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Submit some successful jobs
	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "printer1",
			Type:    "pdf",
			Data:    b64Data,
		}
		q.Submit(req)
	}

	// Wait for jobs to complete
	time.Sleep(500 * time.Millisecond)

	stats = q.Stats()
	if stats.CompletedJobs != 3 {
		t.Errorf("Expected 3 completed jobs, got %d", stats.CompletedJobs)
	}

	// Submit a failing job
	mockSvc.SetPrintError(errors.New("fail"))
	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}
	q.Submit(req)

	// Wait for job to fail
	time.Sleep(500 * time.Millisecond)

	stats = q.Stats()
	if stats.FailedJobs != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs)
	}
}

func TestQueueStats_Struct(t *testing.T) {
	stats := QueueStats{
		QueuedJobs:    5,
		CompletedJobs: 100,
		FailedJobs:    3,
	}

	if stats.QueuedJobs != 5 {
		t.Errorf("Expected QueuedJobs 5, got %d", stats.QueuedJobs)
	}
	if stats.CompletedJobs != 100 {
		t.Errorf("Expected CompletedJobs 100, got %d", stats.CompletedJobs)
	}
	if stats.FailedJobs != 3 {
		t.Errorf("Expected FailedJobs 3, got %d", stats.FailedJobs)
	}
}

func TestQueue_ConcurrentSubmissions(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	var wg sync.WaitGroup
	numJobs := 20

	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &PrintRequest{
				Printer: "printer1",
				Type:    "pdf",
				Data:    b64Data,
			}
			resp, err := q.Submit(req)
			if err != nil {
				t.Errorf("Submit failed: %v", err)
			}
			if resp.JobID == "" {
				t.Error("Empty job ID")
			}
		}()
	}

	wg.Wait()

	// Wait for all jobs to complete with retries
	var stats QueueStats
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		time.Sleep(200 * time.Millisecond)
		stats = q.Stats()
		if stats.CompletedJobs >= numJobs {
			break
		}
	}

	if stats.CompletedJobs != numJobs {
		t.Errorf("Expected %d completed jobs, got %d", numJobs, stats.CompletedJobs)
	}
}

func BenchmarkQueue_Submit(b *testing.B) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Submit(req)
	}
}

func TestQueue_CancelJob(t *testing.T) {
	mockSvc := NewMockPrinterService()
	mockSvc.PrintDelay = 500 * time.Millisecond // Slow processing
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Submit multiple jobs to queue
	var jobIDs []string
	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "printer1",
			Type:    "pdf",
			Data:    b64Data,
			Name:    "test-job-" + string(rune('A'+i)),
		}
		resp, _ := q.Submit(req)
		jobIDs = append(jobIDs, resp.JobID)
	}

	// Try to cancel a queued job (should succeed)
	result, err := q.CancelJob(jobIDs[2])
	if err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}
	if result.Status != printer.JobStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", result.Status)
	}
}

func TestQueue_CancelJob_NotFound(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	_, err := q.CancelJob("nonexistent-job-id")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestQueue_SubmitBatch(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	batchReq := &BatchPrintRequest{
		Printer: "printer1",
		Jobs: []BatchPrintJobItem{
			{Type: "pdf", Data: b64Data, Name: "doc1.pdf"},
			{Type: "pdf", Data: b64Data, Name: "doc2.pdf"},
			{Type: "image", Data: b64Data},
		},
	}

	resp, err := q.SubmitBatch(batchReq)
	if err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("Expected total 3, got %d", resp.Total)
	}
	if resp.Queued != 3 {
		t.Errorf("Expected queued 3, got %d", resp.Queued)
	}
	if resp.Failed != 0 {
		t.Errorf("Expected failed 0, got %d", resp.Failed)
	}
	if len(resp.Jobs) != 3 {
		t.Errorf("Expected 3 job responses, got %d", len(resp.Jobs))
	}
	if resp.BatchID == "" {
		t.Error("Expected batch ID to be set")
	}

	// Verify all jobs have IDs
	for i, job := range resp.Jobs {
		if job.JobID == "" {
			t.Errorf("Job %d has empty ID", i)
		}
		if job.Status != printer.JobStatusQueued {
			t.Errorf("Job %d has unexpected status: %s", i, job.Status)
		}
	}

	// Wait for jobs to complete
	time.Sleep(500 * time.Millisecond)

	stats := q.Stats()
	if stats.CompletedJobs < 3 {
		t.Errorf("Expected at least 3 completed jobs, got %d", stats.CompletedJobs)
	}
}

func TestQueue_SubmitBatch_WithInvalidData(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	batchReq := &BatchPrintRequest{
		Printer: "printer1",
		Jobs: []BatchPrintJobItem{
			{Type: "pdf", Data: b64Data, Name: "doc1.pdf"},
			{Type: "pdf", Data: "invalid-base64!!!", Name: "doc2.pdf"},
		},
	}

	resp, err := q.SubmitBatch(batchReq)
	if err != nil {
		t.Fatalf("SubmitBatch returned error: %v", err)
	}

	// First job should succeed, second should fail
	if resp.Total != 2 {
		t.Errorf("Expected total 2, got %d", resp.Total)
	}
	if resp.Queued != 1 {
		t.Errorf("Expected queued 1, got %d", resp.Queued)
	}
	if resp.Failed != 1 {
		t.Errorf("Expected failed 1, got %d", resp.Failed)
	}
}

func TestQueue_Submit_WithName(t *testing.T) {
	mockSvc := NewMockPrinterService()
	q := NewQueue(mockSvc)
	defer q.Stop()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "printer1",
		Type:    "pdf",
		Data:    b64Data,
		Name:    "my-important-document.pdf",
	}

	resp, err := q.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Verify name is stored
	entry, ok := q.GetJob(resp.JobID)
	if !ok {
		t.Fatal("Job not found")
	}

	if entry.Job.Name != "my-important-document.pdf" {
		t.Errorf("Expected name 'my-important-document.pdf', got '%s'", entry.Job.Name)
	}
	if entry.Result.Name != "my-important-document.pdf" {
		t.Errorf("Expected result name 'my-important-document.pdf', got '%s'", entry.Result.Name)
	}
}
