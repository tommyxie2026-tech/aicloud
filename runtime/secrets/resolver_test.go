package secrets

import (
	"context"
	"testing"
)

func TestParseSecretRef(t *testing.T) {
	ref, err := ParseSecretRef("secret/aicloud-system/openai-public:api-key")
	if err != nil {
		t.Fatalf("ParseSecretRef returned error: %v", err)
	}
	if ref.Namespace != "aicloud-system" || ref.Name != "openai-public" || ref.Key != "api-key" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

func TestParseSecretRefRejectsInvalidFormat(t *testing.T) {
	cases := []string{"", "openai-public", "secret/ns/name", "secret//name:key", "secret/ns/:key", "secret/ns/name:"}
	for _, c := range cases {
		if _, err := ParseSecretRef(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}

func TestMemoryResolverResolvesSecret(t *testing.T) {
	resolver := NewMemoryResolver(map[string]string{"secret/aicloud-system/openai-public:api-key": "test-key"})
	value, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	if err != nil {
		t.Fatalf("ResolveSecret returned error: %v", err)
	}
	if value != "test-key" {
		t.Fatalf("unexpected value: %s", value)
	}
}

func TestMemoryResolverRejectsMissingSecret(t *testing.T) {
	resolver := NewMemoryResolver(map[string]string{})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	if err == nil {
		t.Fatalf("expected missing secret error")
	}
}

func TestMemoryResolverRejectsEmptySecretValue(t *testing.T) {
	resolver := NewMemoryResolver(map[string]string{"secret/aicloud-system/openai-public:api-key": ""})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	if err == nil {
		t.Fatalf("expected empty secret value error")
	}
}

func TestMemoryResolverRejectsInvalidSecretRef(t *testing.T) {
	resolver := NewMemoryResolver(map[string]string{"bad": "value"})
	_, err := resolver.ResolveSecret(context.Background(), "bad")
	if err == nil {
		t.Fatalf("expected invalid secret ref error")
	}
}

func TestMemoryResolverHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resolver := NewMemoryResolver(map[string]string{"secret/aicloud-system/openai-public:api-key": "test-key"})
	_, err := resolver.ResolveSecret(ctx, "secret/aicloud-system/openai-public:api-key")
	if err == nil {
		t.Fatalf("expected canceled context error")
	}
}
