# CLAUDE.md

This file provides guidance for AI assistants working in this codebase.

## Project Overview

OpenPrintHub is a high-performance, cross-platform silent printing service written in Go. It serves as an open-source alternative to JSPrintManager, designed for SaaS developers to overcome browser printing limitations.

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **WebSocket**: gorilla/websocket
- **Frontend**: HTMX + vanilla CSS (admin dashboard)
- **Build**: Makefile

## Project Structure

```
cmd/oph/           # Application entry point
internal/
  api/             # REST API handlers, middleware, router, websocket
  print/           # Print job queue, PDF handling, job management
  printer/         # Platform-specific printer abstraction (darwin/linux/windows)
  web/             # Admin dashboard (templates, static assets)
docs/              # Documentation
build/             # Compiled binaries
```

## Build & Run Commands

```bash
make build         # Build for current platform
make build-all     # Build for all platforms (darwin, windows)
make run           # Run the application
make dev           # Run with hot reload (requires air)
make test          # Run tests
make clean         # Clean build artifacts
make fmt           # Format code
make lint          # Run linter (requires golangci-lint)
```

## Key APIs

- `GET /v1/printers` - List available printers
- `POST /v1/print` - Submit print job (PDF, raw, image)
- `GET /v1/jobs/:id` - Get job status
- `WS /v1/ws` - WebSocket for real-time status updates
- `/` - Admin dashboard (HTMX-powered)

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | 16800 | API server port |
| `-web-port` | port+1 | Web admin dashboard port |
| `-cors` | `*` | Allowed CORS origins |
| `-version` | - | Show version and exit |

## Platform Support

- **macOS**: Uses CUPS/lp command (`internal/printer/printer_darwin.go`)
- **Windows**: Uses winspool API (`internal/printer/printer_windows.go`)
- **Linux**: Uses CUPS/lp command (`internal/printer/printer_linux.go`)

## Development Guidelines

1. Platform-specific code goes in `printer_<platform>.go` files with build tags
2. All print jobs go through the queue system in `internal/print/queue.go`
3. Admin dashboard uses HTMX partials for dynamic updates
4. WebSocket broadcasts job status changes to connected clients
