package taskexec

import "time"

type TaskPhase string
type RunPhase string
type StepType string

const (
	TaskDraft TaskPhase = "draft"
	TaskAccepted TaskPhase = "accepted"
	TaskPlanned TaskPhase = "planned"
	TaskRunning TaskPhase = "running"
	TaskSucceeded TaskPhase = "succeeded"
	TaskFailed TaskPhase = "failed"
	TaskCancelled TaskPhase = "cancelled"
	TaskSuspended TaskPhase = "suspended"
	TaskRejectedByPolicy TaskPhase = "rejected_by_policy"
)

type TaskInput struct { Type, Ref string }
type TaskConstraints struct {
	MaxCostUSD float64
	Timeout time.Duration
	DataClass string
	RegionPolicy string
	HumanApproval string
}
type TaskDefinition struct {
	ID, Tenant, Project, Goal, TaskType string
	Inputs []TaskInput
	Constraints TaskConstraints
	SuccessCriteria []string
}
type PlanStep struct {
	ID string
	Type StepType
	Capability string
	Target string
	DependsOn []string
}
type PolicyDecision struct { ID, Status, Reason string }
type ExecutionPlan struct {
	ID, TaskID string
	Revision int
	Steps []PlanStep
	PolicyDecision PolicyDecision
}
type Usage struct {
	InputTokens, OutputTokens, CacheReadTokens int64
	CostUSD float64
	Latency time.Duration
}
type ExecutionRun struct {
	ID, TaskID, PlanID string
	Attempt int
	Phase RunPhase
	StartedAt, FinishedAt *time.Time
	Usage Usage
	OutcomeSuccess bool
}
