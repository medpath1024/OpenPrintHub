package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockPrinterService for handler tests
type mockPrinterService struct {
	mu             sync.RWMutex
	printers       []printer.PrinterInfo
	defaultPrinter string
	listError      error
	defaultError   error
	statusError    error
	printError     error
	printRawError  error
}

func newMockPrinterService() *mockPrinterService {
	return &mockPrinterService{
		printers: []printer.PrinterInfo{
			{ID: "printer1", Name: "Test Printer 1", Status: printer.StatusReady, IsDefault: true},
			{ID: "printer2", Name: "Test Printer 2", Status: printer.StatusReady, IsDefault: false},
		},
		defaultPrinter: "printer1",
	}
}

func (m *mockPrinterService) List() ([]printer.PrinterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listError != nil {
		return nil, m.listError
	}
	return m.printers, nil
}

func (m *mockPrinterService) GetDefault() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.defaultError != nil {
		return "", m.defaultError
	}
	return m.defaultPrinter, nil
}

func (m *mockPrinterService) Status(printerName string) (*printer.PrinterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.statusError != nil {
		return nil, m.statusError
	}
	for _, p := range m.printers {
		if p.ID == printerName || p.Name == printerName {
			return &p, nil
		}
	}
	return nil, errors.New("printer not found")
}

func (m *mockPrinterService) Print(job *printer.PrintJob) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.printError
}

func (m *mockPrinterService) PrintRaw(printerName string, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.printRawError
}

func (m *mockPrinterService) setListError(err error) {
	m.mu.Lock()
	m.listError = err
	m.mu.Unlock()
}

func (m *mockPrinterService) setDefaultError(err error) {
	m.mu.Lock()
	m.defaultError = err
	m.mu.Unlock()
}

func setupTestRouter(printerSvc printer.Service, queue *print.Queue) *gin.Engine {
	router := gin.New()
	handlers := NewHandlers(printerSvc, queue)

	router.GET("/health", handlers.HealthCheck)
	router.GET("/v1/info", handlers.GetInfo)
	router.GET("/v1/printers", handlers.ListPrinters)
	router.GET("/v1/printers/default", handlers.GetDefaultPrinter)
	router.GET("/v1/printers/:id/status", handlers.GetPrinterStatus)
	router.POST("/v1/print", handlers.SubmitPrintJob)
	router.POST("/v1/print/batch", handlers.SubmitBatchPrintJob)
	router.GET("/v1/jobs", handlers.ListJobs)
	router.GET("/v1/jobs/:id", handlers.GetJobStatus)
	router.POST("/v1/jobs/:id/cancel", handlers.CancelJob)
	router.GET("/v1/stats", handlers.GetStats)

	return router
}

func TestNewHandlers(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	handlers := NewHandlers(mockSvc, queue)
	if handlers == nil {
		t.Fatal("NewHandlers returned nil")
	}
	if handlers.printerSvc == nil {
		t.Error("printerSvc is nil")
	}
	if handlers.printQueue == nil {
		t.Error("printQueue is nil")
	}
}

func TestHandlers_HealthCheck(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", resp["status"])
	}
	if resp["version"] != "0.1.6" {
		t.Errorf("Expected version '0.1.6', got '%v'", resp["version"])
	}
}

func TestHandlers_ListPrinters(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var printers []printer.PrinterInfo
	if err := json.Unmarshal(w.Body.Bytes(), &printers); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(printers) != 2 {
		t.Errorf("Expected 2 printers, got %d", len(printers))
	}
}

func TestHandlers_ListPrinters_Error(t *testing.T) {
	mockSvc := newMockPrinterService()
	mockSvc.setListError(errors.New("list error"))
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandlers_GetPrinterStatus(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers/printer1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status printer.PrinterInfo
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if status.ID != "printer1" {
		t.Errorf("Expected printer ID 'printer1', got '%s'", status.ID)
	}
}

func TestHandlers_GetPrinterStatus_NotFound(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers/nonexistent/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandlers_GetDefaultPrinter(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers/default", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp printer.PrinterInfo
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.ID != "printer1" {
		t.Errorf("Expected default printer 'printer1', got '%s'", resp.ID)
	}
}

func TestHandlers_GetDefaultPrinter_NoDefault(t *testing.T) {
	mockSvc := newMockPrinterService()
	mockSvc.setDefaultError(errors.New("no default printer"))
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/printers/default", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandlers_SubmitPrintJob(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp print.PrintResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.JobID == "" {
		t.Error("Expected job ID to be set")
	}
	if resp.Status != printer.JobStatusQueued {
		t.Errorf("Expected status 'queued', got '%s'", resp.Status)
	}
}

func TestHandlers_SubmitPrintJob_InvalidType(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "invalid",
		"data": "` + b64Data + `"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitPrintJob_InvalidJSON(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitPrintJob_MissingFields(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	// Missing required fields
	reqBody := `{
		"printer": "printer1"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitPrintJob_AllTypes(t *testing.T) {
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	types := []string{"pdf", "raw", "image"}

	for _, jobType := range types {
		t.Run(jobType, func(t *testing.T) {
			mockSvc := newMockPrinterService()
			queue := print.NewQueue(mockSvc)
			defer queue.Stop()

			router := setupTestRouter(mockSvc, queue)

			reqBody := `{
				"printer": "printer1",
				"type": "` + jobType + `",
				"data": "` + b64Data + `"
			}`

			req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusAccepted {
				t.Errorf("Expected status 202 for type %s, got %d", jobType, w.Code)
			}
		})
	}
}

func TestHandlers_GetJobStatus(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	// First submit a job
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var submitResp print.PrintResponse
	json.Unmarshal(w.Body.Bytes(), &submitResp)

	// Get job status
	req2 := httptest.NewRequest("GET", "/v1/jobs/"+submitResp.JobID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	var status printer.JobResult
	if err := json.Unmarshal(w2.Body.Bytes(), &status); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if status.JobID != submitResp.JobID {
		t.Errorf("Expected job ID '%s', got '%s'", submitResp.JobID, status.JobID)
	}
}

func TestHandlers_GetJobStatus_NotFound(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/jobs/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandlers_ListJobs(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	// Submit some jobs
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	for i := 0; i < 3; i++ {
		reqBody := `{
			"printer": "printer1",
			"type": "pdf",
			"data": "` + b64Data + `"
		}`
		req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// List active jobs
	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var jobs []*printer.JobResult
	if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestHandlers_ListJobs_History(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	// Submit a job
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `"
	}`
	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Wait for job to complete
	time.Sleep(500 * time.Millisecond)

	// List job history
	req2 := httptest.NewRequest("GET", "/v1/jobs?history=true", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}
}

func TestHandlers_GetStats(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats print.QueueStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Initially all should be 0
	if stats.QueuedJobs != 0 {
		t.Errorf("Expected 0 queued jobs, got %d", stats.QueuedJobs)
	}
}

func TestHandlers_SubmitPrintJob_WithSettings(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `",
		"settings": {
			"copies": 3,
			"orientation": "landscape",
			"paper_size": "A4",
			"duplex": "long-edge"
		}
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func BenchmarkHandlers_HealthCheck(b *testing.B) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandlers_ListPrinters(b *testing.B) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)
	req := httptest.NewRequest("GET", "/v1/printers", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandlers_SubmitPrintJob(b *testing.B) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test print data")
	b64Data := base64.StdEncoding.EncodeToString(testData)
	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func TestHandlers_GetInfo(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("GET", "/v1/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify expected fields
	if _, ok := resp["version"]; !ok {
		t.Error("Expected 'version' field in response")
	}
	if _, ok := resp["platform"]; !ok {
		t.Error("Expected 'platform' field in response")
	}
	if _, ok := resp["downloads"]; !ok {
		t.Error("Expected 'downloads' field in response")
	}
	if _, ok := resp["uptime"]; !ok {
		t.Error("Expected 'uptime' field in response")
	}
}

func TestHandlers_CancelJob(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	// Submit a job first
	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var submitResp print.PrintResponse
	json.Unmarshal(w.Body.Bytes(), &submitResp)

	// Cancel the job
	req2 := httptest.NewRequest("POST", "/v1/jobs/"+submitResp.JobID+"/cancel", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Job might already be processing, so accept both 200 and 400
	if w2.Code != http.StatusOK && w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w2.Code)
	}
}

func TestHandlers_CancelJob_NotFound(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	req := httptest.NewRequest("POST", "/v1/jobs/nonexistent/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitBatchPrintJob(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"jobs": [
			{"type": "pdf", "data": "` + b64Data + `", "name": "doc1"},
			{"type": "pdf", "data": "` + b64Data + `", "name": "doc2"},
			{"type": "image", "data": "` + b64Data + `"}
		]
	}`

	req := httptest.NewRequest("POST", "/v1/print/batch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp print.BatchPrintResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("Expected total 3, got %d", resp.Total)
	}
	if resp.Queued != 3 {
		t.Errorf("Expected queued 3, got %d", resp.Queued)
	}
	if len(resp.Jobs) != 3 {
		t.Errorf("Expected 3 jobs in response, got %d", len(resp.Jobs))
	}
}

func TestHandlers_SubmitBatchPrintJob_InvalidType(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"jobs": [
			{"type": "pdf", "data": "` + b64Data + `"},
			{"type": "invalid", "data": "` + b64Data + `"}
		]
	}`

	req := httptest.NewRequest("POST", "/v1/print/batch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitBatchPrintJob_EmptyJobs(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	reqBody := `{
		"printer": "printer1",
		"jobs": []
	}`

	req := httptest.NewRequest("POST", "/v1/print/batch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlers_SubmitPrintJob_WithName(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	router := setupTestRouter(mockSvc, queue)

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `",
		"name": "my-document.pdf"
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}

	var resp print.PrintResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Get job and verify name
	req2 := httptest.NewRequest("GET", "/v1/jobs/"+resp.JobID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var jobResult printer.JobResult
	json.Unmarshal(w2.Body.Bytes(), &jobResult)

	if jobResult.Name != "my-document.pdf" {
		t.Errorf("Expected name 'my-document.pdf', got '%s'", jobResult.Name)
	}
}
