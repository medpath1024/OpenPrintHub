package printer

import "time"

// PrinterStatus represents the current status of a printer
type PrinterStatus string

const (
	StatusReady    PrinterStatus = "Ready"
	StatusBusy     PrinterStatus = "Busy"
	StatusOffline  PrinterStatus = "Offline"
	StatusError    PrinterStatus = "Error"
	StatusPaperOut PrinterStatus = "PaperOut"
	StatusPaperJam PrinterStatus = "PaperJam"
	StatusUnknown  PrinterStatus = "Unknown"
)

// PrinterInfo contains information about a printer
type PrinterInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      PrinterStatus `json:"status"`
	IsDefault   bool          `json:"is_default"`
	Location    string        `json:"location,omitempty"`
	Description string        `json:"description,omitempty"`

	DeviceClass            string   `json:"device_class,omitempty"`
	MediaKind              string   `json:"media_kind,omitempty"`
	CommandSets            []string `json:"command_sets,omitempty"`
	SupportedJobTypes      []string `json:"supported_job_types,omitempty"`
	PreferredLabelLanguage string   `json:"preferred_label_language,omitempty"`
	PreferredJobType       string   `json:"preferred_job_type,omitempty"`
	DPI                    int      `json:"dpi,omitempty"`
	CapabilitySource       string   `json:"capability_source,omitempty"`
	CapabilityConfidence   string   `json:"capability_confidence,omitempty"`
}

// PrintJobType represents the type of print data
type PrintJobType string

const (
	JobTypePDF   PrintJobType = "pdf"
	JobTypeRaw   PrintJobType = "raw"
	JobTypeImage PrintJobType = "image"
)

// PrintSettings contains print job settings
type PrintSettings struct {
	Copies      int    `json:"copies,omitempty"`
	Orientation string `json:"orientation,omitempty"` // portrait, landscape
	PaperSize   string `json:"paper_size,omitempty"`  // A4, Letter, etc.
	ColorMode   string `json:"color_mode,omitempty"`  // color, grayscale, mono
	Duplex      string `json:"duplex,omitempty"`      // none, long-edge, short-edge
	FitToPage   bool   `json:"fit_to_page,omitempty"`
	// Image-specific settings
	DPI       int    `json:"dpi,omitempty"`        // Image DPI (default: 300)
	ScaleMode string `json:"scale_mode,omitempty"` // fit, fill, stretch, none
}

// DefaultSettings returns default print settings
func DefaultSettings() PrintSettings {
	return PrintSettings{
		Copies:      1,
		Orientation: "portrait",
		FitToPage:   true,
	}
}

// PrintJob represents a print job to be submitted
type PrintJob struct {
	ID          string        `json:"id"`
	Name        string        `json:"name,omitempty"` // Job name for identification
	PrinterName string        `json:"printer"`
	Type        PrintJobType  `json:"type"`
	Data        []byte        `json:"-"`    // Raw binary data (not serialized in JSON)
	DataBase64  string        `json:"data"` // Base64 encoded data from API
	Settings    PrintSettings `json:"settings"`
	CreatedAt   time.Time     `json:"created_at"`
}

// JobStatus represents the status of a print job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusPrinting   JobStatus = "printing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCanceled   JobStatus = "canceled"
)

// JobResult contains the result of a print job
type JobResult struct {
	JobID       string    `json:"job_id"`
	Name        string    `json:"name,omitempty"` // Job name for identification
	Status      JobStatus `json:"status"`
	PrinterName string    `json:"printer"`
	Message     string    `json:"message,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}
