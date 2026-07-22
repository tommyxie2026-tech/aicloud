package config

import "os"

type Config struct{ HTTPAddr, DatabaseURL, RedisURL, LogLevel string }

func Load() Config {
	return Config{
		HTTPAddr: env("AICLOUD_HTTP_ADDR", ":8080"), DatabaseURL: env("AICLOUD_DATABASE_URL", "postgres://aicloud:aicloud@localhost:5432/aicloud?sslmode=disable"),
		RedisURL: env("AICLOUD_REDIS_URL", "redis://localhost:6379/0"), LogLevel: env("AICLOUD_LOG_LEVEL", "INFO"),
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
