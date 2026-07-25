// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package metadata_test

import (
	"errors"
	"testing"

	"github.com/roberteggl/github-deployment-bridge/pkg/metadata"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

func TestResolvePriorityAndDefaults(t *testing.T) {
	t.Parallel()

	oci := ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "abcdef0123456789",
		Version:  "v2.5.1",
		Title:    "backend",
		Created:  "2026-07-25T12:00:00Z",
	}
	ann := map[string]string{
		metadata.AnnotationAutoReport:  "true",
		metadata.AnnotationEnvironment: "production",
	}
	got, err := metadata.Resolve(ann, oci, metadata.Defaults{
		Cluster:        "staging-cluster",
		Environment:    "staging",
		EnvironmentURL: "https://staging.example.com",
		LogURL:         "https://grafana.example.com/?sha=abcdef0123456789",
		Description:    "Deployed by FluxCD",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Repo.String() != "example/backend" {
		t.Fatalf("repo = %s", got.Repo)
	}
	if got.Commit != "abcdef0123456789" {
		t.Fatalf("commit = %s", got.Commit)
	}
	if got.Environment != "production" {
		t.Fatalf("environment = %s, want production (annotation)", got.Environment)
	}
	if !got.Production {
		t.Fatal("expected production=true from environment name")
	}
	if got.Version != "v2.5.1" || got.Title != "backend" {
		t.Fatalf("optional OCI fields: version=%s title=%s", got.Version, got.Title)
	}
	if got.DeploymentName != "backend" {
		t.Fatalf("deployment-name default = %s, want backend", got.DeploymentName)
	}
	if got.EnvironmentURL != "https://staging.example.com" {
		t.Fatalf("environment-url = %s", got.EnvironmentURL)
	}
	if got.Cluster != "staging-cluster" {
		t.Fatalf("cluster = %s, want staging-cluster", got.Cluster)
	}
}

func TestResolveAnnotationOverrides(t *testing.T) {
	t.Parallel()

	oci := ocilabels.Metadata{
		Source:   "https://github.com/example/shared-image",
		Revision: "aaaaaaaaaaaaaaaa",
	}
	ann := map[string]string{
		metadata.AnnotationAutoReport:     "true",
		metadata.AnnotationRepository:     "example/backend",
		metadata.AnnotationCommit:         "bbbbbbbbbbbbbbbb",
		metadata.AnnotationEnvironment:    "preview",
		metadata.AnnotationEnvironmentURL: "https://preview.example.com",
		metadata.AnnotationLogURL:         "https://grafana.example.com/d/x",
		metadata.AnnotationDescription:    "Customer Portal",
		metadata.AnnotationProduction:     "false",
		metadata.AnnotationDeploymentName: "api",
	}
	got, err := metadata.Resolve(ann, oci, metadata.Defaults{
		Cluster:     "staging-cluster",
		Environment: "staging",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Repo.String() != "example/backend" {
		t.Fatalf("repo = %s", got.Repo)
	}
	if got.Commit != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("commit = %s", got.Commit)
	}
	if got.Environment != "preview" || got.Production {
		t.Fatalf("env/prod = %s/%v", got.Environment, got.Production)
	}
	if got.EnvironmentURL != "https://preview.example.com" || got.LogURL != "https://grafana.example.com/d/x" {
		t.Fatalf("urls = %s / %s", got.EnvironmentURL, got.LogURL)
	}
	if got.Description != "Customer Portal" {
		t.Fatalf("description = %s", got.Description)
	}
	if got.DeploymentName != "api" {
		t.Fatalf("deployment-name = %s", got.DeploymentName)
	}
}

func TestResolveAutoReportRequiresOptIn(t *testing.T) {
	t.Parallel()

	_, err := metadata.Resolve(nil, ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "abcdef0123456789",
	}, metadata.Defaults{
		Cluster:     "staging-cluster",
		Environment: "production",
	})
	var skip *metadata.SkipError
	if !errors.As(err, &skip) {
		t.Fatalf("expected SkipError for missing auto-report, got %v", err)
	}
}

func TestResolveAutoReportFalse(t *testing.T) {
	t.Parallel()

	_, err := metadata.Resolve(map[string]string{
		metadata.AnnotationAutoReport: "false",
	}, ocilabels.Metadata{}, metadata.Defaults{Environment: "production"})
	var skip *metadata.SkipError
	if !errors.As(err, &skip) {
		t.Fatalf("expected SkipError, got %v", err)
	}
}

func TestResolveMissingRequired(t *testing.T) {
	t.Parallel()

	_, err := metadata.Resolve(map[string]string{
		metadata.AnnotationAutoReport: "true",
	}, ocilabels.Metadata{
		Source: "https://github.com/example/backend",
	}, metadata.Defaults{
		Cluster:     "staging-cluster",
		Environment: "production",
	})
	if err == nil {
		t.Fatal("expected error for missing commit")
	}
	var skip *metadata.SkipError
	if errors.As(err, &skip) {
		t.Fatalf("expected validation error, got SkipError: %v", err)
	}
}

func TestResolveInvalidCommit(t *testing.T) {
	t.Parallel()

	_, err := metadata.Resolve(map[string]string{
		metadata.AnnotationAutoReport: "true",
	}, ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "not-a-sha!",
	}, metadata.Defaults{
		Cluster:     "staging-cluster",
		Environment: "production",
	})
	if err == nil {
		t.Fatal("expected invalid SHA error")
	}
}

func TestResolveInvalidHTTPSURL(t *testing.T) {
	t.Parallel()

	_, err := metadata.Resolve(map[string]string{
		metadata.AnnotationAutoReport:     "true",
		metadata.AnnotationEnvironmentURL: "http://insecure.example.com",
	}, ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "abcdef0123456789",
	}, metadata.Defaults{
		Cluster:     "staging-cluster",
		Environment: "production",
	})
	if err == nil {
		t.Fatal("expected HTTPS validation error")
	}
}

func TestValidGitSHA(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"abcdef0":          true,
		"0123456789abcdef": true,
		"ABCDEF0123456789": true,
		"abc":              false,
		"zzzzzzz":          false,
		"":                 false,
		"0123456789abcdef0123456789abcdef01234567":  true,  // 40
		"0123456789abcdef0123456789abcdef012345678": false, // 41
	}
	for in, want := range cases {
		if got := metadata.ValidGitSHA(in); got != want {
			t.Fatalf("ValidGitSHA(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveOptionalPayloadAnnotations(t *testing.T) {
	t.Parallel()

	ann := map[string]string{
		metadata.AnnotationAutoReport:   "true",
		metadata.AnnotationCluster:      "prod-eu-west",
		metadata.AnnotationTeam:         "platform",
		metadata.AnnotationService:      "checkout",
		metadata.AnnotationComponent:    "api",
		metadata.AnnotationSlackChannel: "#deploys",
		metadata.AnnotationOwner:        "alice",
		metadata.AnnotationRelease:      "2026.07.25",
		metadata.AnnotationTag:          "v1.2.3",
	}
	got, err := metadata.Resolve(ann, ocilabels.Metadata{
		Source:   "https://github.com/example/backend",
		Revision: "abcdef0123456789",
	}, metadata.Defaults{
		Cluster:     "fallback-cluster",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Cluster != "prod-eu-west" {
		t.Fatalf("cluster = %q, want annotation override", got.Cluster)
	}
	if got.Team != "platform" || got.Service != "checkout" || got.Component != "api" {
		t.Fatalf("team/service/component = %q/%q/%q", got.Team, got.Service, got.Component)
	}
	if got.SlackChannel != "#deploys" || got.Owner != "alice" || got.Release != "2026.07.25" || got.Tag != "v1.2.3" {
		t.Fatalf("slack/owner/release/tag = %q/%q/%q/%q", got.SlackChannel, got.Owner, got.Release, got.Tag)
	}

	payload := map[string]any{"cluster": got.Cluster}
	metadata.ApplyPayloadExtras(payload, got)
	for _, key := range []string{"team", "service", "component", "slackChannel", "owner", "release", "tag"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload missing %q: %#v", key, payload)
		}
	}
}

func TestIsOptionalPayloadAnnotation(t *testing.T) {
	t.Parallel()
	if !metadata.IsOptionalPayloadAnnotation(metadata.AnnotationTeam) {
		t.Fatal("team should be optional payload annotation")
	}
	if metadata.IsOptionalPayloadAnnotation(metadata.AnnotationEnvironment) {
		t.Fatal("environment should not be optional payload annotation")
	}
}
