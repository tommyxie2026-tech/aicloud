package domain

import (
	"context"
	"time"
)

type ModelLifecycle string

const (
	ModelDraft      ModelLifecycle = "draft"
	ModelActive     ModelLifecycle = "active"
	ModelDegraded   ModelLifecycle = "degraded"
	ModelDeprecated ModelLifecycle = "deprecated"
	ModelRetired    ModelLifecycle = "retired"
	ModelRevoked    ModelLifecycle = "revoked"
)

type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type ApprovalStatus string

const (
	ApprovalDiscovered        ApprovalStatus = "discovered"
	ApprovalEvidenceCollected ApprovalStatus = "evidence-collected"
	ApprovalReviewed          ApprovalStatus = "reviewed"
	ApprovalApproved          ApprovalStatus = "approved"
	ApprovalRestricted        ApprovalStatus = "restricted"
	ApprovalRejected          ApprovalStatus = "rejected"
	ApprovalRevoked           ApprovalStatus = "revoked"
)

type DeploymentMode string

const (
	DeploymentPublicAPI       DeploymentMode = "public-api"
	DeploymentEnterpriseAPI   DeploymentMode = "enterprise-api"
	DeploymentPrivateEndpoint DeploymentMode = "private-endpoint"
	DeploymentSelfHosted      DeploymentMode = "self-hosted"
	DeploymentLocal           DeploymentMode = "local"
)

type InferenceEffort string

const (
	EffortMinimal InferenceEffort = "minimal"
	EffortLow     InferenceEffort = "low"
	EffortMedium  InferenceEffort = "medium"
	EffortHigh    InferenceEffort = "high"
)

type ServiceTier string

const (
	TierStandard    ServiceTier = "standard"
	TierPriority    ServiceTier = "priority"
	TierBatch       ServiceTier = "batch"
	TierProvisioned ServiceTier = "provisioned"
)

type LicenseEvidence struct {
	LicenseID             string     `json:"licenseId,omitempty"`
	LicenseTextRef        string     `json:"licenseTextRef,omitempty"`
	UpstreamModel         string     `json:"upstreamModel,omitempty"`
	CommercialUseAllowed  bool       `json:"commercialUseAllowed"`
	HostedServiceAllowed  bool       `json:"hostedServiceAllowed"`
	RedistributionAllowed bool       `json:"redistributionAllowed"`
	NoticeRequired        bool       `json:"noticeRequired"`
	Reviewer              string     `json:"reviewer,omitempty"`
	ReviewedAt            *time.Time `json:"reviewedAt,omitempty"`
}

type ModelProvenance struct {
	Source       string   `json:"source,omitempty"`
	BaseModels   []string `json:"baseModels,omitempty"`
	Datasets     []string `json:"datasets,omitempty"`
	FineTunings  []string `json:"fineTunings,omitempty"`
	BuildProcess string   `json:"buildProcess,omitempty"`
}

type Model struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	Provider          string            `json:"provider"`
	Endpoint          string            `json:"endpoint,omitempty"`
	DeploymentMode    DeploymentMode    `json:"deploymentMode,omitempty"`
	Lifecycle         ModelLifecycle    `json:"lifecycle"`
	Capabilities      []string          `json:"capabilities,omitempty"`
	Pricing           PricingProfile    `json:"pricing,omitempty"`
	Health            HealthStatus      `json:"health"`
	HealthCheckedAt   *time.Time        `json:"healthCheckedAt,omitempty"`
	P95LatencyMS      int64             `json:"p95LatencyMs,omitempty"`
	ErrorRate         float64           `json:"errorRate,omitempty"`
	QuotaRemaining    int64             `json:"quotaRemaining,omitempty"`
	CapacityAvailable int64             `json:"capacityAvailable,omitempty"`
	QueueDepth        int64             `json:"queueDepth,omitempty"`
	ServiceTiers      []ServiceTier     `json:"serviceTiers,omitempty"`
	InferenceEfforts  []InferenceEffort `json:"inferenceEfforts,omitempty"`
	EvaluationVersion string            `json:"evaluationVersion,omitempty"`
	License           string            `json:"license,omitempty"`
	LicenseEvidence   LicenseEvidence   `json:"licenseEvidence,omitempty"`
	Provenance        ModelProvenance   `json:"provenance,omitempty"`
	ArtifactDigest    string            `json:"artifactDigest,omitempty"`
	ApprovalStatus    ApprovalStatus    `json:"approvalStatus"`
	RiskLevel         string            `json:"riskLevel,omitempty"`
	DataResidency     string            `json:"dataResidency,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

type PricingProfile struct {
	Currency          string  `json:"currency,omitempty"`
	InputPerMillion   float64 `json:"inputPerMillion,omitempty"`
	OutputPerMillion  float64 `json:"outputPerMillion,omitempty"`
	CachedPerMillion  float64 `json:"cachedPerMillion,omitempty"`
	ServiceTierFactor float64 `json:"serviceTierFactor,omitempty"`
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
	ID              string     `json:"id"`
	AgentID         string     `json:"agentId"`
	Input           string     `json:"input"`
	Status          TaskStatus `json:"status"`
	Result          string     `json:"result,omitempty"`
	Cost            float64    `json:"cost,omitempty"`
	EstimatedCost   float64    `json:"estimatedCost,omitempty"`
	ActualCost      float64    `json:"actualCost,omitempty"`
	Currency        string     `json:"currency,omitempty"`
	RouteDecisionID string     `json:"routeDecisionId,omitempty"`
	TraceID         string     `json:"traceId"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type RouteClass string

const (
	RouteDeterministic RouteClass = "deterministic"
	RouteEfficient     RouteClass = "efficient"
	RouteSpecialist    RouteClass = "specialist"
	RouteFlagship      RouteClass = "flagship"
)

type RouteCandidate struct {
	ModelID          string          `json:"modelId,omitempty"`
	ModelVersion     string          `json:"modelVersion,omitempty"`
	RouteClass       RouteClass      `json:"routeClass"`
	InferenceEffort  InferenceEffort `json:"inferenceEffort,omitempty"`
	ServiceTier      ServiceTier     `json:"serviceTier,omitempty"`
	EstimatedCost    float64         `json:"estimatedCost,omitempty"`
	Score            float64         `json:"score,omitempty"`
	RejectionReasons []string        `json:"rejectionReasons,omitempty"`
}

type RouteDecision struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"taskId"`
	Selected        RouteCandidate   `json:"selected"`
	Candidates      []RouteCandidate `json:"candidates"`
	Reason          string           `json:"reason"`
	FallbackChain   []RouteCandidate `json:"fallbackChain,omitempty"`
	EvidenceVersion string           `json:"evidenceVersion,omitempty"`
	PolicyVersion   string           `json:"policyVersion,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
}

type CostComponent string

const (
	CostModelInput  CostComponent = "model-input"
	CostModelOutput CostComponent = "model-output"
	CostModelCache  CostComponent = "model-cache"
	CostServiceTier CostComponent = "service-tier"
	CostTool        CostComponent = "tool"
	CostWorkflow    CostComponent = "workflow"
	CostSandbox     CostComponent = "sandbox"
	CostStorage     CostComponent = "storage"
	CostNetwork     CostComponent = "network"
	CostRetry       CostComponent = "retry"
	CostEvaluation  CostComponent = "evaluation"
	CostHumanReview CostComponent = "human-review"
)

type CostEvent struct {
	ID           string            `json:"id"`
	TaskID       string            `json:"taskId"`
	TraceID      string            `json:"traceId"`
	Component    CostComponent     `json:"component"`
	Provider     string            `json:"provider,omitempty"`
	ModelID      string            `json:"modelId,omitempty"`
	ModelVersion string            `json:"modelVersion,omitempty"`
	Quantity     float64           `json:"quantity"`
	Unit         string            `json:"unit"`
	UnitPrice    float64           `json:"unitPrice"`
	Amount       float64           `json:"amount"`
	Currency     string            `json:"currency"`
	Attempt      int               `json:"attempt,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
}

type ModelRepository interface {
	List(context.Context) ([]Model, error)
	Get(context.Context, string) (Model, error)
	Create(context.Context, Model) (Model, error)
	Update(context.Context, Model) (Model, error)
}

type TaskRepository interface {
	List(context.Context) ([]Task, error)
	Get(context.Context, string) (Task, error)
	Create(context.Context, Task) (Task, error)
	Update(context.Context, Task) (Task, error)
}

type RouteDecisionRepository interface {
	Create(context.Context, RouteDecision) (RouteDecision, error)
	Get(context.Context, string) (RouteDecision, error)
	ListByTask(context.Context, string) ([]RouteDecision, error)
}

type CostEventRepository interface {
	Append(context.Context, CostEvent) (CostEvent, error)
	ListByTask(context.Context, string) ([]CostEvent, error)
}
