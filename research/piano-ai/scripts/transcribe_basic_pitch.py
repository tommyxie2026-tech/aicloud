#!/usr/bin/env python3
"""Run local Basic Pitch inference and write a reproducibility record."""

from __future__ import annotations

import argparse
import hashlib
import importlib.metadata
import json
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output-dir", required=True, type=Path)
    args = parser.parse_args()

    source = args.input.expanduser().resolve()
    output_dir = args.output_dir.expanduser().resolve()
    if not source.is_file():
        parser.error(f"input file does not exist: {source}")
    if source.suffix.lower() not in {".wav", ".mp3", ".flac", ".ogg", ".m4a"}:
        parser.error("supported inputs: wav, mp3, flac, ogg, m4a")

    executable = shutil.which("basic-pitch")
    if not executable:
        parser.error("basic-pitch CLI not found; install requirements-p0.txt first")

    output_dir.mkdir(parents=True, exist_ok=True)
    command = [executable, str(output_dir), str(source)]
    completed = subprocess.run(command, check=False)
    if completed.returncode != 0:
        return completed.returncode

    midi_files = sorted(output_dir.glob("*.mid")) + sorted(output_dir.glob("*.midi"))
    if not midi_files:
        print("Basic Pitch completed but no MIDI output was found", file=sys.stderr)
        return 2

    record = {
        "created_at": datetime.now(timezone.utc).isoformat(),
        "input": str(source),
        "input_sha256": sha256(source),
        "input_bytes": source.stat().st_size,
        "command": command,
        "python": sys.version.split()[0],
        "packages": {},
        "outputs": [
            {"path": str(path), "sha256": sha256(path), "bytes": path.stat().st_size}
            for path in midi_files
        ],
        "notes": "Local inference only. Validate MIDI content and transcription quality separately.",
    }
    for package in ("basic-pitch", "librosa", "pretty-midi", "numpy"):
        try:
            record["packages"][package] = importlib.metadata.version(package)
        except importlib.metadata.PackageNotFoundError:
            record["packages"][package] = None

    record_path = output_dir / "run-metadata.json"
    record_path.write_text(json.dumps(record, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"Wrote {len(midi_files)} MIDI output(s) and {record_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
