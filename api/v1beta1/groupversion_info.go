// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

// Package v1beta1 contains API Schema definitions for the chalk.ocular.crashoverride.run v1beta1 API group.
// +kubebuilder:object:generate=true
// +groupName=chalk.ocular.crashoverride.run
package v1beta1

import (
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "chalk.ocular.crashoverride.run"
	Version = "v1beta1"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	// SchemeGroupVersion is group version used to register these objects.
	// It is the same as GroupVersion and provided for legacy compatibility.
	SchemeGroupVersion = GroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ChalkReportPolicy{}, &ChalkReportPolicyList{},
	)

	scheme.AddKnownTypes(ocularv1beta1.SchemeGroupVersion,
		&ocularv1beta1.Pipeline{}, &ocularv1beta1.PipelineList{},
		&ocularv1beta1.Profile{}, &ocularv1beta1.ProfileList{},
		&ocularv1beta1.Downloader{}, &ocularv1beta1.DownloaderList{},
		&ocularv1beta1.Uploader{}, &ocularv1beta1.UploaderList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
