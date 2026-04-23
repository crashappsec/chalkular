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

var _ = Describe("Policy With no  Test", Ordered, func() {
	Context("When a policy is compiled with match and target", func() {
		policy := CompiledPolicy{}

		matchExpr := "report._TEST == 'PASS'"
		targetExpr := "{'identifier': chalkmark._OCULAR_TEST_IDENTIFIER, 'version': chalkmark._OCULAR_TEST_VERSION}"
		BeforeAll(func() {
			By("compiling the policy")
			compiler, err := NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
			policy.MatchCondition, err = compiler.program(matchExpr)
			Expect(err).To(Not(HaveOccurred()))
			policy.Target, err = compiler.program(targetExpr)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("match should evaluate to a boolean", func() {
			By("executing the CEL match expression with a false condition")
			matches, err := policy.Matches(map[string]any{
				"_TEST": "FAIL",
			}, map[string]any{})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
			By("executing the CEL match expression with a true condition")
			matches, err = policy.Matches(map[string]any{
				"_TEST": "PASS",
			}, map[string]any{})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
		It("target expression should successfully evalute to a v1beta1.Target", func() {
			By("the CEL target expression returning a target with an identifier")
			target, err := policy.ExtractTarget(map[string]any{}, map[string]any{
				"_OCULAR_TEST_IDENTIFIER": "testing-identifier",
				"_OCULAR_TEST_VERSION":    "testing-version",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(BeEquivalentTo(v1beta1.Target{
				Identifier: "testing-identifier",
				Version:    "testing-version",
			}))
		})
		It("should return nothing for downloader/profile params when not set", func() {
			By("attempting to extract parameters via policy")
			profile, downloader, err := policy.ExtractParameters(map[string]any{}, map[string]any{})
			Expect(err).NotTo(HaveOccurred())
			Expect(profile).To(BeEmpty())
			Expect(downloader).To(BeEmpty())

		})
	})
})
