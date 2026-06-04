package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                 string
	AppURL               string
	PostgresURL          string
	PostgresHost         string
	PostgresPort         string
	PostgresUser         string
	PostgresPassword     string
	PostgresDatabase     string
	PostgresSSLMode      string
	NATSURL              string
	OrisunAddress        string
	OrisunBoundary       string
	SessionSecret        string
	BrevoAPIKey          string
	BrevoSenderEmail     string
	BrevoSenderName      string
	StorageProvider      string
	StorageEndpoint      string
	StorageAccessKey     string
	StorageSecretKey     string
	StorageBucket        string
	StoragePublicURL     string
	R2Endpoint           string
	R2AccessKeyID        string
	R2SecretAccessKey    string
	R2Bucket             string
	R2PublicURL          string
	UseSSLForPostgres    bool
	DevelopmentCookie    bool
}

func Load() Config {
	port := env("PORT", "3000")
	useSSL := env("POSTGRES_USE_SSL", "false") == "true"
	sslMode := "disable"
	if useSSL {
		sslMode = "require"
	}

	pgURL := os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			env("POSTGRES_USER", "postgres"),
			env("POSTGRES_PASSWORD", "postgres"),
			env("POSTGRES_HOST", "localhost"),
			env("POSTGRES_PORT", "5432"),
			env("POSTGRES_DB", "hono_event_starter"),
			sslMode,
		)
	}

	return Config{
		Port:              port,
		AppURL:            env("APP_URL", "http://localhost:"+port),
		PostgresURL:       pgURL,
		PostgresHost:      env("POSTGRES_HOST", "localhost"),
		PostgresPort:      env("POSTGRES_PORT", "5432"),
		PostgresUser:      env("POSTGRES_USER", "postgres"),
		PostgresPassword:  env("POSTGRES_PASSWORD", "postgres"),
		PostgresDatabase:  env("POSTGRES_DB", "hono_event_starter"),
		PostgresSSLMode:   sslMode,
		NATSURL:           env("NATS_URL", "nats://localhost:4224"),
		OrisunAddress:     env("ORISUN_HOST", "localhost") + ":" + env("ORISUN_PORT", "5006"),
		OrisunBoundary:    env("ORISUN_GENERAL_BOUNDARY", "hono_event_starter"),
		SessionSecret:     env("BETTER_AUTH_SECRET", "secret-key-that-should-be-very-secret"),
		BrevoAPIKey:       os.Getenv("BREVO_API_KEY"),
		BrevoSenderEmail:  os.Getenv("BREVO_SENDER_EMAIL"),
		BrevoSenderName:   env("BREVO_SENDER_NAME", "Hono Event Starter"),
		StorageProvider:   env("STORAGE_PROVIDER", "garage"),
		StorageEndpoint:   os.Getenv("STORAGE_ENDPOINT"),
		StorageAccessKey:  os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:  os.Getenv("STORAGE_SECRET_KEY"),
		StorageBucket:     os.Getenv("STORAGE_BUCKET"),
		StoragePublicURL:  os.Getenv("STORAGE_PUBLIC_URL"),
		R2Endpoint:        os.Getenv("R2_ENDPOINT"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2PublicURL:       os.Getenv("R2_PUBLIC_URL"),
		UseSSLForPostgres: useSSL,
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
