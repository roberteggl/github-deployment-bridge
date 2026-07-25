// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package metadata

// Annotation prefix for kubernetes workload overrides.
const AnnotationPrefix = "github-deployment-bridge.io/"

// Supported Kubernetes annotations (optional overrides for OCI labels / defaults).
const (
	AnnotationRepository     = AnnotationPrefix + "repository"
	AnnotationCommit         = AnnotationPrefix + "commit"
	AnnotationEnvironment    = AnnotationPrefix + "environment"
	AnnotationEnvironmentURL = AnnotationPrefix + "environment-url"
	AnnotationLogURL         = AnnotationPrefix + "log-url"
	AnnotationDescription    = AnnotationPrefix + "description"
	AnnotationProduction     = AnnotationPrefix + "production"
	AnnotationAutoReport     = AnnotationPrefix + "auto-report" // must be "true" to report (opt-in)
	AnnotationDeploymentName = AnnotationPrefix + "deployment-name"
)

// Future-reserved annotations. Recognized but ignored in v1.
const (
	AnnotationTeam         = AnnotationPrefix + "team"
	AnnotationService      = AnnotationPrefix + "service"
	AnnotationComponent    = AnnotationPrefix + "component"
	AnnotationSlackChannel = AnnotationPrefix + "slack-channel"
	AnnotationOwner        = AnnotationPrefix + "owner"
	AnnotationRelease      = AnnotationPrefix + "release"
	AnnotationTag          = AnnotationPrefix + "tag"
	AnnotationCluster      = AnnotationPrefix + "cluster"
)

// ReservedAnnotations lists annotation keys that are recognized but unused in v1.
var ReservedAnnotations = []string{
	AnnotationTeam,
	AnnotationService,
	AnnotationComponent,
	AnnotationSlackChannel,
	AnnotationOwner,
	AnnotationRelease,
	AnnotationTag,
	AnnotationCluster,
}

// IsReserved reports whether key is a future-reserved annotation.
func IsReserved(key string) bool {
	for _, k := range ReservedAnnotations {
		if k == key {
			return true
		}
	}
	return false
}
