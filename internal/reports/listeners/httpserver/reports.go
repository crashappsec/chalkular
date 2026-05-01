// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import (
	"fmt"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1beta1 "github.com/crashappsec/chalkular/api/v1beta1/httpserver"
	"github.com/crashappsec/chalkular/internal/reports"
	"github.com/gin-gonic/gin"
)

var reportslog = logf.Log.WithName("reports-http")

func scheduleReport(scheduler *reports.SchedulerClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var reports []map[string]any
		if err := c.BindJSON(&reports); err != nil {
			errorResponse(c, http.StatusBadRequest, "unable to parse request")
			return
		}

		reportslog.Info("received report upload", "count", len(reports))
		for _, report := range reports {
			_ = scheduler.NewReport(c, report)
		}
		c.JSON(http.StatusOK, v1beta1.APIResponse[struct{}]{
			Code:    http.StatusOK,
			Message: fmt.Sprintf("processed %d reports", len(reports)),
		})
	}
}
