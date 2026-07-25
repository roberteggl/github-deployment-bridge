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
	got, err := metadata.Resolve(ann, oci, metadata.Defaults{Environment: "staging"})
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
	}, metadata.Defaults{Environment: "production"})
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
	}, metadata.Defaults{Environment: "production"})
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
	}, metadata.Defaults{Environment: "production"})
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
	}, metadata.Defaults{Environment: "production"})
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

func TestIsReserved(t *testing.T) {
	t.Parallel()
	if !metadata.IsReserved(metadata.AnnotationTeam) {
		t.Fatal("team should be reserved")
	}
	if metadata.IsReserved(metadata.AnnotationEnvironment) {
		t.Fatal("environment should not be reserved")
	}
}
