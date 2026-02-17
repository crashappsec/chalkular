// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chalkularocularcrashoverriderunv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/ocular/api/v1beta1"
)

var _ = Describe("MediaTypePolicy Webhook", func() {
	var (
		obj       *chalkularocularcrashoverriderunv1beta1.MediaTypePolicy
		oldObj    *chalkularocularcrashoverriderunv1beta1.MediaTypePolicy
		validator MediaTypePolicyCustomValidator
		defaulter MediaTypePolicyCustomDefaulter
	)

	BeforeEach(func() {
		obj = &chalkularocularcrashoverriderunv1beta1.MediaTypePolicy{}
		oldObj = &chalkularocularcrashoverriderunv1beta1.MediaTypePolicy{}
		validator = MediaTypePolicyCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = MediaTypePolicyCustomDefaulter{
			downloader:     testClusterDownloader,
			downloaderKind: "ClusterDownloader",
		}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating MediaTypePolicy under Defaulting Webhook", func() {
		It("Should apply defaults when a required field is empty", func() {
			By("not setting the downloader ref for the pipeline template")
			obj.Spec.PipelineTemplate.Spec.DownloaderRef = v1beta1.ParameterizedObjectReference{}
			By("calling the Default method to apply defaults")
			Expect(defaulter.Default(ctx, obj)).ToNot(HaveOccurred())
			By("checking the cluster downloader is set")
			Expect(obj.Spec.PipelineTemplate.Spec.DownloaderRef.Name).To(Equal(testClusterDownloader))
			Expect(obj.Spec.PipelineTemplate.Spec.DownloaderRef.Kind).To(Equal("ClusterDownloader"))
		})
	})

	Context("When creating or updating MediaTypePolicy under Validating Webhook", func() {
		It("Should deny creation if target is set for pipeline", func() {
			By("setting the target")
			obj.Spec.PipelineTemplate.Spec.Target.Identifier = "test-identifier"
			By("setting at least one media type")
			obj.Spec.MediaTypes = []string{"test"}
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
		})

		It("Should deny creation if no media types are set", func() {
			By("not setting the target")
			obj.Spec.PipelineTemplate.Spec.Target = v1beta1.Target{}
			By("setting no mediatypes")
			obj.Spec.MediaTypes = []string{}
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
		})

		It("Should admit creation if all required fields are present", func() {
			By("not setting the downloader")
			obj.Spec.PipelineTemplate.Spec.Target = v1beta1.Target{}
			By("setting at least one media type")
			obj.Spec.MediaTypes = []string{"test"}
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		})

		It("Should validate updates correctly", func() {
			By("simulating a valid update scenario")
			oldObj.Spec.PipelineTemplate.Spec.Target = v1beta1.Target{}
			obj.Spec.PipelineTemplate.Spec.Target = v1beta1.Target{}
			oldObj.Spec.MediaTypes = []string{"test1"}
			obj.Spec.MediaTypes = []string{"test2"}
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		})
	})

})
