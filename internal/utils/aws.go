// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package utils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func BuildAWSConfig(ctx context.Context, opts ...func(*config.LoadOptions) error) (aws.Config, error) {
	l := logf.FromContext(ctx)
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		l.Error(err, "Failed to load AWS configuration")
		return aws.Config{}, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	return cfg, nil
}
