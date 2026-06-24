// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crashappsec/chalkular/api/v1beta1/ingest"
	"github.com/crashappsec/chalkular/internal/utils"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	version   = "unknown"
	buildTime = "unknown"
	gitCommit = "unknown"
)

const (
	ChalkMetadataFile = "chalk.json"
)

func main() {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	l := zap.New(zap.UseFlagOptions(&opts)).
		WithValues("version", version, "build-time", buildTime, "git-commit", gitCommit)
	ctrl.SetLogger(l)

	ctx := ctrl.LoggerInto(context.Background(), l)

	// S3 params
	bucketName := os.Getenv("OCULAR_PARAM_S3_BUCKET")
	region := os.Getenv("OCULAR_PARAM_S3_REGION")
	prefix := strings.Trim(os.Getenv("OCULAR_PARAM_S3_PREFIX"), "/")

	// metadata/identifier params
	pipelineName := os.Getenv("OCULAR_PIPELINE_NAME")
	scanTarget := os.Getenv("OCULAR_PROFILE_NAME")
	workspaceID := os.Getenv("OCULAR_PARAM_WORKSPACE_ID")
	metadataID := os.Getenv("OCULAR_PARAM_METADATA_ID")
	actionID := os.Getenv("OCULAR_PARAM_ACTION_ID")

	// ingest params
	ingestHost := os.Getenv("OCULAR_PARAM_INGEST_HOST")
	ingestToken := os.Getenv("CHALKULAR_INGEST_TOKEN")
	l = l.WithValues(
		"workspace-id", workspaceID, "args", os.Args, "ingest-host", ingestHost,
		"s3-prefix", prefix, "s3-region", region, "s3-bucket", bucketName,
		"metadata-id", metadataID, "action-id", actionID,
	)

	var files []string
	for i, arg := range os.Args {
		// files are passed as positional arguments after '--'
		if arg == "--" {
			files = os.Args[i+1:]
			break
		}
	}
	l.Info("parsing files", "files", files)
	var (
		resultFiles  []string
		resultPrefix = os.Getenv(v1beta1.EnvVarResultsDir)
	)
	for _, f := range files {
		if strings.HasPrefix(f, resultPrefix) {
			resultFiles = append(resultFiles, f)
		}
	}
	l = l.WithValues("result-files", resultFiles)
	l.Info("parsed result files")

	// l.Info("parsing chalk metadata")
	// chalkmark, err := parseChalkmark(ctx, path.Join(os.Getenv(v1beta1.EnvVarMetadataDir), ChalkMetadataFile))
	// if err != nil {
	// l.Error(err, "failed to retrieve chalk mark")
	// os.Exit(1)
	// }

	cfg, err := utils.BuildAWSConfig(ctx, config.WithRegion(region))
	if err != nil {
		l.Error(err, "failed to load AWS config")
		os.Exit(1)
	}
	s3Client := s3service.NewFromConfig(cfg)
	var (
		merr          *multierror.Error
		ocularResults []ingest.OcularResult
	)
	for _, file := range resultFiles {
		key := fmt.Sprintf("%s/%s", filepath.Clean(prefix), path.Base(file))
		_, err := uploadToS3(ctx, s3Client, bucketName, key, file)
		if err != nil {
			l.Error(err, "failed to upload result file to s3", "result-file", file)
			merr = multierror.Append(merr, err)
			continue
		}
		scannerName := scannerNameFromFile(file)
		ocularResults = append(ocularResults, ingest.OcularResult{
			MetadataID:  metadataID,
			PipelineID:  pipelineName,
			WorkspaceID: workspaceID,
			ActionID:    actionID,
			ScanType:    scannerName,
			ScanTarget:  scanTarget,
			S3URI:       fmt.Sprintf("s3://%s/%s", bucketName, key),
		})

	}

	if err := merr.ErrorOrNil(); err != nil {
		l.Error(err, "errors reported for S3 upload")
		os.Exit(1)
	}

	if ingestToken != "" && ingestHost != "" {
		err = triggerIngest(ctx, ingestHost, workspaceID, ingestToken, ocularResults)
		if err != nil {
			l.Error(err, "unable to complete ingest upload")
			os.Exit(1)
		}
	} else {
		l.Info("chalkular ingest token not set, skipping ingest")
	}

	l.Info("upload completed successfully")
}

func uploadToS3(ctx context.Context, c *s3service.Client, b, k, f string) (*s3service.PutObjectOutput, error) {
	fileL := logf.FromContext(ctx).WithValues("result-file", f, "bucket", b, "key", k)
	file, err := os.Open(filepath.Clean(f))
	if err != nil {
		return nil, fmt.Errorf("failed to open result file '%s': %w", f, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			fileL.Error(err, "failed to close file")
		}
	}()

	fileL.Info("putting new object into bucket")
	output, err := c.PutObject(ctx, &s3service.PutObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
		Body:   file,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload s3 object '%s': %w", f, err)
	}

	return output, nil
}

// scannerNameFromFile will split the base file name
// by '.' and take the first element. This is because
// scanner 'foo' should output the file '/mnt/results/foo.<outputformat>'
func scannerNameFromFile(file string) string {
	return strings.Split(path.Base(file), ".")[0]
}

func triggerIngest(ctx context.Context, host, workspace, token string, results []ingest.OcularResult) error {
	l := logf.FromContext(ctx, "results", len(results))
	client := &http.Client{
		Timeout: time.Minute * 5,
	}

	var (
		merr *multierror.Error
		b    = &bytes.Buffer{}
	)
	for _, r := range results {
		resultPayload, err := json.Marshal(r)
		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}
		b.Write(resultPayload)
		b.WriteByte('\n')
	}
	if err := merr.ErrorOrNil(); err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		strings.Join([]string{host, "v0.1/ocular/protected/results"}, "/"),
		b,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Workspace", workspace)

	l.Info("triggering ingest for results")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("non-200 response returned from server: %d", resp.StatusCode)
	}

	l.Info("successfully triggered result ingest", "status-code", resp.StatusCode)
	return nil
}

// func parseChalkmark(ctx context.Context, chalkpath string) (map[string]any, error) {
// 	l := logf.FromContext(ctx)
// 	chalkF, err := os.Open(chalkpath)
// 	if err != nil {
// 		l.Error(err, "failed to open chalk metadata file")
// 		return nil, fmt.Errorf("failed to open chalk metadata: %w", err)
// 	}
// 	defer func() {
// 		if err := chalkF.Close(); err != nil {
// 			l.Error(err, "failed to close chalk file")
// 		}
// 	}()

// 	chalkmark := make(map[string]any)
// 	if err := json.NewDecoder(chalkF).Decode(&chalkmark); err != nil {
// 		l.Error(err, "failed to decode chark mark JSON")
// 		return nil, fmt.Errorf("unable to decode chalkmark: %w", err)
// 	}
// 	return chalkmark, nil

// }
