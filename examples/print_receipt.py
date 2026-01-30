#!/usr/bin/env python3
"""
OpenPrintHub - ESC/POS Receipt Printing Example

Usage:
    python print_receipt.py [printer_name]

This script demonstrates how to print thermal receipts using ESC/POS commands.
"""

import base64
import sys
from datetime import datetime
import requests

OPH_URL = "http://localhost:16800"


class ESCPOSBuilder:
    """ESC/POS command builder for thermal receipt printers."""
    
    # Common ESC/POS commands
    ESC = 0x1B
    GS = 0x1D
    
    def __init__(self):
        self.commands = bytearray()
    
    def init(self):
        """Initialize printer."""
        self.commands.extend([self.ESC, 0x40])
        return self
    
    def align(self, position: int):
        """
        Set text alignment.
        0 = Left, 1 = Center, 2 = Right
        """
        self.commands.extend([self.ESC, 0x61, position])
        return self
    
    def left(self):
        return self.align(0)
    
    def center(self):
        return self.align(1)
    
    def right(self):
        return self.align(2)
    
    def bold(self, enabled: bool = True):
        """Enable/disable bold text."""
        self.commands.extend([self.ESC, 0x45, 1 if enabled else 0])
        return self
    
    def underline(self, mode: int = 1):
        """
        Set underline mode.
        0 = Off, 1 = 1-dot, 2 = 2-dot
        """
        self.commands.extend([self.ESC, 0x2D, mode])
        return self
    
    def text_size(self, width: int = 1, height: int = 1):
        """
        Set text size (1-8 for both width and height).
        """
        size = ((width - 1) << 4) | (height - 1)
        self.commands.extend([self.GS, 0x21, size])
        return self
    
    def double_size(self):
        return self.text_size(2, 2)
    
    def normal_size(self):
        return self.text_size(1, 1)
    
    def text(self, content: str):
        """Add text content."""
        self.commands.extend(content.encode('utf-8'))
        return self
    
    def newline(self, count: int = 1):
        """Add line feed(s)."""
        for _ in range(count):
            self.commands.append(0x0A)
        return self
    
    def line(self, char: str = '-', width: int = 32):
        """Print a horizontal line."""
        self.text(char * width)
        return self.newline()
    
    def feed(self, lines: int = 1):
        """Feed paper by number of lines."""
        self.commands.extend([self.ESC, 0x64, lines])
        return self
    
    def cut(self, partial: bool = False):
        """Cut paper."""
        mode = 0x01 if partial else 0x00
        self.commands.extend([self.GS, 0x56, mode])
        return self
    
    def open_drawer(self, pin: int = 0):
        """
        Open cash drawer.
        pin: 0 or 1 for drawer pin selection
        """
        self.commands.extend([self.ESC, 0x70, pin, 0x19, 0xFA])
        return self
    
    def to_base64(self) -> str:
        """Return commands as Base64 encoded string."""
        return base64.b64encode(self.commands).decode('utf-8')


def print_receipt(printer_name: str):
    """Print a sample receipt."""
    
    # Sample receipt data
    store_name = "COFFEE SHOP"
    store_address = "123 Main Street"
    store_phone = "(555) 123-4567"
    
    items = [
        ("Espresso", 3.50),
        ("Latte (Large)", 5.50),
        ("Croissant", 4.00),
        ("Blueberry Muffin", 3.50),
    ]
    
    subtotal = sum(price for _, price in items)
    tax = subtotal * 0.08
    total = subtotal + tax
    
    # Build receipt
    receipt = ESCPOSBuilder()
    receipt.init()
    
    # Header
    receipt.center().double_size().bold()
    receipt.text(store_name).newline()
    receipt.normal_size().bold(False)
    receipt.text(store_address).newline()
    receipt.text(f"Tel: {store_phone}").newline(2)
    
    # Date/Time
    receipt.left()
    receipt.text(datetime.now().strftime("%Y-%m-%d %H:%M:%S")).newline()
    receipt.line("=")
    
    # Items
    for name, price in items:
        line = f"{name:<22}${price:>6.2f}"
        receipt.text(line).newline()
    
    # Totals
    receipt.line("-")
    receipt.text(f"{'Subtotal:':<22}${subtotal:>6.2f}").newline()
    receipt.text(f"{'Tax (8%):':<22}${tax:>6.2f}").newline()
    receipt.line("-")
    receipt.bold()
    receipt.text(f"{'TOTAL:':<22}${total:>6.2f}").newline()
    receipt.bold(False)
    receipt.line("=")
    
    # Footer
    receipt.newline()
    receipt.center()
    receipt.text("Thank you for visiting!").newline()
    receipt.text("Please come again!").newline(3)
    
    # Cut
    receipt.cut()
    
    # Send to printer
    response = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            'printer': printer_name,
            'type': 'raw',
            'data': receipt.to_base64()
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
        
        choice = input("\nSelect printer number (or press Enter for default): ").strip()
        
        if choice:
            try:
                printer_name = printers[int(choice) - 1]['name']
            except (ValueError, IndexError):
                print("Invalid selection")
                sys.exit(1)
        else:
            default = next((p for p in printers if p.get('is_default')), printers[0])
            printer_name = default['name']
    
    try:
        result = print_receipt(printer_name)
        print(f"\nReceipt sent to printer: {printer_name}")
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
