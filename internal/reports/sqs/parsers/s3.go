// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package parsers

import (
	"context"
	"fmt"
	"os"

	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
	sqsapi "github.com/crashappsec/chalkular/api/v1beta1/sqs"
	"github.com/hashicorp/go-multierror"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func S3EventReportParser(cfg aws.Config) ChalkReportParser {
	s3client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if os.Getenv("AWS_S3_USE_PATH_STYLE") != "" {
			o.UsePathStyle = true
		}
	})
	return func(ctx context.Context, msg sqstypes.Message) ([]chalk.Report, error) {
		l := logf.FromContext(ctx)
		l.Info("parsing S3 notification from message body")

		var event sqsapi.S3Event
		err := json.Unmarshal([]byte(aws.ToString(msg.Body)), &event)
		if err != nil {
			l.Error(err, "unable to marshal S3 event")
			return nil, fmt.Errorf("error parsing S3 event: %w", err)
		}
		l.Info("processing S3 records", "records-count", len(event.Records))
		var (
			merr    *multierror.Error
			reports []chalk.Report
		)
		for _, record := range event.Records {
			object := record.S3.Object
			bucket := record.S3.Bucket
			recordL := l.WithValues("object", object, "bucket", bucket)
			recordL.Info("retriveing report from S3 object")

			input := &s3.GetObjectInput{
				Bucket: aws.String(bucket.Name),
				Key:    aws.String(object.URLDecodedKey),
				// VersionId: aws.String(object.VersionID),
				IfMatch: aws.String(object.ETag),
			}

			output, err := s3client.GetObject(ctx, input)
			if err != nil {
				recordL.Error(err, "unable to get object, skipping")
				merr = multierror.Append(err, err)
				continue
			}

			var objectReports []chalk.Report
			err = json.NewDecoder(output.Body).Decode(&objectReports)
			if err != nil {
				recordL.Error(err, "unable to decode object contents, skipping")
				merr = multierror.Append(err, err)
				continue
			}
			reports = append(reports, objectReports...)

		}

		return reports, merr.ErrorOrNil()
	}
}
