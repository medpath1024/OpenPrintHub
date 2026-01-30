package printer

// Service defines the interface for printer operations
// This interface is implemented differently for each platform
type Service interface {
	// List returns all available printers on the system
	List() ([]PrinterInfo, error)

	// GetDefault returns the name of the default printer
	GetDefault() (string, error)

	// Status returns the current status of a specific printer
	Status(printerName string) (*PrinterInfo, error)

	// Print sends a print job to the specified printer
	Print(job *PrintJob) error

	// PrintRaw sends raw data directly to a printer (for ESC/POS, ZPL, etc.)
	PrintRaw(printerName string, data []byte) error
}

// New creates a new platform-specific printer service
// This function is implemented in printer_darwin.go and printer_windows.go
func New() Service {
	return newPlatformService()
}
