// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package config loads bridge configuration from the environment.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for the deployment bridge.
type Config struct {
	ClusterName          string        `envconfig:"CLUSTER_NAME" required:"true"`
	Environment          string        `envconfig:"ENVIRONMENT" required:"true"`
	WatchNamespace       string        `envconfig:"WATCH_NAMESPACE" default:""`
	DatabasePath         string        `envconfig:"DATABASE" default:"/data/cache.db"`
	EnvironmentURL       string        `envconfig:"ENVIRONMENT_URL"`
	LogURLTemplate       string        `envconfig:"LOG_URL_TEMPLATE"`
	MetricsAddr          string        `envconfig:"METRICS_ADDR" default:":8080"`
	ProbeAddr            string        `envconfig:"PROBE_ADDR" default:":8081"`
	LeaderElection       bool          `envconfig:"LEADER_ELECTION" default:"true"`
	LeaderElectionID     string        `envconfig:"LEADER_ELECTION_ID" default:"github-deployment-bridge"`
	GitHubAppID          int64         `envconfig:"GITHUB_APP_ID" required:"true"`
	GitHubInstallationID int64         `envconfig:"GITHUB_INSTALLATION_ID" required:"true"`
	GitHubPrivateKeyPath string        `envconfig:"GITHUB_PRIVATE_KEY_PATH" required:"true"`
	GitHubBaseURL        string        `envconfig:"GITHUB_BASE_URL"` // optional, for GHES
	RetryMaxAttempts     int           `envconfig:"RETRY_MAX_ATTEMPTS" default:"5"`
	RetryInitialBackoff  time.Duration `envconfig:"RETRY_INITIAL_BACKOFF" default:"500ms"`
	RetryMaxBackoff      time.Duration `envconfig:"RETRY_MAX_BACKOFF" default:"30s"`
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	cfg.ClusterName = strings.TrimSpace(cfg.ClusterName)
	if cfg.Environment == "" {
		return Config{}, fmt.Errorf("ENVIRONMENT must not be empty")
	}
	if cfg.ClusterName == "" {
		return Config{}, fmt.Errorf("CLUSTER_NAME must not be empty")
	}
	if cfg.GitHubAppID <= 0 {
		return Config{}, fmt.Errorf("GITHUB_APP_ID must be a positive integer")
	}
	if cfg.GitHubInstallationID <= 0 {
		return Config{}, fmt.Errorf("GITHUB_INSTALLATION_ID must be a positive integer")
	}
	if strings.TrimSpace(cfg.GitHubPrivateKeyPath) == "" {
		return Config{}, fmt.Errorf("GITHUB_PRIVATE_KEY_PATH must not be empty")
	}
	return cfg, nil
}

// IsProduction reports whether the configured environment is production.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// ExpandLogURL substitutes {sha} in the log URL template.
func (c Config) ExpandLogURL(sha string) string {
	if c.LogURLTemplate == "" {
		return ""
	}
	return strings.ReplaceAll(c.LogURLTemplate, "{sha}", sha)
}
