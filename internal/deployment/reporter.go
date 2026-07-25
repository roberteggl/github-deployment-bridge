// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package deployment reports Flux reconciliations to GitHub Deployments.
package deployment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
	"github.com/roberteggl/github-deployment-bridge/internal/config"
	gh "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/internal/metrics"
	"github.com/roberteggl/github-deployment-bridge/internal/registry"
	"github.com/roberteggl/github-deployment-bridge/pkg/metadata"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

// WorkloadImage is a container image discovered from a reconciled workload.
type WorkloadImage struct {
	Namespace   string
	Kind        string
	Name        string
	Image       string
	Annotations map[string]string
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
// Permanent errors (e.g. missing metadata) are logged and skipped.
// Transient errors are returned so controller-runtime can retry.
func (r *Reporter) Report(ctx context.Context, in ReportInput) error {
	seen := make(map[string]struct{})
	var transient error
	for _, img := range in.Images {
		image := strings.TrimSpace(img.Image)
		if image == "" {
			continue
		}
		// Deduplicate per workload+image so distinct deployment-name annotations are preserved.
		dedupeKey := img.Kind + "/" + img.Namespace + "/" + img.Name + "/" + image
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}

		if err := r.reportImage(ctx, in, img); err != nil {
			var skip *metadata.SkipError
			if errors.As(err, &skip) {
				r.log.Info("skipping workload",
					"reason", skip.Reason,
					"cluster", r.cfg.ClusterName,
					"namespace", in.Namespace,
					"kustomization", in.Kustomization,
					"workload_kind", img.Kind,
					"workload_name", img.Name,
					"image", image,
				)
				continue
			}
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

	// Opt-out before touching the registry.
	if raw, ok := img.Annotations[metadata.AnnotationAutoReport]; ok {
		v, parsed, err := metadata.ParseBoolAnnotation(raw)
		if err != nil {
			return retry.Permanent(fmt.Errorf("%s: %w", metadata.AnnotationAutoReport, err))
		}
		if parsed && !v {
			return &metadata.SkipError{Reason: "auto-report=false"}
		}
	}

	ociMeta, err := r.registry.Inspect(ctx, img.Image)
	if err != nil {
		// When annotations supply repository+commit, OCI inspection is optional.
		if hasAnnotationOverrides(img.Annotations) && retry.IsPermanent(err) {
			r.log.Warn("OCI inspect failed; using annotation overrides",
				"error", err,
				"image", img.Image,
				"workload_kind", img.Kind,
				"workload_name", img.Name,
			)
			ociMeta = ocilabels.Metadata{}
		} else {
			return fmt.Errorf("inspect image: %w", err)
		}
	}

	resolved, err := metadata.Resolve(img.Annotations, ociMeta, metadata.Defaults{
		Environment:    r.cfg.Environment,
		EnvironmentURL: r.cfg.EnvironmentURL,
		Description:    "Deployed by FluxCD",
	})
	if err != nil {
		var skip *metadata.SkipError
		if errors.As(err, &skip) {
			return skip
		}
		return retry.Permanent(err)
	}

	// Apply controller log URL template when the annotation is absent.
	if strings.TrimSpace(img.Annotations[metadata.AnnotationLogURL]) == "" {
		if logURL := r.cfg.ExpandLogURL(resolved.Commit); logURL != "" {
			if !metadata.ValidHTTPSURL(logURL) {
				return retry.Permanent(fmt.Errorf("LOG_URL_TEMPLATE expanded to invalid HTTPS URL %q", logURL))
			}
			resolved.LogURL = logURL
		}
	}

	key := cache.Key{
		Owner:          resolved.Repo.Owner,
		Repo:           resolved.Repo.Name,
		Environment:    resolved.Environment,
		CommitSHA:      resolved.Commit,
		DeploymentName: resolved.DeploymentName,
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
			"repository", resolved.Repo.String(),
			"commit", resolved.Commit,
			"environment", resolved.Environment,
			"deployment_name", resolved.DeploymentName,
			"deployment_id", existing.DeploymentID,
		)
		return nil
	}
	if r.metrics != nil {
		r.metrics.CacheMissesTotal.Inc()
	}

	task := ""
	if _, set := img.Annotations[metadata.AnnotationDeploymentName]; set {
		task = resolved.DeploymentName
	}

	dep, err := r.github.CreateDeployment(ctx, gh.DeploymentRequest{
		Owner:                 resolved.Repo.Owner,
		Repo:                  resolved.Repo.Name,
		Ref:                   resolved.Commit,
		Environment:           resolved.Environment,
		ProductionEnvironment: resolved.Production,
		Description:           resolved.Description,
		Task:                  task,
	})
	if err != nil {
		return fmt.Errorf("create github deployment: %w", err)
	}

	if err := r.github.CreateDeploymentStatus(ctx, gh.DeploymentStatusRequest{
		Owner:          resolved.Repo.Owner,
		Repo:           resolved.Repo.Name,
		DeploymentID:   dep.ID,
		State:          "success",
		EnvironmentURL: resolved.EnvironmentURL,
		LogURL:         resolved.LogURL,
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
		"repository", resolved.Repo.String(),
		"commit", resolved.Commit,
		"version", resolved.Version,
		"title", resolved.Title,
		"created", resolved.Created,
		"environment", resolved.Environment,
		"deployment_name", resolved.DeploymentName,
		"deployment_id", dep.ID,
		"image", img.Image,
		"workload_kind", img.Kind,
		"workload_name", img.Name,
		"duration", time.Since(start).String(),
	)
	return nil
}

func hasAnnotationOverrides(ann map[string]string) bool {
	if ann == nil {
		return false
	}
	return strings.TrimSpace(ann[metadata.AnnotationRepository]) != "" &&
		strings.TrimSpace(ann[metadata.AnnotationCommit]) != ""
}
