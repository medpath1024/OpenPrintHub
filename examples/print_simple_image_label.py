#!/usr/bin/env python3
"""
OpenPrintHub - Simple Thermal Label (IMAGE) Print

Goal: print exactly ONE label successfully, without PDF and without extra deps.

This script generates a small black/white BMP label (border + stripes) and
submits it as an "image" job to OpenPrintHub.

Usage:
  OPH_URL=http://192.168.1.4:16800 python3 print_simple_image_label.py
  OPH_URL=http://192.168.1.4:16800 python3 print_simple_image_label.py "DYMO LabelWriter 450 Turbo"
"""

import base64
import os
import sys
import time
from typing import Optional

import requests


def _normalize_base_url(url: str) -> str:
    url = (url or "").strip().rstrip("/")
    if not url:
        return "http://localhost:16800"
    if "://" not in url:
        url = "http://" + url
    return url


OPH_URL = _normalize_base_url(os.environ.get("OPH_URL", "http://localhost:16800"))


def _build_label_bmp(width: int = 400, height: int = 240) -> bytes:
    # 24-bit BMP, uncompressed, bottom-up rows.
    row_bytes = width * 3
    pad = (4 - (row_bytes % 4)) % 4
    pixel_data_size = (row_bytes + pad) * height
    file_size = 14 + 40 + pixel_data_size
    pixel_offset = 14 + 40

    def le_u16(n: int) -> bytes:
        return bytes((n & 0xFF, (n >> 8) & 0xFF))

    def le_u32(n: int) -> bytes:
        return bytes((n & 0xFF, (n >> 8) & 0xFF, (n >> 16) & 0xFF, (n >> 24) & 0xFF))

    # BMP file header (14)
    hdr = bytearray()
    hdr.extend(b"BM")
    hdr.extend(le_u32(file_size))
    hdr.extend(le_u16(0))
    hdr.extend(le_u16(0))
    hdr.extend(le_u32(pixel_offset))

    # DIB header: BITMAPINFOHEADER (40)
    dib = bytearray()
    dib.extend(le_u32(40))  # header size
    dib.extend(le_u32(width))
    dib.extend(le_u32(height))
    dib.extend(le_u16(1))  # planes
    dib.extend(le_u16(24))  # bpp
    dib.extend(le_u32(0))  # BI_RGB
    dib.extend(le_u32(pixel_data_size))
    dib.extend(le_u32(0))  # x ppm
    dib.extend(le_u32(0))  # y ppm
    dib.extend(le_u32(0))  # colors used
    dib.extend(le_u32(0))  # important colors

    # Pixels: BGR
    # Make something visibly non-empty: border + stripe area.
    border = 3
    stripe_y1 = 60
    stripe_y2 = height - 60
    stripe_x1 = 20
    stripe_x2 = width - 20
    stripe_w = 10

    out = bytearray()
    out.extend(hdr)
    out.extend(dib)

    for y_bottom in range(height):  # bottom-up
        y = height - 1 - y_bottom  # top-based y
        row = bytearray()
        for x in range(width):
            black = False
            if x < border or x >= width - border or y < border or y >= height - border:
                black = True
            elif stripe_y1 <= y < stripe_y2 and stripe_x1 <= x < stripe_x2:
                if ((x - stripe_x1) // stripe_w) % 2 == 0:
                    black = True
            # BGR
            if black:
                row.extend((0, 0, 0))
            else:
                row.extend((255, 255, 255))
        if pad:
            row.extend(b"\x00" * pad)
        out.extend(row)

    return bytes(out)


def _get_default_printer_name() -> str:
    r = requests.get(f"{OPH_URL}/v1/printers", timeout=5)
    r.raise_for_status()
    printers = r.json()
    if not printers:
        raise RuntimeError("No printers returned by /v1/printers")
    default = next((p for p in printers if p.get("is_default")), None) or printers[0]
    return default["name"]


def print_image_label(printer_name: Optional[str] = None) -> dict:
    if not printer_name:
        printer_name = _get_default_printer_name()

    bmp = _build_label_bmp()
    img_b64 = base64.b64encode(bmp).decode("ascii")

    r = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            "printer": printer_name,
            "type": "image",
            "data": img_b64,
            "name": "label.bmp",
            "settings": {"copies": 1, "orientation": "portrait", "fit_to_page": True},
        },
        timeout=15,
    )
    r.raise_for_status()
    return r.json()


def get_job(job_id: str) -> dict:
    r = requests.get(f"{OPH_URL}/v1/jobs/{job_id}", timeout=5)
    r.raise_for_status()
    return r.json()


def main() -> None:
    printer = sys.argv[1] if len(sys.argv) > 1 else None
    res = print_image_label(printer)
    job_id = res.get("job_id")
    print(f"Submitted: job_id={job_id} status={res.get('status')}")

    if job_id:
        for _ in range(10):
            time.sleep(0.5)
            j = get_job(job_id)
            print(f"Job: status={j.get('status')} message={j.get('message', '')}")
            if j.get("status") in ("completed", "failed", "cancelled"):
                break


if __name__ == "__main__":
    main()
