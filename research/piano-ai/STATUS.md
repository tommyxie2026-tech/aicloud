# Piano AI Execution Status

> Updated: 2026-11-27  
> Repository: `tommyxie2026-tech/aicloud`  
> Tracking: [P0 baseline PR #59](https://github.com/tommyxie2026-tech/aicloud/pull/59)

## Current state

| Phase | State | Evidence / exit gate |
| --- | --- | --- |
| P0 — environment and baseline | In progress | Workspace, pinned Python dependencies, environment preflight, local inference wrapper, MAESTRO validator, unit tests and CI are in place. Real-audio inference and baseline metrics remain outstanding. |
| P1 — data collection | Not started | Requires paired digital-piano MIDI/audio capture and a manually reviewed acoustic-piano evaluation set. |
| P2 — recognition and scoring | Not started | Requires OMR output, normalized event schema, score alignment, and deterministic error classification. |
| P3 — aicloud integration | Not started | Requires stable dataset, job, artifact, model-registry, and evaluation interfaces. |
| P4 — model adaptation | Not started | Requires measured baselines, rights-cleared data, and an independent holdout test set. |

## P0 evidence

- GitHub Actions workflow `Piano AI P0 Checks` passed on the current PR head.
- Python scripts compile.
- Three MAESTRO validator tests pass.
- Pinned Basic Pitch dependencies install in the CI Python 3.10 environment.
- Environment preflight and CLI help checks pass.

Not yet demonstrated:

- A real piano recording has been transcribed end-to-end.
- Output MIDI quality has been reviewed against a reference.
- GPU inference performance has been measured.
- MAESTRO has been downloaded or validated locally.
- ByteDance legacy training/inference has been reproduced.

## Existing repository CI observation

The repository's separate Go Test workflow currently fails during compilation because `ExecutionFailed` is declared in both `execution/types.go` and `execution/provider.go` on the base `main` branch. The piano PR does not modify either file. This unrelated platform compile issue should be fixed in its own change before treating the full repository check suite as green.

## P0 next action

Run the pinned environment on a short, rights-cleared piano recording:

```bash
python -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements-p0.txt
python scripts/check_environment.py
python scripts/transcribe_basic_pitch.py --input /path/to/piano.wav --output-dir ./outputs/sample-001
```

Review the generated MIDI in a MIDI editor or compare it with the paired MIDI reference. Save only hashes, metadata, evaluation metrics, and permitted derived artifacts in Git; keep raw recordings and datasets outside the repository unless their license and consent explicitly permit redistribution.

## P1 required capture package

For one digital-piano session, retain:

- raw WAV/PCM audio;
- original MIDI file;
- score image/PDF and corrected MusicXML if available;
- instrument model and audio capture route;
- sample rate, bit depth, channels;
- MIDI/audio synchronization calibration;
- consent and data-use scope.

For acoustic-piano evaluation, additionally require a human-reviewed note-event reference. A model's own transcription is not ground truth.

## Stage gates

### P0 → P1

- [ ] Real audio inference succeeds.
- [ ] MIDI output is reviewed and compared against a reference.
- [ ] Inference runtime and hardware are recorded.
- [ ] Dataset/code/weight rights are recorded separately.

### P1 → P2

- [ ] Digital MIDI/audio pair is synchronized.
- [ ] Acoustic-piano holdout has human-reviewed annotations.
- [ ] Train/validation/test split prevents performer and recording leakage.
- [ ] Score source and MusicXML corrections are traceable.

### P2 → P3

- [ ] OMR, AMT/MIDI, and alignment interfaces are stable.
- [ ] Note-level error taxonomy has deterministic tests.
- [ ] Low-confidence recognition is surfaced rather than treated as student error.
- [ ] Evaluation report is reproducible from versioned inputs.

### P3 → P4

- [ ] Dataset and model artifacts have immutable versions.
- [ ] Training jobs record code, data, parameters, and environment.
- [ ] Independent evaluation gates model promotion.
- [ ] Rollback and regression evaluation are demonstrated.

## Data and licensing guardrails

- MAESTRO is CC BY-NC-SA 4.0. Its non-commercial and share-alike terms must be respected; do not assume compatibility with commercial platform use.
- The ByteDance piano transcription repository is archived. Its legacy training stack must remain isolated and security-reviewed.
- Basic Pitch code license, model weights, training data, and user recordings must each be tracked separately.
- No raw user audio, MIDI, or copyrighted sheet music should be committed without explicit rights and consent.
