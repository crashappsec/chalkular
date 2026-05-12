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

	"encoding/json"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/crashappsec/chalkular/api/chalk"
	"github.com/hashicorp/go-multierror"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func S3EventReportParser(s3client *s3.Client) ChalkReportParser {
	return func(ctx context.Context, msg sqsTypes.Message) ([]chalk.Report, error) {
		l := logf.FromContext(ctx)
		l.Info("parsing S3 notification from message body")

		var event S3Event
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

			output, err := s3client.GetObject(ctx, &s3.GetObjectInput{
				Bucket:    aws.String(bucket.Name),
				Key:       aws.String(object.URLDecodedKey),
				VersionId: aws.String(object.VersionID),
				IfMatch:   aws.String(object.ETag),
			})
			if err != nil {
				recordL.Error(err, "unable to get object, skipping")
				merr = multierror.Append(err, err)
				continue
			}

			var report chalk.Report
			err = json.NewDecoder(output.Body).Decode(&report)
			if err != nil {
				recordL.Error(err, "unable to decode object contents, skipping")
				merr = multierror.Append(err, err)
				continue
			}
			reports = append(reports, report)

		}

		return reports, merr.ErrorOrNil()
	}
}

type S3Event struct {
	Records []S3EventRecord `json:"Records"`
}

type S3EventRecord struct {
	EventVersion                string                         `json:"eventVersion"`
	EventSource                 string                         `json:"eventSource"`
	AWSRegion                   string                         `json:"awsRegion"`
	EventTime                   time.Time                      `json:"eventTime"`
	EventName                   string                         `json:"eventName"`
	PrincipalID                 S3UserIdentity                 `json:"userIdentity"`
	RequestParameters           S3RequestParameters            `json:"requestParameters"`
	ResponseElements            map[string]string              `json:"responseElements"`
	S3                          S3Entity                       `json:"s3"`
	GlacierEventData            *S3GlacierEventData            `json:"glacierEventData,omitempty"`
	RestoreEventData            *S3RestoreEventData            `json:"restoreEventData,omitempty"`
	ReplicationEventData        *S3ReplicationEventData        `json:"replicationEventData,omitempty"`
	IntelligentTieringEventData *S3IntelligentTieringEventData `json:"intelligentTieringEventData,omitempty"`
	LifecycleEventData          *S3LifecycleEventData          `json:"lifecycleEventData,omitempty"`
}

type S3UserIdentity struct {
	PrincipalID string `json:"principalId"`
}

type S3RequestParameters struct {
	SourceIPAddress string `json:"sourceIPAddress"`
}

type S3Entity struct {
	SchemaVersion   string   `json:"s3SchemaVersion"`
	ConfigurationID string   `json:"configurationId"`
	Bucket          S3Bucket `json:"bucket"`
	Object          S3Object `json:"object"`
}

type S3Bucket struct {
	Name          string         `json:"name"`
	OwnerIdentity S3UserIdentity `json:"ownerIdentity"`
	Arn           string         `json:"arn"`
}

type S3Object struct {
	Key           string `json:"key"`
	Size          int64  `json:"size,omitempty"`
	URLDecodedKey string `json:"urlDecodedKey"`
	VersionID     string `json:"versionId"`
	ETag          string `json:"eTag"`
	Sequencer     string `json:"sequencer"`
}

func (o *S3Object) UnmarshalJSON(data []byte) error {
	type rawS3Object S3Object
	if err := json.Unmarshal(data, (*rawS3Object)(o)); err != nil {
		return err
	}
	key, err := url.QueryUnescape(o.Key)
	if err != nil {
		return err
	}
	o.URLDecodedKey = key

	return nil
}

type S3GlacierEventData struct {
	RestoreEventData *S3RestoreEventData `json:"restoreEventData"`
}

type S3RestoreEventData struct {
	LifecycleRestorationExpiryTime time.Time `json:"lifecycleRestorationExpiryTime"`
	LifecycleRestoreStorageClass   string    `json:"lifecycleRestoreStorageClass"`
}

type S3ReplicationEventData struct {
	ReplicationRuleID string    `json:"replicationRuleId"`
	DestinationBucket string    `json:"destinationBucket"`
	S3Operation       string    `json:"s3Operation"`
	RequestTime       time.Time `json:"requestTime"`
	FailureReason     string    `json:"failureReason"`
}

type S3IntelligentTieringEventData struct {
	DestinationAccessTier string `json:"destinationAccessTier"`
}

type S3LifecycleEventData struct {
	TransitionEventData *S3TransitionEventData `json:"transitionEventData"`
}

type S3TransitionEventData struct {
	DestinationStorageClass string `json:"destinationStorageClass"`
}

type S3TestEvent struct {
	Service   string    `json:"Service"`
	Bucket    string    `json:"Bucket"`
	Event     string    `json:"Event"`
	Time      time.Time `json:"Time"`
	RequestID string    `json:"RequestId"`
	HostID    string    `json:"HostId"`
}
