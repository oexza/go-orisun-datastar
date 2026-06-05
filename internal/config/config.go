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
		OrisunBoundary:    env("ORISUN_GENERAL_BOUNDARY", "go_orisun_datastar"),
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
