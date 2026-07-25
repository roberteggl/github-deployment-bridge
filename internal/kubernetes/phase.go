// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
)

// knownFailureReasons maps Flux Ready/Stalled reasons that indicate application failure.
var knownFailureReasons = map[string]struct{}{
	"HealthCheckFailed":        {},
	"ProgressDeadlineExceeded": {},
	"ArtifactFailed":           {},
	"BuildFailed":              {},
	"ReconciliationFailed":     {},
	"InstallFailed":            {},
	"UpgradeFailed":            {},
	"UninstallFailed":          {},
	"RollbackFailed":           {},
	"TestFailed":               {},
	"DependencyNotReady":       {},
	"InventoryBuildFailed":     {},
}

// DerivePhase maps Flux status conditions to a desired GitHub Deployment phase.
//
// Rules:
//   - Ready=True and generation observed → success
//   - Ready=False (or Stalled=True) with generation observed, or a known failure reason → failure
//   - Otherwise (reconciling / not yet observed) → in_progress
func DerivePhase(conditions []metav1.Condition, generation, observedGeneration int64) (deployment.Phase, string) {
	ready := apimeta.FindStatusCondition(conditions, fluxmeta.ReadyCondition)
	stalled := apimeta.FindStatusCondition(conditions, fluxmeta.StalledCondition)
	reconciling := apimeta.FindStatusCondition(conditions, fluxmeta.ReconcilingCondition)

	observed := generation == observedGeneration && observedGeneration != 0

	if observed && ready != nil && ready.Status == metav1.ConditionTrue {
		return deployment.PhaseSuccess, ready.Reason
	}

	if stalled != nil && stalled.Status == metav1.ConditionTrue {
		return deployment.PhaseFailure, stalled.Reason
	}

	if ready != nil && ready.Status == metav1.ConditionFalse {
		if observed || isFailureReason(ready.Reason) {
			return deployment.PhaseFailure, ready.Reason
		}
	}

	if reconciling != nil && reconciling.Status == metav1.ConditionTrue {
		return deployment.PhaseInProgress, reconciling.Reason
	}

	if ready != nil {
		return deployment.PhaseInProgress, ready.Reason
	}
	return deployment.PhaseInProgress, ""
}

func isFailureReason(reason string) bool {
	_, ok := knownFailureReasons[reason]
	return ok
}
