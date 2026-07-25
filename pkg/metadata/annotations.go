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

// Optional payload annotations copied into the GitHub Deployment payload when set.
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

// OptionalPayloadAnnotations lists optional annotation keys merged into payload.
var OptionalPayloadAnnotations = []string{
	AnnotationTeam,
	AnnotationService,
	AnnotationComponent,
	AnnotationSlackChannel,
	AnnotationOwner,
	AnnotationRelease,
	AnnotationTag,
	AnnotationCluster,
}

// IsOptionalPayloadAnnotation reports whether key is an optional payload annotation.
func IsOptionalPayloadAnnotation(key string) bool {
	for _, k := range OptionalPayloadAnnotations {
		if k == key {
			return true
		}
	}
	return false
}
