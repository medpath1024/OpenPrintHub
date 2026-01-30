package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// Integration tests for the complete API flow

func TestIntegration_FullPrintJobFlow(t *testing.T) {
	// Setup
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	// 1. Check health
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Health check failed: %d", w.Code)
	}

	// 2. List printers
	req = httptest.NewRequest("GET", "/v1/printers", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List printers failed: %d", w.Code)
	}

	var printers []printer.PrinterInfo
	json.Unmarshal(w.Body.Bytes(), &printers)
	if len(printers) == 0 {
		t.Fatal("No printers returned")
	}

	// 3. Get default printer
	req = httptest.NewRequest("GET", "/v1/printers/default", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get default printer failed: %d", w.Code)
	}

	// 4. Submit a print job
	testData := []byte("test PDF content")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `",
		"settings": {
			"copies": 2,
			"orientation": "portrait"
		}
	}`

	req = httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Submit job failed: %d, body: %s", w.Code, w.Body.String())
	}

	var submitResp print.PrintResponse
	json.Unmarshal(w.Body.Bytes(), &submitResp)
	jobID := submitResp.JobID

	if jobID == "" {
		t.Fatal("No job ID returned")
	}

	// 5. Get job status
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get job status failed: %d", w.Code)
	}

	// 6. Wait for job completion and check history
	time.Sleep(500 * time.Millisecond)

	req = httptest.NewRequest("GET", "/v1/jobs?history=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get job history failed: %d", w.Code)
	}

	// 7. Check stats
	req = httptest.NewRequest("GET", "/v1/stats", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get stats failed: %d", w.Code)
	}

	var stats print.QueueStats
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.CompletedJobs == 0 {
		t.Error("Expected at least 1 completed job")
	}
}

func TestIntegration_WebSocketJobStatusUpdates(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)

	// Create test HTTP server
	testServer := httptest.NewServer(server.APIRouter())
	defer testServer.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/v1/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Wait for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Submit a print job via HTTP
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
	server.APIRouter().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Submit job failed: %d", w.Code)
	}

	// Read WebSocket messages for status updates
	receivedStatuses := make([]string, 0)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	for i := 0; i < 4; i++ { // Expect: queued, processing, printing, completed
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break // Timeout or connection closed
		}
		if msg.Type == "job_status" {
			if data, ok := msg.Data.(map[string]interface{}); ok {
				if status, ok := data["status"].(string); ok {
					receivedStatuses = append(receivedStatuses, status)
				}
			}
		}
	}

	// Verify we received status updates
	if len(receivedStatuses) == 0 {
		t.Error("No status updates received via WebSocket")
	}
}

func TestIntegration_ConcurrentPrintJobs(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	var wg sync.WaitGroup
	numJobs := 20
	jobIDs := make([]string, numJobs)
	var mu sync.Mutex
	errors := 0

	// Submit jobs concurrently
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			reqBody := `{
				"printer": "printer1",
				"type": "pdf",
				"data": "` + b64Data + `"
			}`

			req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			mu.Lock()
			defer mu.Unlock()

			if w.Code != http.StatusAccepted {
				errors++
				return
			}

			var resp print.PrintResponse
			json.Unmarshal(w.Body.Bytes(), &resp)
			jobIDs[idx] = resp.JobID
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("%d jobs failed to submit", errors)
	}

	// Wait for all jobs to complete with retries
	var stats print.QueueStats
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		time.Sleep(200 * time.Millisecond)

		req := httptest.NewRequest("GET", "/v1/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		json.Unmarshal(w.Body.Bytes(), &stats)

		if stats.CompletedJobs >= numJobs {
			break
		}
	}

	if stats.CompletedJobs != numJobs {
		t.Errorf("Expected %d completed jobs, got %d", numJobs, stats.CompletedJobs)
	}
}

func TestIntegration_CORSHeaders(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "http://example.com,http://app.local",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	// Test allowed origin
	req := httptest.NewRequest("GET", "/v1/printers", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected CORS header for allowed origin, got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Test disallowed origin
	req2 := httptest.NewRequest("GET", "/v1/printers", nil)
	req2.Header.Set("Origin", "http://evil.com")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS header for disallowed origin, got '%s'", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestIntegration_SecurityHeaders(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	req := httptest.NewRequest("GET", "/v1/printers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check security headers
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expected := range headers {
		actual := w.Header().Get(header)
		if actual != expected {
			t.Errorf("Header %s: expected '%s', got '%s'", header, expected, actual)
		}
	}
}

func TestIntegration_PreflightRequest(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	// Send preflight OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/v1/print", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for preflight, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
}

func TestIntegration_PrintJobTypes(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	types := []struct {
		name string
		data []byte
	}{
		{"pdf", []byte("%PDF-1.4 test content")},
		{"raw", []byte{0x1B, 0x40, 0x48, 0x65, 0x6C, 0x6C, 0x6F}}, // ESC/POS
		{"image", []byte{0xFF, 0xD8, 0xFF, 0xE0}},                 // JPEG header
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			b64Data := base64.StdEncoding.EncodeToString(tt.data)

			reqBody := `{
				"printer": "printer1",
				"type": "` + tt.name + `",
				"data": "` + b64Data + `"
			}`

			req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusAccepted {
				t.Errorf("Expected 202 for type %s, got %d. Body: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			method:     "POST",
			path:       "/v1/print",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing required fields",
			method:     "POST",
			path:       "/v1/print",
			body:       `{"printer": "printer1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid print type",
			method:     "POST",
			path:       "/v1/print",
			body:       `{"printer": "p", "type": "invalid", "data": "dGVzdA=="}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "job not found",
			method:     "GET",
			path:       "/v1/jobs/nonexistent",
			body:       "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "printer not found",
			method:     "GET",
			path:       "/v1/printers/nonexistent/status",
			body:       "",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestIntegration_PrintSettings(t *testing.T) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	testData := []byte("test")
	b64Data := base64.StdEncoding.EncodeToString(testData)

	// Test with full settings
	reqBody := `{
		"printer": "printer1",
		"type": "pdf",
		"data": "` + b64Data + `",
		"settings": {
			"copies": 5,
			"orientation": "landscape",
			"paper_size": "A4",
			"color_mode": "grayscale",
			"duplex": "long-edge",
			"fit_to_page": true
		}
	}`

	req := httptest.NewRequest("POST", "/v1/print", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp print.PrintResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Verify job was created with settings
	entry, ok := queue.GetJob(resp.JobID)
	if !ok {
		t.Fatal("Job not found")
	}

	if entry.Job.Settings.Copies != 5 {
		t.Errorf("Expected 5 copies, got %d", entry.Job.Settings.Copies)
	}
	if entry.Job.Settings.Orientation != "landscape" {
		t.Errorf("Expected landscape orientation, got %s", entry.Job.Settings.Orientation)
	}
}

func BenchmarkIntegration_SubmitAndComplete(b *testing.B) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	testData := []byte("test")
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

func BenchmarkIntegration_HealthCheck(b *testing.B) {
	mockSvc := newMockPrinterService()
	queue := print.NewQueue(mockSvc)
	defer queue.Stop()

	config := Config{
		Port:         12345,
		WebPort:      12346,
		AllowOrigins: "*",
		PrinterSvc:   mockSvc,
		PrintQueue:   queue,
	}

	server := NewServer(config)
	router := server.APIRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
