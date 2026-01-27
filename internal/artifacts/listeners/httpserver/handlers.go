// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import (
	"net/http"

	v1beta1 "github.com/crashappsec/chalkular/api/v1beta1/httpserver"
	"github.com/crashappsec/chalkular/internal/artifacts"
	"github.com/gin-gonic/gin"
)

func errorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, v1beta1.APIResponse[struct{}]{
		Code:    code,
		Message: message,
	})
}

func analyzeArtifact(scheduler *artifacts.SchedulerClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1beta1.ScheduleArtifactAnalysisRequest
		if err := c.Bind(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, "unable to parse request")
			return
		}
		if err := scheduler.Analyze(c, req.ImageURI, req.Namespace); err != nil {
			errorResponse(c, http.StatusInternalServerError, "unable to schedule request")
			return
		}
		c.JSON(http.StatusOK, v1beta1.APIResponse[struct{}]{
			Code:    http.StatusOK,
			Message: "artifact analysis queued successfully",
		})
	}
}

func health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"health": "ok",
		})
	}
}
