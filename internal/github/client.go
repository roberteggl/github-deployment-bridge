// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package github provides a GitHub App authenticated client for Deployments.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v89/github"

	"github.com/roberteggl/github-deployment-bridge/internal/metrics"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

// DeploymentRequest describes a deployment to create.
type DeploymentRequest struct {
	Owner                 string
	Repo                  string
	Ref                   string
	Environment           string
	ProductionEnvironment bool
	Description           string
	// Task is the GitHub Deployment task (defaults to "deploy" when empty).
	// Used when github-deployment-bridge.io/deployment-name is set.
	Task string
	// Payload is optional JSON metadata attached to the deployment.
	Payload map[string]any
}

// DeploymentStatusRequest describes a deployment status to create.
type DeploymentStatusRequest struct {
	Owner          string
	Repo           string
	DeploymentID   int64
	State          string
	EnvironmentURL string
	LogURL         string
	Description    string
	// AutoInactive asks GitHub to mark prior successes in the same environment
	// inactive. Leave false for monorepo workloads that share an environment but
	// use distinct deployment-name values; the bridge supersedes via markPriorInactive.
	AutoInactive bool
}

// DeploymentResult is returned after creating a deployment.
type DeploymentResult struct {
	ID int64
}

// FindDeploymentRequest locates an existing deployment for crash recovery.
type FindDeploymentRequest struct {
	Owner       string
	Repo        string
	Ref         string
	Environment string
	// Payload must match the deployment payload created by the bridge.
	Payload map[string]any
}

// Client creates GitHub Deployments and statuses.
type Client interface {
	CreateDeployment(ctx context.Context, req DeploymentRequest) (*DeploymentResult, error)
	FindDeployment(ctx context.Context, req FindDeploymentRequest) (*DeploymentResult, error)
	CreateDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) error
}

// AppClient is a GitHub App installation client.
type AppClient struct {
	client  *github.Client
	metrics *metrics.Metrics
	retry   retry.Config
}

// Options configures a new AppClient.
type Options struct {
	AppID          int64
	InstallationID int64
	PrivateKeyPath string
	BaseURL        string
	Metrics        *metrics.Metrics
	Retry          retry.Config
	Transport      http.RoundTripper
}

// NewAppClient builds a GitHub App installation client.
func NewAppClient(opts Options) (*AppClient, error) {
	base := opts.Transport
	if base == nil {
		base = http.DefaultTransport
	}

	itr, err := ghinstallation.NewKeyFromFile(base, opts.AppID, opts.InstallationID, opts.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App transport: %w", err)
	}

	httpClient := &http.Client{Transport: itr, Timeout: 30 * time.Second}
	clientOpts := []github.ClientOptionsFunc{
		github.WithHTTPClient(httpClient),
	}
	if opts.BaseURL != "" {
		// Accept either https://ghe.example.com or https://ghe.example.com/api/v3.
		baseURL := strings.TrimRight(opts.BaseURL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/api/v3")
		itr.BaseURL = baseURL + "/api/v3"
		clientOpts = append(clientOpts, github.WithEnterpriseURLs(baseURL+"/", baseURL+"/"))
	}

	client, err := github.NewClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create GitHub client: %w", err)
	}

	cfg := opts.Retry
	if cfg.MaxAttempts == 0 {
		cfg = retry.Default()
	}

	return &AppClient{
		client:  client,
		metrics: opts.Metrics,
		retry:   cfg,
	}, nil
}

// CreateDeployment creates a GitHub Deployment.
func (c *AppClient) CreateDeployment(ctx context.Context, req DeploymentRequest) (*DeploymentResult, error) {
	var result *DeploymentResult
	err := retry.Do(ctx, c.retry, func(ctx context.Context) error {
		start := time.Now()
		desc := req.Description
		if desc == "" {
			desc = "Deployed by FluxCD"
		}
		ghReq := github.DeploymentRequest{
			Ref:                   req.Ref,
			Environment:           github.Ptr(req.Environment),
			AutoMerge:             github.Ptr(false),
			RequiredContexts:      []string{},
			ProductionEnvironment: github.Ptr(req.ProductionEnvironment),
			Description:           github.Ptr(desc),
		}
		if req.Task != "" {
			ghReq.Task = github.Ptr(req.Task)
		}
		if len(req.Payload) > 0 {
			ghReq.Payload = req.Payload
		}

		dep, resp, err := c.client.Repositories.CreateDeployment(ctx, req.Owner, req.Repo, ghReq)
		c.observe("create_deployment", start, err, resp)
		if err != nil {
			return classifyGitHubError(err, resp)
		}
		if dep == nil || dep.GetID() == 0 {
			return retry.Permanent(fmt.Errorf("github returned deployment without id"))
		}
		result = &DeploymentResult{ID: dep.GetID()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// FindDeployment returns a deployment matching ref, environment, and payload.
// Returns nil when no matching deployment exists.
func (c *AppClient) FindDeployment(ctx context.Context, req FindDeploymentRequest) (*DeploymentResult, error) {
	var result *DeploymentResult
	err := retry.Do(ctx, c.retry, func(ctx context.Context) error {
		start := time.Now()
		opts := &github.DeploymentsListOptions{
			Environment: req.Environment,
			Ref:         req.Ref,
			ListOptions: github.ListOptions{PerPage: 100},
		}
		deployments, resp, err := c.client.Repositories.ListDeployments(ctx, req.Owner, req.Repo, opts)
		c.observe("list_deployments", start, err, resp)
		if err != nil {
			return classifyGitHubError(err, resp)
		}
		for _, dep := range deployments {
			if dep == nil || dep.GetID() == 0 {
				continue
			}
			if !deploymentPayloadMatches(dep.GetPayload(), req.Payload) {
				continue
			}
			result = &DeploymentResult{ID: dep.GetID()}
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateDeploymentStatus creates a deployment status.
func (c *AppClient) CreateDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) error {
	return retry.Do(ctx, c.retry, func(ctx context.Context) error {
		start := time.Now()
		desc := req.Description
		if desc == "" {
			desc = "Flux reconciliation update"
		}
		ghReq := github.DeploymentStatusRequest{
			State:        req.State,
			Description:  github.Ptr(desc),
			AutoInactive: github.Ptr(req.AutoInactive),
		}
		if req.EnvironmentURL != "" {
			ghReq.EnvironmentURL = github.Ptr(req.EnvironmentURL)
		}
		if req.LogURL != "" {
			ghReq.LogURL = github.Ptr(req.LogURL)
		}

		_, resp, err := c.client.Repositories.CreateDeploymentStatus(ctx, req.Owner, req.Repo, req.DeploymentID, ghReq)
		c.observe("create_deployment_status", start, err, resp)
		if err != nil {
			return classifyGitHubError(err, resp)
		}
		return nil
	})
}

func (c *AppClient) observe(operation string, start time.Time, err error, resp *github.Response) {
	if c.metrics == nil {
		return
	}
	c.metrics.GitHubAPILatencySeconds.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	result := "success"
	if err != nil {
		result = "error"
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			result = "client_error"
		}
		c.metrics.GitHubAPIFailuresTotal.Inc()
	}
	c.metrics.GitHubAPIRequestsTotal.WithLabelValues(operation, result).Inc()
}

func deploymentPayloadMatches(raw json.RawMessage, expected map[string]any) bool {
	if len(expected) == 0 || len(raw) == 0 {
		return false
	}
	var actual map[string]any
	if err := json.Unmarshal(raw, &actual); err != nil {
		return false
	}
	for key, want := range expected {
		got, ok := actual[key]
		if !ok || fmt.Sprint(got) != fmt.Sprint(want) {
			return false
		}
	}
	return true
}

func classifyGitHubError(err error, resp *github.Response) error {
	if err == nil {
		return nil
	}

	// Primary and secondary rate limits are retryable even when the status is
	// 403 (common for secondary limits). Prefer Retry-After / rate reset.
	var abuse *github.AbuseRateLimitError
	var rateLimit *github.RateLimitError
	if errors.As(err, &abuse) || errors.As(err, &rateLimit) {
		if d, ok := githubRetryAfter(err, resp); ok {
			return retry.After(err, d)
		}
		return err
	}

	if resp == nil {
		return err
	}
	// Retry rate limits and server errors; treat other 4xx as permanent.
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		if d, ok := githubRetryAfter(err, resp); ok {
			return retry.After(err, d)
		}
		return err
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return retry.Permanent(err)
	}
	return err
}

// githubRetryAfter extracts a wait from AbuseRateLimitError, RateLimitError,
// Retry-After, or X-RateLimit-Reset.
func githubRetryAfter(err error, resp *github.Response) (time.Duration, bool) {
	var abuse *github.AbuseRateLimitError
	if errors.As(err, &abuse) && abuse.RetryAfter != nil && *abuse.RetryAfter > 0 {
		return *abuse.RetryAfter, true
	}

	var rateLimit *github.RateLimitError
	if errors.As(err, &rateLimit) {
		if d := time.Until(rateLimit.Rate.Reset.Time); d > 0 {
			return d, true
		}
	}

	if resp == nil || resp.Response == nil {
		return 0, false
	}
	if d, ok := parseRetryAfterHeader(resp.Header.Get("Retry-After")); ok {
		return d, true
	}
	if !resp.Rate.Reset.IsZero() {
		if d := time.Until(resp.Rate.Reset.Time); d > 0 {
			return d, true
		}
	}
	return 0, false
}

func parseRetryAfterHeader(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(v); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	t, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	d := time.Until(t)
	if d < 0 {
		return 0, false
	}
	return d, true
}
