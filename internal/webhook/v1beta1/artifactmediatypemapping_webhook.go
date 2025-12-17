// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	chalkularocularcrashoverriderunv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var artifactmediatypemappinglog = logf.Log.WithName("artifactmediatypemapping-resource")

// SetupArtifactMediaTypeMappingWebhookWithManager registers the webhook for ArtifactMediaTypeMapping in the manager.
func SetupArtifactMediaTypeMappingWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping{}).
		WithValidator(&ArtifactMediaTypeMappingCustomValidator{}).
		WithDefaulter(&ArtifactMediaTypeMappingCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-chalkular-ocular-crashoverride-run-chalk-ocular-crashoverride-run-v1beta1-artifactmediatypemapping,mutating=true,failurePolicy=fail,sideEffects=None,groups=chalkular.ocular.crashoverride.run,resources=artifactmediatypemappings,verbs=create;update,versions=v1beta1,name=martifactmediatypemapping-v1beta1.kb.io,admissionReviewVersions=v1

// ArtifactMediaTypeMappingCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind ArtifactMediaTypeMapping when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ArtifactMediaTypeMappingCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &ArtifactMediaTypeMappingCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind ArtifactMediaTypeMapping.
func (d *ArtifactMediaTypeMappingCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	artifactmediatypemapping, ok := obj.(*chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping)

	if !ok {
		return fmt.Errorf("expected an ArtifactMediaTypeMapping object but got %T", obj)
	}
	artifactmediatypemappinglog.Info("Defaulting for ArtifactMediaTypeMapping", "name", artifactmediatypemapping.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-chalkular-ocular-crashoverride-run-chalk-ocular-crashoverride-run-v1beta1-artifactmediatypemapping,mutating=false,failurePolicy=fail,sideEffects=None,groups=chalkular.ocular.crashoverride.run,resources=artifactmediatypemappings,verbs=create;update,versions=v1beta1,name=vartifactmediatypemapping-v1beta1.kb.io,admissionReviewVersions=v1

// ArtifactMediaTypeMappingCustomValidator struct is responsible for validating the ArtifactMediaTypeMapping resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ArtifactMediaTypeMappingCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ArtifactMediaTypeMappingCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ArtifactMediaTypeMapping.
func (v *ArtifactMediaTypeMappingCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	artifactmediatypemapping, ok := obj.(*chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping)
	if !ok {
		return nil, fmt.Errorf("expected a ArtifactMediaTypeMapping object but got %T", obj)
	}
	artifactmediatypemappinglog.Info("Validation for ArtifactMediaTypeMapping upon creation", "name", artifactmediatypemapping.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ArtifactMediaTypeMapping.
func (v *ArtifactMediaTypeMappingCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	artifactmediatypemapping, ok := newObj.(*chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping)
	if !ok {
		return nil, fmt.Errorf("expected a ArtifactMediaTypeMapping object for the newObj but got %T", newObj)
	}
	artifactmediatypemappinglog.Info("Validation for ArtifactMediaTypeMapping upon update", "name", artifactmediatypemapping.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ArtifactMediaTypeMapping.
func (v *ArtifactMediaTypeMappingCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	artifactmediatypemapping, ok := obj.(*chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping)
	if !ok {
		return nil, fmt.Errorf("expected a ArtifactMediaTypeMapping object but got %T", obj)
	}
	artifactmediatypemappinglog.Info("Validation for ArtifactMediaTypeMapping upon deletion", "name", artifactmediatypemapping.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
