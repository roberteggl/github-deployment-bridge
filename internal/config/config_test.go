package config_test

import (
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
}
