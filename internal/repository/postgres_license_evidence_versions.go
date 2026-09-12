package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresLicenseEvidenceVersions struct{ db *sql.DB }

func NewPostgresLicenseEvidenceVersions(db *sql.DB) *PostgresLicenseEvidenceVersions {
	if db == nil {
		return nil
	}
	return &PostgresLicenseEvidenceVersions{db: db}
}

const licenseEvidenceColumns = `id, version, model_version_id, license_id,
	weight_availability, commercial_use, hosted_service, redistribution, derivative_works,
	attribution_required, notice_required, thresholds, revenue_share_ref, additional_fee_ref,
	allowed_geographies, blocked_geographies, allowed_customer_tags, blocked_customer_tags,
	effective_from, effective_to, review_after, evidence_ref, evidence_digest, reviewer,
	approval_state, created_at`

func (r *PostgresLicenseEvidenceVersions) Create(ctx context.Context, item domain.LicenseEvidenceVersion) (domain.LicenseEvidenceVersion, error) {
	if err := item.Validate(); err != nil {
		return domain.LicenseEvidenceVersion{}, err
	}
	thresholds, _ := json.Marshal(item.Thresholds)
	allowedGeo, _ := json.Marshal(item.AllowedGeographies)
	blockedGeo, _ := json.Marshal(item.BlockedGeographies)
	allowedTags, _ := json.Marshal(item.AllowedCustomerTags)
	blockedTags, _ := json.Marshal(item.BlockedCustomerTags)
	_, err := r.db.ExecContext(ctx, `INSERT INTO license_evidence_versions (`+licenseEvidenceColumns+`) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
		item.ID, item.Version, item.ModelVersionID, item.LicenseID,
		item.WeightAvailability, item.CommercialUse, item.HostedService, item.Redistribution, item.DerivativeWorks,
		item.AttributionRequired, item.NoticeRequired, thresholds, item.RevenueShareRef, item.AdditionalFeeRef,
		allowedGeo, blockedGeo, allowedTags, blockedTags, item.EffectiveFrom, item.EffectiveTo, item.ReviewAfter,
		item.EvidenceRef, item.EvidenceDigest, item.Reviewer, item.ApprovalState, item.CreatedAt)
	if err != nil {
		return domain.LicenseEvidenceVersion{}, fmt.Errorf("create license evidence version: %w", err)
	}
	return item, nil
}

func (r *PostgresLicenseEvidenceVersions) Get(ctx context.Context, id, version string) (domain.LicenseEvidenceVersion, error) {
	item, err := scanLicenseEvidence(r.db.QueryRowContext(ctx,
		`SELECT `+licenseEvidenceColumns+` FROM license_evidence_versions WHERE id=$1 AND version=$2`, id, version,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LicenseEvidenceVersion{}, ErrNotFound
	}
	return item, err
}

func (r *PostgresLicenseEvidenceVersions) ListByModelVersion(ctx context.Context, modelVersionID string) ([]domain.LicenseEvidenceVersion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+licenseEvidenceColumns+` FROM license_evidence_versions WHERE model_version_id=$1 ORDER BY effective_from DESC, created_at DESC, version DESC`, modelVersionID)
	if err != nil {
		return nil, fmt.Errorf("list license evidence versions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.LicenseEvidenceVersion, 0)
	for rows.Next() {
		item, scanErr := scanLicenseEvidence(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresLicenseEvidenceVersions) Resolve(ctx context.Context, modelVersionID string, at time.Time) (domain.LicenseEvidenceVersion, error) {
	item, err := scanLicenseEvidence(r.db.QueryRowContext(ctx, `SELECT `+licenseEvidenceColumns+` FROM license_evidence_versions
		WHERE model_version_id=$1 AND effective_from <= $2 AND (effective_to IS NULL OR effective_to > $2)
		ORDER BY effective_from DESC, created_at DESC, version DESC LIMIT 1`, modelVersionID, at,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LicenseEvidenceVersion{}, ErrNotFound
	}
	return item, err
}

func scanLicenseEvidence(row scanner) (domain.LicenseEvidenceVersion, error) {
	var item domain.LicenseEvidenceVersion
	var thresholds, allowedGeo, blockedGeo, allowedTags, blockedTags []byte
	if err := row.Scan(
		&item.ID, &item.Version, &item.ModelVersionID, &item.LicenseID,
		&item.WeightAvailability, &item.CommercialUse, &item.HostedService, &item.Redistribution, &item.DerivativeWorks,
		&item.AttributionRequired, &item.NoticeRequired, &thresholds, &item.RevenueShareRef, &item.AdditionalFeeRef,
		&allowedGeo, &blockedGeo, &allowedTags, &blockedTags, &item.EffectiveFrom, &item.EffectiveTo, &item.ReviewAfter,
		&item.EvidenceRef, &item.EvidenceDigest, &item.Reviewer, &item.ApprovalState, &item.CreatedAt,
	); err != nil {
		return domain.LicenseEvidenceVersion{}, err
	}
	for _, target := range []struct {
		body []byte
		into any
	}{
		{thresholds, &item.Thresholds},
		{allowedGeo, &item.AllowedGeographies},
		{blockedGeo, &item.BlockedGeographies},
		{allowedTags, &item.AllowedCustomerTags},
		{blockedTags, &item.BlockedCustomerTags},
	} {
		if err := json.Unmarshal(target.body, target.into); err != nil {
			return domain.LicenseEvidenceVersion{}, fmt.Errorf("decode license evidence JSON: %w", err)
		}
	}
	return item, nil
}
