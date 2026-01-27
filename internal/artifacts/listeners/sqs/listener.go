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
	v1beta1 "github.com/crashappsec/chalkular/api/v1beta1/sqs"
	"github.com/crashappsec/chalkular/internal/artifacts"
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
	scheduler     artifacts.SchedulerClient
	waitTime      time.Duration
	visbilityTime time.Duration
}

// NewListener will construct a new Listener that will listen on the given queue URL.
// When a message is received that contains the namespace and imageURI keys, it will
// schedule a new artifact analysis.
func NewListener(ctx context.Context, scheduler artifacts.SchedulerClient, queueURL string) (*Listener, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	sqsClient := sqs.NewFromConfig(sdkConfig)

	// result, err := sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: &queueName})
	// if err != nil {
	// 	return nil, err
	// }
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
		result, err := l.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(l.queueURL),
			MaxNumberOfMessages:   10,
			MessageAttributeNames: []string{"namespace", "image_uri"},
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
			logger.Info("received new artifact request", "message", *msg.Body, "message_attributes", msg.MessageAttributes)

			var imageURI string
			if imageURIAttr, ok := msg.MessageAttributes[v1beta1.ImageURIKey]; ok {
				imageURI = *imageURIAttr.StringValue
			}

			var namespace string
			if namespaceAttr, ok := msg.MessageAttributes[v1beta1.NamespaceKey]; ok {
				namespace = *namespaceAttr.StringValue
			}

			logger.Info("scheduling new artfiact analysis", "namespace", namespace, "imageURI", imageURI)

			if err = l.scheduler.Analyze(ctx, imageURI, namespace); err != nil {
				logger.Error(err, "unable to schedule artifact analysis", "imageURI", imageURI, "namespace", namespace)
				continue
			}

			_, err := l.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(l.queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			})
			if err != nil {
				logger.Error(err, "unable to remove message from queue", "imageURI", imageURI, "namespace", namespace)
			}
		}
	}

}
