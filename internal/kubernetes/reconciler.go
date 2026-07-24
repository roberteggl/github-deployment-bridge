// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
)

// Reporter reports deployments for a ready Kustomization.
type Reporter interface {
	Report(ctx context.Context, in deployment.ReportInput) error
}

// Reconciler watches Flux Kustomizations and reports successful reconciliations.
type Reconciler struct {
	client.Client
	Finder   *WorkloadFinder
	Reporter Reporter
	Log      *slog.Logger
}

// Reconcile handles a Kustomization event.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.With(
		"namespace", req.Namespace,
		"kustomization", req.Name,
	)

	var ks kustomizev1.Kustomization
	if err := r.Get(ctx, req.NamespacedName, &ks); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get kustomization: %w", err)
	}

	if !isReadyAndCurrent(&ks) {
		log.Debug("kustomization not ready or generation not observed; skipping")
		return ctrl.Result{}, nil
	}

	images, err := r.Finder.ImagesForKustomization(ctx, &ks)
	if err != nil {
		log.Error("failed to resolve workload images", "error", err)
		return ctrl.Result{}, err
	}
	if len(images) == 0 {
		log.Debug("no supported workloads found in inventory")
		return ctrl.Result{}, nil
	}

	if err := r.Reporter.Report(ctx, deployment.ReportInput{
		Namespace:     ks.Namespace,
		Kustomization: ks.Name,
		Images:        images,
	}); err != nil {
		log.Error("failed to report deployments", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Log == nil {
		r.Log = slog.Default()
	}
	if r.Finder == nil {
		r.Finder = &WorkloadFinder{Client: r.Client}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kustomizev1.Kustomization{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		WithEventFilter(readyKustomizationPredicate()).
		Complete(r)
}

func isReadyAndCurrent(ks *kustomizev1.Kustomization) bool {
	if ks.Generation != ks.Status.ObservedGeneration {
		return false
	}
	cond := apimeta.FindStatusCondition(ks.Status.Conditions, fluxmeta.ReadyCondition)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

func readyKustomizationPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			ks, ok := e.Object.(*kustomizev1.Kustomization)
			return ok && isReadyAndCurrent(ks)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newKS, ok := e.ObjectNew.(*kustomizev1.Kustomization)
			if !ok {
				return false
			}
			if !isReadyAndCurrent(newKS) {
				return false
			}
			oldKS, ok := e.ObjectOld.(*kustomizev1.Kustomization)
			if !ok {
				return true
			}
			if !isReadyAndCurrent(oldKS) {
				return true
			}
			return newKS.Status.ObservedGeneration != oldKS.Status.ObservedGeneration ||
				newKS.Status.LastAppliedRevision != oldKS.Status.LastAppliedRevision
		},
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}
