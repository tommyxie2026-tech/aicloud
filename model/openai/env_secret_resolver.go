package openai

import (
	"context"
	"fmt"
	"os"
)

// EnvSecretResolver resolves an environment-variable name stored in SecretRef.
// Production deployments can replace it with Vault, Kubernetes Secrets, or a
// cloud secret manager without changing the provider adapter.
type EnvSecretResolver struct{}

func (EnvSecretResolver) ResolveSecret(_ context.Context, secretRef string) (string, error) {
	if secretRef == "" {
		return "", fmt.Errorf("secret reference is required")
	}
	value := os.Getenv(secretRef)
	if value == "" {
		return "", fmt.Errorf("environment secret %s is not set", secretRef)
	}
	return value, nil
}
