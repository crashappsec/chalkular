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
	"errors"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
	"github.com/crashappsec/chalkular/internal/reports"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// fakeSQSClient implements SQSClientAPI for tests.
type fakeSQSClient struct {
	receive func(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	delete  func(context.Context, *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

func (f *fakeSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return f.receive(ctx, params)
}

func (f *fakeSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	if f.delete == nil {
		return &sqs.DeleteMessageOutput{}, nil
	}
	return f.delete(ctx, params)
}

// fakeScheduler implements reports.SchedulerClient for tests.
// Adjust the Enqueue signature here if the real interface differs.
type fakeScheduler struct {
	enqueue func(context.Context, []chalk.Report) reports.SchedulerResult
}

var _ reports.SchedulerClient = &fakeScheduler{}

func (f *fakeScheduler) Enqueue(ctx context.Context, rs []chalk.Report) reports.SchedulerResult {
	return f.enqueue(ctx, rs)
}

// schedulerResult returns a channel pre-loaded with the given result,
// mimicking a scheduler that has finished evaluating the reports.
func schedulerResult(err error) reports.SchedulerResult {
	ch := make(chan error, 1)
	ch <- err
	return ch
}

// serveOnce returns a ReceiveMessage fake that returns the given messages
// on the first call and an empty result on every call after that.
func serveOnce(msgs ...sqstypes.Message) func(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	var served atomic.Bool
	return func(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
		if served.CompareAndSwap(false, true) {
			return &sqs.ReceiveMessageOutput{Messages: msgs}, nil
		}
		return &sqs.ReceiveMessageOutput{}, nil
	}
}

// counterDelta snapshots a counter and returns a function reporting how much
// it has grown since, so package-level metrics can be asserted per spec.
func counterDelta(c prometheus.Collector) func() float64 {
	start := testutil.ToFloat64(c)
	return func() float64 { return testutil.ToFloat64(c) - start }
}

var _ = Describe("NewListener", func() {
	When("no report parser is supplied", func() {
		It("should return an error", func() {
			_, err := NewListener(&fakeSQSClient{}, &fakeScheduler{}, "queue-url", nil)
			Expect(err).To(MatchError(ContainSubstring("no chalk report parser")))
		})
	})

	When("a report parser is supplied", func() {
		It("should construct a listener with sane defaults", func() {
			l, err := NewListener(&fakeSQSClient{}, &fakeScheduler{}, "queue-url", RawReportParser)

			Expect(err).NotTo(HaveOccurred())
			Expect(l.queueURL).To(Equal("queue-url"))
			Expect(l.waitTime).To(Equal(20 * time.Second))
			Expect(l.visbilityTime).To(Equal(time.Minute))
			Expect(l.sqsClient).NotTo(BeNil())
			Expect(l.scheduler).NotTo(BeNil())
			Expect(l.reportParser).NotTo(BeNil())
		})
	})
})

var _ = Describe("Listener.Start", func() {
	const queueURL = "https://sqs.test/queue"

	var (
		ctx       context.Context
		cancel    context.CancelFunc
		client    *fakeSQSClient
		scheduler *fakeScheduler
		listener  *Listener

		// parse is swapped per spec; the listener holds an indirection to it.
		parse ChalkReportParser

		report  = chalk.Report{"CHALK_ID": "abc123"}
		message = sqstypes.Message{
			MessageId:     aws.String("msg-1"),
			ReceiptHandle: aws.String("rh-1"),
			Body:          aws.String(`{"CHALK_ID":"abc123"}`),
		}
	)

	// startListener runs Start in the background and returns a channel
	// that receives its return value.
	startListener := func() <-chan error {
		done := make(chan error, 1)
		go func() { done <- listener.Start(ctx) }()
		return done
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		client = &fakeSQSClient{
			receive: serveOnce(),
		}
		scheduler = &fakeScheduler{
			enqueue: func(context.Context, []chalk.Report) reports.SchedulerResult {
				return schedulerResult(nil)
			},
		}
		// nolint:unparam
		parse = func(context.Context, sqstypes.Message) ([]chalk.Report, error) {
			return []chalk.Report{report}, nil
		}

		var err error
		listener, err = NewListener(client, scheduler, queueURL,
			func(ctx context.Context, m sqstypes.Message) ([]chalk.Report, error) {
				return parse(ctx, m)
			})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
	})

	When("the context is already cancelled", func() {
		It("should return immediately without error", func() {
			cancel()
			Eventually(startListener()).Should(Receive(BeNil()))
		})
	})

	When("polling the queue", func() {
		It("should request messages with the configured parameters", func() {
			inputs := make(chan *sqs.ReceiveMessageInput, 1)
			client.receive = func(_ context.Context, in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
				select {
				case inputs <- in:
				default:
				}
				return &sqs.ReceiveMessageOutput{}, nil
			}
			done := startListener()

			var input *sqs.ReceiveMessageInput
			Eventually(inputs).Should(Receive(&input))
			Expect(aws.ToString(input.QueueUrl)).To(Equal(queueURL))
			Expect(input.MaxNumberOfMessages).To(Equal(int32(10)))
			Expect(input.WaitTimeSeconds).To(Equal(int32(20)))
			Expect(input.VisibilityTimeout).To(Equal(int32(60)))

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	When("receiving messages fails", func() {
		It("should record the error, back off, and keep running", func() {
			receiveErrors := counterDelta(sqsReceiveErrorsTotal)
			client.receive = func(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
				cancel() // stop the loop once it wakes from its backoff
				return nil, errors.New("boom")
			}
			done := startListener()

			// the loop sleeps 5s after a receive failure before observing ctx
			Eventually(done, "7s").Should(Receive(BeNil()))
			Expect(receiveErrors()).To(Equal(1.0))
		})
	})

	When("no messages are received", func() {
		It("should keep polling without recording received messages", func() {
			received := counterDelta(sqsMessagesReceivedTotal)
			var polls atomic.Int32
			client.receive = func(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
				polls.Add(1)
				return &sqs.ReceiveMessageOutput{}, nil
			}
			done := startListener()

			Eventually(polls.Load).Should(BeNumerically(">=", 2))
			cancel()
			Eventually(done).Should(Receive(BeNil()))
			Expect(received()).To(BeZero())
		})
	})

	When("a message is processed successfully", func() {
		It("should enqueue the parsed reports and delete the message", func() {
			received := counterDelta(sqsMessagesReceivedTotal)
			succeeded := counterDelta(sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "success"}))
			deleted := counterDelta(sqsMessagesDeletedTotal)

			client.receive = serveOnce(message)
			enqueued := make(chan []chalk.Report, 1)
			scheduler.enqueue = func(_ context.Context, rs []chalk.Report) reports.SchedulerResult {
				enqueued <- rs
				return schedulerResult(nil)
			}
			deletes := make(chan *sqs.DeleteMessageInput, 1)
			client.delete = func(_ context.Context, in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
				deletes <- in
				return &sqs.DeleteMessageOutput{}, nil
			}
			done := startListener()

			Eventually(enqueued).Should(Receive(ConsistOf(report)))

			var input *sqs.DeleteMessageInput
			Eventually(deletes).Should(Receive(&input))
			Expect(aws.ToString(input.QueueUrl)).To(Equal(queueURL))
			Expect(aws.ToString(input.ReceiptHandle)).To(Equal("rh-1"))

			Eventually(received).Should(Equal(1.0))
			Eventually(succeeded).Should(Equal(1.0))
			Eventually(deleted).Should(Equal(1.0))

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	When("parsing a message fails", func() {
		It("should skip the message without scheduling or deleting it", func() {
			failed := counterDelta(sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "failure"}))

			client.receive = serveOnce(message)
			parse = func(context.Context, sqstypes.Message) ([]chalk.Report, error) {
				return nil, errors.New("bad report")
			}
			var enqueueCalled, deleteCalled atomic.Bool
			scheduler.enqueue = func(context.Context, []chalk.Report) reports.SchedulerResult {
				enqueueCalled.Store(true)
				return schedulerResult(nil)
			}
			client.delete = func(context.Context, *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
				deleteCalled.Store(true)
				return &sqs.DeleteMessageOutput{}, nil
			}
			done := startListener()

			Eventually(failed).Should(Equal(1.0))
			Consistently(enqueueCalled.Load).Should(BeFalse())
			Consistently(deleteCalled.Load).Should(BeFalse())

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	When("the scheduler reports a failure", func() {
		It("should leave the message on the queue", func() {
			failed := counterDelta(sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "failure"}))

			client.receive = serveOnce(message)
			scheduler.enqueue = func(context.Context, []chalk.Report) reports.SchedulerResult {
				return schedulerResult(errors.New("evaluation failed"))
			}
			var deleteCalled atomic.Bool
			client.delete = func(context.Context, *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
				deleteCalled.Store(true)
				return &sqs.DeleteMessageOutput{}, nil
			}
			done := startListener()

			Eventually(failed).Should(Equal(1.0))
			Consistently(deleteCalled.Load).Should(BeFalse())

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	When("deleting a processed message fails", func() {
		It("should record the success but not count a deletion", func() {
			succeeded := counterDelta(sqsMessagesProcessedTotal.With(prometheus.Labels{"status": "success"}))
			deleted := counterDelta(sqsMessagesDeletedTotal)

			client.receive = serveOnce(message)
			client.delete = func(context.Context, *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
				return nil, errors.New("delete failed")
			}
			done := startListener()

			Eventually(succeeded).Should(Equal(1.0))
			Consistently(deleted).Should(BeZero())

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})
})
