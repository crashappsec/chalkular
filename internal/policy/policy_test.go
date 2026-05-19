// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.
package policy

import (
	"github.com/crashappsec/chalkular/api/v1beta1"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CompliledPolicy test", Ordered, func() {
	Context("policy match and target expressions", func() {
		policy := &v1beta1.ChalkReportPolicy{
			Spec: v1beta1.ChalkReportPolicySpec{
				MatchCondition: "report._MATCH",
				Extraction: v1beta1.ChalkReportPolicyExtraction{
					Target: "{'identifier': report._OCULAR_TEST_IDENTIFIER, 'version': report._OCULAR_TEST_VERSION}",
				},
			},
		}
		var compiled *CompiledPolicy
		BeforeAll(func() {
			By("compiling the policy")
			compiler, err := NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
			compiled, err = compiler.compile(policy)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("match should return an error when not a boolean", func() {
			By("executing the CEL match expression with a invalid payload")
			matches, err := compiled.Matches(map[string]any{
				"_MATCH": map[string]any{
					"complex": []string{"object"},
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("match should evaluate to a boolean", func() {
			By("executing the CEL match expression with a false condition")
			matches, err := compiled.Matches(map[string]any{
				"_MATCH": false,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
			By("executing the CEL match expression with a true condition")
			matches, err = compiled.Matches(map[string]any{
				"_MATCH": true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
		It("extract should fail if the target expression does not evalute", func() {
			By("Running the policy extraction on a payload")
			_, err := compiled.Extract(map[string]any{
				"_MATCH":                  true,
				"_OCULAR_TEST_IDENTIFIER": []string{"string", "list"},
				"_OCULAR_TEST_VERSION":    "testing-version",
			})
			Expect(err).To(HaveOccurred())
		})
		It("target expression should successfully evalute to a target", func() {
			By("Running the policy extraction on a payload")
			vals, err := compiled.Extract(map[string]any{
				"_MATCH":                  true,
				"_OCULAR_TEST_IDENTIFIER": "testing-identifier",
				"_OCULAR_TEST_VERSION":    "testing-version",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(vals).To(HaveLen(1))
			val := vals[0]
			Expect(val.Target).To(BeEquivalentTo(ocularv1beta1.Target{
				Identifier: "testing-identifier",
				Version:    "testing-version",
			}))
			Expect(val.DownloaderParams).To(BeEmpty())
			Expect(val.ProfileParams).To(BeEmpty())
		})
	})

	Context("policy parameter expressions", func() {
		policy := &v1beta1.ChalkReportPolicy{
			Spec: v1beta1.ChalkReportPolicySpec{
				MatchCondition: "true",
				Extraction: v1beta1.ChalkReportPolicyExtraction{
					Target:           "{'identifier': has(report._IDENTIFIER) ? report._IDENTIFIER : 'testing'}",
					ProfileParams:    new("{'ENABLED': report._ENABLED}"),
					DownloaderParams: new("report._DL_PARAMS"),
				},
			},
		}
		var compiled *CompiledPolicy
		BeforeAll(func() {
			By("compiling the policy")
			compiler, err := NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
			compiled, err = compiler.compile(policy)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("extract should return an error when parameter expression fails", func() {
			By("executing the extract expression with a valid payload")
			extract, err := compiled.Extract(map[string]any{
				"_ENABLED":   "YES",
				"_DL_PARAMS": map[string]string{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(extract).To(HaveLen(1))
			val := extract[0]
			Expect(val.DownloaderParams).To(BeEmpty())
			Expect(val.ProfileParams).To(HaveLen(1))
			Expect(val.ProfileParams).To(
				ContainElement(ocularv1beta1.ParameterSetting{
					Name:  "ENABLED",
					Value: "YES",
				}))

			By("getting an error using the wrong type for profile parameter value")
			_, err = compiled.Extract(map[string]any{
				"_ENABLED":   4,
				"_DL_PARAMS": map[string]string{},
			})
			Expect(err).To(HaveOccurred())

			By("getting an error using the wrong type for downloader params")
			_, err = compiled.Extract(map[string]any{
				"_ENABLED":   "YES",
				"_DL_PARAMS": []any{1, "test", true},
			})
			Expect(err).To(HaveOccurred())

		})
	})
	Context("for each policy expressions", func() {
		policy := &v1beta1.ChalkReportPolicy{
			Spec: v1beta1.ChalkReportPolicySpec{
				MatchCondition: "true",
				Extraction: v1beta1.ChalkReportPolicyExtraction{
					ForEach:          new("report._ITEMS"),
					Target:           "{'identifier': each._IDENTIFIER}",
					ProfileParams:    new("{'PROFILE_PARAM': each._PROFILE}"),
					DownloaderParams: new("{'DL_PARAM': report._DL}"),
				},
			},
		}
		var compiled *CompiledPolicy
		BeforeAll(func() {
			By("compiling the policy")
			compiler, err := NewCompiler(5)
			Expect(err).To(Not(HaveOccurred()))
			compiled, err = compiler.compile(policy)
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should fail if the for each does not return a list", func() {
			_, err := compiled.Extract(map[string]any{
				"_ITEMS": "string!",
				"_DL":    "constant",
			})
			Expect(err).To(HaveOccurred())
		})

		It("run the extraction for every item of the for each", func() {
			By("executing the extract expression with a valid payload")
			extract, err := compiled.Extract(map[string]any{
				"_ITEMS": []any{
					map[string]string{
						"_IDENTIFIER": "test1",
						"_PROFILE":    "string1",
					},
					map[string]any{
						"_IDENTIFIER": "test2",
						"_PROFILE":    "string2",
					},
					map[string]any{
						"_IDENTIFIER": "test3",
						"_PROFILE":    "string3",
					},
				},
				"_DL": "constant",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(extract).To(HaveLen(3))
			test1 := extract[0]
			Expect(test1.Target.Identifier).To(Equal("test1"))
			Expect(test1.DownloaderParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "DL_PARAM",
					Value: "constant",
				},
			}))
			Expect(test1.ProfileParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "PROFILE_PARAM",
					Value: "string1",
				},
			}))
			test2 := extract[1]
			Expect(test2.Target.Identifier).To(Equal("test2"))
			Expect(test2.ProfileParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "PROFILE_PARAM",
					Value: "string2",
				},
			}))
			Expect(test2.DownloaderParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "DL_PARAM",
					Value: "constant",
				},
			}))
			test3 := extract[2]
			Expect(test3.Target.Identifier).To(Equal("test3"))
			Expect(test3.ProfileParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "PROFILE_PARAM",
					Value: "string3",
				},
			}))
			Expect(test3.DownloaderParams).To(BeEquivalentTo([]ocularv1beta1.ParameterSetting{
				{
					Name:  "DL_PARAM",
					Value: "constant",
				},
			}))
		})
	})
})
