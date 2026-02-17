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

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
)

var _ = Describe("MediaTypePolicy Controller", func() {
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

		profileNamespacedName := types.NamespacedName{
			Name:      profileName,
			Namespace: "default",
		}
		downloaderNamespacedName := types.NamespacedName{
			Name:      downloaderName,
			Namespace: "default",
		}

		mediatypepolicy := &chalkularv1beta1.MediaTypePolicy{}

		BeforeEach(func() {
			var err error
			By("creating the custom resource for the Kind MediaTypePolicy")

			err = k8sClient.Get(ctx, typeNamespacedName, mediatypepolicy)
			if err != nil && errors.IsNotFound(err) {
				resource := &chalkularv1beta1.MediaTypePolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: chalkularv1beta1.MediaTypePolicySpec{
						MediaTypes: []string{
							"my.custom.mediaType/v1beta1",
						},
						PipelineTemplate: ocularv1beta1.PipelineTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"chalk.ocular.crashoverride.run/test": "true",
								},
							},
							Spec: ocularv1beta1.PipelineSpec{
								ProfileRef: v1.ObjectReference{
									Name: profileName,
								},
								DownloaderRef: ocularv1beta1.ParameterizedObjectReference{
									ObjectReference: v1.ObjectReference{
										Name: downloaderName,
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			var err error

			resource := &chalkularv1beta1.MediaTypePolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance MediaTypePolicy")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

		})
		It("should set the downloader and profile status to false when not available", func() {
			By("Reconciling the created meidatypepolicy when both dont exist")
			controllerReconciler := &MediaTypePolicyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularv1beta1.MediaTypePolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.Profile.Available).To(BeFalse())
			Expect(resource.Status.Downloader.Available).To(BeFalse())
		})
		It("should set the downloader and profile status to true when available", func() {
			By("Creating the profile and downloader, then reconciling the mediatypepolicy")
			profile := &ocularv1beta1.Profile{}
			downloader := &ocularv1beta1.Downloader{}
			err := k8sClient.Get(ctx, profileNamespacedName, profile)
			if err != nil && errors.IsNotFound(err) {
				profile = &ocularv1beta1.Profile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      profileName,
						Namespace: "default",
					},
					Spec: ocularv1beta1.ProfileSpec{
						Containers: []v1.Container{{
							Name:  "my-scanner",
							Image: "my-scanner:latest",
						}},
					},
				}
				Expect(k8sClient.Create(ctx, profile)).To(Succeed())
				err = nil
			}
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, downloaderNamespacedName, downloader)
			if err != nil && errors.IsNotFound(err) {
				downloader = &ocularv1beta1.Downloader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      downloaderName,
						Namespace: "default",
					},
					Spec: ocularv1beta1.DownloaderSpec{
						Container: v1.Container{
							Name:  "my-downloader",
							Image: "my-downloader:latest",
						},
					},
				}
				Expect(k8sClient.Create(ctx, downloader)).To(Succeed())
				err = nil
			}
			Expect(err).NotTo(HaveOccurred())

			controllerReconciler := &MediaTypePolicyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularv1beta1.MediaTypePolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.Profile.Available).To(BeTrue())
			Expect(resource.Status.Downloader.Available).To(BeTrue())

			Expect(k8sClient.Delete(ctx, profile)).To(Succeed())
			Expect(k8sClient.Delete(ctx, downloader)).To(Succeed())
		})
	})
})
