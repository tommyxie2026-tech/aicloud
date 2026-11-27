# Piano AI Research Workspace — P0

This workspace is isolated from the Go platform runtime. It is an experimental vertical workload, not an aicloud core package.

## P0 objective

Establish reproducible audio-to-MIDI inference baselines before training or platform integration:

1. Verify the Python/audio/GPU environment.
2. Run Spotify Basic Pitch as the first maintained, lightweight baseline.
3. Keep the archived ByteDance piano transcription implementation as a separately isolated comparison target.
4. Record dataset, code, weight, environment, and output provenance.

## Important project status

- The ByteDance `piano_transcription` repository was archived on 2025-12-08. Its README describes a legacy Python 3.7 / PyTorch 1.4.0 environment. Do not install its old dependencies into the aicloud runtime or the Basic Pitch environment.
- ByteDance's README describes MAESTRO V2.0.0 and a training configuration requiring about 29 GB GPU memory at batch size 12. Treat this as a historical reference, not a guaranteed requirement for inference or fine-tuning.
- MAESTRO is licensed CC BY-NC-SA 4.0. It is non-commercial and share-alike. Do not use it for commercial training, distribute it with the product, or assume derived artifacts can be commercially used. Obtain a separate rights review before any commercial use.
- Code, pretrained weights, datasets, scores, and user recordings have separate rights. Track them independently.

## Environment

Recommended initial baseline environment: Python 3.10. The Basic Pitch project documents Python 3.7–3.11 compatibility; this workspace pins a conservative Python 3.10 dependency set.

```bash
cd research/piano-ai
python3.10 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r requirements-p0.txt
python scripts/check_environment.py
```

For Linux, install system audio libraries (for example `libsndfile1` and `ffmpeg`) using the operating system package manager.

## Run Basic Pitch

```bash
python scripts/transcribe_basic_pitch.py \
  --input /path/to/piano.wav \
  --output-dir /path/to/output
```

The wrapper invokes the installed `basic-pitch` CLI and verifies that a MIDI output was produced. It does not upload audio or send it to a remote service.

## MAESTRO data

Do not commit MAESTRO audio, MIDI, or downloaded archives to this repository.

Download only after reviewing and accepting the dataset's CC BY-NC-SA 4.0 terms. Keep the data outside the Git worktree and point the validator at the extracted root:

```bash
python scripts/validate_maestro.py --root /data/maestro-v2.0.0
```

The validator checks the expected V2 metadata and referenced files. It does not download data or grant rights to use it.

Official dataset page: https://magenta.withgoogle.com/datasets/maestro

## P0 acceptance checklist

- [ ] Python and dependency versions recorded.
- [ ] Environment checker passes.
- [ ] Basic Pitch CLI help succeeds.
- [ ] One legally usable, locally held WAV sample transcribes to MIDI.
- [ ] Output MIDI opens and has non-zero note events.
- [ ] Run metadata records input hash, package versions, command, and output hash.
- [ ] MAESTRO V2 metadata and files validate, if dataset access is authorized.
- [ ] Baseline report includes hardware, runtime, sample rate, duration, and known limitations.

## P0 boundaries

No model training, fine-tuning, public dataset mirroring, real-time streaming, OMR, user-facing feedback, or production deployment is included in P0. These belong to later stages after the baseline and data rights are verified.
