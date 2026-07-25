package config_test

import (
	"log/slog"
	"testing"

	"github.com/roberteggl/github-deployment-bridge/internal/config"
)

func TestLoadAndExpand(t *testing.T) {
	t.Setenv("CLUSTER_NAME", "production-eu")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("GITHUB_APP_ID", "123")
	t.Setenv("GITHUB_INSTALLATION_ID", "456")
	t.Setenv("GITHUB_PRIVATE_KEY_PATH", "/tmp/key.pem")
	t.Setenv("LOG_URL_TEMPLATE", "https://logs.example.com?commit={sha}")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production")
	}
	if got := cfg.ExpandLogURL("abc"); got != "https://logs.example.com?commit=abc" {
		t.Fatalf("ExpandLogURL = %q", got)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel default = %q, want info", cfg.LogLevel)
	}
	level, err := cfg.SlogLevel()
	if err != nil {
		t.Fatalf("SlogLevel: %v", err)
	}
	if level != slog.LevelInfo {
		t.Fatalf("SlogLevel = %v, want info", level)
	}
}

func TestLogLevel(t *testing.T) {
	base := func(t *testing.T) {
		t.Helper()
		t.Setenv("CLUSTER_NAME", "c")
		t.Setenv("ENVIRONMENT", "staging")
		t.Setenv("GITHUB_APP_ID", "1")
		t.Setenv("GITHUB_INSTALLATION_ID", "2")
		t.Setenv("GITHUB_PRIVATE_KEY_PATH", "/tmp/key.pem")
	}

	t.Run("debug", func(t *testing.T) {
		base(t)
		t.Setenv("LOG_LEVEL", "DEBUG")
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		level, err := cfg.SlogLevel()
		if err != nil {
			t.Fatalf("SlogLevel: %v", err)
		}
		if level != slog.LevelDebug {
			t.Fatalf("got %v", level)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		base(t)
		t.Setenv("LOG_LEVEL", "verbose")
		if _, err := config.Load(); err == nil {
			t.Fatal("expected error for invalid LOG_LEVEL")
		}
	})
}
