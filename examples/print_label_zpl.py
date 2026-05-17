#!/usr/bin/env python3
"""
OpenPrintHub - ZPL Label Printing Example

Usage:
    python print_label_zpl.py [printer_name]

This script demonstrates how to print labels using ZPL (Zebra Programming Language)
for Zebra label printers.
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


class ZPLBuilder:
    """ZPL command builder for Zebra label printers."""

    def __init__(self):
        self.commands = []

    def start(self):
        """Start label format."""
        self.commands.append('^XA')
        return self

    def end(self):
        """End label format."""
        self.commands.append('^XZ')
        return self

    def label_home(self, x: int, y: int):
        """Set label home position."""
        self.commands.append(f'^LH{x},{y}')
        return self

    def field_origin(self, x: int, y: int):
        """Set field origin position."""
        self.commands.append(f'^FO{x},{y}')
        return self

    def font(self, name: str = '0', height: int = 30, width: int = None):
        """
        Select font.
        name: Font name (0-9, A-Z)
        height: Font height in dots
        width: Font width in dots (defaults to height)
        """
        w = width if width else height
        self.commands.append(f'^A{name}N,{height},{w}')
        return self

    def field_data(self, data: str):
        """Add field data."""
        self.commands.append(f'^FD{data}^FS')
        return self

    def barcode_128(self, height: int, data: str, show_text: bool = True):
        """
        Add Code 128 barcode.
        height: Barcode height in dots
        data: Barcode data
        show_text: Whether to show human-readable text
        """
        self.commands.append('^BY2')  # Barcode default settings
        interpretation = 'Y' if show_text else 'N'
        self.commands.append(f'^BCN,{height},{interpretation},N,N')
        self.commands.append(f'^FD{data}^FS')
        return self

    def qr_code(self, data: str, magnification: int = 4):
        """
        Add QR code.
        data: QR code data
        magnification: Size multiplier (1-10)
        """
        self.commands.append(f'^BQN,2,{magnification}')
        self.commands.append(f'^FDQA,{data}^FS')
        return self

    def box(self, width: int, height: int, thickness: int = 1):
        """Draw a box."""
        self.commands.append(f'^GB{width},{height},{thickness}^FS')
        return self

    def line_horizontal(self, width: int, thickness: int = 2):
        """Draw a horizontal line."""
        return self.box(width, thickness, thickness)

    def print_quantity(self, qty: int):
        """Set print quantity."""
        self.commands.append(f'^PQ{qty}')
        return self

    def build(self) -> str:
        """Build ZPL string."""
        return '\n'.join(self.commands)

    def to_base64(self) -> str:
        """Return ZPL as Base64 encoded string."""
        return base64.b64encode(self.build().encode()).decode()


def print_shipping_label(printer_name: str, order: dict, quantity: int = 1):
    """Print a shipping label."""

    label = ZPLBuilder()
    label.start()
    label.label_home(0, 0)

    # Header box
    label.field_origin(30, 20)
    label.box(740, 80, 3)

    # Company name
    label.field_origin(50, 40)
    label.font('0', 50)
    label.field_data('ACME SHIPPING CO.')

    # Divider
    label.field_origin(30, 120)
    label.line_horizontal(740, 2)

    # FROM section
    label.field_origin(40, 140)
    label.font('0', 25)
    label.field_data('FROM:')

    label.field_origin(40, 175)
    label.font('0', 22)
    label.field_data(order['from']['name'])

    label.field_origin(40, 200)
    label.field_data(order['from']['address'])

    label.field_origin(40, 225)
    label.field_data(f"{order['from']['city']}, {order['from']['state']} {order['from']['zip']}")

    # TO section (larger font)
    label.field_origin(40, 280)
    label.font('0', 35)
    label.field_data('SHIP TO:')

    label.field_origin(40, 330)
    label.font('0', 45)
    label.field_data(order['to']['name'])

    label.field_origin(40, 385)
    label.font('0', 35)
    label.field_data(order['to']['address'])

    label.field_origin(40, 430)
    label.field_data(f"{order['to']['city']}, {order['to']['state']} {order['to']['zip']}")

    # Barcode
    label.field_origin(40, 500)
    label.barcode_128(80, order['tracking'])

    # QR Code
    label.field_origin(550, 280)
    label.qr_code(order['tracking'], 6)

    # Order info
    label.field_origin(550, 500)
    label.font('0', 25)
    label.field_data(f"Order: {order['order_id']}")

    label.field_origin(550, 535)
    label.field_data(f"Weight: {order['weight']} lbs")

    label.print_quantity(quantity)
    label.end()

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


def get_printers():
    """Get list of available printers."""
    response = requests.get(f"{OPH_URL}/v1/printers")
    response.raise_for_status()
    return response.json()


def main():
    # Sample order data
    order = {
        'order_id': 'ORD-2026-00456',
        'tracking': '1Z999AA10123456784',
        'weight': 2.5,
        'from': {
            'name': 'ACME Distribution Center',
            'address': '100 Warehouse Drive',
            'city': 'Chicago',
            'state': 'IL',
            'zip': '60601'
        },
        'to': {
            'name': 'John Smith',
            'address': '456 Oak Avenue, Suite 200',
            'city': 'New York',
            'state': 'NY',
            'zip': '10001'
        }
    }

    # Get printer name
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

    try:
        result = print_shipping_label(printer_name, order)
        print(f"\nShipping label sent to: {printer_name}")
        print(f"Job ID: {result['job_id']}")
        print(f"Status: {result['status']}")
        print(f"\nOrder: {order['order_id']}")
        print(f"Tracking: {order['tracking']}")
    except requests.exceptions.ConnectionError:
        print("Error: Cannot connect to OpenPrintHub. Is it running?")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
