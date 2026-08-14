package policy

import (
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestEvaluateLicenseCommercialRights(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	base := domain.LicenseEvidenceVersion{
		ID: "license-model", Version: "v1", ModelVersionID: "model-v1", LicenseID: "test-license",
		WeightAvailability: domain.LicenseAllowed, CommercialUse: domain.LicenseAllowed,
		HostedService: domain.LicenseAllowed, Redistribution: domain.LicenseForbidden,
		DerivativeWorks: domain.LicenseConditional,
		EffectiveFrom: now.Add(-time.Hour), EvidenceRef: "https://example.test/license",
		EvidenceDigest: "sha256:test", Reviewer: "reviewer", ApprovalState: domain.LicenseApproved,
		CreatedAt: now.Add(-time.Hour),
	}

	allowed := EvaluateLicense(base, LicenseUseContext{Commercial: true, HostedService: true, At: now})
	if !allowed.Allowed {
		t.Fatalf("approved commercial use should be allowed: %#v", allowed)
	}

	forbidden := base
	forbidden.CommercialUse = domain.LicenseForbidden
	decision := EvaluateLicense(forbidden, LicenseUseContext{Commercial: true, At: now})
	if decision.Allowed {
		t.Fatal("forbidden commercial use must be rejected")
	}

	conditional := base
	conditional.HostedService = domain.LicenseConditional
	decision = EvaluateLicense(conditional, LicenseUseContext{Commercial: true, HostedService: true, At: now})
	if decision.Allowed {
		t.Fatal("conditional hosted-service rights must fail closed without explicit review")
	}
}

func TestEvaluateLicenseRestrictionsAndApproval(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	evidence := domain.LicenseEvidenceVersion{
		ID: "license-model", Version: "v2", ModelVersionID: "model-v1", LicenseID: "test-license",
		WeightAvailability: domain.LicenseAllowed, CommercialUse: domain.LicenseAllowed,
		HostedService: domain.LicenseAllowed, Redistribution: domain.LicenseAllowed,
		DerivativeWorks: domain.LicenseAllowed,
		BlockedGeographies: []string{"US"}, BlockedCustomerTags: []string{"restricted"},
		Thresholds: []domain.CommercialThreshold{{Metric: "annual_revenue_usd", Operator: "<", Value: 1_000_000, Unit: "USD"}},
		EffectiveFrom: now.Add(-time.Hour), EvidenceRef: "https://example.test/license-v2",
		EvidenceDigest: "sha256:test-v2", Reviewer: "reviewer", ApprovalState: domain.LicenseApproved,
		CreatedAt: now.Add(-time.Hour),
	}
	decision := EvaluateLicense(evidence, LicenseUseContext{
		Commercial: true, Geography: "US", CustomerTags: []string{"restricted"},
		Metrics: map[string]float64{"annual_revenue_usd": 2_000_000}, At: now,
	})
	if decision.Allowed || len(decision.Reasons) < 3 {
		t.Fatalf("geography, customer and threshold violations must reject: %#v", decision)
	}

	evidence.ApprovalState = domain.LicensePending
	decision = EvaluateLicense(evidence, LicenseUseContext{Commercial: true, At: now})
	if decision.Allowed {
		t.Fatal("unapproved evidence must be rejected")
	}
}
