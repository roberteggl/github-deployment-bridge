// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package metadata resolves deployment metadata from OCI labels and Kubernetes annotations.
package metadata

import (
	"fmt"
	"strings"

	"github.com/roberteggl/github-deployment-bridge/pkg/giturl"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

// Defaults are controller-level fallbacks applied when annotations and OCI labels omit a field.
type Defaults struct {
	Cluster        string
	Environment    string
	EnvironmentURL string
	LogURL         string // optional pre-expanded fallback (usually set by the caller after resolve)
	Description    string
}

// Resolved is the final metadata used to create a GitHub Deployment.
type Resolved struct {
	Repo           giturl.Repository
	Commit         string
	Environment    string
	EnvironmentURL string
	LogURL         string
	Description    string
	Production     bool
	DeploymentName string
	Cluster        string
	Team           string
	Service        string
	Component      string
	SlackChannel   string
	Owner          string
	Release        string
	Tag            string
	Version        string // optional; logging/metrics only
	Title          string // optional; logging only
	Created        string // optional; diagnostics only
	AutoReport     bool
}

// SkipError indicates the workload should be ignored without failing reconcile.
type SkipError struct {
	Reason string
}

func (e *SkipError) Error() string {
	return e.Reason
}

// Resolve merges Kubernetes annotations (highest priority), OCI labels, and defaults.
//
// Priority for every field: annotation > OCI label > built-in / controller default.
// Missing required metadata returns an error; callers should skip and warn, never fail reconcile.
func Resolve(annotations map[string]string, oci ocilabels.Metadata, defaults Defaults) (Resolved, error) {
	ann := annotations
	if ann == nil {
		ann = map[string]string{}
	}

	// Opt-in: workloads are ignored unless github-deployment-bridge.io/auto-report=true.
	autoReport := false
	if raw, present := ann[AnnotationAutoReport]; present {
		v, ok, err := ParseBoolAnnotation(raw)
		if err != nil {
			return Resolved{}, fmt.Errorf("%s: %w", AnnotationAutoReport, err)
		}
		if ok {
			autoReport = v
		}
	}
	if !autoReport {
		return Resolved{AutoReport: false}, &SkipError{Reason: "auto-report not enabled (set github-deployment-bridge.io/auto-report=true)"}
	}

	repo, err := resolveRepository(ann, oci)
	if err != nil {
		return Resolved{}, err
	}

	commit, err := resolveCommit(ann, oci)
	if err != nil {
		return Resolved{}, err
	}

	environment := firstNonEmpty(ann[AnnotationEnvironment], defaults.Environment)
	if strings.TrimSpace(environment) == "" {
		return Resolved{}, fmt.Errorf("environment must not be empty")
	}

	envURL, err := resolveHTTPSURL(ann[AnnotationEnvironmentURL], defaults.EnvironmentURL, AnnotationEnvironmentURL)
	if err != nil {
		return Resolved{}, err
	}

	// Annotation log-url may still contain placeholders; the reporter expands them.
	// defaults.LogURL is unused (template applied in the reporter after resolve).
	logURL, err := resolveHTTPSURL(ann[AnnotationLogURL], defaults.LogURL, AnnotationLogURL)
	if err != nil {
		return Resolved{}, err
	}

	description := firstNonEmpty(ann[AnnotationDescription], defaults.Description)
	if description == "" {
		description = "Deployed by FluxCD"
	}

	production, err := resolveProduction(ann, environment)
	if err != nil {
		return Resolved{}, err
	}

	deploymentName := firstNonEmpty(ann[AnnotationDeploymentName], repo.Name)
	if deploymentName == "" {
		return Resolved{}, fmt.Errorf("deployment-name must not be empty")
	}

	cluster := firstNonEmpty(ann[AnnotationCluster], defaults.Cluster)
	if strings.TrimSpace(cluster) == "" {
		return Resolved{}, fmt.Errorf("cluster must not be empty")
	}

	return Resolved{
		Repo:           repo,
		Commit:         commit,
		Environment:    environment,
		EnvironmentURL: envURL,
		LogURL:         logURL,
		Description:    description,
		Production:     production,
		DeploymentName: deploymentName,
		Cluster:        cluster,
		Team:           strings.TrimSpace(ann[AnnotationTeam]),
		Service:        strings.TrimSpace(ann[AnnotationService]),
		Component:      strings.TrimSpace(ann[AnnotationComponent]),
		SlackChannel:   strings.TrimSpace(ann[AnnotationSlackChannel]),
		Owner:          strings.TrimSpace(ann[AnnotationOwner]),
		Release:        strings.TrimSpace(ann[AnnotationRelease]),
		Tag:            strings.TrimSpace(ann[AnnotationTag]),
		Version:        oci.Version,
		Title:          oci.Title,
		Created:        oci.Created,
		AutoReport:     true,
	}, nil
}

func resolveRepository(ann map[string]string, oci ocilabels.Metadata) (giturl.Repository, error) {
	if raw := strings.TrimSpace(ann[AnnotationRepository]); raw != "" {
		repo, err := giturl.Parse(raw)
		if err != nil {
			return giturl.Repository{}, fmt.Errorf("%s: %w", AnnotationRepository, err)
		}
		return repo, nil
	}
	if strings.TrimSpace(oci.Source) == "" {
		return giturl.Repository{}, fmt.Errorf("repository missing: set %s or OCI label %s", AnnotationRepository, ocilabels.LabelSource)
	}
	repo, err := giturl.Parse(oci.Source)
	if err != nil {
		return giturl.Repository{}, fmt.Errorf("OCI %s: %w", ocilabels.LabelSource, err)
	}
	return repo, nil
}

func resolveCommit(ann map[string]string, oci ocilabels.Metadata) (string, error) {
	commit := firstNonEmpty(ann[AnnotationCommit], oci.Revision)
	if commit == "" {
		return "", fmt.Errorf("commit missing: set %s or OCI label %s", AnnotationCommit, ocilabels.LabelRevision)
	}
	if !ValidGitSHA(commit) {
		return "", fmt.Errorf("commit %q is not a valid Git SHA", commit)
	}
	return commit, nil
}

func resolveHTTPSURL(annotationValue, fallback, annotationKey string) (string, error) {
	if raw := strings.TrimSpace(annotationValue); raw != "" {
		if !ValidHTTPSURL(raw) {
			return "", fmt.Errorf("%s must be a valid HTTPS URL", annotationKey)
		}
		return raw, nil
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "", nil
	}
	if !ValidHTTPSURL(fallback) {
		return "", fmt.Errorf("URL %q is not a valid HTTPS URL", fallback)
	}
	return fallback, nil
}

func resolveProduction(ann map[string]string, environment string) (bool, error) {
	if raw, present := ann[AnnotationProduction]; present {
		v, ok, err := ParseBoolAnnotation(raw)
		if err != nil {
			return false, fmt.Errorf("%s: %w", AnnotationProduction, err)
		}
		if ok {
			return v, nil
		}
	}
	return strings.EqualFold(environment, "production"), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
