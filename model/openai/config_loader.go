package openai

import "strings"

const (
	DefaultTimeoutSeconds  = 30
	DefaultMaxRetries      = 2
	DefaultMaxInputTokens  = 32768
	DefaultMaxOutputTokens = 2048
)

type ConfigSource struct {
	Name            string
	Endpoint        string
	EndpointRef     string
	SecretRef       string
	DefaultModel    string
	TimeoutSeconds  int
	MaxRetries      int
	MaxInputTokens  int
	MaxOutputTokens int
	Private         bool
}

func LoadConfig(source ConfigSource) (Config, error) {
	config := Config{
		Name:            strings.TrimSpace(source.Name),
		Endpoint:        strings.TrimSpace(source.Endpoint),
		EndpointRef:     strings.TrimSpace(source.EndpointRef),
		SecretRef:       strings.TrimSpace(source.SecretRef),
		DefaultModel:    strings.TrimSpace(source.DefaultModel),
		TimeoutSeconds:  source.TimeoutSeconds,
		MaxRetries:      source.MaxRetries,
		MaxInputTokens:  source.MaxInputTokens,
		MaxOutputTokens: source.MaxOutputTokens,
		Private:         source.Private,
	}
	applyDefaults(&config)
	if err := ValidateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func applyDefaults(config *Config) {
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = DefaultTimeoutSeconds
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = DefaultMaxRetries
	}
	if config.MaxInputTokens <= 0 {
		config.MaxInputTokens = DefaultMaxInputTokens
	}
	if config.MaxOutputTokens <= 0 {
		config.MaxOutputTokens = DefaultMaxOutputTokens
	}
}

func ValidateConfig(config Config) error {
	if config.Name == "" {
		return NewConfigError("MissingName", "provider name is required")
	}
	if config.DefaultModel == "" {
		return NewConfigError("MissingDefaultModel", "default model is required")
	}
	if config.Endpoint == "" && config.EndpointRef == "" {
		return NewConfigError("MissingEndpoint", "endpoint or endpointRef is required")
	}
	if config.Endpoint != "" && config.EndpointRef != "" {
		return NewConfigError("AmbiguousEndpoint", "endpoint and endpointRef cannot both be set")
	}
	if config.SecretRef == "" {
		return NewConfigError("MissingSecretRef", "secretRef is required; raw credentials must not be stored in provider config")
	}
	if looksLikeRawCredential(config.SecretRef) {
		return NewConfigError("RawCredentialRejected", "secretRef looks like a raw credential; use a secret reference instead")
	}
	if config.TimeoutSeconds <= 0 {
		return NewConfigError("InvalidTimeout", "timeoutSeconds must be > 0")
	}
	if config.MaxRetries < 0 {
		return NewConfigError("InvalidMaxRetries", "maxRetries must be >= 0")
	}
	if config.MaxInputTokens <= 0 {
		return NewConfigError("InvalidMaxInputTokens", "maxInputTokens must be > 0")
	}
	if config.MaxOutputTokens <= 0 {
		return NewConfigError("InvalidMaxOutputTokens", "maxOutputTokens must be > 0")
	}
	return nil
}

func looksLikeRawCredential(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "sk-") || strings.HasPrefix(lower, "bearer ") {
		return true
	}
	if strings.Contains(lower, "api_key=") || strings.Contains(lower, "apikey=") || strings.Contains(lower, "token=") {
		return true
	}
	return false
}

type ConfigError struct {
	Code    string
	Message string
}

func NewConfigError(code string, message string) *ConfigError {
	return &ConfigError{Code: code, Message: message}
}

func (e *ConfigError) Error() string {
	return e.Code + ": " + e.Message
}
