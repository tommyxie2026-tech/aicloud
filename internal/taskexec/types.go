package taskexec

import "time"

// taskexec deliberately does not define a second Task aggregate or Task state
// machine. internal/domain.Task is the canonical Task aggregate. This package
// owns execution-plan and execution-run contracts only.
type RunPhase string
type StepType string

const (
	RunAccepted  RunPhase = "accepted"
	RunRunning   RunPhase = "running"
	RunSuspended RunPhase = "suspended"
	RunSucceeded RunPhase = "succeeded"
	RunFailed    RunPhase = "failed"
	RunCancelled RunPhase = "cancelled"
)

const (
	StepModel     StepType = "model"
	StepAgent     StepType = "agent"
	StepTool      StepType = "tool"
	StepEvaluator StepType = "evaluator"
	StepHuman     StepType = "human"
)

type PlanStep struct {
	ID         string   `json:"id"`
	Type       StepType `json:"type"`
	Capability string   `json:"capability,omitempty"`
	Target     string   `json:"target,omitempty"`
	DependsOn  []string `json:"dependsOn,omitempty"`
}

type PolicyDecision struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ExecutionPlan is a versioned plan for an existing domain.Task. Task intent,
// ownership and lifecycle remain on the domain aggregate.
type ExecutionPlan struct {
	ID             string         `json:"id"`
	TaskID         string         `json:"taskId"`
	Revision       int            `json:"revision"`
	Steps          []PlanStep     `json:"steps"`
	PolicyDecision PolicyDecision `json:"policyDecision"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type Usage struct {
	InputTokens     int64         `json:"inputTokens,omitempty"`
	OutputTokens    int64         `json:"outputTokens,omitempty"`
	CacheReadTokens int64         `json:"cacheReadTokens,omitempty"`
	CostUSD         float64       `json:"costUsd,omitempty"`
	Latency         time.Duration `json:"latency,omitempty"`
}

// ExecutionRun represents one attempt to execute one immutable plan revision.
// Its lifecycle is intentionally independent from domain.TaskStatus: a Task can
// be replanned and can have multiple run attempts while retaining one canonical
// aggregate lifecycle.
type ExecutionRun struct {
	ID             string     `json:"id"`
	TaskID         string     `json:"taskId"`
	PlanID         string     `json:"planId"`
	Attempt        int        `json:"attempt"`
	Phase          RunPhase   `json:"phase"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	Usage          Usage      `json:"usage"`
	OutcomeSuccess bool       `json:"outcomeSuccess"`
}
