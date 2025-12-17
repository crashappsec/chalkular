// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package analyze

import (
	"context"

	"github.com/crashappsec/chalkular/api/v1beta1/artifacts"
)

type Client struct {
	eventBus eventBus
}

func (c *Client) Analyze(ctx context.Context, req artifacts.AnalysisRequest) error {
	c.eventBus <- req
	return nil
}
