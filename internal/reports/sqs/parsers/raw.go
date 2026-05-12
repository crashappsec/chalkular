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
	"encoding/json"

	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/crashappsec/chalkular/api/chalk"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func RawReportParser(ctx context.Context, msg sqsTypes.Message) ([]chalk.Report, error) {
	l := logf.FromContext(ctx)
	l.Info("parsing report from message body")

	report := make(chalk.Report)
	err := json.Unmarshal([]byte(*msg.Body), &report)
	return []chalk.Report{report}, err
}

var _ ChalkReportParser = RawReportParser
