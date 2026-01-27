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
var mediatypepolicylog = logf.Log.WithName("mediatypepolicy-resource")

// SetupMediaTypePolicyWebhookWithManager registers the webhook for MediaTypePolicy in the manager.
func SetupMediaTypePolicyWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&chalkularocularcrashoverriderunv1beta1.MediaTypePolicy{}).
		WithValidator(&MediaTypePolicyCustomValidator{}).
		WithDefaulter(&MediaTypePolicyCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chalk-ocular-crashoverride-run-v1beta1-mediatypepolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies,verbs=create;update,versions=v1beta1,name=mmediatypepolicy-v1beta1.kb.io,admissionReviewVersions=v1

// MediaTypePolicyCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind MediaTypePolicy when those are created or updated.
type MediaTypePolicyCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &MediaTypePolicyCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MediaTypePolicy.
func (d *MediaTypePolicyCustomDefaulter) Default(_ context.Context, obja runtime.Object) error {
	obj, ok := obja.(*chalkularocularcrashoverriderunv1beta1.MediaTypePolicy)

	if !ok {
		return fmt.Errorf("expected an MediaTypePolicy object but got %T", obj)
	}
	mediatypepolicylog.Info("Defaulting for MediaTypePolicy", "name", obj.GetName())

	return nil
}

// +kubebuilder:webhook:path=/validate-chalk-ocular-crashoverride-run-v1beta1-mediatypepolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=chalk.ocular.crashoverride.run,resources=mediatypepolicies,verbs=create;update,versions=v1beta1,name=vmediatypepolicy-v1beta1.kb.io,admissionReviewVersions=v1

// MediaTypePolicyCustomValidator struct is responsible for validating the MediaTypePolicy resource
// when it is created, updated, or deleted.
type MediaTypePolicyCustomValidator struct{}

var _ webhook.CustomValidator = &MediaTypePolicyCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	mediatypepolicy, ok := obj.(*chalkularocularcrashoverriderunv1beta1.MediaTypePolicy)
	if !ok {
		return nil, fmt.Errorf("expected a MediaTypePolicy object but got %T", obj)
	}
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon creation", "name", mediatypepolicy.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	mediatypepolicy, ok := newObj.(*chalkularocularcrashoverriderunv1beta1.MediaTypePolicy)
	if !ok {
		return nil, fmt.Errorf("expected a MediaTypePolicy object for the newObj but got %T", newObj)
	}
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon update", "name", mediatypepolicy.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type MediaTypePolicy.
func (v *MediaTypePolicyCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mediatypepolicy, ok := obj.(*chalkularocularcrashoverriderunv1beta1.MediaTypePolicy)
	if !ok {
		return nil, fmt.Errorf("expected a MediaTypePolicy object but got %T", obj)
	}
	mediatypepolicylog.Info("Validation for MediaTypePolicy upon deletion", "name", mediatypepolicy.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
