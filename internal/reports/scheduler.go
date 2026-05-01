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

	"github.com/crashappsec/chalkular/api/chalk"
	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/internal/policy"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	"github.com/crashappsec/ocular/pkg/generated/clientset"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type ChalkReport = map[string]any
type ChalkMark = map[string]any

type event struct {
	Report ChalkReport
	Result SchedulerResult
}

type eventBus = chan event

type Scheduler struct {
	eventBus       eventBus
	ocularCS       *clientset.Clientset
	mgrClient      client.Client
	policyCompiler *policy.Compiler
}

func NewScheduler(mgrClient client.Client, cfg *rest.Config, policyCompiler *policy.Compiler) (*Scheduler, error) {
	e := make(eventBus)

	ocularCS, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	scheduler := &Scheduler{
		eventBus:       e,
		ocularCS:       ocularCS,
		mgrClient:      mgrClient,
		policyCompiler: policyCompiler,
	}

	return scheduler, nil
}

func (s *Scheduler) GetClient() *SchedulerClient {
	return &SchedulerClient{
		eventBus: s.eventBus,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	l := logf.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-s.eventBus:
			report := event.Report
			actionID, exist := report[chalk.KeyActionID]
			actionIDStr, valid := actionID.(string)
			if !exist || !valid {
				l.Error(fmt.Errorf("missing or invalid key \"%s\" found in report", chalk.KeyActionID), "action ID string was not found for report")
				event.Result <- fmt.Errorf("invalid chalk report, missing or invalid key %s found", chalk.KeyActionID)
				close(event.Result)
				continue
			}
			err := s.createPipelinesForChalkmarks(ctx, actionIDStr, report)
			if err != nil {
				l.Error(err, "failures reported for pipelines created from report", "action-id", actionIDStr)
			}
			event.Result <- err
			close(event.Result)
		}
	}
}

// scheduleAnalysis creates and submits pipelines for scanning the given artifact
// in the given namespace.
func (s *Scheduler) createPipelinesForChalkmarks(ctx context.Context, actionID string, report map[string]any) error {
	chalks, err := extractChalksFromReport(ctx, actionID, report)
	if err != nil {
		return fmt.Errorf("failed to extract chalk marks from report: %w", err)
	}
	var merr *multierror.Error
	for _, chalkmark := range chalks {
		pipelines, err := s.createPipelinesForChalkMark(ctx, actionID, report, chalkmark)
		if err != nil {
			merr = multierror.Append(merr, err)
		}

		for _, pipeline := range pipelines {
			_, err = s.ocularCS.ApiV1beta1().Pipelines(pipeline.Namespace).
				Create(ctx, pipeline, metav1.CreateOptions{})
			if err != nil {
				merr = multierror.Append(merr, fmt.Errorf("unable to create pipeline %s in namespace %s: %w", pipeline.Name, pipeline.Namespace, err))
			}
		}

	}
	return merr.ErrorOrNil()
}

func extractChalksFromReport(ctx context.Context, actionID string, report ChalkReport) ([]ChalkMark, error) {
	l := logf.FromContext(ctx).WithValues("actionID", actionID)
	chalksJSON, exist := report[chalk.KeyChalks]
	if !exist {
		return nil, nil
	}
	chalksList, valid := chalksJSON.([]any)
	if !valid {
		return nil, fmt.Errorf("invalid type for '_CHALKS' key, expected list %T", chalksJSON)
	}

	var chalks []ChalkMark
	for _, c := range chalksList {
		chalkmark, valid := c.(ChalkMark)
		if !valid {
			l.Info("invalid type for chalk mark, skipping", "chalkmark-type", fmt.Sprintf("%T", c))
			continue
		}
		chalks = append(chalks, chalkmark)
	}
	return chalks, nil
}

func (s *Scheduler) createPipelinesForChalkMark(ctx context.Context, actionID string, report ChalkReport, chalkmark ChalkMark) ([]*ocularv1beta1.Pipeline, error) {
	l := logf.FromContext(ctx).WithValues("actionID", actionID)
	var (
		pipelines []*ocularv1beta1.Pipeline
		policies  = &chalkularv1beta1.ChalkReportPolicyList{}
	)

	if err := s.mgrClient.List(ctx, policies); err != nil {
		return nil, fmt.Errorf("unable to list chalk report policies: %w", err)
	}

	var merr *multierror.Error
	for _, reportPolicy := range policies.Items {
		policyLogger := l.WithValues("policy", reportPolicy.Name, "namespace", reportPolicy.Namespace)
		if !reportPolicy.Status.ProfileValid {
			policyLogger.Info("skipping policy, profile unavailable")
			continue
		}
		if !reportPolicy.Status.DownloaderValid {
			policyLogger.Info("skipping policy, downloader unavailable")
			continue
		}

		if !meta.IsStatusConditionTrue(reportPolicy.Status.Conditions, "Ready") {
			policyLogger.Info("skipping policy, not in 'Ready' condition")
			continue
		}

		programs, err := s.policyCompiler.Get(&reportPolicy)
		if err != nil {
			policyLogger.Error(err, "unable to get compiled expressions for policy, skipping")
			continue
		}
		matches, err := programs.Matches(report, chalkmark)
		if err != nil {
			policyLogger.Error(err, "failed to run match expresssion, skipping")
			continue
		}
		if !matches {
			policyLogger.Info("policy match returned false")
			continue
		}

		target, err := programs.ExtractTarget(report, chalkmark)
		if err != nil {
			policyLogger.Error(err, "failed to evalutate policy target, skipping")
			merr = multierror.Append(merr, err)
			continue
		}

		profileParams, dlParams, err := programs.ExtractParameters(report, chalkmark)
		if err != nil {
			policyLogger.Error(err, "failed to evalutate policy parameters, skipping")
			merr = multierror.Append(merr, err)
			continue
		}

		pipeline := &ocularv1beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "chalkular-",
				Namespace:    reportPolicy.Namespace,
				Annotations:  make(map[string]string),
				Labels:       make(map[string]string),
			},
		}
		pipelineTemplate := reportPolicy.Spec.PipelineTemplate
		maps.Copy(pipeline.Labels, pipelineTemplate.Labels)
		maps.Copy(pipeline.Annotations, pipelineTemplate.Annotations)
		pipelineTemplate.Spec.DeepCopyInto(&pipeline.Spec)

		pipeline.Spec.Target = target
		pipeline.Spec.DownloaderRef.Parameters = append(pipeline.Spec.DownloaderRef.Parameters, dlParams...)
		pipeline.Spec.ProfileRef.Parameters = append(pipeline.Spec.ProfileRef.Parameters, profileParams...)

		pipelines = append(pipelines, pipeline)
	}

	l.Info(fmt.Sprintf("generated %d pipelines for chalk mark", len(pipelines)), "pipelines", len(pipelines))
	return pipelines, merr.ErrorOrNil()
}
