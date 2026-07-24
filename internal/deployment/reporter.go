// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package deployment reports Flux reconciliations to GitHub Deployments.
package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
	"github.com/roberteggl/github-deployment-bridge/internal/config"
	gh "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/internal/metrics"
	"github.com/roberteggl/github-deployment-bridge/internal/registry"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

// WorkloadImage is a container image discovered from a reconciled workload.
type WorkloadImage struct {
	Namespace string
	Kind      string
	Name      string
	Image     string
}

// ReportInput is one Flux Kustomization reconciliation to report.
type ReportInput struct {
	Namespace     string
	Kustomization string
	Images        []WorkloadImage
}

// Reporter coordinates OCI inspection, deduplication, and GitHub reporting.
type Reporter struct {
	cfg      config.Config
	cache    cache.Store
	registry registry.Inspector
	github   gh.Client
	metrics  *metrics.Metrics
	log      *slog.Logger
}

// NewReporter constructs a Reporter.
func NewReporter(
	cfg config.Config,
	store cache.Store,
	reg registry.Inspector,
	ghClient gh.Client,
	m *metrics.Metrics,
	log *slog.Logger,
) *Reporter {
	if log == nil {
		log = slog.Default()
	}
	return &Reporter{
		cfg:      cfg,
		cache:    store,
		registry: reg,
		github:   ghClient,
		metrics:  m,
		log:      log,
	}
}

// Report processes images from a ready Kustomization.
// Permanent errors (e.g. malformed OCI labels) are logged and skipped.
// Transient errors are returned so controller-runtime can retry.
func (r *Reporter) Report(ctx context.Context, in ReportInput) error {
	seen := make(map[string]struct{})
	var transient error
	for _, img := range in.Images {
		image := strings.TrimSpace(img.Image)
		if image == "" {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}

		if err := r.reportImage(ctx, in, img); err != nil {
			r.log.Warn("failed to report deployment for image",
				"error", err,
				"cluster", r.cfg.ClusterName,
				"namespace", in.Namespace,
				"kustomization", in.Kustomization,
				"workload_kind", img.Kind,
				"workload_name", img.Name,
				"image", image,
			)
			if r.metrics != nil {
				r.metrics.DeploymentFailuresTotal.Inc()
			}
			if retry.IsPermanent(err) {
				continue
			}
			// Keep processing remaining images, but surface a retryable error.
			transient = err
			continue
		}
	}
	return transient
}

func (r *Reporter) reportImage(ctx context.Context, in ReportInput, img WorkloadImage) error {
	start := time.Now()

	meta, err := r.registry.Inspect(ctx, img.Image)
	if err != nil {
		return fmt.Errorf("inspect image: %w", err)
	}

	key := cache.Key{
		Owner:       meta.Repo.Owner,
		Repo:        meta.Repo.Name,
		Environment: r.cfg.Environment,
		CommitSHA:   meta.Revision,
	}

	existing, err := r.cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("cache lookup: %w", err)
	}
	if cache.AlreadyReported(existing) {
		if r.metrics != nil {
			r.metrics.CacheHitsTotal.Inc()
		}
		r.log.Info("skipping already reported deployment",
			"cluster", r.cfg.ClusterName,
			"namespace", in.Namespace,
			"kustomization", in.Kustomization,
			"repository", meta.Repo.String(),
			"commit", meta.Revision,
			"environment", r.cfg.Environment,
			"deployment_id", existing.DeploymentID,
		)
		return nil
	}
	if r.metrics != nil {
		r.metrics.CacheMissesTotal.Inc()
	}

	dep, err := r.github.CreateDeployment(ctx, gh.DeploymentRequest{
		Owner:                 meta.Repo.Owner,
		Repo:                  meta.Repo.Name,
		Ref:                   meta.Revision,
		Environment:           r.cfg.Environment,
		ProductionEnvironment: r.cfg.IsProduction(),
		Description:           "Deployed by FluxCD",
	})
	if err != nil {
		return fmt.Errorf("create github deployment: %w", err)
	}

	if err := r.github.CreateDeploymentStatus(ctx, gh.DeploymentStatusRequest{
		Owner:          meta.Repo.Owner,
		Repo:           meta.Repo.Name,
		DeploymentID:   dep.ID,
		State:          "success",
		EnvironmentURL: r.cfg.EnvironmentURL,
		LogURL:         r.cfg.ExpandLogURL(meta.Revision),
		Description:    "Flux reconciliation succeeded",
	}); err != nil {
		return fmt.Errorf("create github deployment status: %w", err)
	}

	if err := r.cache.Put(ctx, cache.Entry{
		Key:          key,
		DeploymentID: dep.ID,
		Status:       cache.StatusSuccess,
		ReportedAt:   time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("cache store: %w", err)
	}

	if r.metrics != nil {
		r.metrics.DeploymentReportsTotal.Inc()
	}

	r.log.Info("reported github deployment",
		"cluster", r.cfg.ClusterName,
		"namespace", in.Namespace,
		"kustomization", in.Kustomization,
		"repository", meta.Repo.String(),
		"commit", meta.Revision,
		"version", meta.Version,
		"environment", r.cfg.Environment,
		"deployment_id", dep.ID,
		"image", img.Image,
		"workload_kind", img.Kind,
		"workload_name", img.Name,
		"duration", time.Since(start).String(),
	)
	return nil
}
