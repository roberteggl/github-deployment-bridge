// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes_test

import (
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
	"github.com/roberteggl/github-deployment-bridge/internal/kubernetes"
)

func TestDerivePhase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		conds     []metav1.Condition
		gen, obs  int64
		wantPhase deployment.Phase
	}{
		{
			name: "success",
			conds: []metav1.Condition{{
				Type:   fluxmeta.ReadyCondition,
				Status: metav1.ConditionTrue,
				Reason: "ReconciliationSucceeded",
			}},
			gen: 2, obs: 2,
			wantPhase: deployment.PhaseSuccess,
		},
		{
			name: "failure observed ready false",
			conds: []metav1.Condition{{
				Type:   fluxmeta.ReadyCondition,
				Status: metav1.ConditionFalse,
				Reason: "HealthCheckFailed",
			}},
			gen: 3, obs: 3,
			wantPhase: deployment.PhaseFailure,
		},
		{
			name: "failure stalled",
			conds: []metav1.Condition{{
				Type:   fluxmeta.StalledCondition,
				Status: metav1.ConditionTrue,
				Reason: "BuildFailed",
			}},
			gen: 1, obs: 0,
			wantPhase: deployment.PhaseFailure,
		},
		{
			name: "in progress reconciling",
			conds: []metav1.Condition{{
				Type:   fluxmeta.ReconcilingCondition,
				Status: metav1.ConditionTrue,
				Reason: "Progressing",
			}, {
				Type:   fluxmeta.ReadyCondition,
				Status: metav1.ConditionUnknown,
				Reason: "Progressing",
			}},
			gen: 4, obs: 3,
			wantPhase: deployment.PhaseInProgress,
		},
		{
			name: "in progress not yet observed",
			conds: []metav1.Condition{{
				Type:   fluxmeta.ReadyCondition,
				Status: metav1.ConditionFalse,
				Reason: "Progressing",
			}},
			gen: 5, obs: 4,
			wantPhase: deployment.PhaseInProgress,
		},
		{
			name: "helm install failed",
			conds: []metav1.Condition{{
				Type:   fluxmeta.ReadyCondition,
				Status: metav1.ConditionFalse,
				Reason: "InstallFailed",
			}},
			gen: 1, obs: 1,
			wantPhase: deployment.PhaseFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := kubernetes.DerivePhase(tt.conds, tt.gen, tt.obs)
			if got != tt.wantPhase {
				t.Fatalf("phase = %q, want %q", got, tt.wantPhase)
			}
		})
	}
}
