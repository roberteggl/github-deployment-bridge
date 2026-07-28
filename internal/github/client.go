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
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v89/github"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
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
	client        *github.Client // non-nil when the explicit installation override is used
	appsClient    *github.Client
	appsTransport *ghinstallation.AppsTransport
	baseURL       string
	cache         cache.InstallationStore
	cacheTTL      time.Duration
	log           *slog.Logger
	clients       map[int64]*github.Client
	clientsMu     sync.Mutex
	metrics       *metrics.Metrics
	retry         retry.Config
}

// Options configures a new AppClient.
type Options struct {
	AppID                int64
	InstallationID       int64
	PrivateKeyPath       string
	BaseURL              string
	Metrics              *metrics.Metrics
	Retry                retry.Config
	Transport            http.RoundTripper
	InstallationCache    cache.InstallationStore
	InstallationCacheTTL time.Duration
	Log                  *slog.Logger
}

// NewAppClient builds a GitHub App installation client.
func NewAppClient(opts Options) (*AppClient, error) {
	base := opts.Transport
	if base == nil {
		base = http.DefaultTransport
	}

	atr, err := ghinstallation.NewAppsTransportKeyFromFile(base, opts.AppID, opts.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App transport: %w", err)
	}
	if opts.InstallationID == 0 && opts.InstallationCache == nil {
		return nil, fmt.Errorf("installation cache is required when GITHUB_INSTALLATION_ID is unset")
	}
	baseURL := enterpriseBaseURL(opts.BaseURL)
	atr.BaseURL = strings.TrimRight(baseURL, "/")
	appsClient, err := newGitHubClient(atr, baseURL)
	if err != nil {
		return nil, err
	}
	var client *github.Client
	if opts.InstallationID > 0 {
		client, err = newGitHubClient(ghinstallation.NewFromAppsTransport(atr, opts.InstallationID), baseURL)
		if err != nil {
			return nil, err
		}
	}

	cfg := opts.Retry
	if cfg.MaxAttempts == 0 {
		cfg = retry.Default()
	}

	ttl := opts.InstallationCacheTTL
	if ttl == 0 {
		ttl = time.Hour
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	c := &AppClient{
		client:     client,
		appsClient: appsClient, appsTransport: atr, baseURL: baseURL,
		cache: opts.InstallationCache, cacheTTL: ttl, log: log, clients: make(map[int64]*github.Client),
		metrics: opts.Metrics,
		retry:   cfg,
	}
	if client != nil {
		log.Info("using explicit GitHub App installation", "installation_id", opts.InstallationID)
	}
	return c, nil
}

func enterpriseBaseURL(raw string) string {
	if raw == "" {
		return "https://api.github.com"
	}
	base := strings.TrimRight(raw, "/")
	return strings.TrimSuffix(base, "/api/v3") + "/api/v3"
}

func newGitHubClient(transport http.RoundTripper, baseURL string) (*github.Client, error) {
	options := []github.ClientOptionsFunc{github.WithHTTPClient(&http.Client{Transport: transport, Timeout: 30 * time.Second})}
	if baseURL != "https://api.github.com" {
		root := strings.TrimSuffix(baseURL, "/api/v3")
		options = append(options, github.WithEnterpriseURLs(root+"/", root+"/"))
	}
	client, err := github.NewClient(options...)
	if err != nil {
		return nil, fmt.Errorf("create GitHub client: %w", err)
	}
	return client, nil
}

func (c *AppClient) clientForOwner(ctx context.Context, owner string, force bool) (*github.Client, error) {
	if c.client != nil {
		c.metricResolution("fallback")
		return c.client, nil
	}
	if !force {
		entry, err := c.cache.GetInstallation(ctx, owner)
		if err != nil {
			c.metricResolution("failure")
			return nil, err
		}
		if entry != nil && time.Since(entry.ResolvedAt) < c.cacheTTL {
			c.metricResolution("cache_hit")
			c.log.Debug("GitHub App installation cache hit", "owner", owner, "installation_id", entry.InstallationID)
			return c.installationClient(entry.InstallationID)
		}
	}
	start := time.Now()
	installations, resp, err := c.appsClient.Apps.ListInstallations(ctx, &github.ListOptions{PerPage: 100})
	c.observe("list_installations", start, err, resp)
	if err != nil {
		c.metricResolution("failure")
		return nil, fmt.Errorf("list GitHub App installations: %w", err)
	}
	for {
		for _, installation := range installations {
			if strings.EqualFold(installation.GetAccount().GetLogin(), owner) {
				entry := cache.InstallationEntry{Owner: owner, InstallationID: installation.GetID(), ResolvedAt: time.Now().UTC()}
				if err := c.cache.PutInstallation(ctx, entry); err != nil {
					c.metricResolution("failure")
					return nil, err
				}
				c.metricResolution("resolved")
				c.log.Info("resolved GitHub App installation", "owner", owner, "installation_id", entry.InstallationID)
				return c.installationClient(entry.InstallationID)
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		start = time.Now()
		installations, resp, err = c.appsClient.Apps.ListInstallations(ctx, &github.ListOptions{Page: resp.NextPage, PerPage: 100})
		c.observe("list_installations", start, err, resp)
		if err != nil {
			c.metricResolution("failure")
			return nil, fmt.Errorf("list GitHub App installations: %w", err)
		}
	}
	c.metricResolution("failure")
	return nil, retry.Permanent(fmt.Errorf("no GitHub App installation found for repository owner %q", owner))
}

func (c *AppClient) installationClient(id int64) (*github.Client, error) {
	c.clientsMu.Lock()
	defer c.clientsMu.Unlock()
	if client := c.clients[id]; client != nil {
		return client, nil
	}
	client, err := newGitHubClient(ghinstallation.NewFromAppsTransport(c.appsTransport, id), c.baseURL)
	if err == nil {
		c.clients[id] = client
	}
	return client, err
}

func (c *AppClient) invalidate(ctx context.Context, owner string) {
	if c.client != nil {
		return
	}
	if err := c.cache.DeleteInstallation(ctx, owner); err != nil {
		c.metricResolution("failure")
		c.log.Warn("failed to invalidate GitHub App installation cache", "owner", owner, "error", err)
	}
	c.log.Warn("invalidated GitHub App installation cache", "owner", owner)
}

func (c *AppClient) metricResolution(result string) {
	if c.metrics != nil {
		c.metrics.GitHubInstallationResolutionsTotal.WithLabelValues(result).Inc()
	}
}

// CreateDeployment creates a GitHub Deployment.
func (c *AppClient) CreateDeployment(ctx context.Context, req DeploymentRequest) (*DeploymentResult, error) {
	var result *DeploymentResult
	err := retry.Do(ctx, c.retry, func(ctx context.Context) error {
		client, err := c.clientForOwner(ctx, req.Owner, false)
		if err != nil {
			return err
		}
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

		dep, resp, err := client.Repositories.CreateDeployment(ctx, req.Owner, req.Repo, ghReq)
		if err != nil && installationFailure(resp) && c.client == nil {
			c.invalidate(ctx, req.Owner)
			client, resolveErr := c.clientForOwner(ctx, req.Owner, true)
			if resolveErr != nil {
				return resolveErr
			}
			dep, resp, err = client.Repositories.CreateDeployment(ctx, req.Owner, req.Repo, ghReq)
		}
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
		client, err := c.clientForOwner(ctx, req.Owner, false)
		if err != nil {
			return err
		}
		start := time.Now()
		opts := &github.DeploymentsListOptions{
			Environment: req.Environment,
			Ref:         req.Ref,
			ListOptions: github.ListOptions{PerPage: 100},
		}
		deployments, resp, err := client.Repositories.ListDeployments(ctx, req.Owner, req.Repo, opts)
		if err != nil && installationFailure(resp) && c.client == nil {
			c.invalidate(ctx, req.Owner)
			client, resolveErr := c.clientForOwner(ctx, req.Owner, true)
			if resolveErr != nil {
				return resolveErr
			}
			deployments, resp, err = client.Repositories.ListDeployments(ctx, req.Owner, req.Repo, opts)
		}
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
		client, err := c.clientForOwner(ctx, req.Owner, false)
		if err != nil {
			return err
		}
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

		_, resp, err := client.Repositories.CreateDeploymentStatus(ctx, req.Owner, req.Repo, req.DeploymentID, ghReq)
		if err != nil && installationFailure(resp) && c.client == nil {
			c.invalidate(ctx, req.Owner)
			client, resolveErr := c.clientForOwner(ctx, req.Owner, true)
			if resolveErr != nil {
				return resolveErr
			}
			_, resp, err = client.Repositories.CreateDeploymentStatus(ctx, req.Owner, req.Repo, req.DeploymentID, ghReq)
		}
		c.observe("create_deployment_status", start, err, resp)
		if err != nil {
			return classifyGitHubError(err, resp)
		}
		return nil
	})
}

func installationFailure(resp *github.Response) bool {
	return resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound)
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
