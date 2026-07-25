// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package deployment_test

import (
	"context"
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
}

func (f *fakeGitHub) CreateDeployment(_ context.Context, req gh.DeploymentRequest) (*gh.DeploymentResult, error) {
	f.deployments = append(f.deployments, req)
	f.nextID++
	return &gh.DeploymentResult{ID: f.nextID}, nil
}

func (f *fakeGitHub) CreateDeploymentStatus(_ context.Context, req gh.DeploymentStatusRequest) error {
	f.statuses = append(f.statuses, req)
	return nil
}

func TestReporterCreatesDeploymentAndSkipsDuplicates(t *testing.T) {
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
		LogURLTemplate: "https://grafana.example.com/explore?commit={sha}",
	}

	reg := &fakeRegistry{meta: ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "deadbeefcafebabe",
		Version:  "v1.2.3",
	}}
	g := &fakeGitHub{}
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default())

	in := deployment.ReportInput{
		Namespace:     "apps",
		Kustomization: "backend",
		Images: []deployment.WorkloadImage{{
			Namespace: "apps",
			Kind:      "Deployment",
			Name:      "backend",
			Image:     "ghcr.io/example/backend:v1.2.3",
		}},
	}

	if err := r.Report(context.Background(), in); err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 {
		t.Fatalf("got %d deployments and %d statuses, want 1/1", len(g.deployments), len(g.statuses))
	}
	if g.deployments[0].Ref != "deadbeefcafebabe" {
		t.Fatalf("ref = %q, want deadbeefcafebabe", g.deployments[0].Ref)
	}
	if !g.deployments[0].ProductionEnvironment {
		t.Fatal("expected production_environment=true")
	}
	if g.deployments[0].Environment != "production" {
		t.Fatalf("environment = %q", g.deployments[0].Environment)
	}
	if g.statuses[0].State != "success" {
		t.Fatalf("status state = %q", g.statuses[0].State)
	}
	if g.statuses[0].LogURL != "https://grafana.example.com/explore?commit=deadbeefcafebabe" {
		t.Fatalf("log url = %q", g.statuses[0].LogURL)
	}

	// Second report should hit cache and skip GitHub calls.
	if err := r.Report(context.Background(), in); err != nil {
		t.Fatalf("second report: %v", err)
	}
	if len(g.deployments) != 1 || len(g.statuses) != 1 {
		t.Fatalf("duplicate not prevented: got %d deployments and %d statuses", len(g.deployments), len(g.statuses))
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
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default())

	in := deployment.ReportInput{
		Namespace:     "apps",
		Kustomization: "apps",
		Images: []deployment.WorkloadImage{
			{
				Namespace: "apps",
				Kind:      "Deployment",
				Name:      "api",
				Image:     "ghcr.io/example/shared:1",
				Annotations: map[string]string{
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
	if dep.Environment != "production" || !dep.ProductionEnvironment {
		t.Fatalf("env/prod = %s/%v", dep.Environment, dep.ProductionEnvironment)
	}
	if dep.Description != "API service" {
		t.Fatalf("description = %q", dep.Description)
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
	r := deployment.NewReporter(cfg, store, reg, g, nil, slog.Default())

	err = r.Report(context.Background(), deployment.ReportInput{
		Namespace:     "apps",
		Kustomization: "backend",
		Images: []deployment.WorkloadImage{{
			Namespace: "apps",
			Kind:      "Deployment",
			Name:      "backend",
			Image:     "ghcr.io/example/backend:1",
		}},
	})
	if err != nil {
		t.Fatalf("invalid metadata should not fail reconcile: %v", err)
	}
	if len(g.deployments) != 0 {
		t.Fatalf("expected no deployments, got %d", len(g.deployments))
	}
}
