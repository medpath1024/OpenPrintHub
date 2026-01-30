package print

import (
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

func TestNewJobStore(t *testing.T) {
	tests := []struct {
		name        string
		maxHistory  int
		expectedMax int
	}{
		{"default max history", 0, 100},
		{"negative max history", -1, 100},
		{"custom max history", 50, 50},
		{"large max history", 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewJobStore(tt.maxHistory)
			if store == nil {
				t.Fatal("NewJobStore returned nil")
			}
			if store.maxHistory != tt.expectedMax {
				t.Errorf("Expected maxHistory %d, got %d", tt.expectedMax, store.maxHistory)
			}
			if store.jobs == nil {
				t.Error("jobs map is nil")
			}
			if store.history == nil {
				t.Error("history slice is nil")
			}
		})
	}
}

func TestJobStore_CreateJob(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
		Settings: printer.PrintSettings{
			Copies:      2,
			Orientation: "landscape",
		},
	}

	job, err := store.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Verify job properties
	if job.ID == "" {
		t.Error("Job ID is empty")
	}
	if job.PrinterName != "Test Printer" {
		t.Errorf("Expected PrinterName 'Test Printer', got %s", job.PrinterName)
	}
	if job.Type != printer.PrintJobType("pdf") {
		t.Errorf("Expected Type 'pdf', got %s", job.Type)
	}
	if string(job.Data) != "test print data" {
		t.Errorf("Expected Data 'test print data', got %s", string(job.Data))
	}
	if job.Settings.Copies != 2 {
		t.Errorf("Expected Copies 2, got %d", job.Settings.Copies)
	}
	if job.Settings.Orientation != "landscape" {
		t.Errorf("Expected Orientation 'landscape', got %s", job.Settings.Orientation)
	}
	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// Verify job is stored
	entry, ok := store.GetJob(job.ID)
	if !ok {
		t.Error("Job not found in store")
	}
	if entry.Job.ID != job.ID {
		t.Errorf("Expected Job ID %s, got %s", job.ID, entry.Job.ID)
	}
	if entry.Result.Status != printer.JobStatusQueued {
		t.Errorf("Expected Status queued, got %s", entry.Result.Status)
	}
}

func TestJobStore_CreateJob_DefaultSettings(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer:  "Test Printer",
		Type:     "pdf",
		Data:     b64Data,
		Settings: printer.PrintSettings{}, // Empty settings
	}

	job, err := store.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Verify default settings are applied
	if job.Settings.Copies != 1 {
		t.Errorf("Expected default Copies 1, got %d", job.Settings.Copies)
	}
	if job.Settings.Orientation != "portrait" {
		t.Errorf("Expected default Orientation 'portrait', got %s", job.Settings.Orientation)
	}
}

func TestJobStore_CreateJob_InvalidBase64(t *testing.T) {
	store := NewJobStore(100)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    "invalid-base64!!!",
	}

	_, err := store.CreateJob(req)
	if err == nil {
		t.Error("Expected error for invalid base64 data")
	}
}

func TestJobStore_GetJob(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)

	// Test getting existing job
	entry, ok := store.GetJob(job.ID)
	if !ok {
		t.Error("Expected to find job")
	}
	if entry.Job.ID != job.ID {
		t.Errorf("Expected Job ID %s, got %s", job.ID, entry.Job.ID)
	}

	// Test getting non-existent job
	_, ok = store.GetJob("non-existent-id")
	if ok {
		t.Error("Expected not to find non-existent job")
	}
}

func TestJobStore_UpdateJobStatus(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)

	// Update to processing
	store.UpdateJobStatus(job.ID, printer.JobStatusProcessing, "Processing", nil)
	entry, _ := store.GetJob(job.ID)
	if entry.Result.Status != printer.JobStatusProcessing {
		t.Errorf("Expected status processing, got %s", entry.Result.Status)
	}
	if entry.Result.Message != "Processing" {
		t.Errorf("Expected message 'Processing', got %s", entry.Result.Message)
	}
	if entry.Result.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set")
	}

	// Update to printing
	store.UpdateJobStatus(job.ID, printer.JobStatusPrinting, "Printing", nil)
	entry, _ = store.GetJob(job.ID)
	if entry.Result.Status != printer.JobStatusPrinting {
		t.Errorf("Expected status printing, got %s", entry.Result.Status)
	}

	// Update to completed
	store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Completed", nil)
	entry, _ = store.GetJob(job.ID)
	if entry.Result.Status != printer.JobStatusCompleted {
		t.Errorf("Expected status completed, got %s", entry.Result.Status)
	}
	if entry.Result.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}

	// Verify job was added to history
	history := store.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("Expected 1 job in history, got %d", len(history))
	}
}

func TestJobStore_UpdateJobStatus_WithError(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)

	testErr := &testError{msg: "print failed"}
	store.UpdateJobStatus(job.ID, printer.JobStatusFailed, "Failed", testErr)

	entry, _ := store.GetJob(job.ID)
	if entry.Result.Status != printer.JobStatusFailed {
		t.Errorf("Expected status failed, got %s", entry.Result.Status)
	}
	if entry.Result.Error != "print failed" {
		t.Errorf("Expected error 'print failed', got %s", entry.Result.Error)
	}
}

func TestJobStore_UpdateJobStatus_NonExistent(t *testing.T) {
	store := NewJobStore(100)

	// Should not panic for non-existent job
	store.UpdateJobStatus("non-existent", printer.JobStatusCompleted, "Done", nil)
}

func TestJobStore_GetHistory(t *testing.T) {
	store := NewJobStore(10)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Create and complete multiple jobs
	for i := 0; i < 5; i++ {
		req := &PrintRequest{
			Printer: "Test Printer",
			Type:    "pdf",
			Data:    b64Data,
		}
		job, _ := store.CreateJob(req)
		store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Done", nil)
	}

	// Test getting all history
	history := store.GetHistory(0)
	if len(history) != 5 {
		t.Errorf("Expected 5 jobs in history, got %d", len(history))
	}

	// Test getting limited history
	history = store.GetHistory(3)
	if len(history) != 3 {
		t.Errorf("Expected 3 jobs in history, got %d", len(history))
	}

	// Test history is in reverse order (newest first)
	// The last created job should be first in history
}

func TestJobStore_GetHistory_MaxLimit(t *testing.T) {
	store := NewJobStore(5) // Max 5 items in history

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Create more jobs than max history
	for i := 0; i < 10; i++ {
		req := &PrintRequest{
			Printer: "Test Printer",
			Type:    "pdf",
			Data:    b64Data,
		}
		job, _ := store.CreateJob(req)
		store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Done", nil)
	}

	// History should be trimmed to max
	history := store.GetHistory(0)
	if len(history) != 5 {
		t.Errorf("Expected 5 jobs in history (max), got %d", len(history))
	}
}

func TestJobStore_GetAllJobs(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		req := &PrintRequest{
			Printer: "Test Printer",
			Type:    "pdf",
			Data:    b64Data,
		}
		store.CreateJob(req)
	}

	jobs := store.GetAllJobs()
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestJobStore_Concurrency(t *testing.T) {
	store := NewJobStore(100)
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrently create jobs
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &PrintRequest{
				Printer: "Test Printer",
				Type:    "pdf",
				Data:    b64Data,
			}
			job, err := store.CreateJob(req)
			if err != nil {
				t.Errorf("CreateJob failed: %v", err)
				return
			}
			store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Done", nil)
		}()
	}

	wg.Wait()

	// Verify all jobs were created and completed
	history := store.GetHistory(0)
	if len(history) != numGoroutines {
		t.Errorf("Expected %d jobs in history, got %d", numGoroutines, len(history))
	}
}

func TestPrintRequest_Struct(t *testing.T) {
	req := PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    "dGVzdA==",
		Settings: printer.PrintSettings{
			Copies: 2,
		},
	}

	if req.Printer != "Test Printer" {
		t.Errorf("Expected Printer 'Test Printer', got %s", req.Printer)
	}
	if req.Type != "pdf" {
		t.Errorf("Expected Type 'pdf', got %s", req.Type)
	}
	if req.Data != "dGVzdA==" {
		t.Errorf("Expected Data 'dGVzdA==', got %s", req.Data)
	}
	if req.Settings.Copies != 2 {
		t.Errorf("Expected Copies 2, got %d", req.Settings.Copies)
	}
}

func TestPrintResponse_Struct(t *testing.T) {
	resp := PrintResponse{
		JobID:   "job-123",
		Status:  printer.JobStatusQueued,
		Message: "Job queued",
	}

	if resp.JobID != "job-123" {
		t.Errorf("Expected JobID 'job-123', got %s", resp.JobID)
	}
	if resp.Status != printer.JobStatusQueued {
		t.Errorf("Expected Status queued, got %s", resp.Status)
	}
	if resp.Message != "Job queued" {
		t.Errorf("Expected Message 'Job queued', got %s", resp.Message)
	}
}

func TestJobStore_CreateJob_WithName(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
		Name:    "my-document.pdf",
	}

	job, err := store.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.Name != "my-document.pdf" {
		t.Errorf("Expected name 'my-document.pdf', got '%s'", job.Name)
	}

	// Verify name in result
	entry, _ := store.GetJob(job.ID)
	if entry.Result.Name != "my-document.pdf" {
		t.Errorf("Expected result name 'my-document.pdf', got '%s'", entry.Result.Name)
	}
}

func TestJobStore_CreateJob_AutoName(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
		// No name provided
	}

	job, err := store.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Should auto-generate a name starting with type
	if job.Name == "" {
		t.Error("Expected auto-generated name, got empty string")
	}
	if len(job.Name) < 3 {
		t.Errorf("Expected meaningful auto-generated name, got '%s'", job.Name)
	}
}

func TestJobStore_CreateJob_ImageDPI(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer:  "Test Printer",
		Type:     "image",
		Data:     b64Data,
		Settings: printer.PrintSettings{}, // Empty settings
	}

	job, err := store.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Should have default DPI for image type
	if job.Settings.DPI != 300 {
		t.Errorf("Expected default DPI 300 for image, got %d", job.Settings.DPI)
	}
}

func TestJobStore_CancelJob(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)

	// Cancel the queued job
	result, err := store.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	if result.Status != printer.JobStatusCanceled {
		t.Errorf("Expected status canceled, got %s", result.Status)
	}
	if result.Message != "Canceled by user" {
		t.Errorf("Expected message 'Canceled by user', got '%s'", result.Message)
	}
	if result.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestJobStore_CancelJob_NotFound(t *testing.T) {
	store := NewJobStore(100)

	_, err := store.CancelJob("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestJobStore_CancelJob_AlreadyCompleted(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)

	// Complete the job first
	store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Done", nil)

	// Try to cancel
	_, err := store.CancelJob(job.ID)
	if err == nil {
		t.Error("Expected error when canceling completed job")
	}
}

func TestBatchPrintRequest_Struct(t *testing.T) {
	req := BatchPrintRequest{
		Printer: "Test Printer",
		Jobs: []BatchPrintJobItem{
			{Type: "pdf", Data: "dGVzdA==", Name: "doc1"},
			{Type: "image", Data: "dGVzdA=="},
		},
	}

	if req.Printer != "Test Printer" {
		t.Errorf("Expected Printer 'Test Printer', got %s", req.Printer)
	}
	if len(req.Jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(req.Jobs))
	}
	if req.Jobs[0].Name != "doc1" {
		t.Errorf("Expected first job name 'doc1', got '%s'", req.Jobs[0].Name)
	}
}

// Helper error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func BenchmarkJobStore_CreateJob(b *testing.B) {
	store := NewJobStore(1000)
	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.CreateJob(req)
	}
}

func BenchmarkJobStore_GetJob(b *testing.B) {
	store := NewJobStore(1000)
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}
	job, _ := store.CreateJob(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetJob(job.ID)
	}
}

func BenchmarkJobStore_UpdateJobStatus(b *testing.B) {
	store := NewJobStore(10000)
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Create jobs first
	jobIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		req := &PrintRequest{
			Printer: "Test Printer",
			Type:    "pdf",
			Data:    b64Data,
		}
		job, _ := store.CreateJob(req)
		jobIDs[i] = job.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateJobStatus(jobIDs[i], printer.JobStatusCompleted, "Done", nil)
	}
}

// Add test for time tracking
func TestJobStore_TimeTracking(t *testing.T) {
	store := NewJobStore(100)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	req := &PrintRequest{
		Printer: "Test Printer",
		Type:    "pdf",
		Data:    b64Data,
	}

	job, _ := store.CreateJob(req)
	entry, _ := store.GetJob(job.ID)

	// Initially, times should be zero except CreatedAt
	if entry.Result.StartedAt.IsZero() == false {
		t.Error("StartedAt should be zero initially")
	}
	if entry.Result.CompletedAt.IsZero() == false {
		t.Error("CompletedAt should be zero initially")
	}

	// Update to processing - should set StartedAt
	beforeStart := time.Now()
	store.UpdateJobStatus(job.ID, printer.JobStatusProcessing, "Processing", nil)
	entry, _ = store.GetJob(job.ID)
	afterStart := time.Now()

	if entry.Result.StartedAt.Before(beforeStart) || entry.Result.StartedAt.After(afterStart) {
		t.Error("StartedAt should be between before and after update")
	}

	// Update to completed - should set CompletedAt
	beforeComplete := time.Now()
	store.UpdateJobStatus(job.ID, printer.JobStatusCompleted, "Done", nil)
	entry, _ = store.GetJob(job.ID)
	afterComplete := time.Now()

	if entry.Result.CompletedAt.Before(beforeComplete) || entry.Result.CompletedAt.After(afterComplete) {
		t.Error("CompletedAt should be between before and after update")
	}
}
