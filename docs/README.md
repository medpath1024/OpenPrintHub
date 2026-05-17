# OpenPrintHub Documentation

OpenPrintHub (OPH) is a high-performance, cross-platform silent printing service designed for SaaS developers.

## Table of Contents

- [Getting Started](./getting-started.md)
- [API Reference](./api-reference.md)
- [Configuration Guide](./configuration.md)
- [Print Examples](./examples.md)
- [FAQ](./faq.md)
- [DYMO Label Printing & JSPM Compatibility Design](./dymo-jsprintmanager-design.md)

## Why OpenPrintHub?

| Feature | OpenPrintHub | Traditional Solutions |
|---------|--------------|----------------------|
| Silent Printing | ✅ Supported | ❌ Requires user confirmation |
| Cross-Platform | ✅ Windows/macOS | ⚠️ Limited |
| Deployment | Single binary | Requires runtime installation |
| Resource Usage | < 20MB memory | Usually higher |
| Open Source | ✅ MIT License | Mostly commercial/paid |

## Supported Print Types

- **PDF Documents** - Invoices, reports, prescriptions, etc.
- **Thermal Receipts** - ESC/POS commands (POS terminals, queue machines)
- **Label Printing** - ZPL/TSPL commands (shipping labels, medication labels)
- **Image Printing** - JPEG, PNG, and other formats
