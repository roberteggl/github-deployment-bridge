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
	DeploymentReportsTotal  prometheus.Counter
	DeploymentFailuresTotal prometheus.Counter
	GitHubAPIRequestsTotal  *prometheus.CounterVec
	GitHubAPILatencySeconds *prometheus.HistogramVec
	OCIRequestsTotal        *prometheus.CounterVec
	CacheHitsTotal          prometheus.Counter
	CacheMissesTotal        prometheus.Counter
}

// New registers metrics with the given registerer (or the default if nil).
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	factory := promauto.With(reg)

	return &Metrics{
		DeploymentReportsTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_reports_total",
			Help: "Total number of successful GitHub deployment reports.",
		}),
		DeploymentFailuresTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "deployment_failures_total",
			Help: "Total number of failed GitHub deployment reports.",
		}),
		GitHubAPIRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "github_api_requests_total",
			Help: "Total number of GitHub API requests by operation and result.",
		}, []string{"operation", "result"}),
		GitHubAPILatencySeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "github_api_latency_seconds",
			Help:    "Latency of GitHub API requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		OCIRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "oci_requests_total",
			Help: "Total number of OCI registry requests by result.",
		}, []string{"result"}),
		CacheHitsTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of duplicate-prevention cache hits.",
		}),
		CacheMissesTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of duplicate-prevention cache misses.",
		}),
	}
}
