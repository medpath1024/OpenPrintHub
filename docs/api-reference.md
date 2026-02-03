# API Reference

OpenPrintHub provides RESTful API and WebSocket interfaces.

**Base URL**: `http://localhost:16800`

## Printer Management

### Get Printer List

Get all available printers in the system.

```http
GET /v1/printers
```

**Response Example**:

```json
[
  {
    "id": "Brother_QL_820NWB",
    "name": "Brother QL-820NWB",
    "status": "Ready",
    "is_default": true
  },
  {
    "id": "Epson_TM_T88VI",
    "name": "Epson TM-T88VI",
    "status": "Offline",
    "is_default": false
  }
]
```

**Status Values**:

| Status | Description |
|--------|-------------|
| `Ready` | Ready to accept print jobs |
| `Busy` | Currently printing |
| `Offline` | Offline or disconnected |
| `Error` | Error state |
| `PaperOut` | Out of paper |
| `PaperJam` | Paper jam |

### Get Default Printer

```http
GET /v1/printers/default
```

**Response Example**:

```json
{
  "id": "Brother_QL_820NWB",
  "name": "Brother QL-820NWB",
  "status": "Ready",
  "is_default": true
}
```

### Get Printer Status

```http
GET /v1/printers/:id/status
```

**Path Parameters**:

| Parameter | Description |
|-----------|-------------|
| `id` | Printer ID (URL encoded) |

---

## Print Jobs

### Submit Print Job

```http
POST /v1/print
Content-Type: application/json
```

**Request Body**:

```json
{
  "printer": "Brother QL-820NWB",
  "type": "pdf",
  "data": "JVBERi0xLjQK...",
  "settings": {
    "copies": 1,
    "orientation": "portrait",
    "paper_size": "A4",
    "duplex": "none",
    "fit_to_page": true
  }
}
```

**Request Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `printer` | string | ✅ | Printer name or ID |
| `type` | string | ✅ | Print type: `pdf`, `raw`, `image` |
| `data` | string | ✅ | Base64 encoded print data |
| `settings` | object | ❌ | Print settings |

**Settings Parameters**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `copies` | int | 1 | Number of copies |
| `orientation` | string | portrait | Orientation: `portrait`, `landscape` |
| `paper_size` | string | - | Paper size: `A4`, `Letter`, `Label`, etc. |
| `duplex` | string | none | Duplex: `none`, `long-edge`, `short-edge` |
| `fit_to_page` | bool | true | Fit to page |

**Response Example**:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued"
}
```

### Get Job List

```http
GET /v1/jobs
GET /v1/jobs?history=true
```

**Query Parameters**:

| Parameter | Description |
|-----------|-------------|
| `history` | Set to `true` to return historical jobs |

### Get Job Status

```http
GET /v1/jobs/:id
```

**Response Example**:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "printer": "Brother QL-820NWB",
  "message": "Print completed",
  "started_at": "2026-01-30T10:30:00Z",
  "completed_at": "2026-01-30T10:30:02Z"
}
```

**Job Status Values**:

| Status | Description |
|--------|-------------|
| `queued` | Queued, waiting to be processed |
| `processing` | Being processed |
| `printing` | Currently printing |
| `completed` | Print completed |
| `failed` | Print failed |
| `cancelled` | Cancelled |

---

## WebSocket Real-time Updates

### Connection

```javascript
const ws = new WebSocket('ws://localhost:16800/v1/ws');
```

### Message Format

**Job Status Update**:

```json
{
  "type": "job_status",
  "data": {
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "message": "Print completed"
  }
}
```

### Example Code

```javascript
const ws = new WebSocket('ws://localhost:16800/v1/ws');

ws.onopen = () => {
  console.log('Connected to OpenPrintHub');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'job_status') {
    console.log(`Job ${message.data.job_id}: ${message.data.status}`);
    
    if (message.data.status === 'completed') {
      // Print successful, update UI
    } else if (message.data.status === 'failed') {
      // Print failed, show error
      console.error(message.data.message);
    }
  }
};

ws.onclose = () => {
  console.log('Connection closed, reconnecting in 3 seconds...');
  setTimeout(connectWebSocket, 3000);
};
```

---

## Other Endpoints

### Health Check

```http
GET /health
```

**Response**:

```json
{
  "status": "ok",
  "version": "0.1.5"
}
```

### Statistics

```http
GET /v1/stats
```

**Response**:

```json
{
  "queued_jobs": 2,
  "completed_jobs": 156,
  "failed_jobs": 3
}
```

---

## Error Handling

All error responses follow this format:

```json
{
  "error": "Error description message"
}
```

**HTTP Status Codes**:

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 202 | Accepted (job queued) |
| 400 | Bad request |
| 404 | Resource not found |
| 500 | Internal server error |

---

## Print Type Details

### PDF Printing

The most common print type, suitable for documents, reports, invoices, etc.

```javascript
// Convert PDF file to Base64
const pdfBase64 = btoa(pdfBinaryData);
// Or use FileReader
const reader = new FileReader();
reader.readAsDataURL(pdfFile);
reader.onload = () => {
  const base64 = reader.result.split(',')[1];
  // Send print request
};
```

### RAW Printing (ESC/POS)

Send printer commands directly, suitable for thermal receipt printers.

```javascript
// ESC/POS example: print and cut paper
const escpos = new Uint8Array([
  0x1B, 0x40,        // Initialize printer
  0x1B, 0x61, 0x01,  // Center align
  ...textToBytes('Welcome'),
  0x0A,              // Line feed
  0x1D, 0x56, 0x00   // Cut paper
]);

const base64 = btoa(String.fromCharCode(...escpos));

fetch('http://localhost:16800/v1/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    printer: 'Receipt Printer',
    type: 'raw',
    data: base64
  })
});
```

### Image Printing

Supports JPEG, PNG, GIF, BMP formats.

```javascript
// Convert image to Base64
const canvas = document.createElement('canvas');
// ... bindimage to canvas
const imageBase64 = canvas.toDataURL('image/png').split(',')[1];
```
