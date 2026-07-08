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
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	schedulerLabel = "chalk.ocular.crashoverride.run/scheduled-by"
	schedulerValue = "chalkular-controller"
)

var (
	schedulerPipelinesGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_pipelines_genereated",
			Help: "Total number of pipelines generated from all reports in event",
		},
		[]string{"profile", "policy", "namespace"},
	)
	schedulerPipelinesCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_pipelines_created",
			Help: "Total number of pipelines created",
		},
		[]string{"profile", "policy", "namespace"},
	)
	schedulerPipelineErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_pipeline_errors",
			Help: "Total number of errors when creating pipelines",
		},
		[]string{"profile", "policy", "namespace"},
	)
	schedulerReportsRecieved = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scheduler_reports_received",
			Help: "Total number of reports receieved. (one event can contain multiple reports)",
		},
	)
	schedulerEventsRecieved = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scheduler_events_received",
			Help: "Total number of events received (i.e. SQS message, HTTP request, etc.)",
		},
	)
	schedulerEventErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scheduler_event_errors",
			Help: "Total number of errors reported for event",
		},
	)
	schedulerEventProcessingDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "scheduler_event_processing_duration_seconds",
			Help: "Duration in seconds of processing per event",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		schedulerEventProcessingDurationSeconds,
		schedulerEventsRecieved,
		schedulerEventErrors,
		schedulerReportsRecieved,
		schedulerPipelinesGenerated,
		schedulerPipelinesCreated,
		schedulerPipelineErrors,
	)
}

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
		case e := <-s.eventBus:
			l.Info("chalk reports received, scheduling")
			schedulerEventsRecieved.Inc()
			schedulerReportsRecieved.Add(float64(len(e.Reports)))
			start := time.Now()
			err := s.processReports(ctx, e.Reports)
			e.Result <- err
			close(e.Result)
			duration := time.Since(start)
			schedulerEventProcessingDurationSeconds.Observe(duration.Seconds())
			if err != nil {
				l.Error(err, "error when processing reports")
				schedulerEventErrors.Inc()
			}
			time.Sleep(time.Second)
		}
	}
}

func (s *Scheduler) processReports(ctx context.Context, reports []chalk.Report) error {
	l := logf.FromContext(ctx)
	l.Info("chalk reports received, scheduling")

	if s.rejectPipelineThreshold > 0 {
		active, err := s.countActivePipelines(ctx)
		if err != nil {
			return fmt.Errorf("unable to list active pipelines: %w", err)
		}
		if active >= s.rejectPipelineThreshold {
			l.Info("rejecting reports, pipeline threshold hit")
			return fmt.Errorf("%w: currently %d active which exceeds threshold of %d, rejecting event",
				ErrPipelineThreshold, active, s.rejectPipelineThreshold)
		}
	}

	policies := &chalkularv1beta1.ChalkReportPolicyList{}
	if err := s.mgrClient.List(ctx, policies); err != nil {
		return fmt.Errorf("unable to list chalk report policies: %w", err)
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
			return fmt.Errorf("invalid chalk report, missing or invalid key %s found", chalk.KeyActionID)
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
			metricLabels := prometheus.Labels{
				"profile":   pipeline.Spec.ProfileRef.Name,
				"policy":    g.policy.Name,
				"namespace": pipeline.Namespace,
			}
			schedulerPipelinesGenerated.With(metricLabels).Inc()
			err := s.mgrClient.Create(ctx, pipeline)
			if err != nil {
				l.Error(err, "unable to create pipeline for policy",
					"pipeline", pipeline.Name, "namespace", g.policy.Namespace, "policy", g.policy.Name)
				s.recorder.Eventf(&g.policy, nil,
					corev1.EventTypeWarning,
					"FailedToCreatePipeline",
					"CreatePipelineFromReport",
					"failed to generate pipeline (%d/%d) for report '%s': %s", i, len(g.pipelines), g.actionID, err)
				schedulerPipelineErrors.With(metricLabels).Inc()
			} else {
				schedulerPipelinesCreated.With(metricLabels).Inc()
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
	return nil
}
