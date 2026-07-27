package providerfactory

import (
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/config"
	modelopenai "github.com/tommyxie2026-tech/aicloud/model/openai"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func BuildOpenAICompatible(cfg config.ProviderConfig) (provider.ModelProvider, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.DefaultModel == "" {
		return nil, fmt.Errorf("AICLOUD_PROVIDER_MODEL is required when the provider is enabled")
	}
	adapterConfig := modelopenai.Config{
		Name:            cfg.Name,
		Endpoint:        cfg.Endpoint,
		SecretRef:       cfg.SecretEnv,
		DefaultModel:    cfg.DefaultModel,
		TimeoutSeconds:  cfg.TimeoutSeconds,
		MaxRetries:      cfg.MaxRetries,
		MaxInputTokens:  cfg.MaxInputTokens,
		MaxOutputTokens: cfg.MaxOutputTokens,
		Private:         cfg.Private,
	}
	client, err := modelopenai.NewHTTPClient(adapterConfig, nil, modelopenai.EnvSecretResolver{})
	if err != nil {
		return nil, fmt.Errorf("build OpenAI-compatible client: %w", err)
	}
	return modelopenai.NewProviderWithParser(adapterConfig, client, modelopenai.JSONParser{}), nil
}
