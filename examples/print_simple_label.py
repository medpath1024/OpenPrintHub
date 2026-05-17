#!/usr/bin/env python3
"""
OpenPrintHub - Simple Thermal Label (PDF) Print

This is intentionally minimal: generate a tiny one-page PDF label with one line
of text, then submit exactly one print job.

Usage:
  OPH_URL=http://192.168.1.4:16800 python print_simple_label.py "测试标签"
  OPH_URL=http://192.168.1.4:16800 python print_simple_label.py "TEST" "DYMO LabelWriter 450 Turbo"
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


def _build_simple_pdf(text: str, width_mm: float = 50.0, height_mm: float = 30.0) -> bytes:
    # Minimal PDF generator (no external deps).
    # 1 inch = 72 pt, 1 mm = 72/25.4 pt.
    w = width_mm * 72.0 / 25.4
    h = height_mm * 72.0 / 25.4

    # Keep content ASCII-safe for simplicity.
    safe = text.strip()
    if not safe:
        safe = "TEST"
    safe = safe.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")
    safe = "".join(ch if 32 <= ord(ch) <= 126 else "?" for ch in safe)

    # Place text with a small margin.
    content = f"BT /F1 16 Tf 10 {max(10.0, h - 28.0):.2f} Td ({safe}) Tj ET\n"
    content_bytes = content.encode("ascii", errors="replace")

    # Build objects; we will compute xref offsets after concatenation.
    objects = []

    def add_obj(obj_num: int, payload: bytes) -> None:
        objects.append((obj_num, payload))

    add_obj(1, b"<< /Type /Catalog /Pages 2 0 R >>")
    add_obj(2, b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
    add_obj(
        3,
        (
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 {w:.2f} {h:.2f}] "
            f"/Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>"
        ).encode("ascii"),
    )
    add_obj(4, b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
    add_obj(5, b"<< /Length %d >>\nstream\n" % len(content_bytes) + content_bytes + b"endstream")

    out = bytearray()
    out.extend(b"%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

    offsets = {0: 0}
    for obj_num, payload in objects:
        offsets[obj_num] = len(out)
        out.extend(f"{obj_num} 0 obj\n".encode("ascii"))
        out.extend(payload)
        out.extend(b"\nendobj\n")

    xref_start = len(out)
    max_obj = max(n for n, _ in objects)
    out.extend(f"xref\n0 {max_obj + 1}\n".encode("ascii"))
    out.extend(b"0000000000 65535 f \n")
    for i in range(1, max_obj + 1):
        off = offsets[i]
        out.extend(f"{off:010d} 00000 n \n".encode("ascii"))

    out.extend(
        (
            "trailer\n"
            f"<< /Size {max_obj + 1} /Root 1 0 R >>\n"
            "startxref\n"
            f"{xref_start}\n"
            "%%EOF\n"
        ).encode("ascii")
    )
    return bytes(out)


def _get_default_printer_name() -> str:
    r = requests.get(f"{OPH_URL}/v1/printers", timeout=5)
    r.raise_for_status()
    printers = r.json()
    if not printers:
        raise RuntimeError("No printers returned by /v1/printers")
    default = next((p for p in printers if p.get("is_default")), None) or printers[0]
    return default["name"]


def print_simple_label(text: str, printer_name: Optional[str] = None) -> dict:
    if not printer_name:
        printer_name = _get_default_printer_name()

    pdf_bytes = _build_simple_pdf(text)
    pdf_b64 = base64.b64encode(pdf_bytes).decode("ascii")

    r = requests.post(
        f"{OPH_URL}/v1/print",
        json={
            "printer": printer_name,
            "type": "pdf",
            "data": pdf_b64,
            "name": "label.pdf",
            "settings": {"copies": 1, "orientation": "portrait", "fit_to_page": True},
        },
        timeout=10,
    )
    r.raise_for_status()
    return r.json()


def get_job(job_id: str) -> dict:
    r = requests.get(f"{OPH_URL}/v1/jobs/{job_id}", timeout=5)
    r.raise_for_status()
    return r.json()


def main() -> None:
    if len(sys.argv) < 2:
        print(__doc__.strip())
        raise SystemExit(2)

    text = sys.argv[1]
    printer = sys.argv[2] if len(sys.argv) > 2 else None

    res = print_simple_label(text, printer)
    job_id = res.get("job_id")
    print(f"Submitted: job_id={job_id} status={res.get('status')}")

    # Best-effort: poll a few times so you can see it transitions from queued.
    if job_id:
        for _ in range(10):
            time.sleep(0.5)
            j = get_job(job_id)
            print(f"Job: status={j.get('status')} message={j.get('message', '')}")
            if j.get("status") in ("completed", "failed", "cancelled"):
                break


if __name__ == "__main__":
    main()
