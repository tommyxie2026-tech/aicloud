package kubernetes

import (
	"context"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/runtime/secrets"
)

type SecretGetter interface {
	GetSecret(ctx context.Context, namespace string, name string) (SecretData, error)
}

type Resolver struct {
	getter            SecretGetter
	allowedNamespaces map[string]struct{}
}

func NewResolver(getter SecretGetter, config ResolverConfig) (*Resolver, error) {
	if getter == nil {
		return nil, NewResolverError("MissingSecretGetter", "secret getter is required")
	}
	allowed := map[string]struct{}{}
	for _, ns := range config.AllowedNamespaces {
		ns = strings.TrimSpace(ns)
		if ns != "" {
			allowed[ns] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil, NewResolverError("MissingAllowedNamespace", "at least one allowed namespace is required")
	}
	return &Resolver{getter: getter, allowedNamespaces: allowed}, nil
}

func (r *Resolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return "", NewResolverError("ContextCanceled", ctx.Err().Error())
	default:
	}
	parsed, err := secrets.ParseSecretRef(ref)
	if err != nil {
		return "", NewResolverError("InvalidSecretRef", err.Error())
	}
	if !r.namespaceAllowed(parsed.Namespace) {
		return "", NewResolverError("NamespaceNotAllowed", "namespace is not allowed")
	}
	secret, err := r.getter.GetSecret(ctx, parsed.Namespace, parsed.Name)
	if err != nil {
		return "", NewResolverError("SecretNotFound", "secret not found")
	}
	value, ok := secret.Data[parsed.Key]
	if !ok {
		return "", NewResolverError("SecretKeyNotFound", "secret key not found")
	}
	if strings.TrimSpace(value) == "" {
		return "", NewResolverError("EmptySecretValue", "secret value is empty")
	}
	return value, nil
}

func (r *Resolver) namespaceAllowed(namespace string) bool {
	_, ok := r.allowedNamespaces[namespace]
	return ok
}
