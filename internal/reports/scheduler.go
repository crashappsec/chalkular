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
	"time"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
	"github.com/crashappsec/chalkular/internal/policy"
	ocularv1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	schedulerLabel = "chalk.ocular.crashoverride.run/scheduled-by"
	schedulerValue = "chalkular-controller"
)

type Scheduler struct {
	eventBus                eventBus
	rejectPipelineThreshold int
	maxPipelinesPerPolicy   int

	mgrClient client.Client
	recorder  events.EventRecorder

	policyCompiler *policy.Compiler
}

func NewScheduler(mgr manager.Manager, policyCompiler *policy.Compiler, rejectPipelineThreshold, maxPipelinesPerPolicy int) (*Scheduler, error) {
	e := make(eventBus)

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&ocularv1beta1.Pipeline{},
		"status.active",
		isPipelineActiveIndexer,
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

func (s *Scheduler) GetClient() SchedulerClient {
	return &schedulerClient{
		eventBus: s.eventBus,
	}
}

var ErrPipelineThreshold = errors.New("rejecting report, active pipeline count at or above threshold")

func (s *Scheduler) Start(ctx context.Context) error {
	l := logf.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-s.eventBus:
			l.Info("chalk reports received, scheduling")
			reports := event.Reports

			if s.rejectPipelineThreshold > 0 {
				active, err := s.countActivePipelines(ctx)
				if err != nil {
					event.Result <- fmt.Errorf("unable to list active pipelines: %w", err)
					close(event.Result)
					continue
				}
				if active >= s.rejectPipelineThreshold {
					l.Info("rejecting reports, pipeline threshold hit")
					event.Result <- fmt.Errorf("%w: currently %d active which exceeds threshold of %d, rejecting event",
						ErrPipelineThreshold, active, s.rejectPipelineThreshold)
					close(event.Result)
					continue
				}
			}

			policies := &chalkularv1beta1.ChalkReportPolicyList{}
			if err := s.mgrClient.List(ctx, policies); err != nil {
				event.Result <- fmt.Errorf("unable to list chalk report policies: %w", err)
				close(event.Result)
			}

			// group generated pipelines by report + policy
			// so that we can write events to policies if templated pipeline
			// fails to be created
			var generatedPipelines []policyGeneratedPipelines
			for _, report := range reports {
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

				generated := s.createPipelinesForReport(reportCtx, policies.Items, actionIDStr, report)
				generatedPipelines = append(generatedPipelines, generated...)
			}

			// this is separate incase we fail to process one report,
			// we reject before pipelines are created in order to allow
			// the message to be requeued. If pipelines fail to be created
			// the errors will be logged to the poilicies
			// events and is considered an error with the policy,
			// not the report.
			var createdPipelines []*ocularv1beta1.Pipeline
			for _, g := range generatedPipelines {
				var policyPipelines []*ocularv1beta1.Pipeline
				for i, pipeline := range g.pipelines {
					err := s.mgrClient.Create(ctx, pipeline)
					if err != nil {
						l.Error(err, "unable to create pipeline for policy",
							"pipeline", pipeline.Name, "namespace", g.policy.Namespace, "policy", g.policy.Name)
						s.recorder.Eventf(&g.policy, nil,
							corev1.EventTypeWarning,
							"FailedToCreatePipeline",
							"CreatePipelineFromReport",
							"failed to generate pipeline (%d/%d) for report '%s': %s", i, len(g.pipelines), g.actionID, err)
					} else {
						policyPipelines = append(policyPipelines, pipeline)
					}
				}
				if len(policyPipelines) > 0 {
					s.recorder.Eventf(&g.policy, nil,
						corev1.EventTypeNormal,
						"PipelinesCreated",
						"CreatePipelineFromReport",
						"report '%s' created %d pipeline", g.actionID, len(createdPipelines))
					createdPipelines = append(createdPipelines, policyPipelines...)
				}

			}

			l.Info(fmt.Sprintf("created %d pipelines", len(createdPipelines)), "pipelines", len(createdPipelines))
			event.Result <- nil
			close(event.Result)
			// if we get here lets rest a second to not
			// overwhelm the api server
			time.Sleep(time.Second)
		}
	}
}
