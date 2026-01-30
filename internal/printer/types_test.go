package printer

import (
	"testing"
	"time"
)

func TestPrinterStatus_Constants(t *testing.T) {
	// Test that status constants have expected values
	tests := []struct {
		name     string
		status   PrinterStatus
		expected string
	}{
		{"StatusReady", StatusReady, "Ready"},
		{"StatusBusy", StatusBusy, "Busy"},
		{"StatusOffline", StatusOffline, "Offline"},
		{"StatusError", StatusError, "Error"},
		{"StatusPaperOut", StatusPaperOut, "PaperOut"},
		{"StatusPaperJam", StatusPaperJam, "PaperJam"},
		{"StatusUnknown", StatusUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestPrintJobType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		jobType  PrintJobType
		expected string
	}{
		{"JobTypePDF", JobTypePDF, "pdf"},
		{"JobTypeRaw", JobTypeRaw, "raw"},
		{"JobTypeImage", JobTypeImage, "image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.jobType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.jobType))
			}
		})
	}
}

func TestJobStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected string
	}{
		{"JobStatusQueued", JobStatusQueued, "queued"},
		{"JobStatusProcessing", JobStatusProcessing, "processing"},
		{"JobStatusPrinting", JobStatusPrinting, "printing"},
		{"JobStatusCompleted", JobStatusCompleted, "completed"},
		{"JobStatusFailed", JobStatusFailed, "failed"},
		{"JobStatusCancelled", JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if settings.Copies != 1 {
		t.Errorf("Expected Copies to be 1, got %d", settings.Copies)
	}

	if settings.Orientation != "portrait" {
		t.Errorf("Expected Orientation to be 'portrait', got %s", settings.Orientation)
	}

	if !settings.FitToPage {
		t.Error("Expected FitToPage to be true")
	}
}

func TestPrinterInfo_Struct(t *testing.T) {
	info := PrinterInfo{
		ID:          "printer1",
		Name:        "Test Printer",
		Status:      StatusReady,
		IsDefault:   true,
		Location:    "Office",
		Description: "A test printer",
	}

	if info.ID != "printer1" {
		t.Errorf("Expected ID 'printer1', got %s", info.ID)
	}
	if info.Name != "Test Printer" {
		t.Errorf("Expected Name 'Test Printer', got %s", info.Name)
	}
	if info.Status != StatusReady {
		t.Errorf("Expected Status StatusReady, got %s", info.Status)
	}
	if !info.IsDefault {
		t.Error("Expected IsDefault to be true")
	}
	if info.Location != "Office" {
		t.Errorf("Expected Location 'Office', got %s", info.Location)
	}
	if info.Description != "A test printer" {
		t.Errorf("Expected Description 'A test printer', got %s", info.Description)
	}
}

func TestPrintSettings_Struct(t *testing.T) {
	settings := PrintSettings{
		Copies:      3,
		Orientation: "landscape",
		PaperSize:   "A4",
		ColorMode:   "color",
		Duplex:      "long-edge",
		FitToPage:   false,
		DPI:         300,
		ScaleMode:   "fit",
	}

	if settings.Copies != 3 {
		t.Errorf("Expected Copies 3, got %d", settings.Copies)
	}
	if settings.Orientation != "landscape" {
		t.Errorf("Expected Orientation 'landscape', got %s", settings.Orientation)
	}
	if settings.PaperSize != "A4" {
		t.Errorf("Expected PaperSize 'A4', got %s", settings.PaperSize)
	}
	if settings.ColorMode != "color" {
		t.Errorf("Expected ColorMode 'color', got %s", settings.ColorMode)
	}
	if settings.Duplex != "long-edge" {
		t.Errorf("Expected Duplex 'long-edge', got %s", settings.Duplex)
	}
	if settings.FitToPage {
		t.Error("Expected FitToPage to be false")
	}
	if settings.DPI != 300 {
		t.Errorf("Expected DPI 300, got %d", settings.DPI)
	}
	if settings.ScaleMode != "fit" {
		t.Errorf("Expected ScaleMode 'fit', got %s", settings.ScaleMode)
	}
}

func TestPrintJob_Struct(t *testing.T) {
	now := time.Now()
	job := PrintJob{
		ID:          "job-123",
		Name:        "my-document.pdf",
		PrinterName: "Test Printer",
		Type:        JobTypePDF,
		Data:        []byte("test data"),
		DataBase64:  "dGVzdCBkYXRh",
		Settings:    DefaultSettings(),
		CreatedAt:   now,
	}

	if job.ID != "job-123" {
		t.Errorf("Expected ID 'job-123', got %s", job.ID)
	}
	if job.Name != "my-document.pdf" {
		t.Errorf("Expected Name 'my-document.pdf', got %s", job.Name)
	}
	if job.PrinterName != "Test Printer" {
		t.Errorf("Expected PrinterName 'Test Printer', got %s", job.PrinterName)
	}
	if job.Type != JobTypePDF {
		t.Errorf("Expected Type JobTypePDF, got %s", job.Type)
	}
	if string(job.Data) != "test data" {
		t.Errorf("Expected Data 'test data', got %s", string(job.Data))
	}
	if job.DataBase64 != "dGVzdCBkYXRh" {
		t.Errorf("Expected DataBase64 'dGVzdCBkYXRh', got %s", job.DataBase64)
	}
	if !job.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt %v, got %v", now, job.CreatedAt)
	}
}

func TestJobResult_Struct(t *testing.T) {
	now := time.Now()
	result := JobResult{
		JobID:       "job-123",
		Name:        "my-document.pdf",
		Status:      JobStatusCompleted,
		PrinterName: "Test Printer",
		Message:     "Print completed",
		Error:       "",
		CreatedAt:   now.Add(-2 * time.Minute),
		StartedAt:   now.Add(-time.Minute),
		CompletedAt: now,
	}

	if result.JobID != "job-123" {
		t.Errorf("Expected JobID 'job-123', got %s", result.JobID)
	}
	if result.Name != "my-document.pdf" {
		t.Errorf("Expected Name 'my-document.pdf', got %s", result.Name)
	}
	if result.Status != JobStatusCompleted {
		t.Errorf("Expected Status JobStatusCompleted, got %s", result.Status)
	}
	if result.PrinterName != "Test Printer" {
		t.Errorf("Expected PrinterName 'Test Printer', got %s", result.PrinterName)
	}
	if result.Message != "Print completed" {
		t.Errorf("Expected Message 'Print completed', got %s", result.Message)
	}
	if result.Error != "" {
		t.Errorf("Expected empty Error, got %s", result.Error)
	}
	if result.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestJobResult_WithError(t *testing.T) {
	result := JobResult{
		JobID:       "job-456",
		Status:      JobStatusFailed,
		PrinterName: "Test Printer",
		Message:     "Print failed",
		Error:       "Connection refused",
	}

	if result.Status != JobStatusFailed {
		t.Errorf("Expected Status JobStatusFailed, got %s", result.Status)
	}
	if result.Error != "Connection refused" {
		t.Errorf("Expected Error 'Connection refused', got %s", result.Error)
	}
}
