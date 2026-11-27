# Piano AI / AI Education Platform Status

> Updated: 2026-11-27  
> Confirmed direction: learners at different skill levels; MIDI + audio dual input; aicloud extensible AI education platform.  
> Canonical roadmap: [AI Education Platform Architecture and P0-P4](../../docs/roadmap/2026-11-27-ai-education-platform-architecture-and-p0-p4.zh-CN.md)  
> Tracking PR: [#59](https://github.com/tommyxie2026-tech/aicloud/pull/59)

## Product decision

Piano is the first vertical workload. The reusable platform capability is the learning-evidence loop:

Target/Objectives -> Actual Performance -> Alignment -> Error Diagnosis -> Assessment -> Practice Plan -> New Evidence.

Digital piano MIDI is the high-confidence event reference; digital-piano audio validates AMT; acoustic-piano audio is the real-world generalization path. Sheet images/PDF are processed through OMR into MusicXML/Score IR.

## P0-P4 status

| Phase | State | Exit evidence |
|---|---|---|
| P0 — technical feasibility and reproducible baselines | In progress | Python workspace, Basic Pitch wrapper, MAESTRO metadata validator, tests and CI exist. Real WAV inference, MIDI comparison and OMR sample verification remain outstanding. |
| P1 — dual-route data and benchmark | Not started | Paired digital-piano MIDI/audio, sync calibration, acoustic human-reviewed holdout, score/MusicXML and rights-aware manifests. |
| P2 — complete piano tutor MVP | Not started | OMR + MIDI/AMT + alignment + explainable evaluation + learner-level adaptation + practice feedback + session history. |
| P3 — aicloud integration and reusable education contracts | Not started | Dataset/job/artifact/model/evaluation/policy lifecycle plus a second subject-plugin proof of reuse. |
| P4 — controlled training and personalization | Not started | Rights-cleared domain adaptation, independent holdout gains, model promotion/rollback and measured learning outcomes. |

## P0 evidence currently available

- P0 workspace and dependency pins are committed in the PR branch.
- Environment preflight and local Basic Pitch wrapper are present.
- MAESTRO metadata validator and three standard-library tests are present.
- A previous P0 GitHub Actions run passed compile, dependency, unit-test and CLI checks.
- A subsequent workflow was queued/running at the last status check; do not claim its final result until GitHub reports completion.

Not yet demonstrated:
- Real piano WAV has been transcribed end-to-end.
- Output MIDI has been compared with paired MIDI or human-reviewed reference.
- OMR has been tested on a target score page.
- Digital-piano audio/MIDI synchronization has been measured.
- Acoustic-piano note-level holdout has been annotated.
- End-to-end practice feedback has been validated.
- GPU inference performance or training has been measured.

## Current next actions

1. Run Basic Pitch on one rights-cleared 30-60 second piano recording.
2. Compare the output with the same performance's MIDI reference where available.
3. Run Audiveris on one representative score image and manually inspect MusicXML.
4. Record note-level precision/recall, onset error, failure cases, runtime and environment.
5. Only then close P0 and begin the formal P1 collection protocol.

## Repository CI note

The separate Go Test workflow currently has a base-branch compile failure: ExecutionFailed is declared in both execution/types.go and execution/provider.go. This is unrelated to the piano research changes and should be fixed in a separate platform change. Do not conflate this with the Piano AI P0 workflow.

## Data and rights guardrails

- MAESTRO is CC BY-NC-SA 4.0; do not assume it can be used in a commercial training or product pipeline.
- The archived ByteDance piano transcription stack must be isolated and security-reviewed.
- Code, pretrained weights, training data, sheet music and user recordings each require separate rights/provenance records.
- Do not commit raw recordings, MIDI or copyrighted scores without explicit permission.
- A model's own transcription is not ground truth; acoustic-piano evaluation needs human-reviewed reference annotations.
