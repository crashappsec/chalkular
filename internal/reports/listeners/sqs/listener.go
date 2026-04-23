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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/crashappsec/chalkular/api/chalk"
	"github.com/crashappsec/chalkular/internal/reports"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// A Listener is an SQS listener that will listen
// for Artifact analysis requests. When [Listner.Start]
// is executed, the Listener will continually poll
// the queue for new messages that contain two attributes:
// [github.com/crashappsec/chalkular/api/v1beta1/sqs.NamespaceKey]
// which indicates which namespace to create the pipeline in
// and [github.com/crashappsec/chalkular/api/v1beta1/sqs.ImageURIKey]
// which is the URI for the container image to analyze
type Listener struct {
	sqsClient     *sqs.Client
	queueURL      string
	scheduler     reports.SchedulerClient
	waitTime      time.Duration
	visbilityTime time.Duration
}

// NewListener will construct a new Listener that will listen on the given queue URL.
// When a message is received that contains the namespace and imageURI keys, it will
// schedule a new artifact analysis.
func NewListener(ctx context.Context, scheduler reports.SchedulerClient, queueURL string) (*Listener, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	sqsClient := sqs.NewFromConfig(sdkConfig)

	return &Listener{
		sqsClient:     sqsClient,
		queueURL:      queueURL,
		waitTime:      time.Second * 20,
		visbilityTime: time.Second * 20,
		scheduler:     scheduler,
	}, nil
}

// Start will begin polling the queue for new
// messages on the queue
func (l *Listener) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			result, err := l.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:              aws.String(l.queueURL),
				MaxNumberOfMessages:   10,
				MessageAttributeNames: []string{chalk.KeyActionID},
				WaitTimeSeconds:       int32(l.waitTime.Seconds()),
				VisibilityTimeout:     int32(l.visbilityTime.Seconds()),
			})

			if err != nil {
				logger.Error(err, "couldn't receive messages from SQS queue")
				time.Sleep(5 * time.Second)
				continue
			}

			if len(result.Messages) == 0 {
				fmt.Println("No messages received, polling again...")
				continue
			}

			for _, msg := range result.Messages {
				logger.Info("received new chalk report", "message", *msg.Body, "message_attributes", msg.MessageAttributes)

				var actionID string
				if actionIDAtrr, ok := msg.MessageAttributes[chalk.KeyActionID]; ok && actionIDAtrr.StringValue != nil {
					actionID = *actionIDAtrr.StringValue
				}

				logger.Info("retrieving chalk report from API", "actionID", actionID)
				// TODO(bthuilot): get chalk report from API
				report := make(reports.ChalkReport)

				result := l.scheduler.NewReport(ctx, report)
				fmt.Println("report scheduled", "acitonID", actionID)

				go func() {
					messageResult := <-result
					if messageResult != nil {
						logger.Error(messageResult, "pipeline creation failed for report", "actionID", actionID)
					} else {
						_, err := l.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
							QueueUrl:      aws.String(l.queueURL),
							ReceiptHandle: msg.ReceiptHandle,
						})
						if err != nil {
							logger.Error(err, "unable to remove message from queue", "actionID", actionID)
						}
					}

				}()
			}
		}
	}

}
