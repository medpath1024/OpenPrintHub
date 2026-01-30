# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Linux support (CUPS)
- ZPL/TSPL command parsing
- HTTPS support with self-signed certificates
- Print job persistence (SQLite)

## [0.1.0] - 2026-01-30

### Added
- Initial release
- Cross-platform printer management (macOS/Windows)
- PDF silent printing
- RAW command pass-through (ESC/POS support)
- RESTful API (`/v1/printers`, `/v1/print`, `/v1/jobs`)
- WebSocket real-time status updates
- HTMX-based admin dashboard
- CORS support with configurable origins
- Single binary deployment (Go embed)

### API Endpoints
- `GET /v1/printers` - List available printers
- `GET /v1/printers/default` - Get default printer
- `GET /v1/printers/:id/status` - Get printer status
- `POST /v1/print` - Submit print job
- `GET /v1/jobs` - List print jobs
- `GET /v1/jobs/:id` - Get job status
- `WS /v1/ws` - WebSocket status updates

### Supported Platforms
- macOS 10.15+ (Intel & Apple Silicon)
- Windows 10+

### Print Types
- PDF (Base64 encoded)
- RAW (ESC/POS commands)
- Image (JPEG, PNG, GIF, BMP)

[Unreleased]: https://github.com/medpath1024/OpenPrintHub/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/medpath1024/OpenPrintHub/releases/tag/v0.1.0
