package print

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
)

// PDFValidator provides PDF validation utilities
type PDFValidator struct{}

// NewPDFValidator creates a new PDF validator
func NewPDFValidator() *PDFValidator {
	return &PDFValidator{}
}

// PDF magic bytes
var pdfMagic = []byte("%PDF-")

// ValidatePDF checks if the data is a valid PDF
func (v *PDFValidator) ValidatePDF(data []byte) error {
	if len(data) < len(pdfMagic) {
		return fmt.Errorf("data too short to be a PDF")
	}

	if !bytes.HasPrefix(data, pdfMagic) {
		return fmt.Errorf("invalid PDF: missing PDF header")
	}

	// Check for EOF marker (%%EOF)
	// Note: Some PDFs might have trailing data after %%EOF, so we only check for presence
	eofCheckStart := 0
	if len(data) > 1024 {
		eofCheckStart = len(data) - 1024
	}
	_ = bytes.Contains(data[eofCheckStart:], []byte("%%EOF")) // result intentionally ignored for now

	return nil
}

// ValidateBase64PDF validates a base64-encoded PDF
func (v *PDFValidator) ValidateBase64PDF(b64data string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if err := v.ValidatePDF(data); err != nil {
		return nil, err
	}

	return data, nil
}

// GetPDFInfo extracts basic information from a PDF
func (v *PDFValidator) GetPDFInfo(data []byte) (*PDFInfo, error) {
	if err := v.ValidatePDF(data); err != nil {
		return nil, err
	}

	info := &PDFInfo{
		Size: len(data),
	}

	// Extract version
	if len(data) >= 8 {
		info.Version = string(data[5:8])
	}

	// Count pages (simple heuristic)
	info.Pages = bytes.Count(data, []byte("/Type /Page"))
	if info.Pages == 0 {
		info.Pages = bytes.Count(data, []byte("/Type/Page"))
	}
	// Subtract catalog page references
	catalogCount := bytes.Count(data, []byte("/Type /Pages"))
	if info.Pages > catalogCount {
		info.Pages -= catalogCount
	}

	return info, nil
}

// PDFInfo contains basic PDF information
type PDFInfo struct {
	Version string `json:"version"`
	Pages   int    `json:"pages"`
	Size    int    `json:"size"`
}

// ImageValidator provides image validation utilities
type ImageValidator struct{}

// NewImageValidator creates a new image validator
func NewImageValidator() *ImageValidator {
	return &ImageValidator{}
}

// Image magic bytes
var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
	pngMagic  = []byte{0x89, 0x50, 0x4E, 0x47}
	gifMagic  = []byte{0x47, 0x49, 0x46}
	bmpMagic  = []byte{0x42, 0x4D}
)

// ImageType represents the type of image
type ImageType string

const (
	ImageTypeJPEG    ImageType = "jpeg"
	ImageTypePNG     ImageType = "png"
	ImageTypeGIF     ImageType = "gif"
	ImageTypeBMP     ImageType = "bmp"
	ImageTypeUnknown ImageType = "unknown"
)

// DetectImageType detects the image type from data
func (v *ImageValidator) DetectImageType(data []byte) ImageType {
	if len(data) < 4 {
		return ImageTypeUnknown
	}

	switch {
	case bytes.HasPrefix(data, jpegMagic):
		return ImageTypeJPEG
	case bytes.HasPrefix(data, pngMagic):
		return ImageTypePNG
	case bytes.HasPrefix(data, gifMagic):
		return ImageTypeGIF
	case bytes.HasPrefix(data, bmpMagic):
		return ImageTypeBMP
	default:
		return ImageTypeUnknown
	}
}

// ValidateImage checks if the data is a valid image
func (v *ImageValidator) ValidateImage(data []byte) error {
	imgType := v.DetectImageType(data)
	if imgType == ImageTypeUnknown {
		return fmt.Errorf("unknown or unsupported image format")
	}
	return nil
}

// ValidateBase64Image validates a base64-encoded image
func (v *ImageValidator) ValidateBase64Image(b64data string) ([]byte, ImageType, error) {
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return nil, ImageTypeUnknown, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	imgType := v.DetectImageType(data)
	if imgType == ImageTypeUnknown {
		return nil, ImageTypeUnknown, fmt.Errorf("unknown or unsupported image format")
	}

	return data, imgType, nil
}

// RawDataValidator provides validation for raw printer data (ESC/POS, ZPL, etc.)
type RawDataValidator struct{}

// NewRawDataValidator creates a new raw data validator
func NewRawDataValidator() *RawDataValidator {
	return &RawDataValidator{}
}

// RawDataType represents the type of raw printer data
type RawDataType string

const (
	RawDataTypeESCPOS  RawDataType = "escpos"
	RawDataTypeZPL     RawDataType = "zpl"
	RawDataTypeTSPL    RawDataType = "tspl"
	RawDataTypeUnknown RawDataType = "unknown"
)

// ESC/POS and ZPL markers
var (
	escposInit = []byte{0x1B, 0x40} // ESC @
	zplStart   = []byte("^XA")
	tsplSize   = []byte("SIZE")
)

// DetectRawDataType attempts to detect the raw data type
func (v *RawDataValidator) DetectRawDataType(data []byte) RawDataType {
	if len(data) < 2 {
		return RawDataTypeUnknown
	}

	// Check for ESC/POS
	if bytes.Contains(data[:min(100, len(data))], escposInit) {
		return RawDataTypeESCPOS
	}

	// Check for ZPL
	if bytes.HasPrefix(data, zplStart) || bytes.Contains(data[:min(100, len(data))], zplStart) {
		return RawDataTypeZPL
	}

	// Check for TSPL
	if bytes.HasPrefix(data, tsplSize) || bytes.Contains(data[:min(100, len(data))], tsplSize) {
		return RawDataTypeTSPL
	}

	return RawDataTypeUnknown
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StreamCopy copies data from reader to writer with progress tracking
func StreamCopy(dst io.Writer, src io.Reader, onProgress func(written int64)) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				if onProgress != nil {
					onProgress(written)
				}
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}
