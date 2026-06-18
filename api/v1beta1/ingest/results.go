// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package ingest

type OcularResult struct {
	PipelineID  string `json:"pipeline_id"`
	MetadataID  string `json:"metadata_id"`
	WorkspaceID string `json:"workspace_id"`
	ActionID    string `json:"action_id"`
	ScanType    string `json:"scan_type"`
	ScanTarget  string `json:"scan_target"`
	S3URI       string `json:"s3_uri"`
}
