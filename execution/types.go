package execution

import "time"

// ID types intentionally remain strings in v0.1 so storage/API layers can choose
// UUID/ULID implementations without changing the domain contracts.
type (
	GoalID           string
	ExecutionID      string
	PlanID           string
	NodeID           string
	TargetID         string
	AttemptID        string
	PolicyDecisionID string
	EvidenceID       string
	ArtifactID       string
	OutcomeID        string
)

type Goal struct {
	ID                 GoalID
	Objective          string
	AcceptanceCriteria []AcceptanceCriterion
	Constraints        GoalConstraints
	Budget             BudgetLimit
}

type AcceptanceCriterion struct {
	ID       string
	Text     string
	Required bool
}

type GoalConstraints struct {
	ProductionWriteAllowed bool
	AllowedDataScopes      []string
}

type Execution struct {
	ID            ExecutionID
	GoalRef       GoalID
	PlanRef       PlanRevisionRef
	Identity      Identity
	PolicyContext PolicyContext
	Budget        BudgetState
	Status        ExecutionStatus
	Accounting    Accounting
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Identity struct {
	Principal string
	Tenant    string
}

type PolicyContext struct {
	Environment     string
	DataSensitivity string
}

type ExecutionPhase string

const (
	ExecutionCreated    ExecutionPhase = "CREATED"
	ExecutionPlanning   ExecutionPhase = "PLANNING"
	ExecutionReady      ExecutionPhase = "READY"
	ExecutionRunning    ExecutionPhase = "RUNNING"
	ExecutionVerifying  ExecutionPhase = "VERIFYING"
	ExecutionReplanning ExecutionPhase = "REPLANNING"
	ExecutionSucceeded  ExecutionPhase = "SUCCEEDED"
	ExecutionFailed     ExecutionPhase = "FAILED"
	ExecutionCancelled  ExecutionPhase = "CANCELLED"
)

type ConditionType string

const (
	ConditionApprovalRequired      ConditionType = "ApprovalRequired"
	ConditionPartiallyBlocked      ConditionType = "PartiallyBlocked"
	ConditionRetrying              ConditionType = "Retrying"
	ConditionBudgetPressure        ConditionType = "BudgetPressure"
	ConditionTargetDegraded        ConditionType = "TargetDegraded"
	ConditionWaitingExternalInput  ConditionType = "WaitingExternalInput"
	ConditionCancellationRequested ConditionType = "CancellationRequested"
)

type ExecutionCondition struct {
	Type      ConditionType
	Status    bool
	NodeRef   NodeID
	Reason    string
	UpdatedAt time.Time
}

type ExecutionStatus struct {
	Phase      ExecutionPhase
	Conditions []ExecutionCondition
}

type PlanRevisionRef struct {
	PlanID   PlanID
	Revision int64
}

type ExecutionPlan struct {
	ID             PlanID
	GoalRef        GoalID
	Revision       int64
	ParentRevision int64
	Nodes          []ExecutionNode
	CreatedAt      time.Time
}

type NodeType string

const (
	NodeModel     NodeType = "model"
	NodeTool      NodeType = "tool"
	NodeCode      NodeType = "code"
	NodeAgent     NodeType = "agent"
	NodeBrowser   NodeType = "browser"
	NodeRetrieval NodeType = "retrieval"
	NodeData      NodeType = "data"
	NodeWorkflow  NodeType = "workflow"
	NodeHuman     NodeType = "human"
	NodeVerifier  NodeType = "verifier"
)

type NodeState string

const (
	NodePending         NodeState = "PENDING"
	NodeReady           NodeState = "READY"
	NodeRunning         NodeState = "RUNNING"
	NodeWaitingApproval NodeState = "WAITING_APPROVAL"
	NodeSucceeded       NodeState = "SUCCEEDED"
	NodeFailed          NodeState = "FAILED"
	NodeSkipped         NodeState = "SKIPPED"
	NodeBlocked         NodeState = "BLOCKED"
	NodeCancelled       NodeState = "CANCELLED"
)

type EffectClass string

const (
	EffectPure                  EffectClass = "PURE"
	EffectReadOnly              EffectClass = "READ_ONLY"
	EffectIdempotentMutation    EffectClass = "IDEMPOTENT_MUTATION"
	EffectNonIdempotentMutation EffectClass = "NON_IDEMPOTENT_MUTATION"
)

type ExecutionNode struct {
	ID                    NodeID
	Type                  NodeType
	DependsOn             []NodeID
	CapabilityRequirement CapabilityRequirement
	Timeout                time.Duration
	RetryPolicy            RetryPolicy
	Effect                 EffectSpec
	VerificationRequired   bool
}

type CapabilityRequirement struct {
	Modality         []string
	Reasoning        string
	Coding           string
	Context          string
	ToolCalling      string
	PrivateData      bool
	LatencyClass     string
	ReliabilityClass string
}

type RetryPolicy struct {
	MaxAttempts int
	RetryOn     []ErrorClass
}

type EffectSpec struct {
	Class          EffectClass
	IdempotencyKey string
	RetrySafe      bool
}

type TargetType string

const (
	TargetModel     TargetType = "model"
	TargetTool      TargetType = "tool"
	TargetCode      TargetType = "code"
	TargetAgent     TargetType = "agent"
	TargetBrowser   TargetType = "browser"
	TargetRetrieval TargetType = "retrieval"
	TargetData      TargetType = "data"
	TargetWorkflow  TargetType = "workflow"
	TargetHuman     TargetType = "human"
)

type ExecutionTarget struct {
	ID           TargetID
	Revision     int64
	Type         TargetType
	Capabilities map[string]string
	Policy       TargetPolicy
	Health       TargetHealth
	Snapshot     TargetSnapshot
}

type TargetPolicy struct {
	DataLocality        string
	AllowedSensitivity []string
}

type TargetHealth struct {
	State      string
	ObservedAt time.Time
}

type TargetSnapshot struct {
	Digest          string
	EndpointClass   string
	ModelVersion    string
	RuntimeVersion  string
	DeploymentRef   string
	CapabilityHash  string
}

type ErrorClass string

const (
	ErrorTransient          ErrorClass = "TRANSIENT"
	ErrorTargetUnavailable  ErrorClass = "TARGET_UNAVAILABLE"
	ErrorTimeout            ErrorClass = "TIMEOUT"
	ErrorPolicyDenied       ErrorClass = "POLICY_DENIED"
	ErrorBudgetExceeded     ErrorClass = "BUDGET_EXCEEDED"
	ErrorInvalidOutput      ErrorClass = "INVALID_OUTPUT"
	ErrorVerificationFailed ErrorClass = "VERIFICATION_FAILED"
	ErrorPermanent          ErrorClass = "PERMANENT"
	ErrorUnknown            ErrorClass = "UNKNOWN"
)

type ExecutionAttempt struct {
	ID               AttemptID
	ExecutionRef     ExecutionID
	NodeRef          NodeID
	TargetRef        TargetID
	TargetRevision   int64
	TargetSnapshot   string
	AttemptNumber    int
	Status           string
	ErrorClass       ErrorClass
	Usage            Usage
	IdempotencyKey   string
	EffectStartedAt  *time.Time
	EffectCommittedAt *time.Time
	StartedAt        time.Time
	FinishedAt       *time.Time
}

type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64
	Duration     time.Duration
}

type Accounting struct {
	ConsumedCost float64
	ReservedCost float64
	Elapsed      time.Duration
}

type BudgetLimit struct {
	MaxCost         float64
	MaxDuration     time.Duration
	MaxNodeAttempts int
	MaxFrontierCalls int
	MaxToolCalls    int
}

type BudgetState struct {
	Limit    BudgetLimit
	Reserved AccountingCounters
	Consumed AccountingCounters
}

type AccountingCounters struct {
	Cost          float64
	NodeAttempts  int
	FrontierCalls int
	ToolCalls     int
}

type PolicyDecision string

const (
	PolicyAllow           PolicyDecision = "ALLOW"
	PolicyDeny            PolicyDecision = "DENY"
	PolicyRequireApproval PolicyDecision = "REQUIRE_APPROVAL"
)

type PolicyDecisionRecord struct {
	ID            PolicyDecisionID
	PolicySetRef  string
	PolicyVersion string
	ExecutionRef  ExecutionID
	PlanRevision  int64
	NodeRef       NodeID
	Principal     string
	Action        string
	Resource      string
	RequestedScope string
	Decision      PolicyDecision
	ReasonCodes   []string
	ExpiresAt     *time.Time
}

type Artifact struct {
	ID           ArtifactID
	ExecutionRef ExecutionID
	NodeRef      NodeID
	Type         string
	MediaType    string
	Location     string
	Digest       string
	Immutable    bool
}

type Evidence struct {
	ID               EvidenceID
	ExecutionRef     ExecutionID
	SourceNodeRef    NodeID
	Type             string
	ClaimID          string
	SourceRef        string
	SourceVersion    string
	ObservedAt       time.Time
	ContentDigest    string
	ArtifactRef      ArtifactID
	TargetSnapshot   string
	Strength         string
	Confidence       float64
}

type VerifierVerdict string

const (
	VerifierPass         VerifierVerdict = "PASS"
	VerifierFail         VerifierVerdict = "FAIL"
	VerifierInconclusive VerifierVerdict = "INCONCLUSIVE"
)

type VerifierResult struct {
	ID                string
	ExecutionRef      ExecutionID
	NodeRef           NodeID
	VerifierRef       string
	VerifierVersion   string
	InputEvidenceRefs []EvidenceID
	Verdict           VerifierVerdict
}

type OutcomeStatus string

const (
	OutcomeVerifiedSuccess OutcomeStatus = "VERIFIED_SUCCESS"
	OutcomePartialSuccess  OutcomeStatus = "PARTIAL_SUCCESS"
	OutcomeUnverified      OutcomeStatus = "UNVERIFIED"
	OutcomeFailed          OutcomeStatus = "FAILED"
)

type AcceptanceResult struct {
	CriterionID string
	Result      VerifierVerdict
	EvidenceRefs []EvidenceID
}

type Outcome struct {
	ID                OutcomeID
	ExecutionRef      ExecutionID
	Status            OutcomeStatus
	AcceptanceResults []AcceptanceResult
	Confidence        float64
}
