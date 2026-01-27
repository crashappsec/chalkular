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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chalkularocularcrashoverriderunv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
)

var _ = Describe("ArtifactMediaTypeMapping Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName   = "test-resource"
			profileName    = "test-profile"
			downloaderName = "test-downloader"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		// profile := &ocularcrashoverriderunv1beta1.Profile{}
		// downloader := &ocularcrashoverriderunv1beta1.Downloader{}
		artifactmediatypemapping := &chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping{}

		BeforeEach(func() {
			var err error
			By("creating the custom resource for the Kind ArtifactMediaTypeMapping")

			// err = k8sClient.Get(ctx, typeNamespacedName, profile)
			// if err != nil && errors.IsNotFound(err) {
			// 	resource := &ocularcrashoverriderunv1beta1.Profile{
			// 		ObjectMeta: metav1.ObjectMeta{
			// 			Name:      profileName,
			// 			Namespace: "default",
			// 		},
			// 		Spec: ocularcrashoverriderunv1beta1.ProfileSpec{
			// 			Containers: []v1.Container{{
			// 				Name:  "scanner",
			// 				Image: "my-scanner:latest",
			// 			}},
			// 		},
			// 	}
			// 	Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			// }

			// err = k8sClient.Get(ctx, typeNamespacedName, downloader)
			// if err != nil && errors.IsNotFound(err) {
			// 	resource := &ocularcrashoverriderunv1beta1.Downloader{
			// 		ObjectMeta: metav1.ObjectMeta{
			// 			Name:      downloaderName,
			// 			Namespace: "default",
			// 		},
			// 		Spec: ocularcrashoverriderunv1beta1.DownloaderSpec{
			// 			Container: v1.Container{
			// 				Name:  "scanner",
			// 				Image: "my-scanner:latest",
			// 			},
			// 		},
			// 	}
			// 	Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			// }

			err = k8sClient.Get(ctx, typeNamespacedName, artifactmediatypemapping)
			if err != nil && errors.IsNotFound(err) {
				resource := &chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMappingSpec{
						MediaTypes: []string{
							"my.custom.mediaType/v1beta1",
						},
						Profile: chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMappingProfile{
							ValueFrom: v1.ObjectReference{
								Name: profileName,
							},
						},
						Downloader: chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMappingDownloader{
							ValueFrom: v1.ObjectReference{
								Name: downloaderName,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			var err error
			// profileResource := &ocularcrashoverriderunv1beta1.Profile{}
			// err = k8sClient.Get(ctx, typeNamespacedName, profileResource)
			// Expect(err).NotTo(HaveOccurred())

			// downloaderResource := &ocularcrashoverriderunv1beta1.Downloader{}
			// err = k8sClient.Get(ctx, typeNamespacedName, downloaderResource)
			// Expect(err).NotTo(HaveOccurred())

			resource := &chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ArtifactMediaTypeMapping")
			// Expect(k8sClient.Delete(ctx, profileResource)).To(Succeed())
			// Expect(k8sClient.Delete(ctx, downloaderResource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArtifactMediaTypeMappingReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularocularcrashoverriderunv1beta1.ArtifactMediaTypeMapping{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.Profile.Available).To(BeFalse())
			Expect(resource.Status.Downloader.Available).To(BeFalse())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
