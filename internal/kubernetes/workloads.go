// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
)

var workloadKinds = map[string]schema.GroupVersionKind{
	"Deployment":  {Group: "apps", Version: "v1", Kind: "Deployment"},
	"StatefulSet": {Group: "apps", Version: "v1", Kind: "StatefulSet"},
	"DaemonSet":   {Group: "apps", Version: "v1", Kind: "DaemonSet"},
}

// WorkloadFinder resolves container images for workloads owned by a Kustomization.
type WorkloadFinder struct {
	Client client.Client
}

// ImagesForKustomization returns images for Deployments, StatefulSets, and DaemonSets
// associated with the Kustomization inventory. ReplicaSets are resolved via owner references.
func (f *WorkloadFinder) ImagesForKustomization(ctx context.Context, ks *kustomizev1.Kustomization) ([]deployment.WorkloadImage, error) {
	if ks.Status.Inventory == nil {
		return nil, nil
	}

	var images []deployment.WorkloadImage
	seen := make(map[string]struct{})

	for _, entry := range ks.Status.Inventory.Entries {
		ns, name, group, kind, err := parseInventoryID(entry.ID)
		if err != nil {
			continue
		}

		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			imgs, err := f.imagesFromWorkload(ctx, ns, name, kind)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			for _, img := range imgs {
				key := img.Kind + "/" + img.Namespace + "/" + img.Name + "/" + img.Image
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				images = append(images, img)
			}
		case "ReplicaSet":
			if group != "apps" {
				continue
			}
			imgs, err := f.imagesFromReplicaSet(ctx, ns, name)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			for _, img := range imgs {
				key := img.Kind + "/" + img.Namespace + "/" + img.Name + "/" + img.Image
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				images = append(images, img)
			}
		default:
			// Ignore Jobs, CronJobs, and other kinds.
			continue
		}
	}

	return images, nil
}

func (f *WorkloadFinder) imagesFromWorkload(ctx context.Context, namespace, name, kind string) ([]deployment.WorkloadImage, error) {
	gvk, ok := workloadKinds[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported workload kind %q", kind)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	key := types.NamespacedName{Namespace: namespace, Name: name}
	if err := f.Client.Get(ctx, key, obj); err != nil {
		return nil, err
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil {
		return nil, fmt.Errorf("read containers for %s/%s: %w", kind, name, err)
	}
	if !found {
		return nil, nil
	}

	var out []deployment.WorkloadImage
	for _, c := range containers {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		image, _, _ := unstructured.NestedString(m, "image")
		if image == "" {
			continue
		}
		out = append(out, deployment.WorkloadImage{
			Namespace: namespace,
			Kind:      kind,
			Name:      name,
			Image:     image,
		})
	}

	// Include init containers as well - they may carry the application image.
	initContainers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "initContainers")
	if err != nil {
		return nil, fmt.Errorf("read initContainers for %s/%s: %w", kind, name, err)
	}
	if found {
		for _, c := range initContainers {
			m, ok := c.(map[string]any)
			if !ok {
				continue
			}
			image, _, _ := unstructured.NestedString(m, "image")
			if image == "" {
				continue
			}
			out = append(out, deployment.WorkloadImage{
				Namespace: namespace,
				Kind:      kind,
				Name:      name,
				Image:     image,
			})
		}
	}

	return out, nil
}

func (f *WorkloadFinder) imagesFromReplicaSet(ctx context.Context, namespace, name string) ([]deployment.WorkloadImage, error) {
	var rs appsv1.ReplicaSet
	if err := f.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &rs); err != nil {
		return nil, err
	}

	for _, owner := range rs.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && owner.Kind == "Deployment" {
			return f.imagesFromWorkload(ctx, namespace, owner.Name, "Deployment")
		}
	}
	return nil, nil
}

// parseInventoryID parses Flux inventory IDs: <namespace>_<name>_<group>_<kind>
func parseInventoryID(id string) (namespace, name, group, kind string, err error) {
	parts := strings.Split(id, "_")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid inventory id %q", id)
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}
