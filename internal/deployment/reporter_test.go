// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package deployment_test

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
	"github.com/roberteggl/github-deployment-bridge/internal/config"
	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
	gh "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/pkg/metadata"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

type fakeRegistry struct {
	meta ocilabels.Metadata
	err  error
}

func (f *fakeRegistry) Inspect(context.Context, string) (ocilabels.Metadata, error) {
	return f.meta, f.err
}

type fakeGitHub struct {
	deployments []gh.DeploymentRequest
	statuses    []gh.DeploymentStatusRequest
	nextID      int64
	statusErr   error
	listedDeployments []struct {
		ID          int64
		Ref         string
		Environment string
		Payload     map[string]any
	}
}

func (f *fakeGitHub) CreateDeployment(_ context.Context, req gh.DeploymentRequest) (*gh.DeploymentResult, error) {
	f.deployments = append(f.deployments, req)
	f.nextID++
	return &gh.DeploymentResult{ID: f.nextID}, nil
}

func (f *fakeGitHub) FindDeployment(_ context.Context, req gh.FindDeploymentRequest) (*gh.DeploymentResult, error) {
	for _, dep := range f.listedDeployments {
		if dep.Ref != req.Ref || dep.Environment != req.Environment {
			continue
		}
		if !payloadMatches(dep.Payload, req.Payload) {
			continue
		}
		return &gh.DeploymentResult{ID: dep.ID}, nil
	}
	return nil, nil
}

func payloadMatches(actual, expected map[string]any) bool {
	for key, want := range expected {
		got, ok := actual[key]
		if !ok || fmt.Sprint(got) != fmt.Sprint(want) {
			return false
		}
	}
	return true
}

func (f *fakeGitHub) CreateDeploymentStatus(_ context.Context, req gh.DeploymentStatusRequest) error {
	if f.statusErr != nil {
		return f.statusErr
	}
	f.statuses = append(f.statuses, req)
	return nil
}

func newTestReporter(t *testing.T, g *fakeGitHub) (*deployment.Reporter, *fakeGitHub) {
	t.Helper()
	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{
		ClusterName:    "production-eu",
		Environment:    "production",
		EnvironmentURL: "https://app.example.com",
		LogURLTemplate: "https://grafana.example.com/explore?commit={sha}",
	}
	reg := &fakeRegistry{meta: ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "deadbeefcafebabe",
		Version:  "v1.2.3",
		Digest:   "sha256:abc",
	}}
	if g == nil {
		g = &fakeGitHub{}
	}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "test")
	return r, g
}

func sampleInput(phase deployment.Phase) deployment.ReportInput {
	return deployment.ReportInput{
		Namespace:  "apps",
		SourceKind: "Kustomization",
		SourceName: "backend",
		Phase:      phase,
		Reason:     "test",
		Images: []deployment.WorkloadImage{{
			Namespace: "apps",
			Kind:      "Deployment",
			Name:      "backend",
			Image:     "ghcr.io/example/backend:v1.2.3",
			Annotations: map[string]string{
				metadata.AnnotationAutoReport: "true",
			},
		}},
	}
}

func TestReporterCatchUpSuccessAndSkipsDuplicates(t *testing.T) {
	t.Parallel()
	r, g := newTestReporter(t, nil)

	in := sampleInput(deployment.PhaseSuccess)
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 {
		t.Fatalf("got %d deployments and %d statuses, want 1/1", len(g.deployments), len(g.statuses))
	}
	if g.statuses[0].State != "success" {
		t.Fatalf("status state = %q", g.statuses[0].State)
	}
	if !g.statuses[0].AutoInactive {
		t.Fatal("expected auto_inactive=true")
	}
	if g.deployments[0].Payload["cluster"] != "production-eu" {
		t.Fatalf("payload cluster = %#v", g.deployments[0].Payload["cluster"])
	}
	if g.deployments[0].Payload["kustomization"] != "backend" {
		t.Fatalf("payload kustomization = %#v", g.deployments[0].Payload["kustomization"])
	}

	if err := r.Report(context.Background(), in); err != nil {
		t.Fatalf("second report: %v", err)
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 {
		t.Fatalf("duplicate not prevented: got %d deployments and %d statuses", len(g.deployments), len(g.statuses))
	}
}


func TestReporterRecoversDeploymentAfterCreateCrash(t *testing.T) {
	t.Parallel()

	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{
		ClusterName:    "production-eu",
		Environment:    "production",
		EnvironmentURL: "https://app.example.com",
	}
	reg := &fakeRegistry{meta: ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "deadbeefcafebabe",
		Version:  "v1.2.3",
		Digest:   "sha256:abc",
	}}
	g := &fakeGitHub{
		listedDeployments: []struct {
			ID          int64
			Ref         string
			Environment string
			Payload     map[string]any
		}{{
			ID:          42,
			Ref:         "deadbeefcafebabe",
			Environment: "production",
			Payload: map[string]any{
				"cluster":           "production-eu",
				"namespace":         "apps",
				"deploymentName":    "backend",
				"image":             "ghcr.io/example/backend:v1.2.3",
				"controllerVersion": "test",
				"kustomization":     "backend",
				"digest":            "sha256:abc",
				"version":           "v1.2.3",
			},
		}},
	}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "test")

	key := cache.Key{
		Owner:          "example",
		Repo:           "backend",
		Environment:    "production",
		CommitSHA:      "deadbeefcafebabe",
		DeploymentName: "backend",
	}
	if err := store.Put(context.Background(), cache.Entry{
		Key:          key,
		DeploymentID: 0,
		Status:       "",
	}); err != nil {
		t.Fatalf("seed provisional cache: %v", err)
	}

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseSuccess)); err != nil {
		t.Fatalf("report after crash: %v", err)
	}
	if len(g.deployments) != 0 {
		t.Fatalf("expected recovery without create, got %d deployments", len(g.deployments))
	}
	if len(g.statuses) != 1 || g.statuses[0].DeploymentID != 42 {
		t.Fatalf("statuses = %#v, want one success for deployment 42", g.statuses)
	}
	entry, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("cache get: %v", err)
	}
	if entry == nil || entry.DeploymentID != 42 {
		t.Fatalf("cache deployment_id = %d, want 42", entry.DeploymentID)
	}
}

func TestReporterQueuedInProgressSuccess(t *testing.T) {
	t.Parallel()
	r, g := newTestReporter(t, nil)

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseInProgress)); err != nil {
		t.Fatalf("in_progress: %v", err)
	}
	if len(g.deployments) != 1 {
		t.Fatalf("deployments = %d, want 1", len(g.deployments))
	}
	if len(g.statuses) != 2 {
		t.Fatalf("statuses = %d, want 2 (queued, in_progress)", len(g.statuses))
	}
	if g.statuses[0].State != "queued" || g.statuses[1].State != "in_progress" {
		t.Fatalf("states = %q, %q", g.statuses[0].State, g.statuses[1].State)
	}

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseSuccess)); err != nil {
		t.Fatalf("success: %v", err)
	}
	if len(g.statuses) != 3 || g.statuses[2].State != "success" {
		t.Fatalf("after success statuses = %#v", statusStates(g))
	}
}

func TestReporterQueuedInProgressFailure(t *testing.T) {
	t.Parallel()
	r, g := newTestReporter(t, nil)

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseInProgress)); err != nil {
		t.Fatalf("in_progress: %v", err)
	}
	if err := r.Report(context.Background(), sampleInput(deployment.PhaseFailure)); err != nil {
		t.Fatalf("failure: %v", err)
	}
	if got := statusStates(g); len(got) != 3 || got[2] != "failure" {
		t.Fatalf("states = %#v", got)
	}
}

func TestReporterCatchUpFailure(t *testing.T) {
	t.Parallel()
	r, g := newTestReporter(t, nil)

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseFailure)); err != nil {
		t.Fatalf("failure: %v", err)
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 || g.statuses[0].State != "failure" {
		t.Fatalf("catch-up failure: deps=%d statuses=%#v", len(g.deployments), statusStates(g))
	}
}

func TestReporterNeverSuccessToInProgress(t *testing.T) {
	t.Parallel()
	r, g := newTestReporter(t, nil)

	if err := r.Report(context.Background(), sampleInput(deployment.PhaseSuccess)); err != nil {
		t.Fatalf("success: %v", err)
	}
	if err := r.Report(context.Background(), sampleInput(deployment.PhaseInProgress)); err != nil {
		t.Fatalf("in_progress after success: %v", err)
	}
	if len(g.statuses) != 1 {
		t.Fatalf("expected no additional statuses, got %#v", statusStates(g))
	}
}

func TestReporterInactiveSupersession(t *testing.T) {
	t.Parallel()

	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{ClusterName: "c", Environment: "production"}
	g := &fakeGitHub{}
	reg := &fakeRegistry{}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "test")

	oldMeta := ocilabels.Metadata{Source: "https://github.com/example/backend", Revision: "aaaaaaaaaaaaaaaa"}
	newMeta := ocilabels.Metadata{Source: "https://github.com/example/backend", Revision: "bbbbbbbbbbbbbbbb"}

	reg.meta = oldMeta
	if err := r.Report(context.Background(), sampleInput(deployment.PhaseSuccess)); err != nil {
		t.Fatalf("old success: %v", err)
	}
	reg.meta = newMeta
	if err := r.Report(context.Background(), sampleInput(deployment.PhaseSuccess)); err != nil {
		t.Fatalf("new success: %v", err)
	}

	states := statusStates(g)
	if len(g.deployments) != 2 {
		t.Fatalf("deployments = %d, want 2", len(g.deployments))
	}
	inactiveCount := 0
	for _, s := range states {
		if s == "inactive" {
			inactiveCount++
		}
	}
	if inactiveCount != 1 {
		t.Fatalf("expected 1 inactive status, got %#v", states)
	}
}

func TestReporterAnnotationOverridesAndAutoReport(t *testing.T) {
	t.Parallel()

	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{
		ClusterName: "production-eu",
		Environment: "staging",
	}
	reg := &fakeRegistry{meta: ocilabels.Metadata{
		Source:   "https://github.com/example/shared",
		Revision: "aaaaaaaaaaaaaaaa",
	}}
	g := &fakeGitHub{}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "test")

	in := deployment.ReportInput{
		Namespace:  "apps",
		SourceKind: "Kustomization",
		SourceName: "apps",
		Phase:      deployment.PhaseSuccess,
		Images: []deployment.WorkloadImage{
			{
				Namespace: "apps",
				Kind:      "Deployment",
				Name:      "api",
				Image:     "ghcr.io/example/shared:1",
				Annotations: map[string]string{
					metadata.AnnotationAutoReport:     "true",
					metadata.AnnotationRepository:     "example/backend",
					metadata.AnnotationEnvironment:    "production",
					metadata.AnnotationDeploymentName: "api",
					metadata.AnnotationDescription:    "API service",
					metadata.AnnotationProduction:     "true",
				},
			},
			{
				Namespace: "apps",
				Kind:      "Deployment",
				Name:      "worker",
				Image:     "ghcr.io/example/shared:1",
				Annotations: map[string]string{
					metadata.AnnotationAutoReport: "false",
				},
			},
		},
	}

	if err := r.Report(context.Background(), in); err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(g.deployments) != 1 {
		t.Fatalf("got %d deployments, want 1 (worker auto-report=false)", len(g.deployments))
	}
	dep := g.deployments[0]
	if dep.Owner != "example" || dep.Repo != "backend" {
		t.Fatalf("repo = %s/%s", dep.Owner, dep.Repo)
	}
	if dep.Task != "api" {
		t.Fatalf("task = %q, want api", dep.Task)
	}
}

func TestReporterSkipsInvalidMetadata(t *testing.T) {
	t.Parallel()

	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{ClusterName: "c", Environment: "staging"}
	reg := &fakeRegistry{meta: ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "bad",
	}}
	g := &fakeGitHub{}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "test")

	err = r.Report(context.Background(), sampleInput(deployment.PhaseSuccess))
	if err != nil {
		t.Fatalf("invalid metadata should not fail reconcile: %v", err)
	}
	if len(g.deployments) != 0 {
		t.Fatalf("expected no deployments, got %d", len(g.deployments))
	}
}

func TestCanTransition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from, to deployment.Phase
		want     bool
	}{
		{"", deployment.PhaseQueued, true},
		{deployment.PhaseQueued, deployment.PhaseInProgress, true},
		{deployment.PhaseInProgress, deployment.PhaseSuccess, true},
		{deployment.PhaseSuccess, deployment.PhaseInProgress, false},
		{deployment.PhaseSuccess, deployment.PhaseInactive, true},
		{deployment.PhaseFailure, deployment.PhaseSuccess, false},
		{deployment.PhaseQueued, deployment.PhaseQueued, false},
	}
	for _, tc := range cases {
		if got := deployment.CanTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func statusStates(g *fakeGitHub) []string {
	out := make([]string, len(g.statuses))
	for i, s := range g.statuses {
		out[i] = s.State
	}
	return out
}
