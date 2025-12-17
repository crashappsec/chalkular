// Copyright (C) 2025 Crash Override, Inc.
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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ArtifactMediaTypeMappingReconciler reconciles a ArtifactMediaTypeMapping object
type ArtifactMediaTypeMappingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chalkular.ocular.crashoverride.run,resources=artifactmediatypemappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chalkular.ocular.crashoverride.run,resources=artifactmediatypemappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chalkular.ocular.crashoverride.run,resources=artifactmediatypemappings/finalizers,verbs=update
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=profiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=pipelines,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ArtifactMediaTypeMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	l.Info("reconciling artifact mediatype mapping object", "name", req.Name, "namespace", req.Namespace, "req", req)

	// Fetch the Pipeline instance to be reconciled
	mapping := &chalkularv1beta1.ArtifactMediaTypeMapping{}
	err := r.Get(ctx, req.NamespacedName, mapping)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	profile, err := r.reconcileChildProfile(ctx, mapping)
	if err != nil {
		l.Error(err, "failed to reconcile child profile for artifact mediatype mapping", "name", mapping.Name)
		mapping.Status.Profile = &chalkularv1beta1.ArtifactMediaTypeMappingProfileStatus{
			Available: false,
		}
		if profile != nil {
			mapping.Status.Profile.Name = profile.Name
		}
		errStatus := r.Status().Update(ctx, mapping)
		if errStatus != nil {
			l.Error(errStatus, "failed to update artifact mediatype mapping status after profile reconciliation failure", "name", mapping.Name)
			return ctrl.Result{}, errStatus
		}
		return ctrl.Result{}, err
	}

	mapping.Status.Profile = &chalkularv1beta1.ArtifactMediaTypeMappingProfileStatus{
		Name:      profile.Name,
		Available: true,
	}
	err = r.Status().Update(ctx, mapping)
	return ctrl.Result{}, err
}

func (r *ArtifactMediaTypeMappingReconciler) reconcileChildProfile(ctx context.Context, mapping *chalkularv1beta1.ArtifactMediaTypeMapping) (*ocularv1beta1.Profile, error) {
	found := &ocularv1beta1.Profile{}
	if mapping.Spec.Profile.ValueFrom.Name != "" {
		found = &ocularv1beta1.Profile{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: mapping.Namespace,
				Name:      mapping.Spec.Profile.ValueFrom.Name,
			},
		}
		err := r.Get(ctx, client.ObjectKey{Namespace: mapping.Namespace, Name: mapping.Spec.Profile.ValueFrom.Name}, found)
		return found, err
	}

	if mapping.Spec.Profile.Value == nil {
		// No profile defined
		return nil, fmt.Errorf("no value found for profile %s", mapping.Spec.Profile.ValueFrom.Name)
	}

	// Profile is defined inline, create or update it
	profile := &ocularv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chalkular-" + mapping.Name,
			Namespace: mapping.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mapping, chalkularv1beta1.GroupVersion.WithKind("ArtifactMediaTypeMapping")),
			},
		},
		Spec: *mapping.Spec.Profile.Value,
	}

	profile.DeepCopyInto(found)

	err := r.Get(ctx, client.ObjectKey{Namespace: mapping.Namespace, Name: profile.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the profile
		err = r.Create(ctx, profile)
		if err != nil {
			return nil, err
		}
		return profile, err
	} else if err != nil {
		return nil, err
	}

	// Update the profile if needed
	err = r.Update(ctx, profile)
	return profile, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArtifactMediaTypeMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chalkularv1beta1.ArtifactMediaTypeMapping{}).
		Named("artifactmediatypemapping").
		Owns(&ocularv1beta1.Profile{}).
		Complete(r)
}
