package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JMAP_API_HTTP_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPAddr != "0.0.0.0:8080" {
		t.Fatalf("unexpected http addr: %s", cfg.HTTPAddr)
	}

	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.ShutdownTimeout)
	}
}

func TestLoadInvalidDurations(t *testing.T) {
	t.Setenv("JMAP_API_SHUTDOWN_TIMEOUT", "0s")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error for non-positive shutdown timeout")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("JMAP_API_HTTP_ADDR", "127.0.0.1:9000")
	t.Setenv("JMAP_API_SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("JMAP_API_READ_HEADER_TIMEOUT", "2s")
	t.Setenv("JMAP_API_IDLE_TIMEOUT", "10s")
	t.Setenv("JMAP_API_ENABLE_REQUEST_LOGGING", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:9000" {
		t.Fatalf("http addr override failed: %s", cfg.HTTPAddr)
	}

	if cfg.ShutdownTimeout != 30*time.Second {
		t.Fatalf("shutdown timeout override failed: %s", cfg.ShutdownTimeout)
	}

	if cfg.EnableRequestLogging {
		t.Fatalf("expected logging disabled")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
