# OpenPrintHub Examples

This directory contains example scripts demonstrating how to use OpenPrintHub for various printing tasks.

## Prerequisites

- OpenPrintHub running on `http://localhost:16800`
- Python 3.7+ with `requests` library (for Python examples)

```bash
pip install requests

# Optional: for better quality sample PDFs
pip install reportlab
```

## Examples

### Generate Sample PDF

First, generate a sample invoice PDF for testing:

```bash
# Generate invoice.pdf (uses reportlab if available, otherwise minimal PDF)
python generate_invoice.py

# Or specify output filename
python generate_invoice.py my_invoice.pdf

# Optional: Install reportlab for better quality PDFs
pip install reportlab
```

### PDF Printing

Print PDF documents with customizable settings.

```bash
# Generate and print
python generate_invoice.py invoice.pdf
python print_pdf.py invoice.pdf

# Print with specific printer and copies
python print_pdf.py report.pdf "HP LaserJet Pro" 2
```

### Receipt Printing (ESC/POS)

Print thermal receipts using ESC/POS commands for receipt printers (Epson, Star, etc.).

```bash
python print_receipt.py
python print_receipt.py "Epson TM-T88VI"
```

### Label Printing (ZPL)

Print shipping labels using ZPL for Zebra label printers.

```bash
python print_label_zpl.py
python print_label_zpl.py "Zebra ZD420"
```

### Label Printing (TSPL)

Print product/inventory labels using TSPL for TSC label printers.

```bash
python print_label_tspl.py
python print_label_tspl.py "TSC TTP-244"
```

### Interactive Web Demo

Open `demo.html` in a browser for an interactive demo that supports:

- PDF printing (file upload)
- Receipt printing (ESC/POS)
- Label printing (ZPL)
- Raw command sending
- Real-time job status via WebSocket

```bash
# On macOS
open demo.html

# On Windows
start demo.html

# On Linux
xdg-open demo.html
```

## API Quick Reference

### List Printers

```bash
curl http://localhost:16800/v1/printers
```

### Print PDF

```bash
PDF_BASE64=$(base64 -i document.pdf)
curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d "{
    \"printer\": \"Printer Name\",
    \"type\": \"pdf\",
    \"data\": \"$PDF_BASE64\"
  }"
```

### Print Raw Commands

```bash
curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d "{
    \"printer\": \"Printer Name\",
    \"type\": \"raw\",
    \"data\": \"$(echo -n '^XA^FO50,50^A0N,50,50^FDHello^FS^XZ' | base64)\"
  }"
```

### Check Job Status

```bash
curl http://localhost:16800/v1/jobs/{job_id}
```

## Print Types

| Type | Use Case | Printers |
|------|----------|----------|
| `pdf` | Documents, reports, invoices | Laser, inkjet printers |
| `raw` | ESC/POS, ZPL, TSPL commands | Thermal receipt, label printers |
| `image` | Photos, graphics | Photo printers, inkjet |

## Supported Command Languages

| Language | Description | Common Printers |
|----------|-------------|-----------------|
| ESC/POS | Thermal receipt printing | Epson TM series, Star TSP series |
| ZPL | Zebra label printing | Zebra ZD, ZT, GK series |
| TSPL | TSC label printing | TSC TTP, TE, TX series |

## More Information

- [Full Documentation](../docs/README.md)
- [API Reference](../docs/api-reference.md)
- [Detailed Examples](../docs/examples.md)
