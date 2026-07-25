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

// ReportInput is one Flux source reconciliation to report.
type ReportInput struct {
	Namespace  string
	SourceKind string // Kustomization or HelmRelease
	SourceName string
	Phase      Phase
	Reason     string
	Images     []WorkloadImage
}

// Reporter coordinates OCI inspection, deduplication, and GitHub reporting.
type Reporter struct {
	cfg               config.Config
	cache             cache.Store
	registry          registry.Inspector
	github            gh.Client
	metrics           *metrics.Metrics
	log               *slog.Logger
	controllerVersion string
}

// NewReporter constructs a Reporter.
func NewReporter(
	cfg config.Config,
	store cache.Store,
	reg registry.Inspector,
	ghClient gh.Client,
	m *metrics.Metrics,
	log *slog.Logger,
	controllerVersion string,
) *Reporter {
	if log == nil {
		log = slog.Default()
	}
	if controllerVersion == "" {
		controllerVersion = "dev"
	}
	return &Reporter{
		cfg:               cfg,
		cache:             store,
		registry:          reg,
		github:            ghClient,
		metrics:           m,
		log:               log,
		controllerVersion: controllerVersion,
	}
}

// Report processes images for a Flux source at the desired phase.
// Permanent errors (e.g. missing metadata) are logged and skipped.
// Transient errors are returned so controller-runtime can retry.
func (r *Reporter) Report(ctx context.Context, in ReportInput) error {
	if in.Phase == "" {
		in.Phase = PhaseSuccess
	}
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
				// Opt-in skips are expected for most cluster workloads - stay quiet.
				if strings.HasPrefix(skip.Reason, "auto-report") {
					r.log.Debug("skipping workload",
						"reason", skip.Reason,
						"namespace", img.Namespace,
						"workload_kind", img.Kind,
						"workload_name", img.Name,
					)
					continue
				}
				r.log.Info("skipping workload",
					"reason", skip.Reason,
					"cluster", r.cfg.ClusterName,
					"namespace", in.Namespace,
					"source_kind", in.SourceKind,
					"source_name", in.SourceName,
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
				"source_kind", in.SourceKind,
				"source_name", in.SourceName,
				"workload_kind", img.Kind,
				"workload_name", img.Name,
				"image", image,
			)
			if retry.IsPermanent(err) {
				// After a deployment exists, permanent bridge faults become error status.
				if reportErr := r.tryReportError(ctx, in, img, err); reportErr != nil {
					r.log.Warn("failed to emit error status", "error", reportErr)
				} else if r.metrics != nil {
					r.metrics.DeploymentErrorsTotal.Inc()
				}
				continue
			}
			transient = err
			continue
		}
	}
	return transient
}

func (r *Reporter) reportImage(ctx context.Context, in ReportInput, img WorkloadImage) error {
	start := time.Now()

	resolved, digest, err := r.resolveImage(ctx, img)
	if err != nil {
		return err
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

	current := Phase(cache.LatestStatus(existing))
	desired := in.Phase

	if current == desired {
		if r.metrics != nil {
			r.metrics.DeploymentDuplicatesSkippedTotal.Inc()
		}
		return nil
	}

	// Catch-up: empty cache + terminal desired → emit only the terminal status.
	steps := transitionSteps(current, desired)
	if len(steps) == 0 {
		attrs := []any{
			"previous_state", string(current),
			"new_state", string(desired),
			"repository", resolved.Repo.String(),
			"commit", resolved.Commit,
			"environment", resolved.Environment,
			"deployment_name", resolved.DeploymentName,
		}
		// Flux often re-reconciles after success and briefly looks in_progress;
		// skipping that is expected. Unexpected jumps stay at Warn.
		if benignSkippedTransition(current, desired) {
			r.log.Debug("skipping deployment state transition", attrs...)
		} else {
			r.log.Warn("illegal deployment state transition; skipping", attrs...)
		}
		if r.metrics != nil {
			r.metrics.DeploymentDuplicatesSkippedTotal.Inc()
		}
		return nil
	}

	deploymentID := int64(0)
	if existing != nil {
		deploymentID = existing.DeploymentID
	}

	prev := current
	for _, step := range steps {
		if deploymentID == 0 {
			// Persist a provisional marker before GitHub create so retries can recover
			// the deployment ID instead of creating a duplicate.
			if err := r.cache.Put(ctx, cache.Entry{
				Key:          key,
				DeploymentID: 0,
				Status:       string(prev),
				ReportedAt:   time.Now().UTC(),
			}); err != nil {
				return fmt.Errorf("cache store provisional deployment: %w", err)
			}

			id, err := r.resolveOrCreateDeployment(ctx, in, img, resolved, digest)
			if err != nil {
				return err
			}
			deploymentID = id
			if err := r.cache.Put(ctx, cache.Entry{
				Key:          key,
				DeploymentID: deploymentID,
				Status:       string(prev), // may be empty until first status lands
				ReportedAt:   time.Now().UTC(),
			}); err != nil {
				return fmt.Errorf("cache store deployment id: %w", err)
			}
		}

		if err := r.emitStatus(ctx, resolved, deploymentID, step); err != nil {
			return err
		}

		if err := r.cache.Put(ctx, cache.Entry{
			Key:          key,
			DeploymentID: deploymentID,
			Status:       string(step),
			ReportedAt:   time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("cache store: %w", err)
		}

		r.observeStatusMetric(step)
		r.logTransition(in, img, resolved, deploymentID, prev, step, in.Reason, start)
		prev = step

		if step == PhaseSuccess {
			if err := r.markPriorInactive(ctx, key, resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

func transitionSteps(from, to Phase) []Phase {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) && (from != "" || to == "") {
		// Allow multi-hop catch-up from empty or queued → terminal via intermediates.
		if from != "" && from != PhaseQueued && from != PhaseInProgress {
			return nil
		}
	}

	// Catch-up: no prior state and terminal desired → only terminal.
	if from == "" && IsTerminal(to) && to != PhaseInactive {
		return []Phase{to}
	}

	var steps []Phase
	cur := from
	// When advancing to in_progress from empty, insert queued first.
	if cur == "" && to == PhaseInProgress {
		steps = append(steps, PhaseQueued)
		cur = PhaseQueued
	}
	if cur == to {
		return steps
	}
	if !CanTransition(cur, to) {
		// Multi-hop: queued → in_progress → success/failure
		if cur == PhaseQueued && (to == PhaseSuccess || to == PhaseFailure || to == PhaseError) {
			steps = append(steps, PhaseInProgress)
			cur = PhaseInProgress
		}
		if cur == to {
			return steps
		}
		if !CanTransition(cur, to) {
			return nil
		}
	}
	steps = append(steps, to)
	return steps
}

func (r *Reporter) resolveImage(ctx context.Context, img WorkloadImage) (metadata.Resolved, string, error) {
	// Opt-in before touching the registry - missing/false skips quietly.
	raw, ok := img.Annotations[metadata.AnnotationAutoReport]
	if !ok {
		return metadata.Resolved{}, "", &metadata.SkipError{Reason: "auto-report not enabled"}
	}
	v, parsed, err := metadata.ParseBoolAnnotation(raw)
	if err != nil {
		return metadata.Resolved{}, "", retry.Permanent(fmt.Errorf("%s: %w", metadata.AnnotationAutoReport, err))
	}
	if !parsed || !v {
		return metadata.Resolved{}, "", &metadata.SkipError{Reason: "auto-report not enabled"}
	}

	ociMeta, err := r.registry.Inspect(ctx, img.Image)
	if err != nil {
		if hasAnnotationOverrides(img.Annotations) && retry.IsPermanent(err) {
			r.log.Warn("OCI inspect failed; using annotation overrides",
				"error", err,
				"image", img.Image,
				"workload_kind", img.Kind,
				"workload_name", img.Name,
			)
			ociMeta = ocilabels.Metadata{}
		} else {
			return metadata.Resolved{}, "", fmt.Errorf("inspect image: %w", err)
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
			return metadata.Resolved{}, "", skip
		}
		return metadata.Resolved{}, "", retry.Permanent(err)
	}

	// Apply controller log URL template when the annotation is absent.
	if strings.TrimSpace(img.Annotations[metadata.AnnotationLogURL]) == "" {
		if logURL := r.cfg.ExpandLogURL(resolved.Commit); logURL != "" {
			if !metadata.ValidHTTPSURL(logURL) {
				return metadata.Resolved{}, "", retry.Permanent(fmt.Errorf("LOG_URL_TEMPLATE expanded to invalid HTTPS URL %q", logURL))
			}
			resolved.LogURL = logURL
		}
	}

	return resolved, ociMeta.Digest, nil
}

func (r *Reporter) resolveOrCreateDeployment(
	ctx context.Context,
	in ReportInput,
	img WorkloadImage,
	resolved metadata.Resolved,
	digest string,
) (int64, error) {
	payload := r.buildDeploymentPayload(in, img, resolved, digest)
	found, err := r.github.FindDeployment(ctx, gh.FindDeploymentRequest{
		Owner:       resolved.Repo.Owner,
		Repo:        resolved.Repo.Name,
		Ref:         resolved.Commit,
		Environment: resolved.Environment,
		Payload:     payload,
	})
	if err != nil {
		return 0, fmt.Errorf("find github deployment: %w", err)
	}
	if found != nil && found.ID != 0 {
		return found.ID, nil
	}
	return r.createDeployment(ctx, in, img, resolved, digest, payload)
}

func (r *Reporter) buildDeploymentPayload(
	in ReportInput,
	img WorkloadImage,
	resolved metadata.Resolved,
	digest string,
) map[string]any {
	payload := map[string]any{
		"cluster":           r.cfg.ClusterName,
		"namespace":         in.Namespace,
		"deploymentName":    resolved.DeploymentName,
		"image":             img.Image,
		"controllerVersion": r.controllerVersion,
	}
	switch strings.ToLower(in.SourceKind) {
	case "helmrelease":
		payload["helmRelease"] = in.SourceName
	default:
		payload["kustomization"] = in.SourceName
	}
	if digest != "" {
		payload["digest"] = digest
	}
	if resolved.Version != "" {
		payload["version"] = resolved.Version
	}
	return payload
}

func (r *Reporter) createDeployment(
	ctx context.Context,
	in ReportInput,
	img WorkloadImage,
	resolved metadata.Resolved,
	digest string,
	payload map[string]any,
) (int64, error) {
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
		Payload:               payload,
	})
	if err != nil {
		return 0, fmt.Errorf("create github deployment: %w", err)
	}
	if r.metrics != nil {
		r.metrics.DeploymentsCreatedTotal.Inc()
	}
	return dep.ID, nil
}

func (r *Reporter) emitStatus(ctx context.Context, resolved metadata.Resolved, deploymentID int64, state Phase) error {
	if err := r.github.CreateDeploymentStatus(ctx, gh.DeploymentStatusRequest{
		Owner:          resolved.Repo.Owner,
		Repo:           resolved.Repo.Name,
		DeploymentID:   deploymentID,
		State:          string(state),
		EnvironmentURL: resolved.EnvironmentURL,
		LogURL:         resolved.LogURL,
		Description:    StatusDescription(state),
		AutoInactive:   true,
	}); err != nil {
		return fmt.Errorf("create github deployment status: %w", err)
	}
	if r.metrics != nil {
		r.metrics.DeploymentStatusUpdatesTotal.Inc()
	}
	return nil
}

func (r *Reporter) markPriorInactive(ctx context.Context, key cache.Key, resolved metadata.Resolved) error {
	entries, err := r.cache.ListByIdentity(ctx, cache.Identity{
		Owner:          key.Owner,
		Repo:           key.Repo,
		Environment:    key.Environment,
		DeploymentName: key.DeploymentName,
	})
	if err != nil {
		return fmt.Errorf("list prior deployments: %w", err)
	}
	for _, e := range entries {
		if e.Key.CommitSHA == key.CommitSHA {
			continue
		}
		if e.Status != cache.StatusSuccess {
			continue
		}
		if e.DeploymentID == 0 {
			continue
		}
		if err := r.emitStatus(ctx, resolved, e.DeploymentID, PhaseInactive); err != nil {
			return err
		}
		e.Status = cache.StatusInactive
		e.ReportedAt = time.Now().UTC()
		if err := r.cache.Put(ctx, e); err != nil {
			return fmt.Errorf("cache store inactive: %w", err)
		}
		if r.metrics != nil {
			r.metrics.DeploymentInactiveTotal.Inc()
		}
		r.log.Info("marked prior deployment inactive",
			"repository", resolved.Repo.String(),
			"commit", e.Key.CommitSHA,
			"deployment_id", e.DeploymentID,
			"environment", e.Key.Environment,
			"previous_state", cache.StatusSuccess,
			"new_state", cache.StatusInactive,
			"cluster", r.cfg.ClusterName,
			"reason", "superseded",
		)
	}
	return nil
}

func (r *Reporter) tryReportError(ctx context.Context, in ReportInput, img WorkloadImage, cause error) error {
	resolved, _, err := r.resolveImage(ctx, img)
	if err != nil {
		return err
	}
	key := cache.Key{
		Owner:          resolved.Repo.Owner,
		Repo:           resolved.Repo.Name,
		Environment:    resolved.Environment,
		CommitSHA:      resolved.Commit,
		DeploymentName: resolved.DeploymentName,
	}
	existing, err := r.cache.Get(ctx, key)
	if err != nil || existing == nil || existing.DeploymentID == 0 {
		return cause
	}
	current := Phase(existing.Status)
	if current == PhaseError || !CanTransition(current, PhaseError) {
		return nil
	}
	if err := r.emitStatus(ctx, resolved, existing.DeploymentID, PhaseError); err != nil {
		return err
	}
	existing.Status = cache.StatusError
	existing.ReportedAt = time.Now().UTC()
	return r.cache.Put(ctx, *existing)
}

func (r *Reporter) observeStatusMetric(step Phase) {
	if r.metrics == nil {
		return
	}
	switch step {
	case PhaseFailure:
		r.metrics.DeploymentFailuresTotal.Inc()
	case PhaseError:
		r.metrics.DeploymentErrorsTotal.Inc()
	}
}

func (r *Reporter) logTransition(
	in ReportInput,
	img WorkloadImage,
	resolved metadata.Resolved,
	deploymentID int64,
	prev, next Phase,
	reason string,
	start time.Time,
) {
	r.log.Info("deployment state transition",
		"repository", resolved.Repo.String(),
		"commit", resolved.Commit,
		"deployment_id", deploymentID,
		"environment", resolved.Environment,
		"previous_state", string(prev),
		"new_state", string(next),
		"cluster", r.cfg.ClusterName,
		"namespace", in.Namespace,
		"source_kind", in.SourceKind,
		"source_name", in.SourceName,
		"deployment_name", resolved.DeploymentName,
		"duration", time.Since(start).String(),
		"reason", reason,
		"image", img.Image,
		"workload_kind", img.Kind,
		"workload_name", img.Name,
	)
}

func hasAnnotationOverrides(ann map[string]string) bool {
	if ann == nil {
		return false
	}
	return strings.TrimSpace(ann[metadata.AnnotationRepository]) != "" &&
		strings.TrimSpace(ann[metadata.AnnotationCommit]) != ""
}
