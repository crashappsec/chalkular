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

	"github.com/crashappsec/chalkular/api/v1beta1/chalk"
)

type SchedulerResult chan error

type SchedulerClient interface {
	Enqueue(context.Context, []chalk.Report) SchedulerResult
}

type schedulerClient struct {
	eventBus eventBus
}

type event struct {
	Reports []chalk.Report
	Result  SchedulerResult
}

type eventBus = chan event

func (c *schedulerClient) Enqueue(_ context.Context, reports []chalk.Report) SchedulerResult {
	done := make(SchedulerResult, 1)
	c.eventBus <- event{
		Reports: reports,
		Result:  done,
	}
	return done

}
