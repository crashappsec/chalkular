// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package controller

import (
	"context"
	"fmt"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MediaTypePolicyReconciler reconciles a MediaTypePolicy object
type MediaTypePolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=profiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=downloaders;clusterdownloaders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=pipelines,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MediaTypePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	l.Info("reconciling artifact mediatype mapping object", "name", req.Name, "namespace", req.Namespace, "req", req)

	// Fetch the Pipeline instance to be reconciled
	mapping := &chalkularv1beta1.MediaTypePolicy{}
	err := r.Get(ctx, req.NamespacedName, mapping)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	_, err = r.reconcileChildProfile(ctx, mapping)
	if err != nil && apierrors.IsNotFound(err) {
		mapping.Status.Profile = chalkularv1beta1.MediaTypePolicyProfileStatus{
			Available: false,
		}
	} else if err != nil {
		l.Error(err, "failed to reconcile child profile for artifact mediatype mapping", "name", mapping.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		mapping.Status.Profile = chalkularv1beta1.MediaTypePolicyProfileStatus{
			Available: true,
		}
	}

	_, err = r.reconcileChildDownloader(ctx, mapping)
	if err != nil && apierrors.IsNotFound(err) {
		mapping.Status.Downloader = chalkularv1beta1.MediaTypePolicyDownloaderStatus{
			Available: false,
		}
	} else if err != nil {
		l.Error(err, "failed to reconcile child downloader for artifact mediatype mapping", "name", mapping.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		mapping.Status.Downloader = chalkularv1beta1.MediaTypePolicyDownloaderStatus{
			Available: true,
		}
	}

	err = r.Status().Update(ctx, mapping)
	return ctrl.Result{}, err
}

func (r *MediaTypePolicyReconciler) reconcileChildProfile(ctx context.Context, mapping *chalkularv1beta1.MediaTypePolicy) (*v1.ObjectReference, error) {
	found := &ocularv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mapping.Namespace,
			Name:      mapping.Spec.PipelineTemplate.Spec.ProfileRef.Name,
		},
	}
	err := r.Get(ctx, client.ObjectKey{Namespace: mapping.Namespace, Name: mapping.Spec.PipelineTemplate.Spec.ProfileRef.Name}, found)
	return &v1.ObjectReference{Name: found.Name, Namespace: found.Namespace}, err
}

func (r *MediaTypePolicyReconciler) reconcileChildDownloader(ctx context.Context, mapping *chalkularv1beta1.MediaTypePolicy) (*ocularv1beta1.ParameterizedObjectReference, error) {
	downloaderRef := mapping.Spec.PipelineTemplate.Spec.DownloaderRef
	switch downloaderRef.Kind {
	case "", "Downloader":
		found := &ocularv1beta1.Downloader{}
		err := r.Get(ctx, client.ObjectKey{Namespace: mapping.Namespace, Name: mapping.Spec.PipelineTemplate.Spec.DownloaderRef.Name}, found)
		return &ocularv1beta1.ParameterizedObjectReference{
			ObjectReference: v1.ObjectReference{
				Name: found.Name, Namespace: found.Namespace, Kind: "Downloader"},
			Parameters: downloaderRef.Parameters}, err
	case "ClusterDownloader":
		found := &ocularv1beta1.ClusterDownloader{}
		err := r.Get(ctx, client.ObjectKey{Name: downloaderRef.Name}, found)
		return &ocularv1beta1.ParameterizedObjectReference{
			ObjectReference: v1.ObjectReference{
				Name: found.Name, Kind: "ClusterDownloader",
			},
			Parameters: downloaderRef.Parameters}, err
	default:
		return nil, fmt.Errorf("unknown downloader kind: %s", downloaderRef.Kind)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MediaTypePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chalkularv1beta1.MediaTypePolicy{}).
		Named("mediatypepolicy").
		Owns(&ocularv1beta1.Profile{}).
		Owns(&ocularv1beta1.Downloader{}).
		Complete(r)
}
