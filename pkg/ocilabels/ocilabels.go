// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package ocilabels extracts deployment metadata from OCI image config labels.
package ocilabels

const (
	LabelSource   = "org.opencontainers.image.source"
	LabelRevision = "org.opencontainers.image.revision"
	LabelVersion  = "org.opencontainers.image.version"
	LabelTitle    = "org.opencontainers.image.title"
	LabelCreated  = "org.opencontainers.image.created"
)

// Metadata holds OCI image labels used for GitHub Deployment reporting.
//
// Required for reporting (unless overridden by Kubernetes annotations):
//   - Source (org.opencontainers.image.source)
//   - Revision (org.opencontainers.image.revision)
//
// Optional:
//   - Version, Title, Created — used for logging and diagnostics.
type Metadata struct {
	Source   string
	Revision string
	Version  string
	Title    string
	Created  string
}

// Extract reads known OCI labels from an image config label map.
// Missing labels are left empty; callers merge with annotations and validate.
func Extract(labels map[string]string) Metadata {
	if labels == nil {
		return Metadata{}
	}
	return Metadata{
		Source:   labels[LabelSource],
		Revision: labels[LabelRevision],
		Version:  labels[LabelVersion],
		Title:    labels[LabelTitle],
		Created:  labels[LabelCreated],
	}
}
