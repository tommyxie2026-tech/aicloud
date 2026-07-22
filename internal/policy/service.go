package policy

import "context"

type Decision struct {
	Allowed          bool   `json:"allowed"`
	RequiresApproval bool   `json:"requiresApproval"`
	Reason           string `json:"reason"`
}
type Engine interface {
	Evaluate(context.Context, string, string, string) (Decision, error)
}
type DenyByDefault struct{}

func (DenyByDefault) Evaluate(context.Context, string, string, string) (Decision, error) {
	return Decision{RequiresApproval: true, Reason: "policy engine is fail-closed in the skeleton runtime"}, nil
}
