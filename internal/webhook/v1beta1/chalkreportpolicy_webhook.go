// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package v1beta1

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	chalkocularcrashoverriderunv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var chalkreportpolicylog = logf.Log.WithName("chalkreportpolicy-resource")

// SetupChalkReportPolicyWebhookWithManager registers the webhook for ChalkReportPolicy in the manager.
func SetupChalkReportPolicyWebhookWithManager(mgr ctrl.Manager, clusterDownloaderName string) error {
	return ctrl.NewWebhookManagedBy(mgr, &chalkocularcrashoverriderunv1beta1.ChalkReportPolicy{}).
		WithValidator(&ChalkReportPolicyCustomValidator{}).
		WithDefaulter(&ChalkReportPolicyCustomDefaulter{
			downloader:     clusterDownloaderName,
			downloaderKind: "ClusterDownloader",
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chalk-ocular-crashoverride-run-v1beta1-chalkreportpolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=chalkreportpolicies,verbs=create;update,versions=v1beta1,name=mchalkreportpolicy-v1beta1.kb.io,admissionReviewVersions=v1

// ChalkReportPolicyCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind ChalkReportPolicy when those are created or updated.
type ChalkReportPolicyCustomDefaulter struct {
	downloader     string
	downloaderKind string
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind ChalkReportPolicy.
func (d *ChalkReportPolicyCustomDefaulter) Default(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.ChalkReportPolicy) error {
	chalkreportpolicylog.Info("Defaulting for ChalkReportPolicy", "name", obj.GetName())

	if obj.Spec.PipelineTemplate.Spec.DownloaderRef.Name == "" {
		obj.Spec.PipelineTemplate.Spec.DownloaderRef.Name = d.downloader
		obj.Spec.PipelineTemplate.Spec.DownloaderRef.Kind = d.downloaderKind
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-chalk-ocular-crashoverride-run-v1beta1-chalkreportpolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=chalkreportpolicies,verbs=create;update,versions=v1beta1,name=vchalkreportpolicy-v1beta1.kb.io,admissionReviewVersions=v1

// ChalkReportPolicyCustomValidator struct is responsible for validating the ChalkReportPolicy resource
// when it is created, updated, or deleted.
type ChalkReportPolicyCustomValidator struct{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ChalkReportPolicy.
func (v *ChalkReportPolicyCustomValidator) ValidateCreate(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.ChalkReportPolicy) (admission.Warnings, error) {
	chalkreportpolicylog.Info("Validation for ChalkReportPolicy upon creation", "name", obj.GetName())

	return v.validate(obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ChalkReportPolicy.
func (v *ChalkReportPolicyCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *chalkocularcrashoverriderunv1beta1.ChalkReportPolicy) (admission.Warnings, error) {
	chalkreportpolicylog.Info("Validation for ChalkReportPolicy upon update", "name", newObj.GetName())
	if warnings, err := v.validate(newObj); err != nil {
		return warnings, err
	}

	return nil, nil
}

func (v *ChalkReportPolicyCustomValidator) validate(policy *chalkocularcrashoverriderunv1beta1.ChalkReportPolicy) (admission.Warnings, error) {
	var allErrs field.ErrorList
	target := policy.Spec.PipelineTemplate.Spec.Target
	if target.Identifier != "" || target.Version != "" {
		path := field.NewPath("spec").Child("pipelineTemplate").Child("spec").Child("target")
		allErrs = append(allErrs,
			field.Invalid(path, target, "target should not bet set and instead should be specified by 'extraction.target'"))
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(schema.GroupKind{Group: "chalk.ocular.crashoverride.run", Kind: "ChalkReportPolicy"}, policy.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ChalkReportPolicy.
func (v *ChalkReportPolicyCustomValidator) ValidateDelete(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.ChalkReportPolicy) (admission.Warnings, error) {
	chalkreportpolicylog.Info("Validation for ChalkReportPolicy upon deletion", "name", obj.GetName())

	return nil, nil
}
