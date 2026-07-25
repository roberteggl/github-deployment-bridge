// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
)

// HelmReleaseReconciler watches Flux HelmReleases and reports lifecycle statuses.
type HelmReleaseReconciler struct {
	client.Client
	Finder   *WorkloadFinder
	Reporter Reporter
	Log      *slog.Logger
}

// Reconcile handles a HelmRelease event.
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.With("namespace", req.Namespace, "helmrelease", req.Name)

	var hr helmv2.HelmRelease
	if err := r.Get(ctx, req.NamespacedName, &hr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get helmrelease: %w", err)
	}

	phase, reason := DerivePhase(hr.Status.Conditions, hr.Generation, hr.Status.ObservedGeneration)
	images, err := r.Finder.ImagesForHelmRelease(ctx, &hr)
	if err != nil {
		log.Error("failed to resolve workload images", "error", err)
		return ctrl.Result{}, err
	}
	if len(images) == 0 {
		log.Debug("no supported workloads found in inventory; skipping")
		return ctrl.Result{}, nil
	}

	if err := r.Reporter.Report(ctx, deployment.ReportInput{
		Namespace:  hr.Namespace,
		SourceKind: "HelmRelease",
		SourceName: hr.Name,
		Phase:      phase,
		Reason:     reason,
		Images:     images,
	}); err != nil {
		log.Error("failed to report deployments", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller.
func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Log == nil {
		r.Log = slog.Default()
	}
	if r.Finder == nil {
		r.Finder = &WorkloadFinder{Client: r.Client}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv2.HelmRelease{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		WithEventFilter(fluxSourcePredicate(
			func(obj client.Object) (conditions []metav1.Condition, generation, observed int64, revision string, ok bool) {
				hr, ok := obj.(*helmv2.HelmRelease)
				if !ok {
					return nil, 0, 0, "", false
				}
				return hr.Status.Conditions, hr.Generation, hr.Status.ObservedGeneration, hr.Status.LastAttemptedRevision, true
			},
		)).
		Complete(r)
}
