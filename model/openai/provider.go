package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

// Provider is an OpenAI-compatible provider adapter.
//
// It can represent public model APIs, private model endpoints, or self-hosted
// open-source model servers that expose an OpenAI-compatible interface.
//
// This adapter does not store raw credentials. Runtime implementations should
// resolve SecretRef outside this package.
type Provider struct {
	config Config
	client Client
}

func NewProvider(config Config, client Client) *Provider {
	return &Provider{config: config, client: client}
}

func (p *Provider) Name() string {
	return p.config.Name
}

func (p *Provider) Type() provider.ProviderType {
	if p.config.Private {
		return provider.ProviderTypePrivate
	}
	return provider.ProviderTypeHosted
}

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportsStructuredOutput: true,
		SupportsJSONSchema:      true,
		SupportsStreaming:       false,
		SupportsToolUse:         false,
		SupportsVision:          false,
		SupportsLongContext:     true,
		SupportsChinese:         true,
		SupportsCodeGeneration:  true,
		SupportsLocalDeployment: p.config.Private,
		MaxInputTokens:          p.config.MaxInputTokens,
		MaxOutputTokens:         p.config.MaxOutputTokens,
		RecommendedTasks: []provider.TaskType{
			provider.TaskGeneratePlan,
			provider.TaskGeneratePatch,
			provider.TaskExplainRisk,
			provider.TaskGenerateRollback,
			provider.TaskGenerateValidationReport,
			provider.TaskSummarizeState,
			provider.TaskRepairYAML,
		},
		RestrictedCapabilities: []string{
			provider.RestrictedDirectExecution,
			provider.RestrictedManifestApply,
			provider.RestrictedCredentialRead,
			provider.RestrictedMachineControl,
			provider.RestrictedProductionDelete,
			provider.RestrictedAutoApprove,
			provider.RestrictedAutoMerge,
		},
	}
}

func (p *Provider) Health(ctx context.Context) (*provider.ProviderHealth, error) {
	startedAt := time.Now()
	if p.client == nil {
		return &provider.ProviderHealth{Name: p.config.Name, Available: false, LatencyMs: time.Since(startedAt).Milliseconds(), Message: "client is not configured"}, nil
	}
	if err := p.client.Health(ctx); err != nil {
		return &provider.ProviderHealth{Name: p.config.Name, Available: false, LatencyMs: time.Since(startedAt).Milliseconds(), Message: err.Error()}, nil
	}
	return &provider.ProviderHealth{Name: p.config.Name, Available: true, LatencyMs: time.Since(startedAt).Milliseconds(), ModelNames: []string{p.config.DefaultModel}, Message: "provider is available"}, nil
}

func (p *Provider) Generate(ctx context.Context, req provider.ProviderRequest) (*provider.ProviderResponse, error) {
	startedAt := time.Now()
	if p.client == nil {
		return nil, &provider.ProviderError{Code: provider.ErrProviderUnavailable, Message: "client is not configured", Retryable: false}
	}
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	apiReq := p.mapProviderRequest(req)
	apiResp, err := p.client.Generate(ctx, apiReq)
	if err != nil {
		return nil, normalizeClientError(err)
	}

	structured, err := p.parseStructuredOutput(req.OutputSchema, apiResp.OutputText)
	if err != nil {
		return &provider.ProviderResponse{
			RequestID:        req.RequestID,
			ProviderName:     p.config.Name,
			ModelName:        apiReq.Model,
			TaskType:         req.TaskType,
			RawText:          apiResp.OutputText,
			FinishReason:     apiResp.FinishReason,
			LatencyMs:        time.Since(startedAt).Milliseconds(),
			TokenUsage:       tokenUsage(apiResp),
			ValidationHints:  []string{"structured output parse failed"},
		}, &provider.ProviderError{Code: provider.ErrInvalidOutput, Message: err.Error(), Retryable: true}
	}

	return &provider.ProviderResponse{
		RequestID:    req.RequestID,
		ProviderName: p.config.Name,
		ModelName:    apiReq.Model,
		TaskType:     req.TaskType,
		RawText:      apiResp.OutputText,
		Structured:   structured,
		FinishReason: apiResp.FinishReason,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		TokenUsage:   tokenUsage(apiResp),
	}, nil
}

func (p *Provider) mapProviderRequest(req provider.ProviderRequest) CompatibleRequest {
	return CompatibleRequest{
		Model:           p.config.DefaultModel,
		SystemPrompt:    buildSystemPrompt(req),
		Instruction:     req.Instruction,
		ContextText:     renderContext(req.Context),
		OutputSchema:    req.OutputSchema.Name,
		MaxOutputTokens: chooseMaxOutputTokens(req.MaxOutputTokens, p.config.MaxOutputTokens),
		Temperature:     chooseTemperature(req.Temperature),
		TimeoutSeconds:  p.config.TimeoutSeconds,
	}
}

// Config uses SecretRef rather than raw API keys.
type Config struct {
	Name             string
	Endpoint         string
	EndpointRef      string
	SecretRef        string
	DefaultModel     string
	TimeoutSeconds   int
	MaxRetries       int
	MaxInputTokens   int
	MaxOutputTokens  int
	Private          bool
}

// Client is a narrow interface for an OpenAI-compatible endpoint.
type Client interface {
	Generate(ctx context.Context, req CompatibleRequest) (*CompatibleResponse, error)
	Health(ctx context.Context) error
}

type CompatibleRequest struct {
	Model           string
	SystemPrompt    string
	Instruction     string
	ContextText     string
	OutputSchema    string
	MaxOutputTokens int
	Temperature     float32
	TimeoutSeconds  int
}

type CompatibleResponse struct {
	OutputText   string
	FinishReason string
	InputTokens  int
	OutputTokens int
}

// StructuredParser allows callers to plug in JSON/YAML schema parsing later.
type StructuredParser interface {
	Parse(schemaRef provider.OutputSchemaRef, raw string) (any, error)
}

func validateRequest(req provider.ProviderRequest) error {
	if req.RequestID == "" {
		return &provider.ProviderError{Code: provider.ErrInvalidOutput, Message: "requestId is required", Retryable: false}
	}
	if req.TaskType == "" {
		return &provider.ProviderError{Code: provider.ErrInvalidOutput, Message: "taskType is required", Retryable: false}
	}
	if req.OutputSchema.Name == "" {
		return &provider.ProviderError{Code: provider.ErrSchemaMismatch, Message: "output schema is required", Retryable: false}
	}
	return nil
}

func buildSystemPrompt(req provider.ProviderRequest) string {
	return "You are an infrastructure planning model. Return only structured output for schema: " + req.OutputSchema.Name + ". Do not suggest direct execution."
}

func renderContext(ctx provider.ModelContext) string {
	if ctx.UserIntent != "" {
		return "userIntent: " + ctx.UserIntent
	}
	return ""
}

func (p *Provider) parseStructuredOutput(schemaRef provider.OutputSchemaRef, raw string) (any, error) {
	// MVP placeholder: real implementation will decode strict JSON/YAML into model/schema types.
	// Until that parser exists, this provider is safe to register but should not be used in CI as a trusted planner.
	return nil, fmt.Errorf("structured parser not implemented for schema %s", schemaRef.Name)
}

func normalizeClientError(err error) error {
	if err == nil {
		return nil
	}
	return &provider.ProviderError{Code: provider.ErrProviderUnavailable, Message: err.Error(), Retryable: true}
}

func chooseMaxOutputTokens(requested int, providerMax int) int {
	if providerMax <= 0 {
		providerMax = 2048
	}
	if requested > 0 && requested < providerMax {
		return requested
	}
	return providerMax
}

func chooseTemperature(requested float32) float32 {
	if requested >= 0 {
		return requested
	}
	return 0
}

func tokenUsage(resp *CompatibleResponse) provider.TokenUsage {
	return provider.TokenUsage{InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens, TotalTokens: resp.InputTokens + resp.OutputTokens}
}
