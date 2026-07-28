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
	t.Setenv("DESCRIPTION", "Deployed via GitOps")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production")
	}
	if cfg.Description != "Deployed via GitOps" {
		t.Fatalf("Description = %q, want Deployed via GitOps", cfg.Description)
	}
	if got := cfg.ExpandLogURL(config.LogURLVars{SHA: "abc"}); got != "https://logs.example.com?commit=abc" {
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

func TestExpandLogURLTemplate(t *testing.T) {
	t.Parallel()

	const grafana = "https://grafana.example.com/a/explore/service/{name}/logs" +
		"?var-filters=service_name%7C%3D%7C{service}" +
		"&var-filters=namespace%7C%3D%7C{namespace}" +
		"&cluster={cluster}&env={environment}&sha={sha}"

	tests := []struct {
		name string
		tmpl string
		vars config.LogURLVars
		want string
	}{
		{
			name: "empty",
			tmpl: "",
			vars: config.LogURLVars{SHA: "abc"},
			want: "",
		},
		{
			name: "sha only",
			tmpl: "https://logs.example.com?commit={sha}",
			vars: config.LogURLVars{SHA: "deadbeef"},
			want: "https://logs.example.com?commit=deadbeef",
		},
		{
			name: "grafana loki explore",
			tmpl: grafana,
			vars: config.LogURLVars{
				SHA:         "deadbeef",
				Namespace:   "neuland-app",
				Name:        "neuland-api-prod",
				Environment: "production",
				Cluster:     "neuland",
			},
			want: "https://grafana.example.com/a/explore/service/neuland-api-prod/logs" +
				"?var-filters=service_name%7C%3D%7Cneuland-api-prod" +
				"&var-filters=namespace%7C%3D%7Cneuland-app" +
				"&cluster=neuland&env=production&sha=deadbeef",
		},
		{
			name: "service annotation overrides name",
			tmpl: "https://logs.example.com/s/{service}/n/{name}",
			vars: config.LogURLVars{
				Name:    "deploy-name",
				Service: "loki-service",
			},
			want: "https://logs.example.com/s/loki-service/n/deploy-name",
		},
		{
			name: "service falls back to name",
			tmpl: "https://logs.example.com/s/{service}",
			vars: config.LogURLVars{Name: "backend"},
			want: "https://logs.example.com/s/backend",
		},
		{
			name: "grafana dashboard preset",
			tmpl: "https://grafana.example.com/d/workload/logs?var-cluster={cluster}&var-namespace={namespace}&var-service={service}&var-environment={environment}&var-sha={sha}",
			vars: config.LogURLVars{SHA: "deadbeef", Namespace: "apps", Name: "api", Cluster: "prod", Environment: "production"},
			want: "https://grafana.example.com/d/workload/logs?var-cluster=prod&var-namespace=apps&var-service=api&var-environment=production&var-sha=deadbeef",
		},
		{
			name: "generic flux kubernetes dashboard preset",
			tmpl: "https://ops.example.com/clusters/{cluster}/namespaces/{namespace}/workloads/{name}?environment={environment}&service={service}&revision={sha}",
			vars: config.LogURLVars{SHA: "abc", Namespace: "payments", Name: "worker", Service: "payments-worker", Cluster: "eu-1", Environment: "staging"},
			want: "https://ops.example.com/clusters/eu-1/namespaces/payments/workloads/worker?environment=staging&service=payments-worker&revision=abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := config.ExpandLogURLTemplate(tt.tmpl, tt.vars); got != tt.want {
				t.Fatalf("ExpandLogURLTemplate = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandLogURLTemplateEscaped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		tmpl   string
		vars   config.LogURLVars
		escape bool
		want   string
	}{
		{"backwards compatible literal values", "https://logs.example/{namespace}?q={service}", config.LogURLVars{Namespace: "team one", Service: "api/core"}, false, "https://logs.example/team one?q=api/core"},
		{"escape path and query values", "https://logs.example/{namespace}?q={service}&env={environment}", config.LogURLVars{Namespace: "team one", Service: "api/core?x=1", Environment: "preview & qa"}, true, "https://logs.example/team%20one?q=api%2Fcore%3Fx%3D1&env=preview%20%26%20qa"},
		{"escape fallback service", "https://logs.example/?service={service}", config.LogURLVars{Name: "api/canary"}, true, "https://logs.example/?service=api%2Fcanary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.ExpandLogURLTemplateEscaped(tt.tmpl, tt.vars, tt.escape); got != tt.want {
				t.Fatalf("ExpandLogURLTemplateEscaped = %q, want %q", got, tt.want)
			}
		})
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
