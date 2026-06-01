package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestProviderHealthWithoutClient(t *testing.T) {
	p := NewProvider(Config{Name: "public-test", DefaultModel: "test-model"}, nil)

	health, err := p.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if health == nil {
		t.Fatalf("expected health response")
	}
	if health.Available {
		t.Fatalf("expected provider without client to be unavailable")
	}
}

func TestProviderCapabilities(t *testing.T) {
	p := NewProvider(Config{Name: "private-test", DefaultModel: "test-model", Private: true, MaxInputTokens: 8192, MaxOutputTokens: 2048}, nil)

	caps := p.Capabilities()
	if !caps.SupportsStructuredOutput {
		t.Fatalf("expected structured output support")
	}
	if !caps.SupportsJSONSchema {
		t.Fatalf("expected JSON schema support")
	}
	if !caps.SupportsLocalDeployment {
		t.Fatalf("expected local deployment support for private provider")
	}
	if p.Type() != provider.ProviderTypePrivate {
		t.Fatalf("expected private provider type, got %s", p.Type())
	}
}

func TestProviderGenerateWithoutClient(t *testing.T) {
	p := NewProvider(Config{Name: "public-test", DefaultModel: "test-model"}, nil)

	_, err := p.Generate(context.Background(), provider.ProviderRequest{
		RequestID:   "openai-no-client-001",
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		OutputSchema: provider.OutputSchemaRef{
			Name:    schema.KindChangePlan,
			Version: schema.SchemaVersionV1Alpha1,
		},
	})
	if err == nil {
		t.Fatalf("expected error when client is not configured")
	}
}

func TestProviderGenerateFailsClosedWhenParserMissing(t *testing.T) {
	p := NewProvider(Config{Name: "public-test", DefaultModel: "test-model", MaxOutputTokens: 1024}, &fakeClient{})

	resp, err := p.Generate(context.Background(), provider.ProviderRequest{
		RequestID:   "openai-parser-missing-001",
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		OutputSchema: provider.OutputSchemaRef{
			Name:    schema.KindChangePlan,
			Version: schema.SchemaVersionV1Alpha1,
		},
	})
	if err == nil {
		t.Fatalf("expected structured parser error")
	}
	if resp == nil {
		t.Fatalf("expected response with raw text and validation hint")
	}
	if len(resp.ValidationHints) == 0 {
		t.Fatalf("expected validation hint")
	}
}

type fakeClient struct{}

func (c *fakeClient) Generate(ctx context.Context, req CompatibleRequest) (*CompatibleResponse, error) {
	return &CompatibleResponse{OutputText: `{"kind":"ChangePlan"}`, FinishReason: "stop", InputTokens: 10, OutputTokens: 20}, nil
}

func (c *fakeClient) Health(ctx context.Context) error {
	return nil
}

type failingClient struct{}

func (c *failingClient) Generate(ctx context.Context, req CompatibleRequest) (*CompatibleResponse, error) {
	return nil, errors.New("provider unavailable")
}

func (c *failingClient) Health(ctx context.Context) error {
	return errors.New("provider unavailable")
}
