package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	RedisURL       string
	LogLevel       string
	RepositoryMode string
	RunMigrations  bool
	Auth           AuthConfig
	Provider       ProviderConfig
	Temporal       TemporalConfig
}

type TemporalConfig struct {
	Enabled                  bool
	Address                  string
	Namespace                string
	TaskQueue                string
	WorkerStopTimeoutSeconds int
}

func (c TemporalConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Address) == "" {
		return &ConfigError{Field: "AICLOUD_TEMPORAL_ADDRESS", Message: "is required when Temporal is enabled"}
	}
	if strings.TrimSpace(c.Namespace) == "" {
		return &ConfigError{Field: "AICLOUD_TEMPORAL_NAMESPACE", Message: "is required when Temporal is enabled"}
	}
	if strings.TrimSpace(c.TaskQueue) == "" {
		return &ConfigError{Field: "AICLOUD_TEMPORAL_TASK_QUEUE", Message: "is required when Temporal is enabled"}
	}
	if c.WorkerStopTimeoutSeconds <= 0 {
		return &ConfigError{Field: "AICLOUD_TEMPORAL_WORKER_STOP_TIMEOUT_SECONDS", Message: "must be greater than zero when Temporal is enabled"}
	}
	return nil
}

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	if e == nil {
		return "invalid configuration"
	}
	return e.Field + " " + e.Message
}

type ProviderConfig struct {
	Enabled          bool
	Name             string
	Endpoint         string
	SecretEnv        string
	DefaultModel     string
	ModelVersion     string
	TimeoutSeconds   int
	MaxRetries       int
	MaxInputTokens   int
	MaxOutputTokens  int
	Private          bool
	Approved         bool
	LicenseID        string
	LicenseTextRef   string
	EvidenceReviewer string
	DataResidency    string
	Currency         string
	InputPerMillion  float64
	OutputPerMillion float64
}

func Load() Config {
	return Config{
		HTTPAddr:       env("AICLOUD_HTTP_ADDR", ":8080"),
		DatabaseURL:    env("AICLOUD_DATABASE_URL", "postgres://aicloud:aicloud@localhost:5432/aicloud?sslmode=disable"),
		RedisURL:       env("AICLOUD_REDIS_URL", "redis://localhost:6379/0"),
		LogLevel:       env("AICLOUD_LOG_LEVEL", "INFO"),
		RepositoryMode: env("AICLOUD_REPOSITORY_MODE", "memory"),
		RunMigrations:  envBool("AICLOUD_RUN_MIGRATIONS", false),
		Auth:           loadAuthConfig(),
		Temporal: TemporalConfig{
			Enabled:                  envBool("AICLOUD_TEMPORAL_ENABLED", false),
			Address:                  env("AICLOUD_TEMPORAL_ADDRESS", "localhost:7233"),
			Namespace:                env("AICLOUD_TEMPORAL_NAMESPACE", "default"),
			TaskQueue:                env("AICLOUD_TEMPORAL_TASK_QUEUE", "aicloud-task-v1"),
			WorkerStopTimeoutSeconds: envInt("AICLOUD_TEMPORAL_WORKER_STOP_TIMEOUT_SECONDS", 30),
		},
		Provider: ProviderConfig{
			Enabled:          envBool("AICLOUD_PROVIDER_ENABLED", false),
			Name:             env("AICLOUD_PROVIDER_NAME", "openai-compatible"),
			Endpoint:         env("AICLOUD_PROVIDER_ENDPOINT", "https://api.openai.com/v1"),
			SecretEnv:        env("AICLOUD_PROVIDER_SECRET_ENV", "OPENAI_API_KEY"),
			DefaultModel:     env("AICLOUD_PROVIDER_MODEL", ""),
			ModelVersion:     env("AICLOUD_PROVIDER_MODEL_VERSION", "v1"),
			TimeoutSeconds:   envInt("AICLOUD_PROVIDER_TIMEOUT_SECONDS", 60),
			MaxRetries:       envInt("AICLOUD_PROVIDER_MAX_RETRIES", 2),
			MaxInputTokens:   envInt("AICLOUD_PROVIDER_MAX_INPUT_TOKENS", 128000),
			MaxOutputTokens:  envInt("AICLOUD_PROVIDER_MAX_OUTPUT_TOKENS", 8192),
			Private:          envBool("AICLOUD_PROVIDER_PRIVATE", false),
			Approved:         envBool("AICLOUD_PROVIDER_APPROVED", false),
			LicenseID:        env("AICLOUD_PROVIDER_LICENSE", "commercial-api"),
			LicenseTextRef:   env("AICLOUD_PROVIDER_LICENSE_TEXT_REF", ""),
			EvidenceReviewer: env("AICLOUD_PROVIDER_EVIDENCE_REVIEWER", "runtime-configuration"),
			DataResidency:    env("AICLOUD_PROVIDER_DATA_RESIDENCY", ""),
			Currency:         env("AICLOUD_PROVIDER_CURRENCY", "USD"),
			InputPerMillion:  envFloat("AICLOUD_PROVIDER_INPUT_PER_MILLION", 0),
			OutputPerMillion: envFloat("AICLOUD_PROVIDER_OUTPUT_PER_MILLION", 0),
		},
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
