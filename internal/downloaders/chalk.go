// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func ExtractChalkFromImage(ctx context.Context, ref name.Reference, img v1.Image) error {
	metadataPath := path.Clean(path.Join(os.Getenv(v1beta1.EnvVarMetadataDir), chalkFileName))
	l := logf.FromContext(ctx).WithValues("metadataPath", metadataPath, "image", ref.String())
	l.Info("beginning chalk extraction")

	layers, err := img.Layers()
	if err != nil {
		l.Error(err, "failed to inspect image layers")
		return nil
	}

	if len(layers) == 0 {
		l.Info("image has no layers")
		return nil
	}
	lastLayer := layers[len(layers)-1]
	if lastLayer == nil {
		l.Info("last layer is nil")
		return fmt.Errorf("missing last layer")
	}

	l.Info("attempting to extract chalk metadata from last layer")
	mediaType, err := lastLayer.MediaType()
	if err != nil {
		l.Error(err, "failed to get last layer media type")
		return nil
	}

	if !mediaType.IsLayer() {
		l.Info("last layer media type is not a layer, skipping chalk extraction", "mediaType", mediaType)
		return nil
	}
	l = l.WithValues("mediaType", mediaType)

	rc, err := lastLayer.Uncompressed()
	if err != nil {
		l.Error(err, "failed to get uncompressed last layer")
		return nil
	}

	defer func() {
		if err = rc.Close(); err != nil {
			l.Error(err, "failed to close last layer reader")
		}
	}()

	if err = extractChalkFromTar(ctx, rc, metadataPath); err != nil {
		l.Error(err, "failed to extract chalk metadata")
	}

	return nil
}

const chalkFileName = "chalk.json"

func extractChalkFromTar(ctx context.Context, tr io.Reader, chalkPath string) error {
	l := logf.FromContext(ctx)
	tarReader := tar.NewReader(tr)
	chalk, err := tarReader.Next()
	if err != nil {
		return fmt.Errorf("reading last layer tar: %w", err)
	}

	if chalk.Name != chalkFileName {
		l.Info("chalk metadata file not found in last layer", "expected", chalkFileName, "found", chalk.Name)
		return nil
	}

	chalkFile, err := os.Create(filepath.Clean(chalkPath))
	if err != nil {
		return fmt.Errorf("creating chalk metadata file: %w", err)
	}
	defer func() {
		if err := chalkFile.Close(); err != nil {
			l.Error(err, "failed to close chalk metadata file")
		}
	}()

	_, err = io.Copy(chalkFile, tarReader)
	if err != nil {
		return fmt.Errorf("writing chalk metadata file: %w", err)
	}
	return nil
}
