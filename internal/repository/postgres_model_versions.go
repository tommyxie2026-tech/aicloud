package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresModelVersions struct{ db *sql.DB }

func NewPostgresModelVersions(db *sql.DB) *PostgresModelVersions {
	if db == nil {
		return nil
	}
	return &PostgresModelVersions{db: db}
}

const modelVersionColumns = `id, name, version, lifecycle_state, capabilities,
	evaluation_version, license, license_evidence, license_evidence_version_ref,
	provenance, artifact_digest, approval_status, risk_level, created_at, updated_at`

func (r *PostgresModelVersions) List(ctx context.Context) ([]domain.ModelVersion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+modelVersionColumns+` FROM model_versions ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list model versions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ModelVersion, 0)
	for rows.Next() {
		item, err := scanModelVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresModelVersions) Get(ctx context.Context, id string) (domain.ModelVersion, error) {
	item, err := scanModelVersion(r.db.QueryRowContext(ctx, `SELECT `+modelVersionColumns+` FROM model_versions WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ModelVersion{}, ErrNotFound
	}
	return item, err
}

func (r *PostgresModelVersions) Create(ctx context.Context, item domain.ModelVersion) (domain.ModelVersion, error) {
	capabilities, _ := json.Marshal(item.Capabilities)
	licenseEvidence, _ := json.Marshal(item.LicenseEvidence)
	provenance, _ := json.Marshal(item.Provenance)
	_, err := r.db.ExecContext(ctx, `INSERT INTO model_versions (
		id, name, version, lifecycle_state, capabilities, evaluation_version,
		license, license_evidence, license_evidence_version_ref, provenance,
		artifact_digest, approval_status, risk_level, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		item.ID, item.Name, item.Version, item.Lifecycle, capabilities,
		item.EvaluationVersion, item.License, licenseEvidence, item.LicenseEvidenceVersionRef,
		provenance, item.ArtifactDigest, item.ApprovalStatus, item.RiskLevel,
		item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.ModelVersion{}, fmt.Errorf("create model version: %w", err)
	}
	return item, nil
}

func (r *PostgresModelVersions) Update(ctx context.Context, item domain.ModelVersion) (domain.ModelVersion, error) {
	capabilities, _ := json.Marshal(item.Capabilities)
	licenseEvidence, _ := json.Marshal(item.LicenseEvidence)
	provenance, _ := json.Marshal(item.Provenance)
	result, err := r.db.ExecContext(ctx, `UPDATE model_versions SET
		name=$2, version=$3, lifecycle_state=$4, capabilities=$5,
		evaluation_version=$6, license=$7, license_evidence=$8,
		license_evidence_version_ref=$9, provenance=$10, artifact_digest=$11,
		approval_status=$12, risk_level=$13, updated_at=$14
		WHERE id=$1`, item.ID, item.Name, item.Version, item.Lifecycle,
		capabilities, item.EvaluationVersion, item.License, licenseEvidence,
		item.LicenseEvidenceVersionRef, provenance, item.ArtifactDigest,
		item.ApprovalStatus, item.RiskLevel, item.UpdatedAt)
	if err != nil {
		return domain.ModelVersion{}, fmt.Errorf("update model version: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return domain.ModelVersion{}, err
	}
	return item, nil
}

func scanModelVersion(row scanner) (domain.ModelVersion, error) {
	var item domain.ModelVersion
	var capabilities, licenseEvidence, provenance []byte
	if err := row.Scan(
		&item.ID, &item.Name, &item.Version, &item.Lifecycle, &capabilities,
		&item.EvaluationVersion, &item.License, &licenseEvidence,
		&item.LicenseEvidenceVersionRef, &provenance, &item.ArtifactDigest,
		&item.ApprovalStatus, &item.RiskLevel, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return domain.ModelVersion{}, err
	}
	if err := unmarshalJSON(capabilities, &item.Capabilities); err != nil {
		return domain.ModelVersion{}, err
	}
	if err := unmarshalJSON(licenseEvidence, &item.LicenseEvidence); err != nil {
		return domain.ModelVersion{}, err
	}
	if err := unmarshalJSON(provenance, &item.Provenance); err != nil {
		return domain.ModelVersion{}, err
	}
	return item, nil
}
