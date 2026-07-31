package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/admission"
	"github.com/tommyxie2026-tech/aicloud/internal/evaluation"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
)

type PostgresTraceStore struct{ db *sql.DB }

func NewPostgresTraceStore(db *sql.DB) *PostgresTraceStore { return &PostgresTraceStore{db: db} }

func (s *PostgresTraceStore) Append(ctx context.Context, event tracepkg.Event) error {
	attributes, err := json.Marshal(event.Attributes)
	if err != nil {
		return fmt.Errorf("encode trace attributes: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO trace_events (
		id, trace_id, task_id, span_id, parent_span_id, name, kind, status,
		message, attributes, input_digest, output_digest, started_at, ended_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		event.ID, event.TraceID, event.TaskID, event.SpanID, event.ParentSpanID,
		event.Name, event.Kind, event.Status, event.Message, attributes,
		event.InputDigest, event.OutputDigest, event.StartedAt, event.EndedAt)
	if err != nil {
		return fmt.Errorf("append trace event: %w", err)
	}
	return nil
}

func (s *PostgresTraceStore) ListByTask(ctx context.Context, taskID string) ([]tracepkg.Event, error) {
	return s.list(ctx, `SELECT id, trace_id, task_id, span_id, parent_span_id,
		name, kind, status, message, attributes, input_digest, output_digest,
		started_at, ended_at FROM trace_events WHERE task_id=$1 ORDER BY started_at, id`, taskID)
}

func (s *PostgresTraceStore) ListByTrace(ctx context.Context, traceID string) ([]tracepkg.Event, error) {
	return s.list(ctx, `SELECT id, trace_id, task_id, span_id, parent_span_id,
		name, kind, status, message, attributes, input_digest, output_digest,
		started_at, ended_at FROM trace_events WHERE trace_id=$1 ORDER BY started_at, id`, traceID)
}

func (s *PostgresTraceStore) list(ctx context.Context, query, value string) ([]tracepkg.Event, error) {
	rows, err := s.db.QueryContext(ctx, query, value)
	if err != nil {
		return nil, fmt.Errorf("list trace events: %w", err)
	}
	defer rows.Close()
	items := make([]tracepkg.Event, 0)
	for rows.Next() {
		var event tracepkg.Event
		var attributes []byte
		var endedAt sql.NullTime
		if err := rows.Scan(&event.ID, &event.TraceID, &event.TaskID, &event.SpanID,
			&event.ParentSpanID, &event.Name, &event.Kind, &event.Status,
			&event.Message, &attributes, &event.InputDigest, &event.OutputDigest,
			&event.StartedAt, &endedAt); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			value := endedAt.Time
			event.EndedAt = &value
		}
		if err := unmarshalJSON(attributes, &event.Attributes); err != nil {
			return nil, fmt.Errorf("decode trace attributes: %w", err)
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

type PostgresEvaluationStore struct{ db *sql.DB }

func NewPostgresEvaluationStore(db *sql.DB) *PostgresEvaluationStore {
	return &PostgresEvaluationStore{db: db}
}

func (s *PostgresEvaluationStore) Append(ctx context.Context, run evaluation.Run) error {
	config, err := json.Marshal(run.Config)
	if err != nil {
		return fmt.Errorf("encode evaluation config: %w", err)
	}
	metrics, err := json.Marshal(run.Metrics)
	if err != nil {
		return fmt.Errorf("encode evaluation metrics: %w", err)
	}
	thresholds, err := json.Marshal(run.Thresholds)
	if err != nil {
		return fmt.Errorf("encode evaluation thresholds: %w", err)
	}
	gate, err := json.Marshal(run.Gate)
	if err != nil {
		return fmt.Errorf("encode evaluation gate: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO evaluation_runs (
		id, task_id, trace_id, config, config_digest, raw_output_digest,
		metrics, thresholds, gate_result, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, run.ID, run.TaskID,
		run.TraceID, config, run.ConfigDigest, run.RawOutputDigest, metrics,
		thresholds, gate, run.CreatedAt)
	if err != nil {
		return fmt.Errorf("append evaluation run: %w", err)
	}
	return nil
}

func (s *PostgresEvaluationStore) ListByTask(ctx context.Context, taskID string) ([]evaluation.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, trace_id, config,
		config_digest, raw_output_digest, metrics, thresholds, gate_result,
		created_at FROM evaluation_runs WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list evaluation runs: %w", err)
	}
	defer rows.Close()
	items := make([]evaluation.Run, 0)
	for rows.Next() {
		run, err := scanEvaluationRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, run)
	}
	return items, rows.Err()
}

func (s *PostgresEvaluationStore) Get(ctx context.Context, id string) (evaluation.Run, error) {
	run, err := scanEvaluationRun(s.db.QueryRowContext(ctx, `SELECT id, task_id,
		trace_id, config, config_digest, raw_output_digest, metrics, thresholds,
		gate_result, created_at FROM evaluation_runs WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return evaluation.Run{}, fmt.Errorf("evaluation run not found")
	}
	return run, err
}

func scanEvaluationRun(row scanner) (evaluation.Run, error) {
	var run evaluation.Run
	var config, metrics, thresholds, gate []byte
	if err := row.Scan(&run.ID, &run.TaskID, &run.TraceID, &config,
		&run.ConfigDigest, &run.RawOutputDigest, &metrics, &thresholds, &gate,
		&run.CreatedAt); err != nil {
		return evaluation.Run{}, err
	}
	if err := unmarshalJSON(config, &run.Config); err != nil {
		return evaluation.Run{}, fmt.Errorf("decode evaluation config: %w", err)
	}
	if err := unmarshalJSON(metrics, &run.Metrics); err != nil {
		return evaluation.Run{}, fmt.Errorf("decode evaluation metrics: %w", err)
	}
	if err := unmarshalJSON(thresholds, &run.Thresholds); err != nil {
		return evaluation.Run{}, fmt.Errorf("decode evaluation thresholds: %w", err)
	}
	if err := unmarshalJSON(gate, &run.Gate); err != nil {
		return evaluation.Run{}, fmt.Errorf("decode evaluation gate: %w", err)
	}
	return run, nil
}

type PostgresAdmissionStore struct{ db *sql.DB }

func NewPostgresAdmissionStore(db *sql.DB) *PostgresAdmissionStore {
	return &PostgresAdmissionStore{db: db}
}

func (s *PostgresAdmissionStore) Append(ctx context.Context, evidence admission.Evidence) error {
	upstream, err := json.Marshal(evidence.UpstreamModels)
	if err != nil {
		return fmt.Errorf("encode upstream models: %w", err)
	}
	datasets, err := json.Marshal(evidence.DatasetRefs)
	if err != nil {
		return fmt.Errorf("encode dataset references: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO model_admission_evidence (
		id, model_id, model_version, status, license_id, license_text_ref,
		source_ref, upstream_models, dataset_refs, commercial_use_allowed,
		hosted_service_allowed, redistribution_allowed, notice_required,
		notice_ref, artifact_digest, artifact_signature, security_scan_ref,
		reviewer, reviewed_at, evidence_digest, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		evidence.ID, evidence.ModelID, evidence.ModelVersion, evidence.Status,
		evidence.LicenseID, evidence.LicenseTextRef, evidence.SourceRef, upstream,
		datasets, evidence.CommercialUseAllowed, evidence.HostedServiceAllowed,
		evidence.RedistributionAllowed, evidence.NoticeRequired, evidence.NoticeRef,
		evidence.ArtifactDigest, evidence.ArtifactSignature, evidence.SecurityScanRef,
		evidence.Reviewer, evidence.ReviewedAt, evidence.EvidenceDigest, evidence.CreatedAt)
	if err != nil {
		return fmt.Errorf("append admission evidence: %w", err)
	}
	return nil
}

func (s *PostgresAdmissionStore) Get(ctx context.Context, id string) (admission.Evidence, error) {
	evidence, err := scanAdmissionEvidence(s.db.QueryRowContext(ctx, admissionSelect+` WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return admission.Evidence{}, fmt.Errorf("admission evidence not found")
	}
	return evidence, err
}

func (s *PostgresAdmissionStore) ListByModel(ctx context.Context, modelID, modelVersion string) ([]admission.Evidence, error) {
	rows, err := s.db.QueryContext(ctx, admissionSelect+` WHERE model_id=$1 AND model_version=$2 ORDER BY created_at`, modelID, modelVersion)
	if err != nil {
		return nil, fmt.Errorf("list admission evidence: %w", err)
	}
	defer rows.Close()
	items := make([]admission.Evidence, 0)
	for rows.Next() {
		evidence, err := scanAdmissionEvidence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, evidence)
	}
	return items, rows.Err()
}

const admissionSelect = `SELECT id, model_id, model_version, status, license_id,
	license_text_ref, source_ref, upstream_models, dataset_refs,
	commercial_use_allowed, hosted_service_allowed, redistribution_allowed,
	notice_required, notice_ref, artifact_digest, artifact_signature,
	security_scan_ref, reviewer, reviewed_at, evidence_digest, created_at
	FROM model_admission_evidence`

func scanAdmissionEvidence(row scanner) (admission.Evidence, error) {
	var evidence admission.Evidence
	var upstream, datasets []byte
	var reviewedAt sql.NullTime
	if err := row.Scan(&evidence.ID, &evidence.ModelID, &evidence.ModelVersion,
		&evidence.Status, &evidence.LicenseID, &evidence.LicenseTextRef,
		&evidence.SourceRef, &upstream, &datasets, &evidence.CommercialUseAllowed,
		&evidence.HostedServiceAllowed, &evidence.RedistributionAllowed,
		&evidence.NoticeRequired, &evidence.NoticeRef, &evidence.ArtifactDigest,
		&evidence.ArtifactSignature, &evidence.SecurityScanRef, &evidence.Reviewer,
		&reviewedAt, &evidence.EvidenceDigest, &evidence.CreatedAt); err != nil {
		return admission.Evidence{}, err
	}
	if reviewedAt.Valid {
		value := reviewedAt.Time
		evidence.ReviewedAt = &value
	}
	if err := unmarshalJSON(upstream, &evidence.UpstreamModels); err != nil {
		return admission.Evidence{}, fmt.Errorf("decode upstream models: %w", err)
	}
	if err := unmarshalJSON(datasets, &evidence.DatasetRefs); err != nil {
		return admission.Evidence{}, fmt.Errorf("decode dataset references: %w", err)
	}
	return evidence, nil
}
