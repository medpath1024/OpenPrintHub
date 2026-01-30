# Print Examples

This guide provides complete examples for different print types supported by OpenPrintHub.

## Table of Contents

- [PDF Printing](#pdf-printing)
- [ESC/POS Receipt Printing](#escpos-receipt-printing)
- [ZPL Label Printing](#zpl-label-printing)
- [TSPL Label Printing](#tspl-label-printing)
- [Image Printing](#image-printing)

---

## PDF Printing

PDF is the most common print type, suitable for invoices, reports, documents, etc.

### JavaScript/Browser

```javascript
// Method 1: From file input
async function printPDFFromFile(file, printerName) {
  const reader = new FileReader();
  
  return new Promise((resolve, reject) => {
    reader.onload = async () => {
      const base64 = reader.result.split(',')[1];
      
      const response = await fetch('http://localhost:16800/v1/print', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          printer: printerName,
          type: 'pdf',
          data: base64,
          settings: {
            copies: 1,
            orientation: 'portrait',
            paper_size: 'A4',
            fit_to_page: true
          }
        })
      });
      
      resolve(await response.json());
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

// Method 2: From URL
async function printPDFFromURL(pdfUrl, printerName) {
  // Fetch the PDF
  const response = await fetch(pdfUrl);
  const blob = await response.blob();
  
  // Convert to Base64
  const reader = new FileReader();
  return new Promise((resolve, reject) => {
    reader.onload = async () => {
      const base64 = reader.result.split(',')[1];
      
      const printResponse = await fetch('http://localhost:16800/v1/print', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          printer: printerName,
          type: 'pdf',
          data: base64
        })
      });
      
      resolve(await printResponse.json());
    };
    reader.onerror = reject;
    reader.readAsDataURL(blob);
  });
}

// Method 3: From ArrayBuffer/Uint8Array
async function printPDFFromBytes(pdfBytes, printerName) {
  const base64 = btoa(String.fromCharCode(...new Uint8Array(pdfBytes)));
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'pdf',
      data: base64,
      settings: {
        copies: 2,
        duplex: 'long-edge'  // Double-sided printing
      }
    })
  });
  
  return response.json();
}
```

### cURL

```bash
# Print a local PDF file
PDF_BASE64=$(base64 -i invoice.pdf)

curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d "{
    \"printer\": \"HP LaserJet Pro\",
    \"type\": \"pdf\",
    \"data\": \"$PDF_BASE64\",
    \"settings\": {
      \"copies\": 1,
      \"orientation\": \"portrait\"
    }
  }"
```

### Python

```python
import base64
import requests

def print_pdf(file_path: str, printer_name: str, copies: int = 1):
    """Print a PDF file to the specified printer."""
    
    # Read and encode PDF
    with open(file_path, 'rb') as f:
        pdf_base64 = base64.b64encode(f.read()).decode('utf-8')
    
    # Send print request
    response = requests.post(
        'http://localhost:16800/v1/print',
        json={
            'printer': printer_name,
            'type': 'pdf',
            'data': pdf_base64,
            'settings': {
                'copies': copies,
                'orientation': 'portrait',
                'fit_to_page': True
            }
        }
    )
    
    return response.json()

# Usage
result = print_pdf('invoice.pdf', 'HP LaserJet Pro', copies=2)
print(f"Job ID: {result['job_id']}")
```

### Node.js

```javascript
const fs = require('fs');
const fetch = require('node-fetch');

async function printPDF(filePath, printerName, options = {}) {
  // Read and encode PDF
  const pdfBuffer = fs.readFileSync(filePath);
  const base64 = pdfBuffer.toString('base64');
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'pdf',
      data: base64,
      settings: {
        copies: options.copies || 1,
        orientation: options.orientation || 'portrait',
        paper_size: options.paperSize || 'A4',
        ...options
      }
    })
  });
  
  return response.json();
}

// Usage
printPDF('./invoice.pdf', 'HP LaserJet Pro', { copies: 2 })
  .then(result => console.log('Job ID:', result.job_id));
```

---

## ESC/POS Receipt Printing

ESC/POS is a command language for thermal receipt printers (Epson, Star, etc.).

### JavaScript/Browser

```javascript
class ESCPOSBuilder {
  constructor() {
    this.commands = [];
  }
  
  // Initialize printer
  init() {
    this.commands.push(0x1B, 0x40);
    return this;
  }
  
  // Text alignment: 0=left, 1=center, 2=right
  align(position) {
    this.commands.push(0x1B, 0x61, position);
    return this;
  }
  
  // Text size: 0=normal, 1=double height, 16=double width, 17=double both
  textSize(size) {
    this.commands.push(0x1D, 0x21, size);
    return this;
  }
  
  // Bold: true/false
  bold(enabled) {
    this.commands.push(0x1B, 0x45, enabled ? 1 : 0);
    return this;
  }
  
  // Add text
  text(str) {
    const encoder = new TextEncoder();
    this.commands.push(...encoder.encode(str));
    return this;
  }
  
  // Line feed
  newline(count = 1) {
    for (let i = 0; i < count; i++) {
      this.commands.push(0x0A);
    }
    return this;
  }
  
  // Print horizontal line
  line(char = '-', width = 32) {
    this.text(char.repeat(width));
    return this.newline();
  }
  
  // Feed and cut paper
  cut() {
    this.commands.push(0x1D, 0x56, 0x00);
    return this;
  }
  
  // Open cash drawer
  openDrawer() {
    this.commands.push(0x1B, 0x70, 0x00, 0x19, 0xFA);
    return this;
  }
  
  // Build and return Base64
  toBase64() {
    const bytes = new Uint8Array(this.commands);
    return btoa(String.fromCharCode(...bytes));
  }
}

// Example: Print a receipt
async function printReceipt(printerName, items, total) {
  const receipt = new ESCPOSBuilder()
    .init()
    .align(1)  // Center
    .textSize(17)  // Double size
    .bold(true)
    .text('ACME STORE')
    .newline()
    .textSize(0)  // Normal size
    .bold(false)
    .text('123 Main Street')
    .newline()
    .text('Tel: (555) 123-4567')
    .newline(2)
    .align(0)  // Left
    .line('=')
    .text('SALES RECEIPT')
    .newline()
    .line('-');
  
  // Add items
  items.forEach(item => {
    const price = item.price.toFixed(2).padStart(8);
    const name = item.name.padEnd(22);
    receipt.text(`${name}$${price}`).newline();
  });
  
  receipt
    .line('-')
    .bold(true)
    .text(`TOTAL`.padEnd(22) + `$${total.toFixed(2).padStart(8)}`)
    .newline()
    .bold(false)
    .line('=')
    .newline()
    .align(1)
    .text('Thank you for shopping!')
    .newline()
    .text(new Date().toLocaleString())
    .newline(3)
    .cut();
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'raw',
      data: receipt.toBase64()
    })
  });
  
  return response.json();
}

// Usage
const items = [
  { name: 'Widget A', price: 9.99 },
  { name: 'Widget B', price: 14.50 },
  { name: 'Service Fee', price: 2.00 }
];
printReceipt('Epson TM-T88VI', items, 26.49);
```

### Python

```python
import base64
import requests

class ESCPOSBuilder:
    """Simple ESC/POS command builder."""
    
    def __init__(self):
        self.commands = bytearray()
    
    def init(self):
        """Initialize printer."""
        self.commands.extend([0x1B, 0x40])
        return self
    
    def align(self, position: int):
        """Set alignment: 0=left, 1=center, 2=right."""
        self.commands.extend([0x1B, 0x61, position])
        return self
    
    def bold(self, enabled: bool):
        """Enable/disable bold."""
        self.commands.extend([0x1B, 0x45, 1 if enabled else 0])
        return self
    
    def text_size(self, size: int):
        """Set text size."""
        self.commands.extend([0x1D, 0x21, size])
        return self
    
    def text(self, content: str):
        """Add text."""
        self.commands.extend(content.encode('utf-8'))
        return self
    
    def newline(self, count: int = 1):
        """Add line feeds."""
        self.commands.extend([0x0A] * count)
        return self
    
    def cut(self):
        """Cut paper."""
        self.commands.extend([0x1D, 0x56, 0x00])
        return self
    
    def to_base64(self) -> str:
        """Return commands as Base64."""
        return base64.b64encode(self.commands).decode('utf-8')


def print_receipt(printer_name: str, items: list, total: float):
    """Print a receipt with ESC/POS commands."""
    
    receipt = ESCPOSBuilder()
    receipt.init()
    receipt.align(1).text_size(17).bold(True)
    receipt.text('ACME STORE').newline()
    receipt.text_size(0).bold(False)
    receipt.text('123 Main Street').newline(2)
    
    receipt.align(0)
    receipt.text('-' * 32).newline()
    
    for item in items:
        line = f"{item['name']:<22}${item['price']:>7.2f}"
        receipt.text(line).newline()
    
    receipt.text('-' * 32).newline()
    receipt.bold(True)
    receipt.text(f"{'TOTAL':<22}${total:>7.2f}").newline()
    receipt.bold(False)
    receipt.newline(2).cut()
    
    response = requests.post(
        'http://localhost:16800/v1/print',
        json={
            'printer': printer_name,
            'type': 'raw',
            'data': receipt.to_base64()
        }
    )
    
    return response.json()


# Usage
items = [
    {'name': 'Coffee', 'price': 4.50},
    {'name': 'Sandwich', 'price': 8.99},
    {'name': 'Cookie', 'price': 2.50}
]
result = print_receipt('Epson TM-T88VI', items, 15.99)
print(f"Job ID: {result['job_id']}")
```

---

## ZPL Label Printing

ZPL (Zebra Programming Language) is used for Zebra label printers.

### JavaScript/Browser

```javascript
class ZPLBuilder {
  constructor() {
    this.commands = [];
  }
  
  // Start label format
  start() {
    this.commands.push('^XA');
    return this;
  }
  
  // End label format
  end() {
    this.commands.push('^XZ');
    return this;
  }
  
  // Set label home position
  labelHome(x, y) {
    this.commands.push(`^LH${x},${y}`);
    return this;
  }
  
  // Field origin
  fieldOrigin(x, y) {
    this.commands.push(`^FO${x},${y}`);
    return this;
  }
  
  // Font selection: A-Z, 0-9
  font(name, height, width = null) {
    const w = width || height;
    this.commands.push(`^A${name},${height},${w}`);
    return this;
  }
  
  // Field data
  fieldData(data) {
    this.commands.push(`^FD${data}^FS`);
    return this;
  }
  
  // Barcode Code128
  barcode128(height, data, showText = true) {
    this.commands.push(`^BY2`);
    this.commands.push(`^BC,${height},${showText ? 'Y' : 'N'},N,N`);
    this.commands.push(`^FD${data}^FS`);
    return this;
  }
  
  // QR Code
  qrCode(data, magnification = 4) {
    this.commands.push(`^BQN,2,${magnification}`);
    this.commands.push(`^FDQA,${data}^FS`);
    return this;
  }
  
  // Graphic box (rectangle)
  box(width, height, thickness = 1) {
    this.commands.push(`^GB${width},${height},${thickness}^FS`);
    return this;
  }
  
  // Print quantity
  printQuantity(qty) {
    this.commands.push(`^PQ${qty}`);
    return this;
  }
  
  // Build ZPL string
  build() {
    return this.commands.join('\n');
  }
  
  // Build and return Base64
  toBase64() {
    return btoa(this.build());
  }
}

// Example: Print shipping label
async function printShippingLabel(printerName, order) {
  const label = new ZPLBuilder()
    .start()
    .labelHome(0, 0)
    
    // Company logo area (placeholder box)
    .fieldOrigin(30, 30)
    .box(150, 80, 2)
    
    // Company name
    .fieldOrigin(200, 50)
    .font('0', 40)
    .fieldData('ACME Shipping')
    
    // Divider line
    .fieldOrigin(30, 130)
    .box(700, 2, 2)
    
    // FROM address
    .fieldOrigin(30, 150)
    .font('0', 25)
    .fieldData('FROM:')
    .fieldOrigin(30, 180)
    .font('0', 22)
    .fieldData(order.from.name)
    .fieldOrigin(30, 205)
    .fieldData(order.from.address)
    .fieldOrigin(30, 230)
    .fieldData(`${order.from.city}, ${order.from.state} ${order.from.zip}`)
    
    // TO address (larger font)
    .fieldOrigin(30, 290)
    .font('0', 30)
    .fieldData('SHIP TO:')
    .fieldOrigin(30, 330)
    .font('0', 35)
    .fieldData(order.to.name)
    .fieldOrigin(30, 375)
    .font('0', 28)
    .fieldData(order.to.address)
    .fieldOrigin(30, 410)
    .fieldData(`${order.to.city}, ${order.to.state} ${order.to.zip}`)
    
    // Barcode
    .fieldOrigin(30, 480)
    .barcode128(80, order.trackingNumber)
    
    // QR Code
    .fieldOrigin(550, 300)
    .qrCode(order.trackingNumber, 5)
    
    // Order ID
    .fieldOrigin(550, 480)
    .font('0', 25)
    .fieldData(`Order: ${order.orderId}`)
    
    .printQuantity(1)
    .end();
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'raw',
      data: label.toBase64()
    })
  });
  
  return response.json();
}

// Usage
const order = {
  orderId: 'ORD-2026-00123',
  trackingNumber: '1Z999AA10123456784',
  from: {
    name: 'ACME Warehouse',
    address: '100 Industrial Blvd',
    city: 'Chicago',
    state: 'IL',
    zip: '60601'
  },
  to: {
    name: 'John Smith',
    address: '456 Oak Street, Apt 7',
    city: 'New York',
    state: 'NY',
    zip: '10001'
  }
};

printShippingLabel('Zebra ZD420', order);
```

### cURL

```bash
# Simple ZPL label
ZPL_CONTENT=$(cat << 'EOF'
^XA
^FO50,50^A0N,50,50^FDHello World^FS
^FO50,120^BY2^BCN,100,Y,N,N^FD123456789^FS
^XZ
EOF
)

ZPL_BASE64=$(echo -n "$ZPL_CONTENT" | base64)

curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d "{
    \"printer\": \"Zebra ZD420\",
    \"type\": \"raw\",
    \"data\": \"$ZPL_BASE64\"
  }"
```

---

## TSPL Label Printing

TSPL (TSC Printer Language) is used for TSC label printers.

### JavaScript/Browser

```javascript
class TSPLBuilder {
  constructor(width, height, gap = 3) {
    this.commands = [];
    // Set label size (mm)
    this.commands.push(`SIZE ${width} mm, ${height} mm`);
    this.commands.push(`GAP ${gap} mm, 0 mm`);
  }
  
  // Clear buffer
  clear() {
    this.commands.push('CLS');
    return this;
  }
  
  // Set print direction
  direction(dir = 0) {
    this.commands.push(`DIRECTION ${dir}`);
    return this;
  }
  
  // Text command
  text(x, y, font, rotation, xMul, yMul, content) {
    this.commands.push(`TEXT ${x},${y},"${font}",${rotation},${xMul},${yMul},"${content}"`);
    return this;
  }
  
  // Barcode 128
  barcode128(x, y, height, content, readable = 1) {
    this.commands.push(`BARCODE ${x},${y},"128",${height},${readable},0,2,2,"${content}"`);
    return this;
  }
  
  // QR Code
  qrCode(x, y, content, cellWidth = 6) {
    this.commands.push(`QRCODE ${x},${y},H,${cellWidth},A,0,"${content}"`);
    return this;
  }
  
  // Draw box
  box(x, y, width, height, thickness = 1) {
    this.commands.push(`BOX ${x},${y},${x + width},${y + height},${thickness}`);
    return this;
  }
  
  // Print labels
  print(copies = 1) {
    this.commands.push(`PRINT ${copies}`);
    return this;
  }
  
  // Build TSPL string
  build() {
    return this.commands.join('\r\n');
  }
  
  // Build and return Base64
  toBase64() {
    return btoa(this.build());
  }
}

// Example: Print product label
async function printProductLabel(printerName, product, quantity = 1) {
  const label = new TSPLBuilder(50, 30)  // 50mm x 30mm label
    .clear()
    .direction(0)
    
    // Product name
    .text(20, 20, '3', 0, 1, 1, product.name)
    
    // SKU
    .text(20, 60, '2', 0, 1, 1, `SKU: ${product.sku}`)
    
    // Price
    .text(20, 90, '4', 0, 1, 1, `$${product.price.toFixed(2)}`)
    
    // Barcode
    .barcode128(20, 130, 60, product.barcode)
    
    // QR Code with product URL
    .qrCode(280, 20, product.url, 4)
    
    .print(quantity);
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'raw',
      data: label.toBase64()
    })
  });
  
  return response.json();
}

// Usage
const product = {
  name: 'Wireless Mouse',
  sku: 'WM-2026-BLK',
  price: 29.99,
  barcode: '4901234567890',
  url: 'https://shop.example.com/p/wm2026'
};

printProductLabel('TSC TTP-244', product, 2);
```

### Python

```python
import base64
import requests

class TSPLBuilder:
    """TSPL command builder for TSC printers."""
    
    def __init__(self, width_mm: int, height_mm: int, gap_mm: int = 3):
        self.commands = [
            f'SIZE {width_mm} mm, {height_mm} mm',
            f'GAP {gap_mm} mm, 0 mm'
        ]
    
    def clear(self):
        self.commands.append('CLS')
        return self
    
    def direction(self, d: int = 0):
        self.commands.append(f'DIRECTION {d}')
        return self
    
    def text(self, x: int, y: int, font: str, rotation: int, 
             x_mul: int, y_mul: int, content: str):
        self.commands.append(
            f'TEXT {x},{y},"{font}",{rotation},{x_mul},{y_mul},"{content}"'
        )
        return self
    
    def barcode128(self, x: int, y: int, height: int, 
                   content: str, readable: int = 1):
        self.commands.append(
            f'BARCODE {x},{y},"128",{height},{readable},0,2,2,"{content}"'
        )
        return self
    
    def qrcode(self, x: int, y: int, content: str, cell_width: int = 6):
        self.commands.append(
            f'QRCODE {x},{y},H,{cell_width},A,0,"{content}"'
        )
        return self
    
    def print_label(self, copies: int = 1):
        self.commands.append(f'PRINT {copies}')
        return self
    
    def to_base64(self) -> str:
        content = '\r\n'.join(self.commands)
        return base64.b64encode(content.encode()).decode()


def print_inventory_label(printer_name: str, item: dict, qty: int = 1):
    """Print inventory label using TSPL."""
    
    label = TSPLBuilder(60, 40)
    label.clear().direction(0)
    label.text(20, 20, '3', 0, 1, 1, item['name'][:20])
    label.text(20, 60, '2', 0, 1, 1, f"Loc: {item['location']}")
    label.text(20, 90, '2', 0, 1, 1, f"Qty: {item['quantity']}")
    label.barcode128(20, 130, 50, item['barcode'])
    label.print_label(qty)
    
    response = requests.post(
        'http://localhost:16800/v1/print',
        json={
            'printer': printer_name,
            'type': 'raw',
            'data': label.to_base64()
        }
    )
    
    return response.json()


# Usage
item = {
    'name': 'Resistor 10K Ohm',
    'location': 'A-12-3',
    'quantity': 500,
    'barcode': '1234567890123'
}
result = print_inventory_label('TSC TTP-244', item, qty=2)
print(f"Job ID: {result['job_id']}")
```

---

## Image Printing

Print images directly (JPEG, PNG, GIF, BMP).

### JavaScript/Browser

```javascript
// Method 1: From file input
async function printImageFromFile(file, printerName) {
  const reader = new FileReader();
  
  return new Promise((resolve, reject) => {
    reader.onload = async () => {
      const base64 = reader.result.split(',')[1];
      
      const response = await fetch('http://localhost:16800/v1/print', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          printer: printerName,
          type: 'image',
          data: base64,
          settings: {
            fit_to_page: true
          }
        })
      });
      
      resolve(await response.json());
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

// Method 2: From canvas
async function printCanvas(canvas, printerName) {
  // Get image data as PNG Base64
  const dataUrl = canvas.toDataURL('image/png');
  const base64 = dataUrl.split(',')[1];
  
  const response = await fetch('http://localhost:16800/v1/print', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      printer: printerName,
      type: 'image',
      data: base64,
      settings: {
        copies: 1,
        orientation: 'portrait'
      }
    })
  });
  
  return response.json();
}

// Method 3: From image URL
async function printImageFromURL(imageUrl, printerName) {
  // Create image element
  const img = new Image();
  img.crossOrigin = 'anonymous';
  
  return new Promise((resolve, reject) => {
    img.onload = async () => {
      // Draw to canvas
      const canvas = document.createElement('canvas');
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext('2d');
      ctx.drawImage(img, 0, 0);
      
      // Get Base64
      const base64 = canvas.toDataURL('image/png').split(',')[1];
      
      const response = await fetch('http://localhost:16800/v1/print', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          printer: printerName,
          type: 'image',
          data: base64
        })
      });
      
      resolve(await response.json());
    };
    img.onerror = reject;
    img.src = imageUrl;
  });
}
```

### Python

```python
import base64
import requests

def print_image(file_path: str, printer_name: str, fit_to_page: bool = True):
    """Print an image file."""
    
    with open(file_path, 'rb') as f:
        image_base64 = base64.b64encode(f.read()).decode('utf-8')
    
    response = requests.post(
        'http://localhost:16800/v1/print',
        json={
            'printer': printer_name,
            'type': 'image',
            'data': image_base64,
            'settings': {
                'fit_to_page': fit_to_page,
                'orientation': 'portrait'
            }
        }
    )
    
    return response.json()


# Usage
result = print_image('photo.jpg', 'Canon PIXMA', fit_to_page=True)
print(f"Job ID: {result['job_id']}")
```

### cURL

```bash
# Print an image file
IMG_BASE64=$(base64 -i photo.png)

curl -X POST http://localhost:16800/v1/print \
  -H "Content-Type: application/json" \
  -d "{
    \"printer\": \"Canon PIXMA\",
    \"type\": \"image\",
    \"data\": \"$IMG_BASE64\",
    \"settings\": {
      \"fit_to_page\": true
    }
  }"
```

---

## WebSocket Job Monitoring

Monitor print job status in real-time.

```javascript
class PrintJobMonitor {
  constructor(baseUrl = 'localhost:16800') {
    this.ws = null;
    this.baseUrl = baseUrl;
    this.callbacks = new Map();
  }
  
  connect() {
    this.ws = new WebSocket(`ws://${this.baseUrl}/v1/ws`);
    
    this.ws.onopen = () => {
      console.log('Connected to OpenPrintHub');
    };
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'job_status') {
        const { job_id, status, message: msg } = message.data;
        
        // Call registered callback
        const callback = this.callbacks.get(job_id);
        if (callback) {
          callback(status, msg);
          
          // Remove callback when job is done
          if (['completed', 'failed', 'cancelled'].includes(status)) {
            this.callbacks.delete(job_id);
          }
        }
      }
    };
    
    this.ws.onclose = () => {
      console.log('Disconnected, reconnecting in 3s...');
      setTimeout(() => this.connect(), 3000);
    };
  }
  
  // Register callback for a job
  watch(jobId, callback) {
    this.callbacks.set(jobId, callback);
  }
  
  // Print and watch
  async printAndWatch(printRequest) {
    const response = await fetch(`http://${this.baseUrl}/v1/print`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(printRequest)
    });
    
    const result = await response.json();
    
    return new Promise((resolve, reject) => {
      this.watch(result.job_id, (status, message) => {
        if (status === 'completed') {
          resolve({ job_id: result.job_id, status, message });
        } else if (status === 'failed') {
          reject(new Error(message));
        }
      });
    });
  }
}

// Usage
const monitor = new PrintJobMonitor();
monitor.connect();

// Print and wait for completion
try {
  const result = await monitor.printAndWatch({
    printer: 'HP LaserJet',
    type: 'pdf',
    data: pdfBase64
  });
  console.log('Print completed:', result);
} catch (error) {
  console.error('Print failed:', error.message);
}
```
