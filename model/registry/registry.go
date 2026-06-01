package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

// Registry stores model providers and exposes lookup / listing APIs.
type Registry interface {
	Register(p provider.ModelProvider) error
	Get(name string) (provider.ModelProvider, bool)
	List() []ProviderInfo
	Health(ctx context.Context) []ProviderHealthInfo
}

// MemoryRegistry is an in-memory provider registry for the MVP.
type MemoryRegistry struct {
	mu        sync.RWMutex
	providers map[string]provider.ModelProvider
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{providers: map[string]provider.ModelProvider{}}
}

func (r *MemoryRegistry) Register(p provider.ModelProvider) error {
	if p == nil {
		return NewRegistryError("NilProvider", "provider is nil")
	}
	name := p.Name()
	if name == "" {
		return NewRegistryError("MissingProviderName", "provider name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[name]; exists {
		return NewRegistryError("ProviderAlreadyExists", fmt.Sprintf("provider %s already exists", name))
	}
	r.providers[name] = p
	return nil
}

func (r *MemoryRegistry) Get(name string) (provider.ModelProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *MemoryRegistry) List() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		caps := p.Capabilities()
		infos = append(infos, ProviderInfo{
			Name:                  p.Name(),
			Type:                  p.Type(),
			SupportsStructuredOutput: caps.SupportsStructuredOutput,
			SupportsJSONSchema:    caps.SupportsJSONSchema,
			SupportsLocalDeployment: caps.SupportsLocalDeployment,
			MaxInputTokens:        caps.MaxInputTokens,
			MaxOutputTokens:       caps.MaxOutputTokens,
			RecommendedTasks:      caps.RecommendedTasks,
			RestrictedCapabilities: caps.RestrictedCapabilities,
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos
}

func (r *MemoryRegistry) Health(ctx context.Context) []ProviderHealthInfo {
	r.mu.RLock()
	providers := make([]provider.ModelProvider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	r.mu.RUnlock()

	health := make([]ProviderHealthInfo, 0, len(providers))
	for _, p := range providers {
		h, err := p.Health(ctx)
		info := ProviderHealthInfo{Name: p.Name(), Type: p.Type()}
		if err != nil {
			info.Available = false
			info.Message = err.Error()
		} else if h != nil {
			info.Available = h.Available
			info.LatencyMs = h.LatencyMs
			info.ModelNames = h.ModelNames
			info.Message = h.Message
		}
		health = append(health, info)
	}
	sort.Slice(health, func(i, j int) bool { return health[i].Name < health[j].Name })
	return health
}

type ProviderInfo struct {
	Name                    string
	Type                    provider.ProviderType
	SupportsStructuredOutput bool
	SupportsJSONSchema      bool
	SupportsLocalDeployment bool
	MaxInputTokens          int
	MaxOutputTokens         int
	RecommendedTasks        []provider.TaskType
	RestrictedCapabilities  []string
}

type ProviderHealthInfo struct {
	Name       string
	Type       provider.ProviderType
	Available  bool
	LatencyMs  int64
	ModelNames []string
	Message    string
}

type RegistryError struct {
	Code    string
	Message string
}

func NewRegistryError(code string, message string) *RegistryError {
	return &RegistryError{Code: code, Message: message}
}

func (e *RegistryError) Error() string {
	return e.Code + ": " + e.Message
}
