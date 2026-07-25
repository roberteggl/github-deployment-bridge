// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
)

// Reporter reports deployments for a Flux source.
type Reporter interface {
	Report(ctx context.Context, in deployment.ReportInput) error
}

// KustomizationReconciler watches Flux Kustomizations and reports lifecycle statuses.
type KustomizationReconciler struct {
	client.Client
	Finder   *WorkloadFinder
	Reporter Reporter
	Log      *slog.Logger
}

// Reconcile handles a Kustomization event.
func (r *KustomizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.With("namespace", req.Namespace, "kustomization", req.Name)

	var ks kustomizev1.Kustomization
	if err := r.Get(ctx, req.NamespacedName, &ks); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get kustomization: %w", err)
	}

	phase, reason := DerivePhase(ks.Status.Conditions, ks.Generation, ks.Status.ObservedGeneration)
	images, err := r.Finder.ImagesForKustomization(ctx, &ks)
	if err != nil {
		log.Error("failed to resolve workload images", "error", err)
		return ctrl.Result{}, err
	}
	if len(images) == 0 {
		log.Debug("no supported workloads found in inventory; skipping")
		return ctrl.Result{}, nil
	}

	if err := r.Reporter.Report(ctx, deployment.ReportInput{
		Namespace:  ks.Namespace,
		SourceKind: "Kustomization",
		SourceName: ks.Name,
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
func (r *KustomizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Log == nil {
		r.Log = slog.Default()
	}
	if r.Finder == nil {
		r.Finder = &WorkloadFinder{Client: r.Client}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kustomizev1.Kustomization{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		WithEventFilter(fluxSourcePredicate(
			func(obj client.Object) (conditions []metav1.Condition, generation, observed int64, revision string, ok bool) {
				ks, ok := obj.(*kustomizev1.Kustomization)
				if !ok {
					return nil, 0, 0, "", false
				}
				return ks.Status.Conditions, ks.Generation, ks.Status.ObservedGeneration, ks.Status.LastAppliedRevision, true
			},
		)).
		Complete(r)
}

// Reconciler is an alias kept for backward-compatible main wiring.
type Reconciler = KustomizationReconciler

func fluxSourcePredicate(
	extract func(obj client.Object) (conditions []metav1.Condition, generation, observed int64, revision string, ok bool),
) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			_, _, _, _, ok := extract(e.Object)
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newCond, newGen, newObs, newRev, ok := extract(e.ObjectNew)
			if !ok {
				return false
			}
			oldCond, oldGen, oldObs, oldRev, ok := extract(e.ObjectOld)
			if !ok {
				return true
			}
			if newGen != oldGen || newObs != oldObs || newRev != oldRev {
				return true
			}
			return !reflect.DeepEqual(newCond, oldCond)
		},
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}
