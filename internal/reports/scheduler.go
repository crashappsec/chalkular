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
	"errors"
	"fmt"
	"maps"
	"time"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
	"github.com/crashappsec/chalkular/internal/policy"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	schedulerLabel = "chalk.ocular.crashoverride.run/scheduled-by"
	schedulerValue = "chalkular-controller"
)

type event struct {
	Report chalk.Report
	Result SchedulerResult
}

type eventBus = chan event

type Scheduler struct {
	eventBus                eventBus
	rejectPipelineThreshold int
	maxPipelinesPerPolicy   int

	mgrClient client.Client
	recorder  events.EventRecorder

	policyCompiler *policy.Compiler
}

func isActiveIndexer(o client.Object) []string {
	p := o.(*ocularv1beta1.Pipeline)
	if p.Status.StartTime != nil && p.Status.CompletionTime == nil {
		return []string{"true"}
	}
	return []string{"false"}
}

func NewScheduler(mgr manager.Manager, policyCompiler *policy.Compiler, rejectPipelineThreshold, maxPipelinesPerPolicy int) (*Scheduler, error) {
	e := make(eventBus)

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&ocularv1beta1.Pipeline{},
		"status.active",
		isActiveIndexer,
	); err != nil {
		return nil, err
	}

	scheduler := &Scheduler{
		eventBus: e,

		maxPipelinesPerPolicy:   maxPipelinesPerPolicy,
		rejectPipelineThreshold: rejectPipelineThreshold,

		policyCompiler: policyCompiler,

		mgrClient: mgr.GetClient(),
		recorder:  mgr.GetEventRecorder("chalkular-report-scheduler"),
	}

	return scheduler, nil
}

func (s *Scheduler) GetClient() *SchedulerClient {
	return &SchedulerClient{
		eventBus: s.eventBus,
	}
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

var ErrPipelineThreshold = errors.New("rejecting report, active pipeline count at or above threshold")

func (s *Scheduler) Start(ctx context.Context) error {
	l := logf.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-s.eventBus:
			l.Info("chalk report received, scheduling")
			report := event.Report

			actionID, exist := report[chalk.KeyActionID]
			actionIDStr, valid := actionID.(string)
			if !exist || !valid {
				l.Error(fmt.Errorf("missing or invalid key \"%s\" found in report", chalk.KeyActionID), "action ID string was not found for report")
				event.Result <- fmt.Errorf("invalid chalk report, missing or invalid key %s found", chalk.KeyActionID)
				close(event.Result)
				continue
			}
			reportL := l.WithValues("action-id", actionID)
			reportCtx := logf.IntoContext(ctx, reportL)

			if s.rejectPipelineThreshold > 0 {
				active, err := s.countActivePipelines(reportCtx)
				if err != nil {
					event.Result <- fmt.Errorf("unable to list active pipelines: %w", err)
					close(event.Result)
					continue
				}
				if active >= s.rejectPipelineThreshold {
					reportL.Info("rejecting report, pipeline threshold hit")
					event.Result <- fmt.Errorf("%w: currently %d active which exceeds threshold of %d, rejecting",
						ErrPipelineThreshold, active, s.rejectPipelineThreshold)
					close(event.Result)
					continue
				}
			}

			pipelines, err := s.createPipelinesForReport(reportCtx, actionIDStr, report)
			if err != nil {
				reportL.Error(err, "failures reported for pipelines created from report", "action-id", actionIDStr)
			}
			var merr *multierror.Error
			for _, pipeline := range pipelines {
				if err = s.mgrClient.Create(reportCtx, pipeline); err != nil {
					reportL.Error(err, "unable to create pipeline", "pipeline-name", pipeline.Name, "pipeline-namespace", pipeline.Namespace)
					merr = multierror.Append(merr, fmt.Errorf("unable to create pipeline %s in namespace %s: %w", pipeline.Name, pipeline.Namespace, err))
				}
			}

			err = merr.ErrorOrNil()
			event.Result <- err
			close(event.Result)
			// if we get here lets rest a second to not
			// overwhelm the api server
			time.Sleep(time.Second)
		}
	}
}

func (s *Scheduler) createPipelinesForReport(ctx context.Context, actionID string, report chalk.Report) ([]*ocularv1beta1.Pipeline, error) {
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

		p, err := s.policyCompiler.Get(&reportPolicy)
		if err != nil {
			policyLogger.Error(err, "unable to get compiled expressions for policy, skipping")
			continue
		}
		matches, err := p.Matches(report)
		if err != nil {
			policyLogger.Error(err, "failed to run match expresssion, skipping")
			s.recorder.Eventf(&reportPolicy, nil,
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

		pipelineTemplate := reportPolicy.Spec.PipelineTemplate

		values, err := p.Extract(report)
		if err != nil {
			policyLogger.Error(err, "failed to extract pipeline values")
			s.recorder.Eventf(&reportPolicy, nil,
				corev1.EventTypeWarning,
				"PolicyExtractFailed",
				"ExtractPipelineValues",
				"failed to extract pipeline values for action %s: %s", actionID, err)
			continue
		}

		if s.maxPipelinesPerPolicy > 0 && len(values) > s.maxPipelinesPerPolicy {
			policyLogger.Error(err, "policy generated too many pipelines")
			s.recorder.Eventf(&reportPolicy, nil,
				corev1.EventTypeWarning,
				"TooManyPipelinesGenerated",
				"ExtractPipelineValues",
				"more than %d pipelines were generated for action %s: exceededs limit of %d", len(values), actionID, s.maxPipelinesPerPolicy)
			continue
		}

		policyLogger.Info(fmt.Sprintf("policy generated %d values", len(values)), "values", len(values))
		for _, vs := range values {
			pipeline := &ocularv1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "chalkular-",
					Namespace:    reportPolicy.Namespace,
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

	}

	l.Info(fmt.Sprintf("generated %d pipelines for chalk report", len(pipelines)), "pipelines", len(pipelines))
	return pipelines, merr.ErrorOrNil()
}
