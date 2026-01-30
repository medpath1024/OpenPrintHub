package print

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewPDFValidator(t *testing.T) {
	v := NewPDFValidator()
	if v == nil {
		t.Fatal("NewPDFValidator returned nil")
	}
}

func TestPDFValidator_ValidatePDF_Valid(t *testing.T) {
	v := NewPDFValidator()

	// Valid PDF data - needs to be larger than 1024 bytes for EOF check
	validPDF := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 100 >>
stream
BT
/F1 12 Tf
100 700 Td
(Hello World) Tj
ET
endstream
endobj
5 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
6 0 obj
<< /Type /FontDescriptor /FontName /Helvetica /Flags 32 /ItalicAngle 0 /Ascent 718 /Descent -207 /CapHeight 718 /StemV 88 >>
endobj
xref
0 7
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000214 00000 n 
0000000367 00000 n 
0000000444 00000 n 
trailer
<< /Root 1 0 R /Size 7 >>
startxref
600
%%EOF
` + strings.Repeat(" ", 500)) // Pad to ensure > 1024 bytes

	err := v.ValidatePDF(validPDF)
	if err != nil {
		t.Errorf("Expected valid PDF to pass, got error: %v", err)
	}
}

func TestPDFValidator_ValidatePDF_TooShort(t *testing.T) {
	v := NewPDFValidator()

	shortData := []byte("%PD")
	err := v.ValidatePDF(shortData)
	if err == nil {
		t.Error("Expected error for data too short")
	}
}

func TestPDFValidator_ValidatePDF_InvalidHeader(t *testing.T) {
	v := NewPDFValidator()

	invalidPDF := []byte("Not a PDF file content")
	err := v.ValidatePDF(invalidPDF)
	if err == nil {
		t.Error("Expected error for invalid PDF header")
	}
}

func TestPDFValidator_ValidateBase64PDF(t *testing.T) {
	v := NewPDFValidator()

	validPDF := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
trailer
<< /Root 1 0 R >>
startxref
0
%%EOF`)

	b64Data := base64.StdEncoding.EncodeToString(validPDF)

	data, err := v.ValidateBase64PDF(b64Data)
	if err != nil {
		t.Errorf("Expected valid base64 PDF to pass, got error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected decoded data to be non-empty")
	}
}

func TestPDFValidator_ValidateBase64PDF_InvalidBase64(t *testing.T) {
	v := NewPDFValidator()

	_, err := v.ValidateBase64PDF("invalid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestPDFValidator_ValidateBase64PDF_InvalidPDF(t *testing.T) {
	v := NewPDFValidator()

	invalidData := base64.StdEncoding.EncodeToString([]byte("Not a PDF"))
	_, err := v.ValidateBase64PDF(invalidData)
	if err == nil {
		t.Error("Expected error for invalid PDF content")
	}
}

func TestPDFValidator_GetPDFInfo(t *testing.T) {
	v := NewPDFValidator()

	validPDF := []byte(`%PDF-1.7
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj
4 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj
trailer
<< /Root 1 0 R >>
startxref
0
%%EOF`)

	info, err := v.GetPDFInfo(validPDF)
	if err != nil {
		t.Fatalf("GetPDFInfo failed: %v", err)
	}

	if info.Version != "1.7" {
		t.Errorf("Expected version '1.7', got '%s'", info.Version)
	}
	if info.Size != len(validPDF) {
		t.Errorf("Expected size %d, got %d", len(validPDF), info.Size)
	}
	// Note: Page counting is a heuristic
	if info.Pages < 1 {
		t.Errorf("Expected at least 1 page, got %d", info.Pages)
	}
}

func TestPDFValidator_GetPDFInfo_InvalidPDF(t *testing.T) {
	v := NewPDFValidator()

	_, err := v.GetPDFInfo([]byte("Not a PDF"))
	if err == nil {
		t.Error("Expected error for invalid PDF")
	}
}

func TestPDFInfo_Struct(t *testing.T) {
	info := PDFInfo{
		Version: "1.5",
		Pages:   10,
		Size:    1024,
	}

	if info.Version != "1.5" {
		t.Errorf("Expected Version '1.5', got '%s'", info.Version)
	}
	if info.Pages != 10 {
		t.Errorf("Expected Pages 10, got %d", info.Pages)
	}
	if info.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", info.Size)
	}
}

// Image Validator Tests

func TestNewImageValidator(t *testing.T) {
	v := NewImageValidator()
	if v == nil {
		t.Fatal("NewImageValidator returned nil")
	}
}

func TestImageValidator_DetectImageType(t *testing.T) {
	v := NewImageValidator()

	tests := []struct {
		name     string
		data     []byte
		expected ImageType
	}{
		{
			name:     "JPEG",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			expected: ImageTypeJPEG,
		},
		{
			name:     "PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A},
			expected: ImageTypePNG,
		},
		{
			name:     "GIF",
			data:     []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
			expected: ImageTypeGIF,
		},
		{
			name:     "BMP",
			data:     []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00},
			expected: ImageTypeBMP,
		},
		{
			name:     "Unknown",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			expected: ImageTypeUnknown,
		},
		{
			name:     "Too short",
			data:     []byte{0xFF, 0xD8},
			expected: ImageTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.DetectImageType(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestImageValidator_ValidateImage(t *testing.T) {
	v := NewImageValidator()

	// Valid JPEG
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	err := v.ValidateImage(jpegData)
	if err != nil {
		t.Errorf("Expected valid JPEG to pass, got error: %v", err)
	}

	// Invalid image
	invalidData := []byte("not an image")
	err = v.ValidateImage(invalidData)
	if err == nil {
		t.Error("Expected error for invalid image data")
	}
}

func TestImageValidator_ValidateBase64Image(t *testing.T) {
	v := NewImageValidator()

	// Valid PNG
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	b64Data := base64.StdEncoding.EncodeToString(pngData)

	data, imgType, err := v.ValidateBase64Image(b64Data)
	if err != nil {
		t.Errorf("Expected valid PNG to pass, got error: %v", err)
	}
	if imgType != ImageTypePNG {
		t.Errorf("Expected ImageTypePNG, got %s", imgType)
	}
	if len(data) == 0 {
		t.Error("Expected decoded data to be non-empty")
	}
}

func TestImageValidator_ValidateBase64Image_InvalidBase64(t *testing.T) {
	v := NewImageValidator()

	_, _, err := v.ValidateBase64Image("invalid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestImageValidator_ValidateBase64Image_InvalidImage(t *testing.T) {
	v := NewImageValidator()

	invalidData := base64.StdEncoding.EncodeToString([]byte("not an image"))
	_, _, err := v.ValidateBase64Image(invalidData)
	if err == nil {
		t.Error("Expected error for invalid image content")
	}
}

func TestImageType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		imgType  ImageType
		expected string
	}{
		{"JPEG", ImageTypeJPEG, "jpeg"},
		{"PNG", ImageTypePNG, "png"},
		{"GIF", ImageTypeGIF, "gif"},
		{"BMP", ImageTypeBMP, "bmp"},
		{"Unknown", ImageTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.imgType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.imgType))
			}
		})
	}
}

// Raw Data Validator Tests

func TestNewRawDataValidator(t *testing.T) {
	v := NewRawDataValidator()
	if v == nil {
		t.Fatal("NewRawDataValidator returned nil")
	}
}

func TestRawDataValidator_DetectRawDataType(t *testing.T) {
	v := NewRawDataValidator()

	tests := []struct {
		name     string
		data     []byte
		expected RawDataType
	}{
		{
			name:     "ESC/POS",
			data:     []byte{0x1B, 0x40, 0x48, 0x65, 0x6C, 0x6C, 0x6F},
			expected: RawDataTypeESCPOS,
		},
		{
			name:     "ZPL",
			data:     []byte("^XA^FO50,50^A0N,50,50^FDTest^FS^XZ"),
			expected: RawDataTypeZPL,
		},
		{
			name:     "TSPL",
			data:     []byte("SIZE 4,3\nGAP 0,0\nCLS\nTEXT 100,100,\"3\",0,1,1,\"HELLO\"\nPRINT 1\n"),
			expected: RawDataTypeTSPL,
		},
		{
			name:     "Unknown",
			data:     []byte("unknown printer language"),
			expected: RawDataTypeUnknown,
		},
		{
			name:     "Too short",
			data:     []byte{0x00},
			expected: RawDataTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.DetectRawDataType(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRawDataType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		dataType RawDataType
		expected string
	}{
		{"ESCPOS", RawDataTypeESCPOS, "escpos"},
		{"ZPL", RawDataTypeZPL, "zpl"},
		{"TSPL", RawDataTypeTSPL, "tspl"},
		{"Unknown", RawDataTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dataType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.dataType))
			}
		})
	}
}

// min function test
func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkPDFValidator_ValidatePDF(b *testing.B) {
	v := NewPDFValidator()
	validPDF := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj
trailer
<< /Root 1 0 R >>
startxref
0
%%EOF`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidatePDF(validPDF)
	}
}

func BenchmarkImageValidator_DetectImageType(b *testing.B) {
	v := NewImageValidator()
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.DetectImageType(jpegData)
	}
}

func BenchmarkRawDataValidator_DetectRawDataType(b *testing.B) {
	v := NewRawDataValidator()
	zplData := []byte("^XA^FO50,50^A0N,50,50^FDTest^FS^XZ")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.DetectRawDataType(zplData)
	}
}
