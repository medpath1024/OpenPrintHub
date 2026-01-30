# OpenPrintHub

A high-performance, cross-platform silent printing service written in Go. An open-source alternative to JSPrintManager, designed for SaaS developers to overcome browser printing limitations.

## Features

- **Silent Printing**: Print PDF documents without user interaction
- **Cross-Platform**: Supports Windows, macOS, and Linux
- **RESTful API**: Easy integration with any web application
- **WebSocket**: Real-time print job status updates
- **Admin Dashboard**: Built-in web interface for printer management
- **Lightweight**: Single binary, < 20MB memory footprint

## Quick Start

### From Source

```bash
# Clone the repository
git clone https://github.com/medpath1024/OpenPrintHub.git
cd OpenPrintHub

# Build
make build

# Run
./build/oph
```

### Usage

Once running, OpenPrintHub listens on two ports:

- **API Server**: `http://localhost:16800` - For application integration
- **Admin Dashboard**: `http://localhost:16801` - Web interface for printer management

See [API Reference](#api-reference) below for API documentation.

## API Reference

### List Printers

```http
GET /v1/printers
```

Response:
```json
[
  {
    "id": "printer_1",
    "name": "Brother QL-820NWB",
    "status": "Ready",
    "is_default": true
  }
]
```

### Submit Print Job

```http
POST /v1/print
Content-Type: application/json

{
  "printer": "Brother QL-820NWB",
  "type": "pdf",
  "data": "JVBERi0xLjQK...",
  "settings": {
    "copies": 1,
    "orientation": "portrait"
  }
}
```

Response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued"
}
```

### Get Job Status

```http
GET /v1/jobs/:id
```

### WebSocket Status Updates

```javascript
const ws = new WebSocket('ws://localhost:16800/v1/ws');
ws.onmessage = (event) => {
  const status = JSON.parse(event.data);
  console.log('Job status:', status);
};
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | 16800 | API server port |
| `-web-port` | port+1 | Web admin dashboard port (default: 16801) |
| `-cors` | `*` | Allowed CORS origins (comma-separated) |
| `-version` | - | Show version and exit |

## Supported Print Types

| Type | Description |
|------|-------------|
| `pdf` | PDF document (Base64 encoded) |
| `raw` | Raw printer commands (ESC/POS, ZPL, TSPL) |
| `image` | Image file (Base64 encoded) |

## Platform Support

| Platform | Print Method | Status |
|----------|--------------|--------|
| macOS | CUPS/lp command | ✅ Supported |
| Windows | winspool API | ✅ Supported |
| Linux | CUPS/lp command | ✅ Supported |

## Development

```bash
# Run tests
make test

# Build for all platforms
make build-all

# Run in development mode
make dev
```

## License

MIT License - See [LICENSE](LICENSE) for details.

## Acknowledgments

OpenPrintHub is designed as an open-source alternative to commercial solutions like JSPrintManager, providing modern web developers with a free, lightweight option for silent printing capabilities.
