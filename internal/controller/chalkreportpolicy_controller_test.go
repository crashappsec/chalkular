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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/internal/policy"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
)

var _ = Describe("ChalkReportPolicy Controller", Ordered, func() {
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

		reportPolicy := &chalkularv1beta1.ChalkReportPolicy{}
		var policyCompiler *policy.Compiler

		BeforeAll(func() {
			var err error
			By("creating the custom resource for the Kind ChalkReportPolicy")

			err = k8sClient.Get(ctx, typeNamespacedName, reportPolicy)
			if err != nil && errors.IsNotFound(err) {
				resource := &chalkularv1beta1.ChalkReportPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: chalkularv1beta1.ChalkReportPolicySpec{
						MatchCondition: "report['_ACTION_ID'] == 'test'",
						Extraction: chalkularv1beta1.ChalkReportPolicyExtraction{
							Target: "{'identifier': 'testing', 'version': '1'}",
						},
						PipelineTemplate: ocularv1beta1.PipelineTemplate{
							// ObjectMeta: metav1.ObjectMeta{
							// 	Labels: map[string]string{
							// 		"chalk.ocular.crashoverride.run/test": "true",
							// 	},
							// },
							Spec: ocularv1beta1.PipelineSpec{
								ProfileRef: ocularv1beta1.ParameterizedLocalObjectReference{
									Name: profileName,
								},
								DownloaderRef: ocularv1beta1.ParameterizedLocalObjectReference{
									Name: downloaderName,
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("creating policy compiler")
			policyCompiler, err = policy.NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterAll(func() {
			var err error
			resource := &chalkularv1beta1.ChalkReportPolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ChalkReportPolicy")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

		})
		It("should add the cache finalizer to the resource", func() {
			By("Reconciling the created chalk report policy")
			controllerReconciler := &ChalkReportPolicyReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				PolicyCompiler: policyCompiler,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularv1beta1.ChalkReportPolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Finalizers).To(ContainElement(policyCacheFinalizer))
		})
		It("should set the downloader and profile status to false when not available", func() {
			By("Reconciling the created meidatypepolicy when both dont exist")
			controllerReconciler := &ChalkReportPolicyReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				PolicyCompiler: policyCompiler,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularv1beta1.ChalkReportPolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.ProfileValid).To(BeFalse())
			Expect(resource.Status.DownloaderValid).To(BeFalse())
		})
		It("should set the downloader and profile status to true when available", func() {
			By("Creating the profile and downloader, then reconciling the report policy")
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
						Containers: []ocularv1beta1.ConditionalContainer{
							{
								Container: v1.Container{
									Name:  "my-scanner",
									Image: "my-scanner:latest",
								},
							},
						},
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

			controllerReconciler := &ChalkReportPolicyReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				PolicyCompiler: policyCompiler,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &chalkularv1beta1.ChalkReportPolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.ProfileValid).To(BeTrue())
			Expect(resource.Status.DownloaderValid).To(BeTrue())

			Expect(k8sClient.Delete(ctx, profile)).To(Succeed())
			Expect(k8sClient.Delete(ctx, downloader)).To(Succeed())
		})

		It("should compile the policy and store it in the cache", func() {
			By("reconciling the object after status is updated")
			controllerReconciler := &ChalkReportPolicyReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				PolicyCompiler: policyCompiler,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			resource := &chalkularv1beta1.ChalkReportPolicy{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(resource.Status.Conditions, "Ready")).To(BeTrue(), "report policy not in Ready status")

		})
	})
})
