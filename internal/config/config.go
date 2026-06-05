package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port              string
	AppURL            string
	SQLitePath        string
	OrisunSQLiteDir   string
	NATSURL           string
	OrisunBoundary    string
	SessionSecret     string
	UploadDir         string
	UploadBaseURL     string
	DevelopmentCookie bool
}

func Load() Config {
	port := env("PORT", "3001")

	return Config{
		Port:              port,
		AppURL:            env("APP_URL", "http://localhost:"+port),
		SQLitePath:        env("SQLITE_PATH", "data/app.sqlite"),
		OrisunSQLiteDir:   env("ORISUN_SQLITE_DIR", "data/orisun"),
		NATSURL:           env("NATS_URL", "nats://localhost:4224"),
		OrisunBoundary:    env("ORISUN_GENERAL_BOUNDARY", "hono_event_starter"),
		SessionSecret:     env("BETTER_AUTH_SECRET", "secret-key-that-should-be-very-secret"),
		UploadDir:         env("UPLOAD_DIR", "static/uploads"),
		UploadBaseURL:     env("UPLOAD_BASE_URL", "/static/uploads"),
		DevelopmentCookie: env("NODE_ENV", "development") != "production",
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func IntEnv(key string, fallback int) int {
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
