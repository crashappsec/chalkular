// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package artifacts

import (
	"context"
)

type SchedulerClient struct {
	eventBus eventBus
}

func (c *SchedulerClient) Analyze(ctx context.Context, imageRef string, namespace string) error {
	c.eventBus <- analysisRequest{
		ImageRef:  imageRef,
		Namespace: namespace,
	}
	return nil
}
