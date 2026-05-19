// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.
package policy

import (
	"github.com/crashappsec/ocular/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Standard Policy Test", Ordered, func() {
	Context("When a policy is compiled with match and target", func() {
		policy := CompiledPolicy{}
		var compiler *Compiler

		matchExpr := "report._TEST == 'PASS'"
		targetExprSingleton := "{'identifier': report._OCULAR_TEST_IDENTIFIER, 'version': report._OCULAR_TEST_VERSION}"
		targetExprList := "report.chalks.map(c, {'identifier': c.ident})"
		BeforeAll(func() {
			By("compiling the policy")
			var err error
			compiler, err = NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("match should evaluate to a boolean", func() {
			By("executing the CEL match expression with a false condition")
			var err error
			policy.MatchCondition, err = compiler.program(matchExpr)
			Expect(err).To(Not(HaveOccurred()))
			matches, err := policy.Matches(map[string]any{
				"_TEST": "FAIL",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
			By("executing the CEL match expression with a true condition")
			matches, err = policy.Matches(map[string]any{
				"_TEST": "PASS",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
		It("single target expression should successfully evalute to a []v1beta1.Target", func() {
			By("the CEL target expression returning a target with an identifier")
			var err error
			policy.Target, err = compiler.program(targetExprSingleton)
			Expect(err).To(Not(HaveOccurred()))
			target, err := policy.ExtractTargets(map[string]any{
				"_OCULAR_TEST_IDENTIFIER": "testing-identifier",
				"_OCULAR_TEST_VERSION":    "testing-version",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(BeEquivalentTo([]v1beta1.Target{{
				Identifier: "testing-identifier",
				Version:    "testing-version",
			}}))
		})
		It("list of target expression should successfully evalute to a []v1beta1.Target", func() {
			By("the CEL target expression returning a target with an identifier")
			var err error
			policy.Target, err = compiler.program(targetExprList)
			Expect(err).To(Not(HaveOccurred()))
			target, err := policy.ExtractTargets(map[string]any{
				"chalks": []map[string]any{
					{"ident": "test1"},
					{"ident": "test2"},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(BeEquivalentTo([]v1beta1.Target{{
				Identifier: "test1",
			}, {
				Identifier: "test2",
			}}))
		})
		It(" ", func() {
			By("the CEL target expression returning a target with an identifier")
			var err error
			policy.Target, err = compiler.program(targetExprList)
			Expect(err).To(Not(HaveOccurred()))
			target, err := policy.ExtractTargets(map[string]any{
				"chalks": []map[string]any{
					{"ident": "test1"},
					{"ident": "test2"},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(BeEquivalentTo([]v1beta1.Target{{
				Identifier: "test1",
			}, {
				Identifier: "test2",
			}}))
		})
		It("should return nothing for downloader/profile params when not set", func() {
			By("attempting to extract parameters via policy")
			profile, downloader, err := policy.ExtractParameters(map[string]any{})
			Expect(err).NotTo(HaveOccurred())
			Expect(profile).To(BeEmpty())
			Expect(downloader).To(BeEmpty())
		})
	})
})

var _ = Describe("Chalk report '_REPO_DIGESTS' Policy Test", Ordered, func() {
	Context("Policy should be able to extract ", func() {
		policy := CompiledPolicy{}
		var compiler *Compiler

		matchExpr := "report._CHALKS.exists(c, c._OP_ARTIFACT_TYPE == \"Docker Image\")"
		targetExpression := `report._CHALKS.map(c, c._OP_ARTIFACT_TYPE == "Docker Image",
			  c._REPO_DIGESTS.map(d, 
				  c._REPO_DIGESTS[d].map(r,
					  c._REPO_DIGESTS[d][r].map(h, 
					    { 'identifier': d + "/" + r,
						  'version': h }
					  )
				  )
			  )
		  ).flatten(3)
`
		marshalledReport := map[string]any{
			"_CHALKS": []map[string]any{
				{
					"_OP_ARTIFACT_TYPE": "Docker Image",
					"_REPO_DIGESTS": map[string]any{
						"docker.io": map[string]any{
							"my-org/my-image": []string{
								"version-1",
								"version-2",
							},
							"my-org/my-image-2": []string{
								"version-1",
							},
						},
						"gcr.io": map[string]any{
							"my-org/my-image": []string{
								"version-3",
							},
						},
					},
				},
				{
					"_OP_ARTIFACT_TYPE": "Docker Image",
					"_REPO_DIGESTS": map[string]any{
						"docker.io": map[string]any{
							"my-org/my-image-3": []string{
								"version-1",
							},
						},
					},
				},
			},
		}
		BeforeAll(func() {
			By("compiling the policy")
			var err error
			compiler, err = NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
			policy.Target, err = compiler.program(targetExpression)
			Expect(err).To(Not(HaveOccurred()))
			policy.MatchCondition, err = compiler.program(matchExpr)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("match should evaluate to a boolean", func() {
			By("executing the CEL match expression with a true condition")
			matches, err := policy.Matches(marshalledReport)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
		It("target expression should successfully evalute to a []v1beta1.Target", func() {
			By("the CEL target expression using macros")
			targets, err := policy.ExtractTargets(marshalledReport)
			Expect(err).NotTo(HaveOccurred())
			Expect(targets).To(HaveLen(5))
			Expect(targets).To(ContainElements(
				v1beta1.Target{
					Identifier: "docker.io/my-org/my-image",
					Version:    "version-1",
				},
				v1beta1.Target{
					Identifier: "docker.io/my-org/my-image",
					Version:    "version-2",
				},
				v1beta1.Target{
					Identifier: "docker.io/my-org/my-image-2",
					Version:    "version-1",
				},
				v1beta1.Target{
					Identifier: "gcr.io/my-org/my-image",
					Version:    "version-3",
				},
				v1beta1.Target{
					Identifier: "docker.io/my-org/my-image-3",
					Version:    "version-1",
				},
			))
		})
	})
})
