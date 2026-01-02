package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config captures runtime configuration for the JMAP API service.
type Config struct {
	HTTPAddr             string
	ShutdownTimeout      time.Duration
	ReadHeaderTimeout    time.Duration
	IdleTimeout          time.Duration
	EnableRequestLogging bool
}

// Load reads configuration values from environment variables, applying sane defaults.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:             defaultString("JMAP_API_HTTP_ADDR", "0.0.0.0:8080"),
		ShutdownTimeout:      defaultDuration("JMAP_API_SHUTDOWN_TIMEOUT", 15*time.Second),
		ReadHeaderTimeout:    defaultDuration("JMAP_API_READ_HEADER_TIMEOUT", 5*time.Second),
		IdleTimeout:          defaultDuration("JMAP_API_IDLE_TIMEOUT", 2*time.Minute),
		EnableRequestLogging: defaultBool("JMAP_API_ENABLE_REQUEST_LOGGING", true),
	}

	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("shutdown timeout must be > 0")
	}

	if cfg.ReadHeaderTimeout <= 0 {
		return Config{}, fmt.Errorf("read header timeout must be > 0")
	}

	if cfg.IdleTimeout <= 0 {
		return Config{}, fmt.Errorf("idle timeout must be > 0")
	}

	return cfg, nil
}

func defaultString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func defaultDuration(key string, fallback time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		d, err := time.ParseDuration(val)
		if err == nil {
			return d
		}
	}
	return fallback
}

func defaultBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		parsed, err := strconv.ParseBool(val)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
