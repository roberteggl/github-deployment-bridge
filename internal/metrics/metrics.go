// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package metrics exposes Prometheus instrumentation for the bridge.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus collectors.
type Metrics struct {
	DeploymentsCreatedTotal            prometheus.Counter
	DeploymentStatusUpdatesTotal       prometheus.Counter
	DeploymentFailuresTotal            prometheus.Counter
	DeploymentErrorsTotal              prometheus.Counter
	DeploymentDuplicatesSkippedTotal   prometheus.Counter
	DeploymentInactiveTotal            prometheus.Counter
	GitHubAPIRequestsTotal             *prometheus.CounterVec
	GitHubAPIFailuresTotal             prometheus.Counter
	GitHubAPILatencySeconds            *prometheus.HistogramVec
	GitHubInstallationResolutionsTotal *prometheus.CounterVec
	OCIRequestsTotal                   *prometheus.CounterVec
}

// New registers metrics with the given registerer (or the default if nil).
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	factory := promauto.With(reg)

	return &Metrics{
		DeploymentsCreatedTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployments_created_total",
			Help: "Total number of GitHub Deployments created.",
		}),
		DeploymentStatusUpdatesTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_status_updates_total",
			Help: "Total number of GitHub Deployment status updates sent.",
		}),
		DeploymentFailuresTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_failures_total",
			Help: "Total number of GitHub Deployment failure statuses emitted.",
		}),
		DeploymentErrorsTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_errors_total",
			Help: "Total number of GitHub Deployment error statuses emitted.",
		}),
		DeploymentDuplicatesSkippedTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_duplicates_skipped_total",
			Help: "Total number of duplicate deployment status updates skipped.",
		}),
		DeploymentInactiveTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_inactive_total",
			Help: "Total number of GitHub Deployment inactive statuses emitted.",
		}),
		GitHubAPIRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "github_api_requests_total",
			Help: "Total number of GitHub API requests by operation and result.",
		}, []string{"operation", "result"}),
		GitHubAPIFailuresTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "github_api_failures_total",
			Help: "Total number of failed GitHub API requests.",
		}),
		GitHubAPILatencySeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "github_api_latency_seconds",
			Help:    "Latency of GitHub API requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		GitHubInstallationResolutionsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "github_installation_resolutions_total",
			Help: "Total GitHub App installation resolution outcomes.",
		}, []string{"result"}),
		OCIRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "oci_requests_total",
			Help: "Total number of OCI registry requests by result.",
		}, []string{"result"}),
	}
}
