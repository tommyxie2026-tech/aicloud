package openai

import "testing"

func TestLoadConfigAppliesDefaults(t *testing.T) {
	config, err := LoadConfig(ConfigSource{
		Name:         "openai-public",
		Endpoint:     "https://api.openai.example/v1",
		SecretRef:    "secret/openai-public",
		DefaultModel: "gpt-test",
	})
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if config.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Fatalf("expected default timeout %d, got %d", DefaultTimeoutSeconds, config.TimeoutSeconds)
	}
	if config.MaxInputTokens != DefaultMaxInputTokens {
		t.Fatalf("expected default max input tokens %d, got %d", DefaultMaxInputTokens, config.MaxInputTokens)
	}
	if config.MaxOutputTokens != DefaultMaxOutputTokens {
		t.Fatalf("expected default max output tokens %d, got %d", DefaultMaxOutputTokens, config.MaxOutputTokens)
	}
}

func TestLoadConfigAllowsEndpointRef(t *testing.T) {
	config, err := LoadConfig(ConfigSource{
		Name:         "private-openai-compatible",
		EndpointRef:  "endpoint/private-model-gateway",
		SecretRef:    "secret/private-model-gateway",
		DefaultModel: "qwen-test",
		Private:      true,
	})
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if !config.Private {
		t.Fatalf("expected private provider config")
	}
	if config.EndpointRef == "" {
		t.Fatalf("expected endpointRef")
	}
}

func TestValidateConfigRejectsMissingEndpoint(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Name: "p", SecretRef: "secret/p", DefaultModel: "m"})
	if err == nil {
		t.Fatalf("expected missing endpoint error")
	}
}

func TestValidateConfigRejectsAmbiguousEndpoint(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Name: "p", Endpoint: "https://example", EndpointRef: "endpoint/p", SecretRef: "secret/p", DefaultModel: "m"})
	if err == nil {
		t.Fatalf("expected ambiguous endpoint error")
	}
}

func TestValidateConfigRejectsMissingSecretRef(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Name: "p", Endpoint: "https://example", DefaultModel: "m"})
	if err == nil {
		t.Fatalf("expected missing secretRef error")
	}
}

func TestValidateConfigRejectsRawCredential(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Name: "p", Endpoint: "https://example", SecretRef: "sk-test-secret", DefaultModel: "m"})
	if err == nil {
		t.Fatalf("expected raw credential rejection")
	}
}

func TestValidateConfigRejectsMissingName(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Endpoint: "https://example", SecretRef: "secret/p", DefaultModel: "m"})
	if err == nil {
		t.Fatalf("expected missing name error")
	}
}

func TestValidateConfigRejectsMissingDefaultModel(t *testing.T) {
	_, err := LoadConfig(ConfigSource{Name: "p", Endpoint: "https://example", SecretRef: "secret/p"})
	if err == nil {
		t.Fatalf("expected missing default model error")
	}
}
