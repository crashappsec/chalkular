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
var mediatypepolicylog = logf.Log.WithName("mediatypepolicy-resource")

// SetupMediaTypePolicyWebhookWithManager registers the webhook for MediaTypePolicy in the manager.
func SetupMediaTypePolicyWebhookWithManager(mgr ctrl.Manager, clusterDownloaderName string) error {
	return ctrl.NewWebhookManagedBy(mgr, &chalkocularcrashoverriderunv1beta1.MediaTypePolicy{}).
		WithValidator(&MediaTypePolicyCustomValidator{}).
		WithDefaulter(&MediaTypePolicyCustomDefaulter{
			downloader:     clusterDownloaderName,
			downloaderKind: "ClusterDownloader",
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chalk-ocular-crashoverride-run-v1beta1-mediatypepolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies,verbs=create;update,versions=v1beta1,name=mmediatypepolicy-v1beta1.kb.io,admissionReviewVersions=v1

// MediaTypePolicyCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind MediaTypePolicy when those are created or updated.
type MediaTypePolicyCustomDefaulter struct {
	downloader     string
	downloaderKind string
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MediaTypePolicy.
func (d *MediaTypePolicyCustomDefaulter) Default(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.MediaTypePolicy) error {
	mediatypepolicylog.Info("Defaulting for MediaTypePolicy", "name", obj.GetName())

	if obj.Spec.PipelineTemplate.Spec.DownloaderRef.Name == "" {
		obj.Spec.PipelineTemplate.Spec.DownloaderRef.Name = d.downloader
		obj.Spec.PipelineTemplate.Spec.DownloaderRef.Kind = d.downloaderKind
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-chalk-ocular-crashoverride-run-v1beta1-mediatypepolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies,verbs=create;update,versions=v1beta1,name=vmediatypepolicy-v1beta1.kb.io,admissionReviewVersions=v1

// MediaTypePolicyCustomValidator struct is responsible for validating the MediaTypePolicy resource
// when it is created, updated, or deleted.
type MediaTypePolicyCustomValidator struct{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateCreate(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.MediaTypePolicy) (admission.Warnings, error) {
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon creation", "name", obj.GetName())

	return v.validate(obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *chalkocularcrashoverriderunv1beta1.MediaTypePolicy) (admission.Warnings, error) {
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon update", "name", newObj.GetName())
	if warnings, err := v.validate(newObj); err != nil {
		return warnings, err
	}

	return nil, nil
}

func (v *MediaTypePolicyCustomValidator) validate(policy *chalkocularcrashoverriderunv1beta1.MediaTypePolicy) (admission.Warnings, error) {
	var allErrs field.ErrorList
	target := policy.Spec.PipelineTemplate.Spec.Target
	if target.Identifier != "" || target.Version != "" {
		path := field.NewPath("spec").Child("pipelineTemplate").Child("spec").Child("target")
		allErrs = append(allErrs,
			field.Invalid(path, target, "target should not be set as it will be set to the received artifact from the listener"))
	}

	if len(policy.Spec.MediaTypes) == 0 {
		path := field.NewPath("spec").Child("mediaTypes")
		allErrs = append(allErrs,
			field.Invalid(path, policy.Spec.MediaTypes, "media type policy should have at least 1 media type set"))
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(schema.GroupKind{Group: "chalk.ocular.crashoverride.run", Kind: "MediaTypePolicy"}, policy.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateDelete(_ context.Context, obj *chalkocularcrashoverriderunv1beta1.MediaTypePolicy) (admission.Warnings, error) {
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon deletion", "name", obj.GetName())

	return nil, nil
}
