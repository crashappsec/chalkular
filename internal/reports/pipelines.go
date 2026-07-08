// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package reports

import (
	"context"
	"fmt"
	"maps"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type policyGeneratedPipelines struct {
	report    chalk.Report
	actionID  string
	policy    chalkularv1beta1.ChalkReportPolicy
	pipelines []*ocularv1beta1.Pipeline
}

func (s *Scheduler) createPipelinesForReport(ctx context.Context, policies []chalkularv1beta1.ChalkReportPolicy, actionID string, report chalk.Report) []policyGeneratedPipelines {
	l := logf.FromContext(ctx)

	var generatedPipelines []policyGeneratedPipelines
	for _, policy := range policies {
		policyLogger := l.WithValues("policy", policy.Name, "namespace", policy.Namespace)

		if err := s.isDownloaderValid(ctx, &policy); err != nil {
			policyLogger.Info("skipping policy, unable to validate downloader: %w", err)
		}

		if err := s.isProfileValid(ctx, &policy); err != nil {
			policyLogger.Info("skipping policy, unable to validate profile: %w", err)
		}

		if !meta.IsStatusConditionTrue(policy.Status.Conditions, "Ready") {
			policyLogger.Info("skipping policy, not in 'Ready' condition")
			continue
		}

		p, err := s.policyCompiler.Get(&policy)
		if err != nil {
			policyLogger.Error(err, "unable to get compiled expressions for policy, skipping")
			continue
		}
		matches, err := p.Matches(report)
		if err != nil {
			policyLogger.Error(err, "failed to run match expresssion, skipping")
			s.recorder.Eventf(&policy, nil,
				corev1.EventTypeWarning,
				"PolicyEvalFailed",
				"MatchConditionEval",
				"failed to evaluate match condition for action %s: %s", actionID, err)
			continue
		}
		if !matches {
			policyLogger.Info("policy match returned false")
			continue
		}

		pipelineTemplate := policy.Spec.PipelineTemplate

		values, err := p.Extract(report)
		if err != nil {
			policyLogger.Error(err, "failed to extract pipeline values")
			s.recorder.Eventf(&policy, nil,
				corev1.EventTypeWarning,
				"PolicyExtractFailed",
				"ExtractPipelineValues",
				"failed to extract pipeline values for action %s: %s", actionID, err)
			continue
		}

		if s.maxPipelinesPerPolicy > 0 && len(values) > s.maxPipelinesPerPolicy {
			policyLogger.Error(err, "policy generated too many pipelines")
			s.recorder.Eventf(&policy, nil,
				corev1.EventTypeWarning,
				"TooManyPipelinesGenerated",
				"ExtractPipelineValues",
				"more than %d pipelines were generated for action %s: exceededs limit of %d", len(values), actionID, s.maxPipelinesPerPolicy)
			continue
		}

		var pipelines []*ocularv1beta1.Pipeline
		policyLogger.Info(fmt.Sprintf("policy generated %d values", len(values)), "values", len(values))
		for _, vs := range values {
			pipeline := &ocularv1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("chalkular-%s-", actionID),
					Namespace:    policy.Namespace,
					Annotations:  make(map[string]string),
					Labels:       make(map[string]string),
				},
			}
			maps.Copy(pipeline.Labels, pipelineTemplate.Labels)
			maps.Copy(pipeline.Annotations, pipelineTemplate.Annotations)
			pipelineTemplate.Spec.DeepCopyInto(&pipeline.Spec)

			pipeline.Spec.DownloaderRef.Parameters = append(pipeline.Spec.DownloaderRef.Parameters, vs.DownloaderParams...)
			pipeline.Spec.ProfileRef.Parameters = append(pipeline.Spec.ProfileRef.Parameters, vs.ProfileParams...)
			pipeline.Spec.Target = vs.Target

			pipeline.Labels[schedulerLabel] = schedulerValue
			pipelines = append(pipelines, pipeline)
		}
		generatedPipelines = append(generatedPipelines, policyGeneratedPipelines{
			report:    report,
			actionID:  actionID,
			policy:    policy,
			pipelines: pipelines,
		})

	}

	l.Info(fmt.Sprintf("generated %d pipelines for chalk report", len(generatedPipelines)), "pipelines", len(generatedPipelines))
	return generatedPipelines
}

func (s *Scheduler) isProfileValid(ctx context.Context, reportPolicy *chalkularv1beta1.ChalkReportPolicy) error {
	found := &ocularv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: reportPolicy.Namespace,
			Name:      reportPolicy.Spec.PipelineTemplate.Spec.ProfileRef.Name,
		},
	}
	return s.mgrClient.Get(ctx, client.ObjectKey{Namespace: reportPolicy.Namespace, Name: reportPolicy.Spec.PipelineTemplate.Spec.ProfileRef.Name}, found)
}

func (s *Scheduler) isDownloaderValid(ctx context.Context, reportPolicy *chalkularv1beta1.ChalkReportPolicy) error {
	downloaderRef := reportPolicy.Spec.PipelineTemplate.Spec.DownloaderRef
	switch downloaderRef.Kind {
	case "", "Downloader":
		found := &ocularv1beta1.Downloader{}
		return s.mgrClient.Get(ctx, client.ObjectKey{Namespace: reportPolicy.Namespace, Name: reportPolicy.Spec.PipelineTemplate.Spec.DownloaderRef.Name}, found)
	case "ClusterDownloader":
		found := &ocularv1beta1.ClusterDownloader{}
		return s.mgrClient.Get(ctx, client.ObjectKey{Name: downloaderRef.Name}, found)
	default:
		return fmt.Errorf("unknown downloader kind: %s", downloaderRef.Kind)
	}
}

func isPipelineActiveIndexer(o client.Object) []string {
	p := o.(*ocularv1beta1.Pipeline)
	if p.Status.CompletionTime == nil {
		return []string{"true"}
	}
	return []string{"false"}
}

func (s *Scheduler) countActivePipelines(ctx context.Context) (int, error) {
	list := &ocularv1beta1.PipelineList{}
	err := s.mgrClient.List(ctx, list,
		client.MatchingLabels{schedulerLabel: schedulerValue},
		client.MatchingFields{"status.active": "true"},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to list pipelines: %w", err)
	}
	return len(list.Items), nil
}
