//go:build windows

package printer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	winspool              = syscall.NewLazyDLL("winspool.drv")
	procEnumPrinters      = winspool.NewProc("EnumPrintersW")
	procGetDefaultPrinter = winspool.NewProc("GetDefaultPrinterW")
	procOpenPrinter       = winspool.NewProc("OpenPrinterW")
	procClosePrinter      = winspool.NewProc("ClosePrinter")
	procStartDocPrinter   = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter     = winspool.NewProc("EndDocPrinter")
	procStartPagePrinter  = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter    = winspool.NewProc("EndPagePrinter")
	procWritePrinter      = winspool.NewProc("WritePrinter")
	procGetPrinter        = winspool.NewProc("GetPrinterW")

	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetLastError = kernel32.NewProc("GetLastError")

	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

const (
	PRINTER_ENUM_LOCAL       = 0x00000002
	PRINTER_ENUM_CONNECTIONS = 0x00000004
	PRINTER_STATUS_READY     = 0x00000000
	PRINTER_STATUS_PAUSED    = 0x00000001
	PRINTER_STATUS_ERROR     = 0x00000002
	PRINTER_STATUS_OFFLINE   = 0x00000080
	PRINTER_STATUS_PAPER_JAM = 0x00000008
	PRINTER_STATUS_PAPER_OUT = 0x00000010
	PRINTER_STATUS_BUSY      = 0x00000200
)

// PRINTER_INFO_2 structure
type PRINTER_INFO_2 struct {
	ServerName         *uint16
	PrinterName        *uint16
	ShareName          *uint16
	PortName           *uint16
	DriverName         *uint16
	Comment            *uint16
	Location           *uint16
	DevMode            uintptr
	SepFile            *uint16
	PrintProcessor     *uint16
	Datatype           *uint16
	Parameters         *uint16
	SecurityDescriptor uintptr
	Attributes         uint32
	Priority           uint32
	DefaultPriority    uint32
	StartTime          uint32
	UntilTime          uint32
	Status             uint32
	Jobs               uint32
	AveragePPM         uint32
}

// DOC_INFO_1 structure for StartDocPrinter
type DOC_INFO_1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

// windowsService implements the Service interface for Windows
type windowsService struct{}

// newPlatformService creates a new Windows printer service
func newPlatformService() Service {
	return &windowsService{}
}

// utf16PtrToString converts a UTF-16 pointer to a Go string
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	// Find null terminator
	end := unsafe.Pointer(p)
	n := 0
	for *(*uint16)(unsafe.Pointer(uintptr(end) + uintptr(n)*2)) != 0 {
		n++
	}
	s := make([]uint16, n)
	for i := 0; i < n; i++ {
		s[i] = *(*uint16)(unsafe.Pointer(uintptr(end) + uintptr(i)*2))
	}
	return syscall.UTF16ToString(s)
}

// stringToUTF16Ptr converts a Go string to a UTF-16 pointer
func stringToUTF16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// List returns all available printers on Windows
func (s *windowsService) List() ([]PrinterInfo, error) {
	var needed, returned uint32
	flags := uint32(PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS)

	// First call to get required buffer size
	procEnumPrinters.Call(
		uintptr(flags),
		0, // Name
		2, // Level (PRINTER_INFO_2)
		0, // pPrinterEnum
		0, // cbBuf
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if needed == 0 {
		return []PrinterInfo{}, nil
	}

	// Allocate buffer and call again
	buf := make([]byte, needed)
	ret, _, _ := procEnumPrinters.Call(
		uintptr(flags),
		0,
		2,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("EnumPrinters failed")
	}

	// Get default printer
	defaultPrinter, _ := s.GetDefault()

	// Parse results
	printers := make([]PrinterInfo, 0, returned)
	infoSize := unsafe.Sizeof(PRINTER_INFO_2{})
	for i := uint32(0); i < returned; i++ {
		info := (*PRINTER_INFO_2)(unsafe.Pointer(&buf[uintptr(i)*infoSize]))
		name := utf16PtrToString(info.PrinterName)
		printers = append(printers, PrinterInfo{
			ID:          name,
			Name:        name,
			Status:      s.parseStatus(info.Status),
			IsDefault:   name == defaultPrinter,
			Location:    utf16PtrToString(info.Location),
			Description: utf16PtrToString(info.Comment),
		})
	}

	return printers, nil
}

// GetDefault returns the default printer name on Windows
func (s *windowsService) GetDefault() (string, error) {
	var size uint32 = 256
	buf := make([]uint16, size)

	ret, _, _ := procGetDefaultPrinter.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return "", fmt.Errorf("no default printer set")
	}

	return syscall.UTF16ToString(buf), nil
}

// Status returns the current status of a specific printer
func (s *windowsService) Status(printerName string) (*PrinterInfo, error) {
	var hPrinter syscall.Handle
	pName := stringToUTF16Ptr(printerName)

	ret, _, _ := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pName)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to open printer: %s", printerName)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	// Get printer info
	var needed uint32
	procGetPrinter.Call(
		uintptr(hPrinter),
		2,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
	)

	if needed == 0 {
		return nil, fmt.Errorf("failed to get printer info size")
	}

	buf := make([]byte, needed)
	ret, _, _ = procGetPrinter.Call(
		uintptr(hPrinter),
		2,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to get printer info")
	}

	info := (*PRINTER_INFO_2)(unsafe.Pointer(&buf[0]))
	defaultPrinter, _ := s.GetDefault()

	return &PrinterInfo{
		ID:          printerName,
		Name:        printerName,
		Status:      s.parseStatus(info.Status),
		IsDefault:   printerName == defaultPrinter,
		Location:    utf16PtrToString(info.Location),
		Description: utf16PtrToString(info.Comment),
	}, nil
}

// parseStatus converts Windows printer status flags to PrinterStatus
func (s *windowsService) parseStatus(status uint32) PrinterStatus {
	switch {
	case status == PRINTER_STATUS_READY:
		return StatusReady
	case status&PRINTER_STATUS_OFFLINE != 0:
		return StatusOffline
	case status&PRINTER_STATUS_ERROR != 0:
		return StatusError
	case status&PRINTER_STATUS_PAPER_JAM != 0:
		return StatusPaperJam
	case status&PRINTER_STATUS_PAPER_OUT != 0:
		return StatusPaperOut
	case status&PRINTER_STATUS_BUSY != 0:
		return StatusBusy
	case status&PRINTER_STATUS_PAUSED != 0:
		return StatusOffline
	default:
		return StatusUnknown
	}
}

// Print sends a print job to the specified printer
// For PDF files on Windows, we use ShellExecute with "print" verb
func (s *windowsService) Print(job *PrintJob) error {
	// Create temp file for print data
	tmpFile, err := os.CreateTemp("", "oph-print-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Note: We don't delete immediately as the print spooler needs access
	// The file will be cleaned up on next run or by the OS

	if _, err := tmpFile.Write(job.Data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write print data: %w", err)
	}
	tmpFile.Close()

	// For PDF, use ShellExecute to print via the default PDF handler
	// This requires a PDF reader like Adobe Reader, Foxit, or SumatraPDF
	if job.Type == JobTypePDF {
		return s.printPDFWithShell(tmpPath, job.PrinterName, job.Settings)
	}

	// For other types, use direct printing
	return s.printDirect(job.PrinterName, job.Data)
}

// printPDFWithShell prints a PDF using the shell print verb
func (s *windowsService) printPDFWithShell(filePath, printerName string, settings PrintSettings) error {
	// Try SumatraPDF first (if available) for better control
	sumatraPath := s.findSumatraPDF()
	if sumatraPath != "" {
		return s.printWithSumatra(sumatraPath, filePath, printerName, settings)
	}

	// Fall back to ShellExecute print verb
	verb := stringToUTF16Ptr("print")
	file := stringToUTF16Ptr(filePath)

	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		0, // SW_HIDE
	)

	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed with code: %d", ret)
	}

	return nil
}

// findSumatraPDF looks for SumatraPDF installation
func (s *windowsService) findSumatraPDF() string {
	paths := []string{
		`C:\Program Files\SumatraPDF\SumatraPDF.exe`,
		`C:\Program Files (x86)\SumatraPDF\SumatraPDF.exe`,
		`SumatraPDF.exe`, // In PATH
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Check if in PATH
		if p, err := exec.LookPath(path); err == nil {
			return p
		}
	}

	return ""
}

// printWithSumatra uses SumatraPDF for silent printing
func (s *windowsService) printWithSumatra(sumatraPath, filePath, printerName string, settings PrintSettings) error {
	args := []string{"-print-to", printerName}

	// Add print settings
	var printSettings []string
	if settings.Copies > 1 {
		printSettings = append(printSettings, fmt.Sprintf("%dx", settings.Copies))
	}
	if settings.Orientation == "landscape" {
		printSettings = append(printSettings, "landscape")
	}
	if settings.Duplex == "long-edge" {
		printSettings = append(printSettings, "duplex")
	} else if settings.Duplex == "short-edge" {
		printSettings = append(printSettings, "duplexshort")
	}

	if len(printSettings) > 0 {
		args = append(args, "-print-settings", strings.Join(printSettings, ","))
	}

	args = append(args, filePath)

	cmd := exec.Command(sumatraPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SumatraPDF failed: %w, output: %s", err, string(output))
	}

	return nil
}

// printDirect sends raw data directly to the printer using WritePrinter
func (s *windowsService) printDirect(printerName string, data []byte) error {
	var hPrinter syscall.Handle
	pName := stringToUTF16Ptr(printerName)

	ret, _, _ := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pName)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)

	if ret == 0 {
		return fmt.Errorf("failed to open printer: %s", printerName)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	// Start document
	docInfo := DOC_INFO_1{
		DocName:  stringToUTF16Ptr("OpenPrintHub Document"),
		Datatype: stringToUTF16Ptr("RAW"),
	}

	ret, _, _ = procStartDocPrinter.Call(
		uintptr(hPrinter),
		1,
		uintptr(unsafe.Pointer(&docInfo)),
	)

	if ret == 0 {
		return fmt.Errorf("StartDocPrinter failed")
	}
	defer procEndDocPrinter.Call(uintptr(hPrinter))

	// Start page
	ret, _, _ = procStartPagePrinter.Call(uintptr(hPrinter))
	if ret == 0 {
		return fmt.Errorf("StartPagePrinter failed")
	}
	defer procEndPagePrinter.Call(uintptr(hPrinter))

	// Write data
	var written uint32
	ret, _, _ = procWritePrinter.Call(
		uintptr(hPrinter),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)

	if ret == 0 {
		return fmt.Errorf("WritePrinter failed")
	}

	return nil
}

// PrintRaw sends raw data directly to a printer (for ESC/POS, ZPL, etc.)
func (s *windowsService) PrintRaw(printerName string, data []byte) error {
	return s.printDirect(printerName, data)
}
