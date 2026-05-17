#!/usr/bin/env python3
"""
OpenPrintHub - TSPL Label Printing Example

Usage:
    python print_label_tspl.py [printer_name]

This script demonstrates how to print labels using TSPL (TSC Printer Language)
for TSC label printers.
"""

import base64
import os
import sys
import requests

def _normalize_base_url(url: str) -> str:
    url = (url or "").strip().rstrip("/")
    if not url:
        return "http://localhost:16800"
    if "://" not in url:
        url = "http://" + url
    return url


OPH_URL = _normalize_base_url(os.environ.get("OPH_URL", "http://localhost:16800"))


class TSPLBuilder:
    """TSPL command builder for TSC label printers."""

    def __init__(self, width_mm: int, height_mm: int, gap_mm: int = 3):
        """
        Initialize TSPL builder.

        Args:
            width_mm: Label width in millimeters
            height_mm: Label height in millimeters
            gap_mm: Gap between labels in millimeters
        """
        self.commands = [
            f'SIZE {width_mm} mm, {height_mm} mm',
            f'GAP {gap_mm} mm, 0 mm'
        ]

    def clear(self):
        """Clear image buffer."""
        self.commands.append('CLS')
        return self

    def direction(self, d: int = 0):
        """
        Set print direction.
        0 = Normal, 1 = Reversed
        """
        self.commands.append(f'DIRECTION {d}')
        return self

    def density(self, level: int = 8):
        """
        Set print density (darkness).
        level: 0-15
        """
        self.commands.append(f'DENSITY {level}')
        return self

    def speed(self, level: int = 4):
        """
        Set print speed.
        level: 1-6
        """
        self.commands.append(f'SPEED {level}')
        return self

    def text(self, x: int, y: int, font: str, rotation: int,
             x_mul: int, y_mul: int, content: str):
        """
        Add text.

        Args:
            x, y: Position in dots
            font: Font name ("1"-"8", "TSS24.BF2", etc.)
            rotation: 0, 90, 180, or 270
            x_mul, y_mul: Horizontal/vertical magnification
            content: Text content
        """
        self.commands.append(
            f'TEXT {x},{y},"{font}",{rotation},{x_mul},{y_mul},"{content}"'
        )
        return self

    def barcode_128(self, x: int, y: int, height: int, content: str,
                    readable: int = 1, rotation: int = 0):
        """
        Add Code 128 barcode.

        Args:
            x, y: Position in dots
            height: Barcode height in dots
            content: Barcode data
            readable: 0=No text, 1=Show text
            rotation: 0, 90, 180, or 270
        """
        self.commands.append(
            f'BARCODE {x},{y},"128",{height},{readable},{rotation},2,2,"{content}"'
        )
        return self

    def barcode_ean13(self, x: int, y: int, height: int, content: str,
                      readable: int = 1, rotation: int = 0):
        """Add EAN-13 barcode."""
        self.commands.append(
            f'BARCODE {x},{y},"EAN13",{height},{readable},{rotation},2,2,"{content}"'
        )
        return self

    def qrcode(self, x: int, y: int, content: str,
               cell_width: int = 6, ecc: str = 'H', rotation: int = 0):
        """
        Add QR code.

        Args:
            x, y: Position in dots
            content: QR code data
            cell_width: Module size (1-10)
            ecc: Error correction level (L, M, Q, H)
            rotation: 0, 90, 180, or 270
        """
        self.commands.append(
            f'QRCODE {x},{y},{ecc},{cell_width},A,{rotation},"{content}"'
        )
        return self

    def box(self, x: int, y: int, width: int, height: int, thickness: int = 1):
        """Draw a rectangle."""
        x_end = x + width
        y_end = y + height
        self.commands.append(f'BOX {x},{y},{x_end},{y_end},{thickness}')
        return self

    def line(self, x1: int, y1: int, x2: int, y2: int, thickness: int = 1):
        """Draw a line."""
        self.commands.append(f'BAR {x1},{y1},{x2 - x1},{thickness}')
        return self

    def print_label(self, copies: int = 1, sets: int = 1):
        """
        Print labels.

        Args:
            copies: Number of copies per set
            sets: Number of sets
        """
        self.commands.append(f'PRINT {sets},{copies}')
        return self

    def build(self) -> str:
        """Build TSPL command string."""
        return '\r\n'.join(self.commands)

    def to_base64(self) -> str:
        """Return TSPL as Base64 encoded string."""
        return base64.b64encode(self.build().encode()).decode()


def print_product_label(printer_name: str, product: dict, quantity: int = 1):
    """Print a product label."""

    # 60mm x 40mm label
    label = TSPLBuilder(60, 40)
    label.clear()
    label.direction(0)
    label.density(10)
    label.speed(4)

    # Border
    label.box(10, 10, 460, 300, 2)

    # Product name (large)
    label.text(20, 25, '4', 0, 1, 1, product['name'][:25])

    # Category
    label.text(20, 70, '2', 0, 1, 1, f"Category: {product['category']}")

    # SKU
    label.text(20, 100, '3', 0, 1, 1, f"SKU: {product['sku']}")

    # Price (large, bold effect using size)
    label.text(20, 140, '5', 0, 1, 1, f"${product['price']:.2f}")

    # Barcode
    label.barcode_128(20, 190, 60, product['barcode'])

    # QR Code (product URL)
    label.qrcode(350, 70, product['url'], 4)

    # Date
    label.text(350, 200, '1', 0, 1, 1, product['date'])

    label.print_label(quantity)

    # Send to printer
    response = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            'printer': printer_name,
            'type': 'raw',
            'data': label.to_base64()
        }
    )
    response.raise_for_status()
    return response.json()


def print_inventory_label(printer_name: str, item: dict, quantity: int = 1):
    """Print an inventory/warehouse label."""

    # 50mm x 30mm label
    label = TSPLBuilder(50, 30)
    label.clear()
    label.direction(0)

    # Item name
    label.text(15, 15, '3', 0, 1, 1, item['name'][:18])

    # Location
    label.text(15, 55, '2', 0, 1, 1, f"Loc: {item['location']}")

    # Quantity
    label.text(15, 85, '2', 0, 1, 1, f"Qty: {item['quantity']}")

    # Barcode
    label.barcode_128(15, 120, 50, item['barcode'])

    # QR with full info
    qr_data = f"{item['barcode']}|{item['location']}|{item['quantity']}"
    label.qrcode(300, 15, qr_data, 3)

    label.print_label(quantity)

    response = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            'printer': printer_name,
            'type': 'raw',
            'data': label.to_base64()
        }
    )
    response.raise_for_status()
    return response.json()


def get_printers():
    """Get list of available printers."""
    response = requests.get(f"{OPH_URL}/v1/printers")
    response.raise_for_status()
    return response.json()


def main():
    from datetime import datetime

    # Sample data
    product = {
        'name': 'Wireless Bluetooth Mouse',
        'category': 'Electronics',
        'sku': 'WBM-2026-BLK',
        'price': 29.99,
        'barcode': '4901234567890',
        'url': 'https://shop.example.com/p/wbm2026',
        'date': datetime.now().strftime('%Y-%m-%d')
    }

    inventory_item = {
        'name': 'Resistor 10K Ohm',
        'location': 'A-12-3',
        'quantity': 500,
        'barcode': '1234567890123'
    }

    # Get printer
    if len(sys.argv) > 1:
        printer_name = sys.argv[1]
    else:
        printers = get_printers()
        if not printers:
            print("Error: No printers available")
            sys.exit(1)

        print("Available printers:")
        for i, p in enumerate(printers):
            default = " (default)" if p.get('is_default') else ""
            print(f"  {i + 1}. {p['name']}{default}")

        choice = input("\nSelect printer number: ").strip()

        try:
            printer_name = printers[int(choice) - 1]['name']
        except (ValueError, IndexError):
            print("Invalid selection")
            sys.exit(1)

    print("\nLabel types:")
    print("  1. Product label (60x40mm)")
    print("  2. Inventory label (50x30mm)")

    label_type = input("\nSelect label type: ").strip()

    try:
        if label_type == '1':
            result = print_product_label(printer_name, product)
            print(f"\nProduct label sent to: {printer_name}")
            print(f"Product: {product['name']}")
        elif label_type == '2':
            result = print_inventory_label(printer_name, inventory_item)
            print(f"\nInventory label sent to: {printer_name}")
            print(f"Item: {inventory_item['name']}")
        else:
            print("Invalid selection")
            sys.exit(1)

        print(f"Job ID: {result['job_id']}")
        print(f"Status: {result['status']}")

    except requests.exceptions.ConnectionError:
        print("Error: Cannot connect to OpenPrintHub. Is it running?")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
