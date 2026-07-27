package policy

import (
	"context"
	"strings"
)

type Decision struct {
	Allowed          bool   `json:"allowed"`
	RequiresApproval bool   `json:"requiresApproval"`
	Reason           string `json:"reason"`
	PolicyVersion    string `json:"policyVersion,omitempty"`
	MatchedRule      string `json:"matchedRule,omitempty"`
}

type Engine interface {
	Evaluate(context.Context, string, string, string) (Decision, error)
}

type DenyByDefault struct{}

func (DenyByDefault) Evaluate(context.Context, string, string, string) (Decision, error) {
	return Decision{Allowed: false, RequiresApproval: true, Reason: "policy engine is fail-closed in the skeleton runtime", PolicyVersion: "deny-by-default-v1"}, nil
}

type Rule struct {
	Name             string
	Subject          string
	Action           string
	Resource         string
	Allowed          bool
	RequireApproval  bool
	Reason           string
}

type StaticEngine struct {
	Version string
	Rules   []Rule
}

func (e StaticEngine) Evaluate(_ context.Context, subject, action, resource string) (Decision, error) {
	for _, rule := range e.Rules {
		if matches(rule.Subject, subject) && matches(rule.Action, action) && matches(rule.Resource, resource) {
			return Decision{
				Allowed:          rule.Allowed,
				RequiresApproval: rule.RequireApproval,
				Reason:           rule.Reason,
				PolicyVersion:    e.Version,
				MatchedRule:      rule.Name,
			}, nil
		}
	}
	return Decision{Allowed: false, Reason: "no policy rule matched", PolicyVersion: e.Version}, nil
}

func matches(pattern, value string) bool {
	return pattern == "*" || strings.EqualFold(pattern, value)
}
