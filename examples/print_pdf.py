#!/usr/bin/env python3
"""
OpenPrintHub - PDF Printing Example

Usage:
    python print_pdf.py <pdf_file> [printer_name] [copies] [job_name]

Examples:
    python print_pdf.py invoice.pdf
    python print_pdf.py report.pdf "HP LaserJet Pro" 2
    python print_pdf.py report.pdf "HP LaserJet Pro" 1 "Monthly Report"
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


def get_printers():
    """Get list of available printers."""
    response = requests.get(f"{OPH_URL}/v1/printers")
    response.raise_for_status()
    return response.json()


def print_pdf(file_path: str, printer_name: str = None, copies: int = 1, job_name: str = None):
    """
    Print a PDF file.

    Args:
        file_path: Path to the PDF file
        printer_name: Name of printer (uses default if not specified)
        copies: Number of copies to print
        job_name: Optional name for the print job (for identification)

    Returns:
        Job response dict with job_id and status
    """
    # Get default printer if not specified
    if not printer_name:
        printers = get_printers()
        default_printer = next((p for p in printers if p.get('is_default')), None)
        if not default_printer:
            default_printer = printers[0] if printers else None
        if not default_printer:
            raise Exception("No printers available")
        printer_name = default_printer['name']
        print(f"Using default printer: {printer_name}")

    # Use filename as job name if not specified
    if not job_name:
        job_name = file_path.split('/')[-1].split('\\')[-1]

    # Read and encode PDF
    with open(file_path, 'rb') as f:
        pdf_base64 = base64.b64encode(f.read()).decode('utf-8')

    # Submit print job
    response = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            'printer': printer_name,
            'type': 'pdf',
            'data': pdf_base64,
            'name': job_name,
            'settings': {
                'copies': copies,
                'orientation': 'portrait',
                'fit_to_page': True
            }
        }
    )
    response.raise_for_status()
    return response.json()


def cancel_job(job_id: str):
    """Cancel a print job."""
    response = requests.post(f"{OPH_URL}/v1/jobs/{job_id}/cancel")
    response.raise_for_status()
    return response.json()


def get_service_info():
    """Get OpenPrintHub service information."""
    response = requests.get(f"{OPH_URL}/v1/info")
    response.raise_for_status()
    return response.json()


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    file_path = sys.argv[1]
    printer_name = sys.argv[2] if len(sys.argv) > 2 else None
    copies = int(sys.argv[3]) if len(sys.argv) > 3 else 1
    job_name = sys.argv[4] if len(sys.argv) > 4 else None

    try:
        # Show service info
        info = get_service_info()
        print(f"Connected to OpenPrintHub v{info['version']} ({info['platform']})")

        result = print_pdf(file_path, printer_name, copies, job_name)
        print(f"Print job submitted successfully!")
        print(f"  Job ID: {result['job_id']}")
        print(f"  Status: {result['status']}")
    except FileNotFoundError:
        print(f"Error: File not found: {file_path}")
        sys.exit(1)
    except requests.exceptions.ConnectionError:
        print("Error: Cannot connect to OpenPrintHub. Is it running?")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
