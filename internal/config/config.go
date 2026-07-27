package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	RedisURL      string
	LogLevel      string
	RepositoryMode string
	RunMigrations bool
	Provider      ProviderConfig
}

type ProviderConfig struct {
	Enabled         bool
	Name            string
	Endpoint        string
	SecretEnv       string
	DefaultModel    string
	TimeoutSeconds  int
	MaxRetries      int
	MaxInputTokens  int
	MaxOutputTokens int
	Private         bool
}

func Load() Config {
	return Config{
		HTTPAddr:       env("AICLOUD_HTTP_ADDR", ":8080"),
		DatabaseURL:    env("AICLOUD_DATABASE_URL", "postgres://aicloud:aicloud@localhost:5432/aicloud?sslmode=disable"),
		RedisURL:       env("AICLOUD_REDIS_URL", "redis://localhost:6379/0"),
		LogLevel:       env("AICLOUD_LOG_LEVEL", "INFO"),
		RepositoryMode: env("AICLOUD_REPOSITORY_MODE", "memory"),
		RunMigrations:  envBool("AICLOUD_RUN_MIGRATIONS", false),
		Provider: ProviderConfig{
			Enabled:         envBool("AICLOUD_PROVIDER_ENABLED", false),
			Name:            env("AICLOUD_PROVIDER_NAME", "openai-compatible"),
			Endpoint:        env("AICLOUD_PROVIDER_ENDPOINT", "https://api.openai.com/v1"),
			SecretEnv:       env("AICLOUD_PROVIDER_SECRET_ENV", "OPENAI_API_KEY"),
			DefaultModel:    env("AICLOUD_PROVIDER_MODEL", ""),
			TimeoutSeconds:  envInt("AICLOUD_PROVIDER_TIMEOUT_SECONDS", 60),
			MaxRetries:      envInt("AICLOUD_PROVIDER_MAX_RETRIES", 2),
			MaxInputTokens:  envInt("AICLOUD_PROVIDER_MAX_INPUT_TOKENS", 128000),
			MaxOutputTokens: envInt("AICLOUD_PROVIDER_MAX_OUTPUT_TOKENS", 8192),
			Private:         envBool("AICLOUD_PROVIDER_PRIVATE", false),
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
