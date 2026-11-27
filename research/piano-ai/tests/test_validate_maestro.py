from __future__ import annotations

import csv
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).parents[1] / "scripts" / "validate_maestro.py"


class ValidateMaestroTests(unittest.TestCase):
    def make_dataset(self, root: Path) -> None:
        (root / "maestro-v2.0.0.json").write_text("{}", encoding="utf-8")
        audio = root / "audio.wav"
        midi = root / "score.mid"
        audio.write_bytes(b"audio-fixture")
        midi.write_bytes(b"midi-fixture")
        with (root / "maestro-v2.0.0.csv").open("w", encoding="utf-8", newline="") as stream:
            writer = csv.DictWriter(stream, fieldnames=["audio_filename", "midi_filename", "split"])
            writer.writeheader()
            writer.writerow({
                "audio_filename": "audio.wav",
                "midi_filename": "score.mid",
                "split": "train",
            })

    def test_metadata_validation_without_file_walk(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.make_dataset(root)
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--root", str(root)],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('"metadata_rows": 1', result.stdout)

    def test_full_file_validation(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.make_dataset(root)
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--root", str(root), "--check-files"],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('"missing_files_count": 0', result.stdout)

    def test_missing_metadata_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--root", tmp],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn('"ok": false', result.stdout)


if __name__ == "__main__":
    unittest.main()
