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
	"github.com/crashappsec/chalkular/internal/policy"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ChalkReportPolicyReconciler reconciles a ChalkReportPolicy object
type ChalkReportPolicyReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	PolicyCompiler *policy.Compiler
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChalkReportPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chalkularv1beta1.ChalkReportPolicy{}).
		Named("chalkreportpolicy").
		Owns(&ocularv1beta1.Profile{}).
		Owns(&ocularv1beta1.Downloader{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=chalkreportpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=chalkreportpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chalk.ocular.crashoverride.run,resources=chalkreportpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=profiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=downloaders;clusterdownloaders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ocular.crashoverride.run,resources=pipelines,verbs=get;list;watch;create;update;patch;delete

const policyCacheFinalizer = "chalk.ocular.crashoverride.run/cel-cache"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ChalkReportPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)

	l.Info("reconciling chalk report policy")

	// Fetch the Pipeline instance to be reconciled
	reportPolicy := &chalkularv1beta1.ChalkReportPolicy{}
	err := r.Get(ctx, req.NamespacedName, reportPolicy)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !reportPolicy.DeletionTimestamp.IsZero() {
		if err := r.PolicyCompiler.Remove(reportPolicy); err != nil {
			l.Error(err, "failed to remove compiled reportPolicy")
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(reportPolicy, policyCacheFinalizer)
		return ctrl.Result{}, r.Update(ctx, reportPolicy)
	}

	// Ensure our finalizer is present.
	if !controllerutil.ContainsFinalizer(reportPolicy, policyCacheFinalizer) {
		controllerutil.AddFinalizer(reportPolicy, policyCacheFinalizer)
		return ctrl.Result{}, r.Update(ctx, reportPolicy)
	}

	var downloaderAvailable, profileAvailable bool
	_, err = r.reconcileChildProfile(ctx, reportPolicy)
	if err != nil && apierrors.IsNotFound(err) {
		profileAvailable = false
	} else if err != nil {
		l.Error(err, "failed to reconcile child profile for chalk report reportPolicy", "name", reportPolicy.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		profileAvailable = true
	}

	_, err = r.reconcileChildDownloader(ctx, reportPolicy)
	if err != nil && apierrors.IsNotFound(err) {
		downloaderAvailable = false
	} else if err != nil {
		l.Error(err, "failed to reconcile child downloader for chalk report reportPolicy", "name", reportPolicy.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		downloaderAvailable = true
	}

	if reportPolicy.Status.DownloaderValid != downloaderAvailable || reportPolicy.Status.ProfileValid != profileAvailable {
		reportPolicy.Status.DownloaderValid = downloaderAvailable
		reportPolicy.Status.ProfileValid = profileAvailable
		return ctrl.Result{}, r.Status().Update(ctx, reportPolicy)
	}
	// compile reportPolicy
	var metaChanged bool
	_, err = r.PolicyCompiler.Get(reportPolicy)
	if err != nil {
		l.Error(err, "unable to compile reportPolicy")
		metaChanged = meta.SetStatusCondition(&reportPolicy.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "CELCompileFailed",
			Message:            err.Error(),
			ObservedGeneration: reportPolicy.Generation,
		})
	} else {
		metaChanged = meta.SetStatusCondition(&reportPolicy.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "CELCompiled",
			Message:            "",
			ObservedGeneration: reportPolicy.Generation,
		})
	}

	if metaChanged {
		return ctrl.Result{}, r.Status().Update(ctx, reportPolicy)
	}

	return ctrl.Result{}, nil
}

func (r *ChalkReportPolicyReconciler) reconcileChildProfile(ctx context.Context, mapping *chalkularv1beta1.ChalkReportPolicy) (*v1.ObjectReference, error) {
	found := &ocularv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mapping.Namespace,
			Name:      mapping.Spec.PipelineTemplate.Spec.ProfileRef.Name,
		},
	}
	err := r.Get(ctx, client.ObjectKey{Namespace: mapping.Namespace, Name: mapping.Spec.PipelineTemplate.Spec.ProfileRef.Name}, found)
	return &v1.ObjectReference{Name: found.Name, Namespace: found.Namespace}, err
}

func (r *ChalkReportPolicyReconciler) reconcileChildDownloader(ctx context.Context, mapping *chalkularv1beta1.ChalkReportPolicy) (*ocularv1beta1.ParameterizedObjectReference, error) {
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
