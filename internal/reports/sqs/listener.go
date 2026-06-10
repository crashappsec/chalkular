// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package sqs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/crashappsec/chalkular/internal/reports"
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	sqsMessagesReceivedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_messages_received_total",
			Help: "Total messages pulled from SQS",
		},
	)
	sqsMessagesProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqs_messages_processed_total",
			Help: "Number of SQS messages handled",
		},
		[]string{"status"},
	)
	sqsMessagesDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_messages_deleted_total",
			Help: "Messages successfully deleted after processing",
		},
	)
	sqsMessageProcessingDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "sqs_message_processing_duration_seconds",
			Help: "Number of scan pods ocular has created",
		},
	)
	sqsReceiveErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_receive_errors_total",
			Help: "Failures calling ReceiveMessage",
		},
	)
)

type SQSClientAPI interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)

	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// A Listener is an SQS listener that will listen
// for Artifact analysis requests. When [Listner.Start]
// is executed, the Listener will continually poll
// the queue for new messages that contain two attributes:
// [github.com/crashappsec/chalkular/api/v1beta1/sqs.NamespaceKey]
// which indicates which namespace to create the pipeline in
// and [github.com/crashappsec/chalkular/api/v1beta1/sqs.ImageURIKey]
// which is the URI for the container image to analyze
type Listener struct {
	sqsClient     SQSClientAPI
	queueURL      string
	scheduler     reports.SchedulerClient
	waitTime      time.Duration
	visbilityTime time.Duration
	reportParser  ChalkReportParser
}

// NewListener will construct a new Listener that will listen on the given queue URL.
// When a message is received that contains the namespace and imageURI keys, it will
// schedule a new artifact analysis.
func NewListener(sqsClient SQSClientAPI, scheduler reports.SchedulerClient, queueURL string, reportParser ChalkReportParser) (*Listener, error) {
	if reportParser == nil {
		return nil, fmt.Errorf("no chalk report parser supplied")
	}

	return &Listener{
		sqsClient:     sqsClient,
		queueURL:      queueURL,
		waitTime:      time.Second * 20,
		visbilityTime: time.Minute,
		scheduler:     scheduler,
		reportParser:  reportParser,
	}, nil
}

// Start will begin polling the queue for new
// messages on the queue
func (l *Listener) Start(ctx context.Context) error {
	logger := logf.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			result, err := l.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(l.queueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     int32(l.waitTime.Seconds()),
				VisibilityTimeout:   int32(l.visbilityTime.Seconds()),
			})

			if err != nil {
				logger.Error(err, "couldn't receive messages from SQS queue")
				sqsReceiveErrorsTotal.Add(1)
				time.Sleep(5 * time.Second)
				continue
			}

			if len(result.Messages) == 0 {
				logger.Info("no messages received, polling again...")
				continue
			}

			sqsMessagesReceivedTotal.Add(float64(len(result.Messages)))
			for _, msg := range result.Messages {
				processingStartTime := time.Now()

				msgLogger := logger.WithValues(
					"attributes", msg.Attributes,
					"message-attributes", msg.MessageAttributes,
					"message-id", aws.ToString(msg.MessageId),
				)
				msgLogger.Info("received new queue message")
				msgCtx := logf.IntoContext(ctx, msgLogger)

				rs, err := l.reportParser(msgCtx, msg)
				if err != nil {
					msgLogger.Error(err, "failed to parse report from SQS message, skipping")
					sqsMessageProcessingDurationSeconds.Observe(time.Since(processingStartTime).Seconds())
					sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "failure"}).Add(1)
					continue
				}

				result := l.scheduler.Enqueue(msgCtx, rs)
				go func() {
					msgLogger.Info("reports scheduled, awaiting result")
					messageErr := <-result
					sqsMessageProcessingDurationSeconds.Observe(time.Since(processingStartTime).Seconds())
					if messageErr != nil {
						sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "failure"}).Add(1)
						msgLogger.Error(messageErr, "failed to evaluate reports from message, not deleting message")
					} else {
						sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "success"}).Add(1)
						_, err := l.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
							QueueUrl:      aws.String(l.queueURL),
							ReceiptHandle: msg.ReceiptHandle,
						})
						if err != nil {
							msgLogger.Error(err, "unable to remove message from queue")
						} else {
							sqsMessagesDeletedTotal.Add(1)
						}
					}
				}()
			}
		}
	}

}
