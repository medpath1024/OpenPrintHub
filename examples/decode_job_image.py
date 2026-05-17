#!/usr/bin/env python3
"""
Decode OpenPrintHub job "data" (Base64) into an image file.

Supports input formats:
  1) JSON with either:
     - {"job": {"data": "...", "name": "label.bmp", ...}}
     - {"data": "...", "name": "label.bmp", ...}
  2) Plain base64 string (no JSON)

Usage:
  python3 decode_job_image.py job.json
  cat job.json | python3 decode_job_image.py -
  pbpaste | python3 decode_job_image.py - -o label.bmp   # macOS clipboard
"""

import argparse
import base64
import json
import os
import sys
from typing import Optional, Tuple


def _read_all(path: str) -> str:
    if path == "-":
        return sys.stdin.read()
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


def _guess_ext(data: bytes) -> str:
    if data.startswith(b"BM"):
        return ".bmp"
    if data.startswith(b"\x89PNG\r\n\x1a\n"):
        return ".png"
    if len(data) >= 3 and data[0] == 0xFF and data[1] == 0xD8 and data[2] == 0xFF:
        return ".jpg"
    if data.startswith(b"GIF87a") or data.startswith(b"GIF89a"):
        return ".gif"
    if data.startswith(b"RIFF") and len(data) >= 12 and data[8:12] == b"WEBP":
        return ".webp"
    return ".bin"


def _extract_b64_and_name(text: str) -> Tuple[str, Optional[str]]:
    # Try JSON first.
    stripped = text.lstrip()
    if stripped.startswith("{") or stripped.startswith("["):
        obj = json.loads(text)
        if isinstance(obj, dict) and "job" in obj and isinstance(obj["job"], dict):
            job = obj["job"]
            data_b64 = job.get("data")
            name = job.get("name")
        elif isinstance(obj, dict):
            data_b64 = obj.get("data")
            name = obj.get("name")
        else:
            raise ValueError("JSON root must be an object with 'data' or 'job.data'")

        if not isinstance(data_b64, str) or not data_b64.strip():
            raise ValueError("Missing base64 data at 'data' or 'job.data'")
        return data_b64, name if isinstance(name, str) and name.strip() else None

    # Fall back: treat as plain base64.
    b64 = text.strip()
    if not b64:
        raise ValueError("Input is empty")
    return b64, None


def _resolve_output_path(out: Optional[str], name: Optional[str], decoded: bytes) -> str:
    if out:
        out = out.strip()
        if not out:
            raise ValueError("Output path is empty")
        # If user passed a directory, write into it.
        if os.path.isdir(out):
            base = name or ("image" + _guess_ext(decoded))
            return os.path.join(out, base)
        # If user passed no extension, append a guessed one.
        root, ext = os.path.splitext(out)
        if ext:
            return out
        return out + _guess_ext(decoded)

    if name:
        return name
    return "image" + _guess_ext(decoded)


def main() -> None:
    ap = argparse.ArgumentParser(add_help=True)
    ap.add_argument("input", help="Path to JSON/base64 file, or '-' for stdin")
    ap.add_argument("-o", "--out", help="Output file path (or directory)", default=None)
    args = ap.parse_args()

    text = _read_all(args.input)
    data_b64, name = _extract_b64_and_name(text)

    decoded = base64.b64decode(data_b64, validate=False)
    out_path = _resolve_output_path(args.out, name, decoded)

    with open(out_path, "wb") as f:
        f.write(decoded)

    print(out_path)


if __name__ == "__main__":
    main()
