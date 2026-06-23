package kubernetes

import (
	"context"
	"strings"
	"testing"
)

func TestResolverResolvesSecret(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{
		"aicloud-system/openai-public": {Namespace: "aicloud-system", Name: "openai-public", Data: map[string]string{"api-key": "test-value"}},
	})
	value, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	if err != nil {
		t.Fatalf("ResolveSecret returned error: %v", err)
	}
	if value != "test-value" {
		t.Fatalf("unexpected value: %s", value)
	}
}

func TestResolverRejectsNamespaceOutsideAllowlist(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{})
	_, err := resolver.ResolveSecret(context.Background(), "secret/other/openai-public:api-key")
	assertResolverError(t, err, "NamespaceNotAllowed")
}

func TestResolverRejectsMissingSecret(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/missing:api-key")
	assertResolverError(t, err, "SecretNotFound")
}

func TestResolverRejectsMissingKey(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{
		"aicloud-system/openai-public": {Namespace: "aicloud-system", Name: "openai-public", Data: map[string]string{"other": "value"}},
	})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	assertResolverError(t, err, "SecretKeyNotFound")
}

func TestResolverRejectsEmptySecretValue(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{
		"aicloud-system/openai-public": {Namespace: "aicloud-system", Name: "openai-public", Data: map[string]string{"api-key": "   "}},
	})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:api-key")
	assertResolverError(t, err, "EmptySecretValue")
}

func TestResolverRejectsInvalidSecretRef(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{})
	_, err := resolver.ResolveSecret(context.Background(), "invalid")
	assertResolverError(t, err, "InvalidSecretRef")
}

func TestResolverHonorsCanceledContext(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := resolver.ResolveSecret(ctx, "secret/aicloud-system/openai-public:api-key")
	assertResolverError(t, err, "ContextCanceled")
}

func TestResolverErrorDoesNotExposeSecretValue(t *testing.T) {
	resolver := newTestResolver(t, map[string]SecretData{
		"aicloud-system/openai-public": {Namespace: "aicloud-system", Name: "openai-public", Data: map[string]string{"api-key": "super-sensitive-value"}},
	})
	_, err := resolver.ResolveSecret(context.Background(), "secret/aicloud-system/openai-public:missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "super-sensitive-value") {
		t.Fatalf("error leaked secret value: %v", err)
	}
}

func TestNewResolverRequiresGetter(t *testing.T) {
	_, err := NewResolver(nil, ResolverConfig{AllowedNamespaces: []string{"aicloud-system"}})
	assertResolverError(t, err, "MissingSecretGetter")
}

func TestNewResolverRequiresAllowedNamespace(t *testing.T) {
	_, err := NewResolver(memorySecretGetter{}, ResolverConfig{})
	assertResolverError(t, err, "MissingAllowedNamespace")
}

func newTestResolver(t *testing.T, values map[string]SecretData) *Resolver {
	t.Helper()
	resolver, err := NewResolver(memorySecretGetter{values: values}, ResolverConfig{AllowedNamespaces: []string{"aicloud-system"}})
	if err != nil {
		t.Fatalf("NewResolver returned error: %v", err)
	}
	return resolver
}

func assertResolverError(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s", code)
	}
	resolverErr, ok := err.(*ResolverError)
	if !ok {
		t.Fatalf("expected ResolverError, got %T", err)
	}
	if resolverErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, resolverErr.Code)
	}
}

type memorySecretGetter struct {
	values map[string]SecretData
}

func (g memorySecretGetter) GetSecret(ctx context.Context, namespace string, name string) (SecretData, error) {
	secret, ok := g.values[namespace+"/"+name]
	if !ok {
		return SecretData{}, NewResolverError("SecretNotFound", "not found")
	}
	return secret, nil
}
