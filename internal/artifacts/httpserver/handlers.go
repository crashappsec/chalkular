// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import (
	"net/http"

	artifacts "github.com/crashappsec/chalkular/api/v1beta1/artifacts"
	"github.com/crashappsec/chalkular/internal/artifacts/analyze"
	"github.com/gin-gonic/gin"
)

func errorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, artifacts.APIResponse[struct{}]{
		Code:    code,
		Message: message,
	})
}

func analyzeArtifact(artifactScheduler *analyze.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req artifacts.AnalysisRequest
		if err := c.Bind(req); err != nil {
			errorResponse(c, http.StatusBadRequest, "unable to parse request")
			return
		}
		if err := artifactScheduler.Analyze(c, req); err != nil {
			errorResponse(c, http.StatusInternalServerError, "unable to schedule request")
			return
		}
		c.JSON(http.StatusOK, artifacts.APIResponse[struct{}]{
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
