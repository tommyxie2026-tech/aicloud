package domain

import (
	"context"
	"fmt"
	"time"
)

type LicensePermission string

const (
	LicenseAllowed     LicensePermission = "allowed"
	LicenseConditional LicensePermission = "conditional"
	LicenseForbidden   LicensePermission = "forbidden"
)

type LicenseApprovalState string

const (
	LicensePending  LicenseApprovalState = "pending"
	LicenseApproved LicenseApprovalState = "approved"
	LicenseRejected LicenseApprovalState = "rejected"
	LicenseRevoked  LicenseApprovalState = "revoked"
)

type CommercialThreshold struct {
	Metric   string  `json:"metric,omitempty"`
	Operator string  `json:"operator,omitempty"`
	Value    float64 `json:"value,omitempty"`
	Unit     string  `json:"unit,omitempty"`
}

type LicenseEvidenceVersion struct {
	ID                  string                `json:"id"`
	Version             string                `json:"version"`
	ModelVersionID      string                `json:"modelVersionId"`
	LicenseID           string                `json:"licenseId"`
	WeightAvailability  LicensePermission     `json:"weightAvailability"`
	CommercialUse       LicensePermission     `json:"commercialUse"`
	HostedService       LicensePermission     `json:"hostedService"`
	Redistribution      LicensePermission     `json:"redistribution"`
	DerivativeWorks     LicensePermission     `json:"derivativeWorks"`
	AttributionRequired bool                  `json:"attributionRequired"`
	NoticeRequired      bool                  `json:"noticeRequired"`
	Thresholds          []CommercialThreshold `json:"thresholds,omitempty"`
	RevenueShareRef     string                `json:"revenueShareRef,omitempty"`
	AdditionalFeeRef    string                `json:"additionalFeeRef,omitempty"`
	AllowedGeographies  []string              `json:"allowedGeographies,omitempty"`
	BlockedGeographies  []string              `json:"blockedGeographies,omitempty"`
	AllowedCustomerTags []string              `json:"allowedCustomerTags,omitempty"`
	BlockedCustomerTags []string              `json:"blockedCustomerTags,omitempty"`
	EffectiveFrom       time.Time             `json:"effectiveFrom"`
	EffectiveTo         *time.Time            `json:"effectiveTo,omitempty"`
	ReviewAfter         *time.Time            `json:"reviewAfter,omitempty"`
	EvidenceRef         string                `json:"evidenceRef"`
	EvidenceDigest      string                `json:"evidenceDigest"`
	Reviewer            string                `json:"reviewer,omitempty"`
	ApprovalState       LicenseApprovalState  `json:"approvalState"`
	CreatedAt           time.Time             `json:"createdAt"`
}

func (e LicenseEvidenceVersion) Ref() string { return e.ID + "@" + e.Version }

func (e LicenseEvidenceVersion) Validate() error {
	if e.ID == "" || e.Version == "" || e.ModelVersionID == "" {
		return fmt.Errorf("license evidence id, version and model version id are required")
	}
	if e.LicenseID == "" || e.EvidenceRef == "" || e.EvidenceDigest == "" {
		return fmt.Errorf("license id and authoritative evidence reference/digest are required")
	}
	if e.EffectiveFrom.IsZero() {
		return fmt.Errorf("license evidence effective time is required")
	}
	if e.EffectiveTo != nil && !e.EffectiveTo.After(e.EffectiveFrom) {
		return fmt.Errorf("license evidence effective end must follow effective start")
	}
	for _, permission := range []LicensePermission{e.WeightAvailability, e.CommercialUse, e.HostedService, e.Redistribution, e.DerivativeWorks} {
		if permission != LicenseAllowed && permission != LicenseConditional && permission != LicenseForbidden {
			return fmt.Errorf("invalid license permission %q", permission)
		}
	}
	return nil
}

type LicenseEvidenceVersionRepository interface {
	Create(context.Context, LicenseEvidenceVersion) (LicenseEvidenceVersion, error)
	Get(context.Context, string, string) (LicenseEvidenceVersion, error)
	ListByModelVersion(context.Context, string) ([]LicenseEvidenceVersion, error)
	Resolve(context.Context, string, time.Time) (LicenseEvidenceVersion, error)
}
