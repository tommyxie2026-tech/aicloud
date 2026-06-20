package secrets

import (
	"context"
	"strings"
)

type Resolver interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)
}

type SecretRef struct {
	Namespace string
	Name      string
	Key       string
}

func ParseSecretRef(ref string) (SecretRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return SecretRef{}, NewSecretError("EmptySecretRef", "secret ref is required")
	}
	parts := strings.Split(ref, "/")
	if len(parts) != 3 || parts[0] != "secret" {
		return SecretRef{}, NewSecretError("InvalidSecretRef", "secret ref must use secret/<namespace>/<name>:<key>")
	}
	nameAndKey := strings.Split(parts[2], ":")
	if len(nameAndKey) != 2 {
		return SecretRef{}, NewSecretError("InvalidSecretRef", "secret ref must include key as secret/<namespace>/<name>:<key>")
	}
	parsed := SecretRef{Namespace: strings.TrimSpace(parts[1]), Name: strings.TrimSpace(nameAndKey[0]), Key: strings.TrimSpace(nameAndKey[1])}
	if parsed.Namespace == "" || parsed.Name == "" || parsed.Key == "" {
		return SecretRef{}, NewSecretError("InvalidSecretRef", "namespace, name and key are required")
	}
	return parsed, nil
}

type MemoryResolver struct {
	values map[string]string
}

func NewMemoryResolver(values map[string]string) *MemoryResolver {
	copyValues := map[string]string{}
	for k, v := range values {
		copyValues[strings.TrimSpace(k)] = v
	}
	return &MemoryResolver{values: copyValues}
}

func (r *MemoryResolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return "", NewSecretError("ContextCanceled", ctx.Err().Error())
	default:
	}
	if _, err := ParseSecretRef(ref); err != nil {
		return "", err
	}
	value, ok := r.values[strings.TrimSpace(ref)]
	if !ok {
		return "", NewSecretError("SecretNotFound", "secret ref not found")
	}
	if strings.TrimSpace(value) == "" {
		return "", NewSecretError("EmptySecretValue", "secret value is empty")
	}
	return value, nil
}

type SecretError struct {
	Code    string
	Message string
}

func NewSecretError(code string, message string) *SecretError {
	return &SecretError{Code: code, Message: message}
}

func (e *SecretError) Error() string {
	return e.Code + ": " + e.Message
}
