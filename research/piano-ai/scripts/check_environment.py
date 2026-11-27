#!/usr/bin/env python3
"""Report the local P0 audio/MIDI environment without downloading anything."""

from __future__ import annotations

import importlib.metadata
import importlib.util
import json
import platform
import shutil
import sys


PACKAGES = ("basic-pitch", "librosa", "pretty-midi", "numpy")


def package_version(name: str) -> str | None:
    try:
        return importlib.metadata.version(name)
    except importlib.metadata.PackageNotFoundError:
        return None


def main() -> int:
    report = {
        "python": sys.version.split()[0],
        "python_executable": sys.executable,
        "platform": platform.platform(),
        "packages": {name: package_version(name) for name in PACKAGES},
        "commands": {
            name: shutil.which(name)
            for name in ("basic-pitch", "ffmpeg")
        },
        "optional_modules": {
            name: importlib.util.find_spec(name) is not None
            for name in ("torch", "torchaudio", "mido")
        },
    }

    if report["optional_modules"]["torch"]:
        try:
            import torch  # type: ignore

            report["torch"] = {
                "version": torch.__version__,
                "cuda_available": torch.cuda.is_available(),
                "cuda_version": torch.version.cuda,
                "device_count": torch.cuda.device_count(),
            }
        except Exception as exc:  # diagnostic must not hide import failures
            report["torch_error"] = repr(exc)

    print(json.dumps(report, indent=2, ensure_ascii=False))

    missing = [name for name, version in report["packages"].items() if version is None]
    if missing:
        print("Missing packages: " + ", ".join(missing), file=sys.stderr)
        return 1
    if not report["commands"]["basic-pitch"]:
        print("basic-pitch CLI was not found on PATH", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
