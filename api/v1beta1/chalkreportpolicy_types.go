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

// ChalkReportPolicySpec defines the desired state of ChalkReportPolicy.
// The policy uses CEL expressions to both match the policy to an incoming
// Chalk mark event & to dervice parameters for resulting pipelines.
// CEL expressions will have the following variables available to them
// - `chalkmark`: A chark mark object for the artifact to scan
// - `report`: The chalk report the chalk was received from
// (see https://chalkproject.io/docs/glossary/ for more info)
type ChalkReportPolicySpec struct {
	// MatchCondition is the CEL expression to
	// match on incoming reports & chalk marks.
	// The expression should return a boolean,
	// where `true` will result in a "match"
	// +required
	MatchCondition string `json:"matchCondition" description:"boolean CEL expression to indicate if the policy matches"`

	// Extraction contains the CEL expressions for extracting
	// inputs for the created pipeline.
	// +required
	Extraction ChalkReportPolicyExtraction `json:"extraction"`

	// PipelineTemplate is the specification of the desired behavior of the
	// pipeline created for resources
	// The target field will be set to the target read from
	// the listener. If not set, the downloader will default
	// to the chalkular-artifacts cluster downloader.
	// +required
	PipelineTemplate v1beta1.PipelineTemplate `json:"pipelineTemplate"`
}

type ChalkReportPolicyExtraction struct {
	// Target is a CEL expression to extract the
	// [v1beta1.Target] from the chalk report.
	// The expression should return a string map
	// with two keys: 'identifier' and (optionally) 'version'
	// +required
	Target string `json:"target"`
	// DownloaderParams is a CEL expression to extract
	// dynamic parameters from the chalk report to
	// apply to the downloader. The expression should
	// return a string map.
	// +optional
	DownloaderParams *string `json:"downloaderParams"`
	// ProfileParams is a CEL expression to extract
	// dynamic parameters from the chalk report to
	// apply to the profile. The expression should
	// return a string map.
	// +optional
	ProfileParams *string `json:"profileParams"`
}

// ChalkReportPolicyStatus defines the observed state of ChalkReportPolicy.
type ChalkReportPolicyStatus struct {
	// conditions represent the current state of the ChalkReportPolicy resource.
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

	// ProfileValid will be true if the referenced profile in the pipeline
	// template is valid, and false otherwise
	// +optional
	ProfileValid bool `json:"profileValid"`

	// DownloaderValid will be true if the referenced downloader in the pipeline
	// template is valid, and false otherwise
	// +optional
	DownloaderValid bool `json:"downloaderValid"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ChalkReportPolicy is a policy evalutor for creating
// [github.com/crashappsec/ocular/api/v1beta1.Pipeline] resources that
// can scan chalk marked artifacts.
// See [CahlkReportPolicySpec] for the full list of available configuration options
type ChalkReportPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ChalkReportPolicy
	// +required
	Spec ChalkReportPolicySpec `json:"spec"`

	// status defines the observed state of ChalkReportPolicy
	// +optional
	Status ChalkReportPolicyStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ChalkReportPolicyList contains a list of ChalkReportPolicy
type ChalkReportPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChalkReportPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChalkReportPolicy{}, &ChalkReportPolicyList{})
}
