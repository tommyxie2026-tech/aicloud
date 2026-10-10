package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type LicenseUseContext struct {
	Commercial      bool
	HostedService   bool
	Redistribution  bool
	DerivativeWorks bool
	Geography       string
	CustomerTags    []string
	Metrics         map[string]float64
	At              time.Time
}

type LicenseDecision struct {
	Allowed  bool
	Evidence string
	Reasons  []string
}

func EvaluateLicense(e domain.LicenseEvidenceVersion, use LicenseUseContext) LicenseDecision {
	decision := LicenseDecision{Evidence: e.Ref()}
	if err := e.Validate(); err != nil {
		decision.Reasons = append(decision.Reasons, "invalid license evidence: "+err.Error())
		return decision
	}
	if e.ApprovalState != domain.LicenseApproved {
		decision.Reasons = append(decision.Reasons, "license evidence is not approved")
		return decision
	}
	at := use.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if at.Before(e.EffectiveFrom) || (e.EffectiveTo != nil && !at.Before(*e.EffectiveTo)) {
		decision.Reasons = append(decision.Reasons, "license evidence is not effective at use time")
		return decision
	}
	checkPermission := func(required bool, permission domain.LicensePermission, name string) {
		if !required {
			return
		}
		if permission == domain.LicenseForbidden {
			decision.Reasons = append(decision.Reasons, name+" is forbidden")
		} else if permission == domain.LicenseConditional {
			decision.Reasons = append(decision.Reasons, name+" requires conditional review")
		}
	}
	checkPermission(use.Commercial, e.CommercialUse, "commercial use")
	checkPermission(use.HostedService, e.HostedService, "hosted service")
	checkPermission(use.Redistribution, e.Redistribution, "redistribution")
	checkPermission(use.DerivativeWorks, e.DerivativeWorks, "derivative works")
	if geographyBlocked(use.Geography, e.AllowedGeographies, e.BlockedGeographies) {
		decision.Reasons = append(decision.Reasons, "geography is not permitted")
	}
	if customerBlocked(use.CustomerTags, e.AllowedCustomerTags, e.BlockedCustomerTags) {
		decision.Reasons = append(decision.Reasons, "customer class is not permitted")
	}
	for _, threshold := range e.Thresholds {
		if threshold.Metric == "" {
			continue
		}
		value, ok := use.Metrics[threshold.Metric]
		if !ok || !thresholdSatisfied(value, threshold.Operator, threshold.Value) {
			decision.Reasons = append(decision.Reasons, fmt.Sprintf("commercial threshold %s %s %g is not satisfied", threshold.Metric, threshold.Operator, threshold.Value))
		}
	}
	decision.Allowed = len(decision.Reasons) == 0
	return decision
}

func geographyBlocked(value string, allowed, blocked []string) bool {
	if value == "" {
		return len(allowed) > 0
	}
	for _, item := range blocked {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	if len(allowed) == 0 {
		return false
	}
	for _, item := range allowed {
		if strings.EqualFold(item, value) {
			return false
		}
	}
	return true
}

func customerBlocked(tags, allowed, blocked []string) bool {
	set := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		set[strings.ToLower(tag)] = struct{}{}
	}
	for _, tag := range blocked {
		if _, ok := set[strings.ToLower(tag)]; ok {
			return true
		}
	}
	if len(allowed) == 0 {
		return false
	}
	for _, tag := range allowed {
		if _, ok := set[strings.ToLower(tag)]; ok {
			return false
		}
	}
	return true
}

func thresholdSatisfied(actual float64, operator string, expected float64) bool {
	switch operator {
	case "<":
		return actual < expected
	case "<=":
		return actual <= expected
	case ">":
		return actual > expected
	case ">=":
		return actual >= expected
	case "=", "==":
		return actual == expected
	default:
		return false
	}
}
