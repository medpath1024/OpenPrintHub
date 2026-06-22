//go:build linux

package printer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// linuxService implements the Service interface for Linux using CUPS/lp commands
type linuxService struct{}

// newPlatformService creates a new Linux printer service
func newPlatformService() Service {
	return &linuxService{}
}

// List returns all available printers on Linux using lpstat
func (s *linuxService) List() ([]PrinterInfo, error) {
	// Get list of printers
	cmd := exec.Command("lpstat", "-p")
	output, err := cmd.Output()
	if err != nil {
		// No printers or lpstat not available
		return []PrinterInfo{}, nil
	}

	// Get default printer
	defaultPrinter, _ := s.GetDefault()

	var printers []PrinterInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Parse lines like: "printer Brother_QL_820NWB is idle.  enabled since..."
		if strings.HasPrefix(line, "printer ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				status := s.parseStatus(line)
				printers = append(printers, ApplyCapabilities(PrinterInfo{
					ID:        name,
					Name:      strings.ReplaceAll(name, "_", " "),
					Status:    status,
					IsDefault: name == defaultPrinter,
				}))
			}
		}
	}

	return printers, nil
}

// GetDefault returns the default printer name on Linux
func (s *linuxService) GetDefault() (string, error) {
	cmd := exec.Command("lpstat", "-d")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default printer: %w", err)
	}

	// Parse output like: "system default destination: Brother_QL_820NWB"
	line := strings.TrimSpace(string(output))
	if strings.HasPrefix(line, "system default destination:") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1]), nil
		}
	}

	return "", fmt.Errorf("no default printer set")
}

// Status returns the current status of a specific printer
func (s *linuxService) Status(printerName string) (*PrinterInfo, error) {
	// Normalize printer name (replace spaces with underscores for CUPS)
	cupsName := strings.ReplaceAll(printerName, " ", "_")

	cmd := exec.Command("lpstat", "-p", cupsName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("printer not found: %s", printerName)
	}

	defaultPrinter, _ := s.GetDefault()
	line := strings.TrimSpace(string(output))
	status := s.parseStatus(line)

	info := ApplyCapabilities(PrinterInfo{
		ID:        cupsName,
		Name:      printerName,
		Status:    status,
		IsDefault: cupsName == defaultPrinter,
	})

	return &info, nil
}

// parseStatus extracts the printer status from lpstat output
func (s *linuxService) parseStatus(line string) PrinterStatus {
	lineLower := strings.ToLower(line)
	switch {
	case strings.Contains(lineLower, "idle"):
		return StatusReady
	case strings.Contains(lineLower, "printing"):
		return StatusBusy
	case strings.Contains(lineLower, "disabled"):
		return StatusOffline
	case strings.Contains(lineLower, "offline"):
		return StatusOffline
	case strings.Contains(lineLower, "paper"):
		if strings.Contains(lineLower, "jam") {
			return StatusPaperJam
		}
		return StatusPaperOut
	default:
		return StatusUnknown
	}
}

// Print sends a print job to the specified printer using lp command
func (s *linuxService) Print(job *PrintJob) error {
	// Normalize printer name
	printerName := strings.ReplaceAll(job.PrinterName, " ", "_")

	tempPattern := "oph-print-*.pdf"
	if job.Type == JobTypeImage {
		tempPattern = "oph-image-*" + detectImageExtension(job.Data)
	}

	// Create temp file for print data
	tmpFile, err := os.CreateTemp("", tempPattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write data to temp file
	if _, err := tmpFile.Write(job.Data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write print data: %w", err)
	}
	_ = tmpFile.Close()

	// Build lp command arguments
	args := []string{"-d", printerName}

	// Add copies
	if job.Settings.Copies > 1 {
		args = append(args, "-n", fmt.Sprintf("%d", job.Settings.Copies))
	}

	// Add options
	var options []string

	// Orientation
	if job.Settings.Orientation == "landscape" {
		options = append(options, "orientation-requested=4")
	}

	// Fit to page
	if job.Settings.FitToPage {
		options = append(options, "fit-to-page")
	}

	// Paper size
	if job.Settings.PaperSize != "" {
		options = append(options, fmt.Sprintf("media=%s", job.Settings.PaperSize))
	}

	// Duplex
	switch job.Settings.Duplex {
	case "long-edge":
		options = append(options, "sides=two-sided-long-edge")
	case "short-edge":
		options = append(options, "sides=two-sided-short-edge")
	}

	// Add all options
	for _, opt := range options {
		args = append(args, "-o", opt)
	}

	// Add file path
	args = append(args, tmpFile.Name())

	// Execute lp command
	cmd := exec.Command("lp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lp command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// PrintRaw sends raw data directly to a printer (for ESC/POS, ZPL, etc.)
func (s *linuxService) PrintRaw(printerName string, data []byte) error {
	// Normalize printer name
	cupsName := strings.ReplaceAll(printerName, " ", "_")

	// Create temp file for raw data
	tmpFile, err := os.CreateTemp("", "oph-raw-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write raw data: %w", err)
	}
	_ = tmpFile.Close()

	// Use lp with raw option
	cmd := exec.Command("lp", "-d", cupsName, "-o", "raw", tmpPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lp raw command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// GetPrinterURI gets the device URI for a printer (useful for direct connections)
func (s *linuxService) GetPrinterURI(printerName string) (string, error) {
	cupsName := strings.ReplaceAll(printerName, " ", "_")

	// Use lpstat -v to get printer URIs
	cmd := exec.Command("lpstat", "-v", cupsName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get printer URI: %w", err)
	}

	// Parse output like: "device for Brother_QL_820NWB: usb://Brother/QL-820NWB..."
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1]), nil
	}

	return "", fmt.Errorf("could not parse printer URI")
}
