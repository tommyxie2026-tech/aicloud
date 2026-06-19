package openai

import (
	"context"
	"os"
	"testing"
)

const (
	envIntegrationEnabled = "AICLOUD_OPENAI_INTEGRATION_TEST"
	envIntegrationEndpoint = "AICLOUD_OPENAI_ENDPOINT"
	envIntegrationModel    = "AICLOUD_OPENAI_MODEL"
	envIntegrationAPIKey   = "AICLOUD_OPENAI_API_KEY"
)

func TestEnvGuardedOpenAICompatibleIntegration(t *testing.T) {
	if os.Getenv(envIntegrationEnabled) != "1" {
		t.Skip("set AICLOUD_OPENAI_INTEGRATION_TEST=1 to run OpenAI-compatible integration test")
	}

	endpoint := os.Getenv(envIntegrationEndpoint)
	model := os.Getenv(envIntegrationModel)
	apiKey := os.Getenv(envIntegrationAPIKey)
	if endpoint == "" || model == "" || apiKey == "" {
		t.Skip("integration test requires AICLOUD_OPENAI_ENDPOINT, AICLOUD_OPENAI_MODEL and AICLOUD_OPENAI_API_KEY")
	}

	config, err := LoadConfig(ConfigSource{
		Name:            "env-openai-compatible",
		Endpoint:        endpoint,
		SecretRef:       "env/aicloud-openai-api-key",
		DefaultModel:    model,
		TimeoutSeconds:  30,
		MaxRetries:      1,
		MaxInputTokens:  4096,
		MaxOutputTokens: 512,
		Private:         false,
	})
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	client, err := NewHTTPClient(config, nil, envSecretResolver{value: apiKey})
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}

	resp, err := client.Generate(context.Background(), CompatibleRequest{
		Model:           model,
		SystemPrompt:    "Return only raw JSON. Do not use markdown fences.",
		Instruction:     "Return a JSON object with a single field named ok set to true.",
		OutputSchema:    "json_object",
		MaxOutputTokens: 128,
		Temperature:     0,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp == nil || resp.OutputText == "" {
		t.Fatalf("expected non-empty output")
	}
}

type envSecretResolver struct {
	value string
}

func (r envSecretResolver) ResolveSecret(ctx context.Context, secretRef string) (string, error) {
	return r.value, nil
}
