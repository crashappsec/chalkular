// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package v1beta1

import (
	"github.com/crashappsec/ocular/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MediaTypePolicySpec defines the desired state of MediaTypePolicy.
// The spec includes the media types to watch for, the [github.com/crashappsec/ocular/api/v1beta1.Profile]
// that should be used, and additional options for the created [github.com/crashappsec/ocular/api/v1beta1.Pipeline]
type MediaTypePolicySpec struct {
	// MediaTypes is the media type of the artifact.
	// +kubebuilder:validation:items:Pattern=`^[\w.-]+/[\w\.\-\+]+$`
	// +required
	MediaTypes []string `json:"mediaTypes"`

	// PipelineTemplate is the specification of the desired behavior of the
	// pipeline created for resources
	// The target field will be set to the target read from
	// the listener. If not set, the downloader will default
	// to the chalkular-artifacts cluster downloader.
	PipelineTemplate v1beta1.PipelineTemplate `json:"pipelineTemplate"`
}

// MediaTypePolicyProfileStatus represents the status of the managed or referenced profile
type MediaTypePolicyProfileStatus struct {
	// Available indicates whether the Profile resource is available.
	// +optional
	Available bool `json:"available"`
}

// MediaTypePolicyDownloaderStatus represents the status of the managed or referenced downloader
type MediaTypePolicyDownloaderStatus struct {
	// Available indicates whether the Downloader resource is available.
	// +optional
	Available bool `json:"available"`
}

// MediaTypePolicyStatus defines the observed state of MediaTypePolicy.
type MediaTypePolicyStatus struct {
	// conditions represent the current state of the MediaTypePolicy resource.
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

	// Profile represents the status of the Profile resource associated with this MediaTypePolicy.
	// +optional
	Profile MediaTypePolicyProfileStatus `json:"profile,omitempty"`

	// Downloader represents the status of the Downloader resource associated with this MediaTypePolicy.
	// +optional
	Downloader MediaTypePolicyDownloaderStatus `json:"downloader,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// MediaTypePolicy represents a mapping of OCI media types to the desired
// [github.com/crashappsec/ocular/api/v1beta1.Pipeline] that should be created
// when a container image is registered with Chalkular.
// See [MediaTypePolicySpec] for the full list of available configuration options
type MediaTypePolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of MediaTypePolicy
	// +required
	Spec MediaTypePolicySpec `json:"spec"`

	// status defines the observed state of MediaTypePolicy
	// +optional
	Status MediaTypePolicyStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// MediaTypePolicyList contains a list of MediaTypePolicy
type MediaTypePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MediaTypePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MediaTypePolicy{}, &MediaTypePolicyList{})
}
