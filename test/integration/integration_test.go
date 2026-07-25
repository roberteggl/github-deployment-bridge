// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
	"github.com/roberteggl/github-deployment-bridge/internal/config"
	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
	gh "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

// Integration tests exercise the reporter lifecycle against fakes.
// A full kind+Flux harness can be layered on later; run with:
//
//	go test ./test/integration -tags=integration -count=1

type fakeRegistry struct {
	meta ocilabels.Metadata
}

func (f *fakeRegistry) Inspect(context.Context, string) (ocilabels.Metadata, error) {
	return f.meta, nil
}

type fakeGitHub struct {
	deployments []gh.DeploymentRequest
	statuses    []gh.DeploymentStatusRequest
	nextID      int64
}

func (f *fakeGitHub) CreateDeployment(_ context.Context, req gh.DeploymentRequest) (*gh.DeploymentResult, error) {
	f.deployments = append(f.deployments, req)
	f.nextID++
	return &gh.DeploymentResult{ID: f.nextID}, nil
}

func (f *fakeGitHub) FindDeployment(_ context.Context, _ gh.FindDeploymentRequest) (*gh.DeploymentResult, error) {
	return nil, nil
}

func (f *fakeGitHub) CreateDeploymentStatus(_ context.Context, req gh.DeploymentStatusRequest) error {
	f.statuses = append(f.statuses, req)
	return nil
}

func TestLifecycleQueuedInProgressSuccess(t *testing.T) {
	r, g := newReporter(t, "aaaaaaaaaaaaaaaa")

	in := input(deployment.PhaseInProgress)
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	in.Phase = deployment.PhaseSuccess
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	assertStates(t, g, "queued", "in_progress", "success")
}

func TestLifecycleQueuedInProgressFailure(t *testing.T) {
	r, g := newReporter(t, "bbbbbbbbbbbbbbbb")

	in := input(deployment.PhaseInProgress)
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	in.Phase = deployment.PhaseFailure
	in.Reason = "HealthCheckFailed"
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	assertStates(t, g, "queued", "in_progress", "failure")
}

func TestLifecycleCatchUpSuccess(t *testing.T) {
	r, g := newReporter(t, "cccccccccccccccc")
	if err := r.Report(context.Background(), input(deployment.PhaseSuccess)); err != nil {
		t.Fatal(err)
	}
	assertStates(t, g, "success")
}

func TestLifecycleRepeatedEventsIdempotent(t *testing.T) {
	r, g := newReporter(t, "dddddddddddddddd")
	in := input(deployment.PhaseSuccess)
	for i := 0; i < 5; i++ {
		if err := r.Report(context.Background(), in); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 {
		t.Fatalf("deps=%d statuses=%d", len(g.deployments), len(g.statuses))
	}
}

func TestLifecycleSuccessThenInactive(t *testing.T) {
	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{ClusterName: "c", Environment: "production"}
	g := &fakeGitHub{}
	reg := &fakeRegistry{}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default(), "integration")

	reg.meta = ocilabels.Metadata{Source: "https://github.com/example/app", Revision: "1111111111111111"}
	if err := r.Report(context.Background(), input(deployment.PhaseSuccess)); err != nil {
		t.Fatal(err)
	}
	reg.meta = ocilabels.Metadata{Source: "https://github.com/example/app", Revision: "2222222222222222"}
	if err := r.Report(context.Background(), input(deployment.PhaseSuccess)); err != nil {
		t.Fatal(err)
	}

	var inactive int
	for _, s := range g.statuses {
		if s.State == "inactive" {
			inactive++
		}
	}
	if inactive != 1 {
		t.Fatalf("expected 1 inactive, statuses=%v", g.statuses)
	}
}

func newReporter(t *testing.T, revision string) (*deployment.Reporter, *fakeGitHub) {
	t.Helper()
	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	g := &fakeGitHub{}
	r := deployment.NewReporter(
		config.Config{ClusterName: "c", Environment: "production"},
		store,
		&fakeRegistry{meta: ocilabels.Metadata{
			Source:   "https://github.com/example/app",
			Revision: revision,
		}},
		g,
		nil,
		slog.Default(),
		"integration",
	)
	return r, g
}

func input(phase deployment.Phase) deployment.ReportInput {
	return deployment.ReportInput{
		Namespace:  "apps",
		SourceKind: "Kustomization",
		SourceName: "app",
		Phase:      phase,
		Images: []deployment.WorkloadImage{{
			Namespace: "apps",
			Kind:      "Deployment",
			Name:      "app",
			Image:     "ghcr.io/example/app:1",
		}},
	}
}

func assertStates(t *testing.T, g *fakeGitHub, want ...string) {
	t.Helper()
	if len(g.statuses) != len(want) {
		t.Fatalf("status count = %d, want %d (%v)", len(g.statuses), len(want), want)
	}
	for i, s := range want {
		if g.statuses[i].State != s {
			t.Fatalf("status[%d] = %q, want %q", i, g.statuses[i].State, s)
		}
	}
}
