// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package ocilabels extracts deployment metadata from OCI image config labels.
package ocilabels

import (
	"fmt"

	"github.com/roberteggl/github-deployment-bridge/pkg/giturl"
)

const (
	LabelSource   = "org.opencontainers.image.source"
	LabelRevision = "org.opencontainers.image.revision"
	LabelVersion  = "org.opencontainers.image.version"
)

// Metadata holds the OCI labels required to report a GitHub Deployment.
type Metadata struct {
	Source   string
	Revision string
	Version  string
	Repo     giturl.Repository
}

// Extract reads required OCI labels and parses the GitHub repository from source.
func Extract(labels map[string]string) (Metadata, error) {
	if labels == nil {
		return Metadata{}, fmt.Errorf("image has no labels")
	}

	source := labels[LabelSource]
	revision := labels[LabelRevision]
	version := labels[LabelVersion]

	var missing []string
	if source == "" {
		missing = append(missing, LabelSource)
	}
	if revision == "" {
		missing = append(missing, LabelRevision)
	}
	if version == "" {
		missing = append(missing, LabelVersion)
	}
	if len(missing) > 0 {
		return Metadata{}, fmt.Errorf("missing required OCI labels: %v", missing)
	}

	repo, err := giturl.Parse(source)
	if err != nil {
		return Metadata{}, fmt.Errorf("parse %s: %w", LabelSource, err)
	}

	return Metadata{
		Source:   source,
		Revision: revision,
		Version:  version,
		Repo:     repo,
	}, nil
}
