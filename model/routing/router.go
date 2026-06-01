package routing

import (
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

// Router selects a safe provider for a model task.
type Router interface {
	Route(req RouteRequest) (*RouteDecision, error)
}

// StaticRouter is a policy-driven router for the MVP.
type StaticRouter struct {
	providers map[string]provider.ModelProvider
	scores    map[string]ProviderScore
	policy    RoutingPolicy
}

func NewStaticRouter(providers []provider.ModelProvider, scores map[string]ProviderScore, policy RoutingPolicy) *StaticRouter {
	providerMap := map[string]provider.ModelProvider{}
	for _, p := range providers {
		providerMap[p.Name()] = p
	}
	return &StaticRouter{providers: providerMap, scores: scores, policy: policy}
}

func (r *StaticRouter) Route(req RouteRequest) (*RouteDecision, error) {
	if isRestrictedTask(req.TaskType) {
		return blockedDecision(req, "restricted task is not routed to model providers"), NewRoutingError("RestrictedTask", "restricted task is not routed to model providers")
	}
	if req.DataSensitivity == DataSensitivityRestricted {
		return blockedDecision(req, "restricted data is not routed to model providers"), NewRoutingError("RestrictedData", "restricted data is not routed to model providers")
	}
	if req.TaskType == TaskRiskClassification || req.TaskType == TaskApprovalDecision {
		return &RouteDecision{
			RoutingDecisionID:  newRouteID(),
			RequestID:          req.RequestID,
			TaskType:           req.TaskType,
			SelectedProvider:   "deterministic-policy",
			SelectedModel:      "deterministic-policy",
			ValidationRequired: []string{"policy"},
			Reason:            []string{"risk and approval decisions use deterministic policy"},
			CreatedAt:         time.Now(),
		}, nil
	}

	rules := r.matchRules(req)
	if len(rules) == 0 {
		return blockedDecision(req, "no routing rule matched"), NewRoutingError("NoRouteMatched", "no routing rule matched")
	}

	for _, rule := range rules {
		providerNames := append([]string{rule.PreferredProvider}, rule.FallbackProviders...)
		for _, name := range providerNames {
			p, ok := r.providers[name]
			if !ok {
				continue
			}
			if !r.providerAllowedForRequest(p, req, rule) {
				continue
			}
			return &RouteDecision{
				RoutingDecisionID:  newRouteID(),
				RequestID:          req.RequestID,
				TaskType:           req.TaskType,
				RiskHint:           req.RiskHint,
				Environment:        req.Environment,
				DataSensitivity:    req.DataSensitivity,
				SelectedProvider:   p.Name(),
				SelectedModel:      "default",
				FallbackProviders:  rule.FallbackProviders,
				ValidationRequired: []string{"schema", "safety", "policy"},
				Reason: []string{
					"provider matched routing rule",
					"provider passed evaluation threshold",
					"provider supports requested task",
				},
				CreatedAt: time.Now(),
			}, nil
		}
	}

	return blockedDecision(req, "no safe provider available"), NewRoutingError("NoSafeProvider", "no safe provider available")
}

func (r *StaticRouter) matchRules(req RouteRequest) []RouteRule {
	var matched []RouteRule
	for _, rule := range r.policy.Routes {
		if rule.TaskType != req.TaskType {
			continue
		}
		if !riskInRange(req.RiskHint, rule.MinRisk, rule.MaxRisk) {
			continue
		}
		if rule.Environment != "" && rule.Environment != req.Environment {
			continue
		}
		matched = append(matched, rule)
	}
	return matched
}

func (r *StaticRouter) providerAllowedForRequest(p provider.ModelProvider, req RouteRequest, rule RouteRule) bool {
	caps := p.Capabilities()
	if !providerSupportsTask(caps, req.TaskType) {
		return false
	}
	if req.RequiresPrivateProvider && p.Type() == provider.ProviderTypeHosted {
		return false
	}
	if req.DataSensitivity == DataSensitivityConfidential && p.Type() == provider.ProviderTypeHosted {
		return false
	}
	if req.Environment == EnvironmentProduction && req.RiskHint == RiskHigh && p.Type() == provider.ProviderTypeLocal {
		return false
	}
	if score, ok := r.scores[p.Name()]; ok {
		if score.SafetyFailures > 0 {
			return false
		}
		if score.AverageScore < rule.MinEvaluationScore {
			return false
		}
	} else if r.policy.RequireEvaluation {
		return false
	}
	return true
}

func providerSupportsTask(caps provider.ProviderCapabilities, task RouteTaskType) bool {
	providerTask, ok := toProviderTaskType(task)
	if !ok {
		return false
	}
	for _, t := range caps.RecommendedTasks {
		if t == providerTask {
			return true
		}
	}
	return false
}

func toProviderTaskType(task RouteTaskType) (provider.TaskType, bool) {
	switch task {
	case TaskGeneratePlan:
		return provider.TaskGeneratePlan, true
	case TaskGeneratePatch:
		return provider.TaskGeneratePatch, true
	case TaskExplainRisk:
		return provider.TaskExplainRisk, true
	case TaskGenerateRollback:
		return provider.TaskGenerateRollback, true
	case TaskGenerateValidationReport:
		return provider.TaskGenerateValidationReport, true
	case TaskSummarizeState:
		return provider.TaskSummarizeState, true
	case TaskRepairYAML:
		return provider.TaskRepairYAML, true
	case TaskExplainPolicyFailure:
		return provider.TaskExplainPolicyFailure, true
	default:
		return "", false
	}
}

type RouteRequest struct {
	RequestID              string
	TaskType               RouteTaskType
	RiskHint               RiskLevel
	Environment            Environment
	DataSensitivity         DataSensitivity
	RequiredSchema          string
	LatencyBudgetMs         int
	CostBudget              CostBudget
	Language                string
	ContextSize             string
	RequiresPrivateProvider bool
}

type RouteDecision struct {
	RoutingDecisionID  string
	RequestID          string
	TaskType           RouteTaskType
	RiskHint           RiskLevel
	Environment        Environment
	DataSensitivity    DataSensitivity
	SelectedProvider   string
	SelectedModel      string
	FallbackProviders  []string
	ValidationRequired []string
	Reason             []string
	Blocked            bool
	BlockedReason      string
	CreatedAt          time.Time
}

type RoutingPolicy struct {
	FailClosed           bool
	RequireEvaluation    bool
	BlockOnSafetyFailure bool
	Routes               []RouteRule
}

type RouteRule struct {
	TaskType           RouteTaskType
	MinRisk            RiskLevel
	MaxRisk            RiskLevel
	Environment        Environment
	PreferredProvider  string
	FallbackProviders  []string
	MinEvaluationScore int
}

type ProviderScore struct {
	ProviderName     string
	AverageScore     int
	SafetyFailures   int
	SchemaFailures   int
	RecommendedTasks []RouteTaskType
}

type RouteTaskType string

const (
	TaskGeneratePlan             RouteTaskType = "GeneratePlan"
	TaskGeneratePatch            RouteTaskType = "GeneratePatch"
	TaskExplainRisk              RouteTaskType = "ExplainRisk"
	TaskGenerateRollback         RouteTaskType = "GenerateRollback"
	TaskGenerateValidationReport RouteTaskType = "GenerateValidationReport"
	TaskSummarizeState           RouteTaskType = "SummarizeState"
	TaskRepairYAML               RouteTaskType = "RepairYAML"
	TaskExplainPolicyFailure     RouteTaskType = "ExplainPolicyFailure"
	TaskRiskClassification       RouteTaskType = "RiskClassification"
	TaskApprovalDecision         RouteTaskType = "ApprovalDecision"
	TaskRestrictedOperation      RouteTaskType = "RestrictedOperation"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "Low"
	RiskMedium   RiskLevel = "Medium"
	RiskHigh     RiskLevel = "High"
	RiskCritical RiskLevel = "Critical"
)

type Environment string

const (
	EnvironmentDev        Environment = "dev"
	EnvironmentStaging    Environment = "staging"
	EnvironmentProduction Environment = "production"
)

type DataSensitivity string

const (
	DataSensitivityPublic       DataSensitivity = "Public"
	DataSensitivityInternal     DataSensitivity = "Internal"
	DataSensitivityConfidential DataSensitivity = "Confidential"
	DataSensitivityRestricted   DataSensitivity = "Restricted"
)

type CostBudget string

const (
	CostBudgetLow    CostBudget = "low"
	CostBudgetNormal CostBudget = "normal"
	CostBudgetHigh   CostBudget = "high"
)

func DefaultRoutingPolicy() RoutingPolicy {
	return RoutingPolicy{
		FailClosed:           true,
		RequireEvaluation:    true,
		BlockOnSafetyFailure: true,
		Routes: []RouteRule{
			{TaskType: TaskGeneratePlan, MinRisk: RiskLow, MaxRisk: RiskLow, PreferredProvider: "local-small", FallbackProviders: []string{"mock", "hosted-strong"}, MinEvaluationScore: 80},
			{TaskType: TaskGeneratePlan, MinRisk: RiskMedium, MaxRisk: RiskHigh, PreferredProvider: "hosted-strong", FallbackProviders: []string{"private-strong", "mock"}, MinEvaluationScore: 90},
			{TaskType: TaskGenerateRollback, MinRisk: RiskMedium, MaxRisk: RiskHigh, PreferredProvider: "hosted-strong", FallbackProviders: []string{"private-strong", "mock"}, MinEvaluationScore: 90},
			{TaskType: TaskGenerateValidationReport, MinRisk: RiskLow, MaxRisk: RiskMedium, PreferredProvider: "local-small", FallbackProviders: []string{"mock"}, MinEvaluationScore: 75},
			{TaskType: TaskExplainRisk, MinRisk: RiskLow, MaxRisk: RiskHigh, PreferredProvider: "local-small", FallbackProviders: []string{"mock", "hosted-strong"}, MinEvaluationScore: 75},
		},
	}
}

func isRestrictedTask(task RouteTaskType) bool {
	return task == TaskRestrictedOperation
}

func riskInRange(risk RiskLevel, min RiskLevel, max RiskLevel) bool {
	return riskRank(risk) >= riskRank(min) && riskRank(risk) <= riskRank(max)
}

func riskRank(risk RiskLevel) int {
	switch risk {
	case RiskLow:
		return 1
	case RiskMedium:
		return 2
	case RiskHigh:
		return 3
	case RiskCritical:
		return 4
	default:
		return 0
	}
}

func blockedDecision(req RouteRequest, reason string) *RouteDecision {
	return &RouteDecision{RoutingDecisionID: newRouteID(), RequestID: req.RequestID, TaskType: req.TaskType, RiskHint: req.RiskHint, Environment: req.Environment, DataSensitivity: req.DataSensitivity, Blocked: true, BlockedReason: reason, Reason: []string{reason}, CreatedAt: time.Now()}
}

func newRouteID() string {
	return fmt.Sprintf("route-%d", time.Now().UnixNano())
}

type RoutingError struct {
	Code    string
	Message string
}

func NewRoutingError(code string, message string) *RoutingError {
	return &RoutingError{Code: code, Message: message}
}

func (e *RoutingError) Error() string {
	return e.Code + ": " + e.Message
}
