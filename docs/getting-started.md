# Getting Started

This guide will help you install and configure OpenPrintHub in 5 minutes.

## System Requirements

- **macOS**: 10.15 (Catalina) or later
- **Windows**: Windows 10 or later
- At least one configured printer

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Go to the [Releases page](https://github.com/medpath1024/OpenPrintHub/releases)
2. Download the version for your system:
   - macOS Intel: `oph-darwin-amd64`
   - macOS Apple Silicon: `oph-darwin-arm64`
   - Windows: `oph-windows-amd64.exe`

3. Set execute permission (macOS only):

```bash
chmod +x oph-darwin-*
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/medpath1024/OpenPrintHub.git
cd OpenPrintHub

# Build
make build

# Run
./build/oph
```

## Running

### Basic Startup

```bash
# macOS
./oph-darwin-arm64

# Windows
oph-windows-amd64.exe
```

### Custom Port

```bash
./oph -port 8080
```

### Specify CORS Origins

```bash
./oph -cors "https://your-app.com,https://admin.your-app.com"
```

## Verify Installation

After starting, open your browser and visit:

```
http://localhost:16800
```

You should see the OpenPrintHub admin interface displaying a list of connected printers.

## Test Printing

### Using cURL to Test the API

```bash
# Get service info
curl http://localhost:16800/v1/info

# Get printer list
curl http://localhost:16800/v1/printers

# Submit a test print (replace BASE64_PDF_DATA with actual data)
curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d '{
    "printer": "Your Printer Name",
    "type": "pdf",
    "data": "BASE64_PDF_DATA",
    "name": "test-document.pdf",
    "settings": {"copies": 1}
  }'

# Submit batch print
curl -X POST http://localhost:16800/v1/print/batch \
  -H "Content-Type: application/json" \
  -d '{
    "printer": "Your Printer Name",
    "jobs": [
      {"type": "pdf", "data": "BASE64_PDF_DATA", "name": "doc1.pdf"},
      {"type": "pdf", "data": "BASE64_PDF_DATA", "name": "doc2.pdf"}
    ]
  }'

# Cancel a job
curl -X POST http://localhost:16800/v1/jobs/{job_id}/cancel
```

### JavaScript Integration

```javascript
// Get service info
const infoResponse = await fetch('http://localhost:16800/v1/info');
const info = await infoResponse.json();
console.log('OpenPrintHub version:', info.version);

// Get printer list
const response = await fetch('http://localhost:16800/v1/printers');
const printers = await response.json();
console.log('Available printers:', printers);

// Print PDF with name
const printResponse = await fetch('http://localhost:16800/v1/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    printer: printers[0].name,
    type: 'pdf',
    data: btoa(pdfBinaryData), // Base64 encode
    name: 'my-document.pdf',   // Job name for identification
    settings: { copies: 1 }
  })
});
const result = await printResponse.json();
console.log('Print job:', result.job_id);

// Batch print multiple documents
const batchResponse = await fetch('http://localhost:16800/v1/print/batch', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    printer: printers[0].name,
    jobs: [
      { type: 'pdf', data: btoa(pdf1Data), name: 'invoice-001.pdf' },
      { type: 'pdf', data: btoa(pdf2Data), name: 'invoice-002.pdf' }
    ]
  })
});
const batchResult = await batchResponse.json();
console.log('Batch submitted:', batchResult.batch_id);

// Cancel a job
const cancelResponse = await fetch(`http://localhost:16800/v1/jobs/${result.job_id}/cancel`, {
  method: 'POST'
});
const cancelResult = await cancelResponse.json();
console.log('Job cancelled:', cancelResult.status);
```

## Next Steps

- [API Reference](./api-reference.md) - Complete API documentation
- [Configuration Guide](./configuration.md) - Advanced configuration options
- [FAQ](./faq.md) - Frequently asked questions
