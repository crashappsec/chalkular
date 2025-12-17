// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package v1beta1

import (
	"github.com/crashappsec/ocular/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ArtifactMediaTypeMappingSpec defines the desired state of ArtifactMediaTypeMapping
type ArtifactMediaTypeMappingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// MediaTypes is the media type of the artifact.
	// +kubebuilder:validation:items:Pattern=`^[\w.-]+/[\w\.\-\+]+$`
	// +required
	MediaTypes []string `json:"mediaTypes"`

	// Profile is the profile to use for the given media type.
	// See ArtifactMediaTypeMappingProfile resource for more information.
	// +required
	Profile ArtifactMediaTypeMappingProfile `json:"profile"`

	// ScanServiceAccountName is the name of the service account that will be used to run the scan job.
	// If not set, the default service account of the namespace will be used.
	// +optional
	ScanServiceAccountName string `json:"scanServiceAccountName,omitempty" protobuf:"bytes,4,opt,name=scanServiceAccountName" description:"The name of the service account that will be used to run the scan job."`

	// UploadServiceAccountName is the name of the service account that will be used to run the upload job.
	// If not set, the default service account of the namespace will be used.
	// +optional
	UploadServiceAccountName string `json:"uploadServiceAccountName,omitempty" protobuf:"bytes,5,opt,name=uploadServiceAccountName" description:"The name of the service account that will be used to run the upload job."`

	// TTLSecondsAfterFinished
	// If set, the pipeline and its associated resources will be automatically deleted
	// after the specified number of seconds have passed since the pipeline finished.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"  protobuf:"bytes,6,opt,name=ttlSecondsAfterFinished"`

	// TTLSecondsMaxLifetime
	// If set, the pipeline and its associated resources will be automatically deleted
	// after the specified number of seconds have passed since the pipeline was created,
	// regardless of its state.
	// +optional
	TTLSecondsMaxLifetime *int32 `json:"ttlSecondsMaxLifetime,omitempty" protobuf:"bytes,7,opt,name=TTLSecondsMaxLifetime" description:"If set, the pipeline and its associated resources will be automatically deleted after the specified number of seconds have passed since the pipeline was created, regardless of its state."`
}

// ArtifactMediaTypeMappingProfile defines a reference to a Profile resource
// that can be specified either directly or via a reference to a Profile resource.
// If Value is given, a profile will be created and owned by the ArtifactMediaTypeMapping resource.
// If ValueFrom is given, the Profile resource will be referenced.
// Exactly one of Value or ValueFrom must be specified.
// +kubebuilder:validation:XValidation:message="exactly one of 'value' or 'valueFrom' must be set",rule="(has(self.value) && !has(self.valueFrom)) || (!has(self.value) && has(self.valueFrom))"
type ArtifactMediaTypeMappingProfile struct {
	// Value is the Profile resource to use.
	// +optional
	Value *v1beta1.ProfileSpec `json:"value,omitempty,omitzero" yaml:"value,omitempty,omitzero"`

	// ValueFrom is a reference to a Profile resource.
	// +optional
	ValueFrom v1.LocalObjectReference `json:"valueFrom,omitempty,omitzero" yaml:"valueFrom,omitempty,omitzero"`
}

type ArtifactMediaTypeMappingProfileStatus struct {
	// Name is the name of the Profile resource.
	Name string `json:"name,omitempty"`
	// Available indicates whether the Profile resource is available.
	Available bool `json:"available,omitempty"`
}

// ArtifactMediaTypeMappingStatus defines the observed state of ArtifactMediaTypeMapping.
type ArtifactMediaTypeMappingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ArtifactMediaTypeMapping resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Profile represents the status of the Profile resource associated with this ArtifactMediaTypeMapping.
	// +optional
	Profile *ArtifactMediaTypeMappingProfileStatus `json:"profile,omitempty,omitzero"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ArtifactMediaTypeMapping is the Schema for the artifactmediatypemappings API
type ArtifactMediaTypeMapping struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ArtifactMediaTypeMapping
	// +required
	Spec ArtifactMediaTypeMappingSpec `json:"spec"`

	// status defines the observed state of ArtifactMediaTypeMapping
	// +optional
	Status ArtifactMediaTypeMappingStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ArtifactMediaTypeMappingList contains a list of ArtifactMediaTypeMapping
type ArtifactMediaTypeMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactMediaTypeMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArtifactMediaTypeMapping{}, &ArtifactMediaTypeMappingList{})
}
