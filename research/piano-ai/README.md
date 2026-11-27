# Piano AI Research Workspace

Piano AI is the first vertical workload for the aicloud extensible AI education platform. The confirmed direction is to support learners at different skill levels, both MIDI and audio capture, and sheet music image/PDF input.

## Canonical product roadmap

See [aicloud 可扩展 AI 教育平台：产品架构与 P0–P4 实施路线](../../docs/roadmap/2026-11-27-ai-education-platform-architecture-and-p0-p4.zh-CN.md).

The stages are:
- P0: technical feasibility and reproducible OMR/AMT/MIDI baselines.
- P1: dual-route data collection, synchronization calibration, human-reviewed acoustic holdout, rights and benchmark manifests.
- P2: complete offline piano tutor MVP — score recognition, MIDI/audio analysis, alignment, explainable evaluation, learner-level adaptation, practice feedback and session history.
- P3: aicloud integration and reusable subject-plugin contracts, including a second subject proof of reuse.
- P4: controlled training, domain adaptation, personalization, independent promotion gates and (only after offline quality is stable) real-time practice.

## Workspace scope

This workspace contains isolated research code and reproducible experiments. It is not an aicloud core package. The platform owns datasets, jobs, artifacts, model versions, evaluation evidence, policy and deployment governance. The piano vertical owns music-domain semantics, OMR/AMT, MIDI parsing, score alignment, performance evaluation and practice strategy.

## P0 baseline environment

Recommended initial baseline: Python 3.10 in an isolated virtual environment.

```bash
cd research/piano-ai
python3.10 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r requirements-p0.txt
python scripts/check_environment.py
```

For Linux, install system audio libraries such as ffmpeg and libsndfile using the operating system package manager.

## Run Basic Pitch

```bash
python scripts/transcribe_basic_pitch.py \
  --input /path/to/piano.wav \
  --output-dir /path/to/output
```

This invokes the local Basic Pitch CLI and records input/output hashes and environment metadata. It does not upload audio to a remote service. A generated MIDI file must still be reviewed and compared against a paired MIDI or human-reviewed reference.

## P0 current boundaries

The current code and CI provide environment preflight, dependency installation, script compilation, MAESTRO metadata validation tests and CLI checks. They do not yet demonstrate:
- successful transcription of a real piano recording;
- note-level quality against a paired MIDI/reference;
- OMR quality on the target sheet-music types;
- MIDI/audio synchronization calibration;
- acoustic-piano holdout accuracy;
- complete end-to-end practice feedback;
- GPU performance or model training.

A passing CI job is not equivalent to passing P0 product acceptance.

## Dataset and rights

- Do not commit MAESTRO audio, MIDI, or archives to this repository.
- MAESTRO is CC BY-NC-SA 4.0; commercial use is not assumed.
- The archived ByteDance piano transcription stack must remain isolated and security-reviewed.
- Code, pretrained weights, training data, sheet music and user recordings have separate rights and provenance.
- Do not commit raw user audio, MIDI or copyrighted scores without explicit permission.
- Acoustic-piano evaluation requires human-reviewed reference annotations; model predictions are not ground truth.

## Local data and artifact handling

Keep raw data and generated artifacts outside the Git worktree unless their rights explicitly permit redistribution. Store manifests, hashes, version metadata, consent scope and evaluation reports in the research record. Do not store secrets, personal data or identifiable learner recordings in public CI artifacts.
