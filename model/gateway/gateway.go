package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// ModelGateway exposes task-level APIs to agent workflows.
type ModelGateway interface {
	GeneratePlan(ctx context.Context, req GeneratePlanRequest) (*schema.ChangePlan, *AuditRecord, error)
}

// SafetyValidator is the model safety boundary used by Gateway.
type SafetyValidator interface {
	ValidateRequest(req provider.ProviderRequest) error
	ValidateResponse(resp *provider.ProviderResponse) error
}

// Gateway is the default model gateway implementation.
type Gateway struct {
	provider  provider.ModelProvider
	validator schema.Validator
	safety    SafetyValidator
	auditor   Auditor
}

func NewGateway(p provider.ModelProvider, validator schema.Validator, safety SafetyValidator, auditor Auditor) *Gateway {
	return &Gateway{provider: p, validator: validator, safety: safety, auditor: auditor}
}

func (g *Gateway) GeneratePlan(ctx context.Context, req GeneratePlanRequest) (*schema.ChangePlan, *AuditRecord, error) {
	requestID := ensureRequestID(req.RequestID)
	startedAt := time.Now()

	providerReq := provider.ProviderRequest{
		RequestID:   requestID,
		TaskType:    provider.TaskGeneratePlan,
		UserID:      req.UserID,
		RiskHint:    req.RiskHint,
		Instruction: req.UserIntent,
		Context:     req.Context,
		OutputSchema: provider.OutputSchemaRef{
			Name:    schema.KindChangePlan,
			Version: schema.SchemaVersionV1Alpha1,
		},
		SafetyPolicy: provider.SafetyPolicyRef{Name: "default", Version: "v1alpha1"},
	}

	if g.safety != nil {
		if err := g.safety.ValidateRequest(providerReq); err != nil {
			return nil, g.auditBlocked(requestID, provider.TaskGeneratePlan, startedAt, err), err
		}
	}

	resp, err := g.provider.Generate(ctx, providerReq)
	if err != nil {
		return nil, g.auditProviderError(requestID, provider.TaskGeneratePlan, startedAt, resp, err), err
	}

	if g.safety != nil {
		if err := g.safety.ValidateResponse(resp); err != nil {
			return nil, g.auditProviderError(requestID, provider.TaskGeneratePlan, startedAt, resp, err), err
		}
	}

	plan, ok := resp.Structured.(schema.ChangePlan)
	if !ok {
		return nil, g.auditProviderError(requestID, provider.TaskGeneratePlan, startedAt, resp, ErrUnexpectedStructuredOutput), ErrUnexpectedStructuredOutput
	}

	if err := g.validator.ValidateChangePlan(&plan); err != nil {
		return nil, g.auditProviderError(requestID, provider.TaskGeneratePlan, startedAt, resp, err), err
	}

	audit := g.auditSuccess(requestID, provider.TaskGeneratePlan, startedAt, resp, schema.KindChangePlan)
	return &plan, audit, nil
}

type GeneratePlanRequest struct {
	RequestID  string
	UserID     string
	UserIntent string
	RiskHint   string
	Context    provider.ModelContext
}

type Auditor interface {
	Record(ctx context.Context, record AuditRecord) error
}

type AuditRecord struct {
	RequestID        string
	TaskType         provider.TaskType
	ProviderName     string
	ModelName        string
	OutputKind       string
	ValidationResult string
	SafetySignals    []provider.SafetySignal
	LatencyMs        int64
	TokenUsage       provider.TokenUsage
	Error            string
	CreatedAt        time.Time
}

func (g *Gateway) auditSuccess(requestID string, task provider.TaskType, startedAt time.Time, resp *provider.ProviderResponse, outputKind string) *AuditRecord {
	record := &AuditRecord{
		RequestID:        requestID,
		TaskType:         task,
		ProviderName:     resp.ProviderName,
		ModelName:        resp.ModelName,
		OutputKind:       outputKind,
		ValidationResult: "Passed",
		SafetySignals:    resp.SafetySignals,
		LatencyMs:        time.Since(startedAt).Milliseconds(),
		TokenUsage:       resp.TokenUsage,
		CreatedAt:        time.Now(),
	}
	_ = g.recordAudit(record)
	return record
}

func (g *Gateway) auditBlocked(requestID string, task provider.TaskType, startedAt time.Time, err error) *AuditRecord {
	record := &AuditRecord{
		RequestID:        requestID,
		TaskType:         task,
		ValidationResult: "Blocked",
		LatencyMs:        time.Since(startedAt).Milliseconds(),
		Error:            err.Error(),
		CreatedAt:        time.Now(),
	}
	_ = g.recordAudit(record)
	return record
}

func (g *Gateway) auditProviderError(requestID string, task provider.TaskType, startedAt time.Time, resp *provider.ProviderResponse, err error) *AuditRecord {
	record := &AuditRecord{
		RequestID:        requestID,
		TaskType:         task,
		ValidationResult: "Failed",
		LatencyMs:        time.Since(startedAt).Milliseconds(),
		Error:            err.Error(),
		CreatedAt:        time.Now(),
	}
	if resp != nil {
		record.ProviderName = resp.ProviderName
		record.ModelName = resp.ModelName
		record.SafetySignals = resp.SafetySignals
		record.TokenUsage = resp.TokenUsage
	}
	_ = g.recordAudit(record)
	return record
}

func (g *Gateway) recordAudit(record *AuditRecord) error {
	if g.auditor == nil {
		return nil
	}
	return g.auditor.Record(context.Background(), *record)
}

func ensureRequestID(id string) string {
	if id != "" {
		return id
	}
	return fmt.Sprintf("modelreq-%d", time.Now().UnixNano())
}

var ErrUnexpectedStructuredOutput = fmt.Errorf("unexpected structured output type")
