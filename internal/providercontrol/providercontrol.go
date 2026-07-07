package providercontrol

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

type LimitConfig struct {
	Burst       int
	BurstWindow time.Duration
	Daily       int
}

func TimeoutFromEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
		return parsed
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func WithTimeout(ctx context.Context, envKey string, fallback time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, TimeoutFromEnv(envKey, fallback))
}

func Call[T any](ctx context.Context, envKey string, fallback time.Duration, call func(context.Context) (T, error)) (T, error) {
	callCtx, cancel := WithTimeout(ctx, envKey, fallback)
	defer cancel()
	return call(callCtx)
}

func LimitsFromEnv(prefix string, defaults LimitConfig) LimitConfig {
	return LimitConfig{
		Burst:       positiveInt(prefix+"_BURST_LIMIT", defaults.Burst),
		BurstWindow: TimeoutFromEnv(prefix+"_BURST_WINDOW", defaults.BurstWindow),
		Daily:       positiveInt(prefix+"_DAILY_LIMIT", defaults.Daily),
	}
}

func positiveInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
