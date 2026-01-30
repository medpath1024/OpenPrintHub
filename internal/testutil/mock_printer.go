package testutil

import (
	"errors"
	"sync"

	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

// MockPrinterService is a mock implementation of printer.Service for testing
type MockPrinterService struct {
	mu sync.RWMutex

	// Printers to return
	Printers []printer.PrinterInfo

	// Default printer name
	DefaultPrinter string

	// Error to return from methods
	ListError     error
	DefaultError  error
	StatusError   error
	PrintError    error
	PrintRawError error

	// Track calls for verification
	PrintCalls    []PrintCall
	PrintRawCalls []PrintRawCall
}

// PrintCall records a call to Print
type PrintCall struct {
	Job *printer.PrintJob
}

// PrintRawCall records a call to PrintRaw
type PrintRawCall struct {
	PrinterName string
	Data        []byte
}

// NewMockPrinterService creates a new mock printer service with default printers
func NewMockPrinterService() *MockPrinterService {
	return &MockPrinterService{
		Printers: []printer.PrinterInfo{
			{
				ID:        "printer1",
				Name:      "Test Printer 1",
				Status:    printer.StatusReady,
				IsDefault: true,
			},
			{
				ID:        "printer2",
				Name:      "Test Printer 2",
				Status:    printer.StatusReady,
				IsDefault: false,
			},
			{
				ID:        "offline_printer",
				Name:      "Offline Printer",
				Status:    printer.StatusOffline,
				IsDefault: false,
			},
		},
		DefaultPrinter: "printer1",
		PrintCalls:     make([]PrintCall, 0),
		PrintRawCalls:  make([]PrintRawCall, 0),
	}
}

// List returns all available printers
func (m *MockPrinterService) List() ([]printer.PrinterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ListError != nil {
		return nil, m.ListError
	}
	return m.Printers, nil
}

// GetDefault returns the default printer name
func (m *MockPrinterService) GetDefault() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.DefaultError != nil {
		return "", m.DefaultError
	}
	if m.DefaultPrinter == "" {
		return "", errors.New("no default printer set")
	}
	return m.DefaultPrinter, nil
}

// Status returns the status of a specific printer
func (m *MockPrinterService) Status(printerName string) (*printer.PrinterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.StatusError != nil {
		return nil, m.StatusError
	}

	for _, p := range m.Printers {
		if p.ID == printerName || p.Name == printerName {
			return &p, nil
		}
	}
	return nil, errors.New("printer not found: " + printerName)
}

// Print sends a print job to the specified printer
func (m *MockPrinterService) Print(job *printer.PrintJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PrintError != nil {
		return m.PrintError
	}

	m.PrintCalls = append(m.PrintCalls, PrintCall{Job: job})
	return nil
}

// PrintRaw sends raw data directly to a printer
func (m *MockPrinterService) PrintRaw(printerName string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PrintRawError != nil {
		return m.PrintRawError
	}

	m.PrintRawCalls = append(m.PrintRawCalls, PrintRawCall{
		PrinterName: printerName,
		Data:        data,
	})
	return nil
}

// Reset resets all recorded calls and errors
func (m *MockPrinterService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PrintCalls = make([]PrintCall, 0)
	m.PrintRawCalls = make([]PrintRawCall, 0)
	m.ListError = nil
	m.DefaultError = nil
	m.StatusError = nil
	m.PrintError = nil
	m.PrintRawError = nil
}

// GetPrintCallCount returns the number of Print calls
func (m *MockPrinterService) GetPrintCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.PrintCalls)
}

// GetPrintRawCallCount returns the number of PrintRaw calls
func (m *MockPrinterService) GetPrintRawCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.PrintRawCalls)
}

// SetPrinterStatus updates the status of a printer
func (m *MockPrinterService) SetPrinterStatus(printerID string, status printer.PrinterStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Printers {
		if p.ID == printerID {
			m.Printers[i].Status = status
			return
		}
	}
}

// AddPrinter adds a new printer to the mock service
func (m *MockPrinterService) AddPrinter(p printer.PrinterInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Printers = append(m.Printers, p)
}

// RemovePrinter removes a printer from the mock service
func (m *MockPrinterService) RemovePrinter(printerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Printers {
		if p.ID == printerID {
			m.Printers = append(m.Printers[:i], m.Printers[i+1:]...)
			return
		}
	}
}
