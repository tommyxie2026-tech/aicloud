#!/usr/bin/env python3
"""Validate an already downloaded MAESTRO V2 tree; never downloads data."""

from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path


EXPECTED_CSV = "maestro-v2.0.0.csv"
EXPECTED_JSON = "maestro-v2.0.0.json"
REQUIRED_COLUMNS = {"audio_filename", "midi_filename", "split"}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True, type=Path)
    parser.add_argument("--check-files", action="store_true", help="verify every audio/MIDI path")
    args = parser.parse_args()

    root = args.root.expanduser().resolve()
    csv_path = root / EXPECTED_CSV
    json_path = root / EXPECTED_JSON
    missing = [str(path) for path in (csv_path, json_path) if not path.is_file()]
    if missing:
        print(json.dumps({"ok": False, "missing_metadata": missing}, indent=2))
        return 1

    with csv_path.open("r", encoding="utf-8", newline="") as stream:
        reader = csv.DictReader(stream)
        columns = set(reader.fieldnames or [])
        if not REQUIRED_COLUMNS.issubset(columns):
            print(json.dumps({
                "ok": False,
                "missing_columns": sorted(REQUIRED_COLUMNS - columns),
                "actual_columns": sorted(columns),
            }, indent=2))
            return 1
        rows = list(reader)

    split_counts: dict[str, int] = {}
    missing_files: list[str] = []
    for row in rows:
        split = row.get("split", "unknown")
        split_counts[split] = split_counts.get(split, 0) + 1
        if args.check_files:
            for field in ("audio_filename", "midi_filename"):
                candidate = root / row[field]
                if not candidate.is_file():
                    missing_files.append(str(candidate))

    report = {
        "ok": not missing_files,
        "dataset": "MAESTRO V2.0.0",
        "root": str(root),
        "metadata_rows": len(rows),
        "split_counts": split_counts,
        "missing_files_count": len(missing_files),
        "missing_files_sample": missing_files[:20],
        "license_notice": "CC BY-NC-SA 4.0; non-commercial/share-alike restrictions apply.",
        "downloaded": False,
    }
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 1 if missing_files else 0


if __name__ == "__main__":
    raise SystemExit(main())
