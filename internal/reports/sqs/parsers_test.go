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
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeS3Client implements S3ClientAPI for tests.
type fakeS3Client struct {
	getObject func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return f.getObject(ctx, params)
}

func s3Body(contents string) *s3.GetObjectOutput {
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(contents))}
}

func msgWithBody(body string) sqstypes.Message {
	return sqstypes.Message{Body: aws.String(body)}
}

var _ = Describe("RawReportParser", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	When("the message body is a valid report", func() {
		It("should return a single report", func() {
			reports, err := RawReportParser(ctx, msgWithBody(`{"CHALK_ID":"abc123"}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0]).To(HaveKeyWithValue("CHALK_ID", "abc123"))
		})
	})
	When("when the message body is invalid JSON", func() {
		It("should return an error for malformed JSON", func() {
			body := `{not valid json`
			msg := sqstypes.Message{Body: aws.String(body)}

			_, err := RawReportParser(ctx, msg)

			Expect(err).To(HaveOccurred())

		})

		It("should return an error for a JSON array instead of object", func() {
			body := `[{"key":"value"}]`
			msg := sqstypes.Message{Body: aws.String(body)}

			_, err := RawReportParser(ctx, msg)

			Expect(err).To(HaveOccurred())
		})

		It("should return an error for an empty body", func() {
			body := ``
			msg := sqstypes.Message{Body: aws.String(body)}

			_, err := RawReportParser(ctx, msg)

			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = Describe("S3EventReportParser", func() {
	const (
		oneRecordEvent = `{"Records":[
			{"s3":{"bucket":{"name":"reports"},"object":{"key":"report.json","eTag":"etag-1"}}}
		]}`
		twoRecordEvent = `{"Records":[
			{"s3":{"bucket":{"name":"reports"},"object":{"key":"a.json","eTag":"etag-a"}}},
			{"s3":{"bucket":{"name":"reports"},"object":{"key":"b.json","eTag":"etag-b"}}}
		]}`
	)

	var (
		ctx    context.Context
		client *fakeS3Client
		parse  ChalkReportParser
	)

	BeforeEach(func() {
		ctx = context.Background()
		client = &fakeS3Client{}
		parse = S3EventReportParser(client)
	})

	When("the message body is not a valid S3 event", func() {
		It("should return an error without calling S3", func() {
			_, err := parse(ctx, msgWithBody(`not-json`))

			Expect(err).To(MatchError(ContainSubstring("error parsing S3 event")))
		})
	})

	When("the event contains no records", func() {
		It("should return no reports and no error", func() {
			reports, err := parse(ctx, msgWithBody(`{"Records":[]}`))

			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})
	})

	When("the event contains a valid record", func() {
		It("should fetch the object and return its reports", func() {
			var input *s3.GetObjectInput
			client.getObject = func(_ context.Context, in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				input = in
				return s3Body(`[{"CHALK_ID":"a"},{"CHALK_ID":"b"}]`), nil
			}

			reports, err := parse(ctx, msgWithBody(oneRecordEvent))

			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0]).To(HaveKeyWithValue("CHALK_ID", "a"))
			Expect(reports[1]).To(HaveKeyWithValue("CHALK_ID", "b"))

			Expect(input).NotTo(BeNil())
			Expect(aws.ToString(input.Bucket)).To(Equal("reports"))
			Expect(aws.ToString(input.Key)).To(Equal("report.json"))
			Expect(aws.ToString(input.IfMatch)).To(Equal("etag-1"))
		})
	})

	When("fetching the object fails", func() {
		It("should return the error", func() {
			client.getObject = func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return nil, errors.New("boom")
			}

			reports, err := parse(ctx, msgWithBody(oneRecordEvent))

			Expect(err).To(MatchError(ContainSubstring("boom")))
			Expect(reports).To(BeEmpty())
		})
	})

	When("the object contents are not valid reports", func() {
		It("should return a decode error", func() {
			client.getObject = func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return s3Body(`not-json`), nil
			}

			reports, err := parse(ctx, msgWithBody(oneRecordEvent))

			Expect(err).To(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})
	})

	When("one record fails and another succeeds", func() {
		It("should return the successful reports along with the error", func() {
			client.getObject = func(_ context.Context, in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				if aws.ToString(in.Key) == "a.json" {
					return nil, errors.New("boom")
				}
				return s3Body(`[{"CHALK_ID":"b"}]`), nil
			}

			reports, err := parse(ctx, msgWithBody(twoRecordEvent))

			Expect(err).To(MatchError(ContainSubstring("boom")))
			Expect(reports).To(HaveLen(1))
			Expect(reports[0]).To(HaveKeyWithValue("CHALK_ID", "b"))
		})
	})
})
