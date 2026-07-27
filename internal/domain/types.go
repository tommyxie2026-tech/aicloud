package domain

import (
	"context"
	"time"
)

type Model struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`
	Capabilities []string       `json:"capabilities,omitempty"`
	Pricing      PricingProfile `json:"pricing,omitempty"`
	License      string         `json:"license,omitempty"`
	RiskLevel    string         `json:"riskLevel,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type PricingProfile struct {
	InputPerMillion  float64 `json:"inputPerMillion,omitempty"`
	OutputPerMillion float64 `json:"outputPerMillion,omitempty"`
}

type Agent struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	ModelID  string   `json:"modelId"`
	Workflow string   `json:"workflow,omitempty"`
	Tools    []string `json:"tools,omitempty"`
	Sandbox  string   `json:"sandbox,omitempty"`
	PolicyID string   `json:"policyId,omitempty"`
}

type Tool struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	RiskLevel      string `json:"riskLevel"`
	Permission     string `json:"permission,omitempty"`
	CredentialMode string `json:"credentialMode,omitempty"`
}

type Policy struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RequireApproval bool   `json:"requireApproval"`
	Network         string `json:"network,omitempty"`
}

type TaskStatus string

const (
	TaskPending  TaskStatus = "PENDING"
	TaskRunning  TaskStatus = "RUNNING"
	TaskComplete TaskStatus = "COMPLETED"
	TaskFailed   TaskStatus = "FAILED"
)

type Task struct {
	ID        string     `json:"id"`
	AgentID   string     `json:"agentId"`
	Input     string     `json:"input"`
	Status    TaskStatus `json:"status"`
	Result    string     `json:"result,omitempty"`
	Cost      float64    `json:"cost,omitempty"`
	TraceID   string     `json:"traceId"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type ModelRepository interface {
	List(context.Context) ([]Model, error)
	Get(context.Context, string) (Model, error)
	Create(context.Context, Model) (Model, error)
}

type TaskRepository interface {
	List(context.Context) ([]Task, error)
	Get(context.Context, string) (Task, error)
	Create(context.Context, Task) (Task, error)
	Update(context.Context, Task) (Task, error)
}
