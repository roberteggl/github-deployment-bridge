// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package config loads bridge configuration from the environment.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for the deployment bridge.
type Config struct {
	ClusterName                string        `envconfig:"CLUSTER_NAME" required:"true"`
	Environment                string        `envconfig:"ENVIRONMENT" required:"true"`
	WatchNamespace             string        `envconfig:"WATCH_NAMESPACE" default:""`
	DatabasePath               string        `envconfig:"DATABASE" default:"/data/cache.db"`
	EnvironmentURL             string        `envconfig:"ENVIRONMENT_URL"`
	Description                string        `envconfig:"DESCRIPTION"`
	LogURLTemplate             string        `envconfig:"LOG_URL_TEMPLATE"`
	LogURLTemplateEscape       bool          `envconfig:"LOG_URL_TEMPLATE_ESCAPE" default:"false"`
	LogLevel                   string        `envconfig:"LOG_LEVEL" default:"info"`
	MetricsAddr                string        `envconfig:"METRICS_ADDR" default:":8080"`
	ProbeAddr                  string        `envconfig:"PROBE_ADDR" default:":8081"`
	LeaderElection             bool          `envconfig:"LEADER_ELECTION" default:"true"`
	LeaderElectionID           string        `envconfig:"LEADER_ELECTION_ID" default:"github-deployment-bridge"`
	GitHubAppID                int64         `envconfig:"GITHUB_APP_ID" required:"true"`
	GitHubInstallationID       int64         `envconfig:"GITHUB_INSTALLATION_ID"`
	GitHubInstallationCacheTTL time.Duration `envconfig:"GITHUB_INSTALLATION_CACHE_TTL" default:"1h"`
	GitHubPrivateKeyPath       string        `envconfig:"GITHUB_PRIVATE_KEY_PATH" required:"true"`
	GitHubBaseURL              string        `envconfig:"GITHUB_BASE_URL"` // optional, for GHES
	RetryMaxAttempts           int           `envconfig:"RETRY_MAX_ATTEMPTS" default:"5"`
	RetryInitialBackoff        time.Duration `envconfig:"RETRY_INITIAL_BACKOFF" default:"500ms"`
	RetryMaxBackoff            time.Duration `envconfig:"RETRY_MAX_BACKOFF" default:"30s"`
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	cfg.ClusterName = strings.TrimSpace(cfg.ClusterName)
	cfg.LogLevel = strings.TrimSpace(cfg.LogLevel)
	if cfg.Environment == "" {
		return Config{}, fmt.Errorf("ENVIRONMENT must not be empty")
	}
	if cfg.ClusterName == "" {
		return Config{}, fmt.Errorf("CLUSTER_NAME must not be empty")
	}
	if _, err := cfg.SlogLevel(); err != nil {
		return Config{}, err
	}
	if cfg.GitHubAppID <= 0 {
		return Config{}, fmt.Errorf("GITHUB_APP_ID must be a positive integer")
	}
	if cfg.GitHubInstallationID < 0 {
		return Config{}, fmt.Errorf("GITHUB_INSTALLATION_ID must be a positive integer when set")
	}
	if cfg.GitHubInstallationCacheTTL <= 0 {
		return Config{}, fmt.Errorf("GITHUB_INSTALLATION_CACHE_TTL must be positive")
	}
	if strings.TrimSpace(cfg.GitHubPrivateKeyPath) == "" {
		return Config{}, fmt.Errorf("GITHUB_PRIVATE_KEY_PATH must not be empty")
	}
	return cfg, nil
}

// SlogLevel parses LOG_LEVEL into an slog level.
// Accepted values: debug, info, warn (or warning), error (case-insensitive).
func (c Config) SlogLevel() (slog.Level, error) {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}

// IsProduction reports whether the configured environment is production.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// LogURLVars are placeholders for ExpandLogURL / ExpandLogURLTemplate.
type LogURLVars struct {
	SHA         string // {sha} — commit SHA
	Namespace   string // {namespace} — workload namespace
	Name        string // {name} — workload name (Deployment/StatefulSet/DaemonSet)
	Service     string // {service} — annotation service, else Name
	Environment string // {environment} — resolved GitHub environment
	Cluster     string // {cluster} — resolved cluster name
}

// ExpandLogURL substitutes placeholders in LOG_URL_TEMPLATE.
func (c Config) ExpandLogURL(v LogURLVars) string {
	return ExpandLogURLTemplateEscaped(c.LogURLTemplate, v, c.LogURLTemplateEscape)
}

// ExpandLogURLTemplate substitutes log URL placeholders in tmpl.
//
// Supported: {sha}, {namespace}, {name}, {service}, {environment}, {cluster}.
// {service} falls back to {name} when Service is empty.
func ExpandLogURLTemplate(tmpl string, v LogURLVars) string {
	return ExpandLogURLTemplateEscaped(tmpl, v, false)
}

// ExpandLogURLTemplateEscaped substitutes placeholders, optionally percent-encoding
// each value so it is safe in both URL paths and query parameters.
func ExpandLogURLTemplateEscaped(tmpl string, v LogURLVars, escape bool) string {
	if strings.TrimSpace(tmpl) == "" {
		return ""
	}
	service := strings.TrimSpace(v.Service)
	if service == "" {
		service = v.Name
	}
	value := func(s string) string {
		if escape {
			// QueryEscape covers URL delimiters (including '/', '&', '?' and '#').
			// Use %20 rather than '+' so the result is also safe in path segments.
			return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
		}
		return s
	}
	return strings.NewReplacer(
		"{sha}", value(v.SHA),
		"{namespace}", value(v.Namespace),
		"{name}", value(v.Name),
		"{service}", value(service),
		"{environment}", value(v.Environment),
		"{cluster}", value(v.Cluster),
	).Replace(tmpl)
}
