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

	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/crashappsec/chalkular/api/chalk"
)

type ChalkReportParser = func(context.Context, sqsTypes.Message) ([]chalk.Report, error)
