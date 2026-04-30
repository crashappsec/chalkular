// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	s3service "github.com/aws/aws-sdk-go-v2/service/s3"
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
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	ctrl.SetLogger(l)

	ctx := ctrl.LoggerInto(context.Background(), l)

	bucketName := os.Getenv("OCULAR_PARAM_BUCKET")
	region := os.Getenv("OCULAR_PARAM_REGION")

	var files []string
	for i, arg := range os.Args {
		// files are passed as positional arguments after '--'
		if arg == "--" {
			files = os.Args[i+1:]
			break
		}
	}
	l.Info("parsing files", "files", files)
	var resultFiles []string
	for _, f := range files {
		if strings.HasPrefix(f, os.Getenv(v1beta1.EnvVarResultsDir)) {
			resultFiles = append(resultFiles, f)
		}
	}
	l = l.WithValues("result-files", resultFiles)
	l.Info("parsed result files")

	l.Info("parsing chalk metadata")
	chalkmark, err := parseChalkmark(ctx, path.Join(os.Getenv(v1beta1.EnvVarMetadataDir), ChalkMetadataFile))
	if err != nil {
		l.Error(err, "failed to retrieve chalk mark")
		os.Exit(1)
	}

	chalkIDJSON, exists := chalkmark["CHALK_ID"]
	chalkID, ok := chalkIDJSON.(string)
	if !exists || !ok {
		l.Error(fmt.Errorf("invalid or missing chalkID, got %v", chalkIDJSON), "invalid chalk ID in chalk mark")
		os.Exit(1)
	}

	cfg, err := utils.BuildAWSConfig(ctx, config.WithRegion(region))
	if err != nil {
		l.Error(err, "failed to load AWS config")
		os.Exit(1)
	}
	s3Client := s3service.NewFromConfig(cfg)
	var merr *multierror.Error
	for _, file := range resultFiles {
		fileL := l.WithValues("result-file", file)
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			fileL.Error(err, "failed to open result file")
			merr = multierror.Append(merr, fmt.Errorf("failed to open result file '%s': %w", file, err))
		}

		key := fmt.Sprintf("chalkular/%s/%s", chalkID, path.Base(file))
		fileL.Info("putting new object into bucket", "bucket", bucketName, "key", key)
		_, err = s3Client.PutObject(ctx, &s3service.PutObjectInput{
			Bucket: &bucketName,
			Key:    &key,
			Body:   f,
			Metadata: map[string]string{
				"CHALK_ID": chalkID,
			},
		})
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("failed to upload file %s: %w", file, err))
		}
		if err = f.Close(); err != nil {
			l.Error(err, "Failed to close file", "file", file)
		}
	}

	if err := merr.ErrorOrNil(); err != nil {
		l.Error(err, "errors reported for S3 upload")
		os.Exit(1)
	}
	l.Info("upload completed successfully")
}

func parseChalkmark(ctx context.Context, chalkpath string) (map[string]any, error) {
	l := logf.FromContext(ctx)
	chalkF, err := os.Open(chalkpath)
	if err != nil {
		l.Error(err, "failed to open chalk metadata file")
		return nil, fmt.Errorf("failed to open chalk metadata: %w", err)
	}
	defer func() {
		if err := chalkF.Close(); err != nil {
			l.Error(err, "failed to close chalk file")
		}
	}()

	chalkmark := make(map[string]any)
	if err := json.NewDecoder(chalkF).Decode(&chalkmark); err != nil {
		l.Error(err, "failed to decode chark mark JSON")
		return nil, fmt.Errorf("unable to decode chalkmark: %w", err)
	}
	return chalkmark, nil

}
