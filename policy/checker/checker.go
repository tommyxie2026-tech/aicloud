package checker

import (
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
)

// Checker performs deterministic policy evaluation for ChangeProposal.
//
// It must be the source of truth for risk and approval decisions. Model risk
// hints are treated only as hints and must not decide approval by themselves.
type Checker struct {
	policy Policy
}

func NewChecker(policy Policy) *Checker {
	if len(policy.Rules) == 0 {
		policy = DefaultPolicy()
	}
	return &Checker{policy: policy}
}

func (c *Checker) Evaluate(p *proposal.ChangeProposal) (proposal.PolicyResult, error) {
	if p == nil {
		return proposal.PolicyResult{}, NewPolicyError("NilProposal", "proposal is nil")
	}
	if len(p.Changes) == 0 {
		return proposal.PolicyResult{}, NewPolicyError("MissingChanges", "proposal changes must not be empty")
	}

	for _, rule := range c.policy.Rules {
		if rule.matches(p) {
			return proposal.PolicyResult{
				RiskLevel:        string(rule.RiskLevel),
				ApprovalRequired: rule.ApprovalRequired,
				PolicyName:       c.policy.Name,
				MatchedRule:      rule.Name,
				Result:           "PASS",
				Reason:           rule.Reason,
			}, nil
		}
	}

	return proposal.PolicyResult{
		RiskLevel:        string(RiskHigh),
		ApprovalRequired: true,
		PolicyName:       c.policy.Name,
		MatchedRule:      "fail-closed",
		Result:           "REVIEW_REQUIRED",
		Reason:           "no explicit policy rule matched; fail closed",
	}, nil
}

type Policy struct {
	Name  string
	Rules []Rule
}

type Rule struct {
	Name             string
	Environment      string
	OperationType    string
	TargetKind       string
	AllowedField     string
	MaxReplicaDelta  int
	RiskLevel        RiskLevel
	ApprovalRequired bool
	Reason           string
}

type RiskLevel string

const (
	RiskLow      RiskLevel = "Low"
	RiskMedium   RiskLevel = "Medium"
	RiskHigh     RiskLevel = "High"
	RiskCritical RiskLevel = "Critical"
)

func DefaultPolicy() Policy {
	return Policy{
		Name: "default-risk-policy",
		Rules: []Rule{
			{
				Name:             "dev-managedcluster-small-scale",
				Environment:      "dev",
				OperationType:    "ScaleOut",
				TargetKind:       "ManagedCluster",
				AllowedField:     "spec.workers[name=gpu-workers].replicas",
				MaxReplicaDelta:  3,
				RiskLevel:        RiskMedium,
				ApprovalRequired: false,
				Reason:           "dev ManagedCluster scale-out within small replica delta",
			},
			{
				Name:             "staging-managedcluster-small-scale",
				Environment:      "staging",
				OperationType:    "ScaleOut",
				TargetKind:       "ManagedCluster",
				AllowedField:     "spec.workers[name=gpu-workers].replicas",
				MaxReplicaDelta:  3,
				RiskLevel:        RiskMedium,
				ApprovalRequired: true,
				Reason:           "staging infrastructure change requires review",
			},
		},
	}
}

func (r Rule) matches(p *proposal.ChangeProposal) bool {
	if r.Environment != "" && r.Environment != p.Environment {
		return false
	}
	if r.OperationType != "" && r.OperationType != p.OperationType {
		return false
	}
	if r.TargetKind != "" && r.TargetKind != p.Target.Kind {
		return false
	}
	for _, change := range p.Changes {
		if r.AllowedField != "" && change.Field != r.AllowedField {
			return false
		}
		if r.MaxReplicaDelta > 0 && replicaDelta(change.From, change.To) > r.MaxReplicaDelta {
			return false
		}
	}
	return true
}

func replicaDelta(from any, to any) int {
	fromInt, okFrom := toInt(from)
	toIntValue, okTo := toInt(to)
	if !okFrom || !okTo {
		return 0
	}
	if toIntValue >= fromInt {
		return toIntValue - fromInt
	}
	return fromInt - toIntValue
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	default:
		return 0, false
	}
}

type PolicyError struct {
	Code    string
	Message string
}

func NewPolicyError(code string, message string) *PolicyError {
	return &PolicyError{Code: code, Message: message}
}

func (e *PolicyError) Error() string {
	return e.Code + ": " + e.Message
}

func Explain(result proposal.PolicyResult) string {
	parts := []string{result.Result, result.RiskLevel, result.MatchedRule, result.Reason}
	return strings.Join(parts, " | ")
}

func ResultSummary(result proposal.PolicyResult) string {
	return fmt.Sprintf("risk=%s approvalRequired=%v rule=%s", result.RiskLevel, result.ApprovalRequired, result.MatchedRule)
}
