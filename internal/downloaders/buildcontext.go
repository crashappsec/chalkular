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
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func DownloadBuildContext(ctx context.Context, ref name.Reference, img v1.Image, dir string) error {
	l := log.FromContext(ctx)
	l.Info("writing build context to tarball", "ref", ref.String(), "outdir", dir)

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("unable to list layers: %w", err)
	}

	var merr *multierror.Error
	for i, tarLayer := range layers {
		tarL := l.WithValues("layer-index", i)
		tarRC, err := tarLayer.Uncompressed()
		if err != nil {
			tarL.Error(err, "unable to retrieve are from layer")
			merr = multierror.Append(merr, err)
			continue
		}

		if err := writeTar(ctx, tarRC, dir); err != nil {
			tarL.Error(err, "failed to write tar to disk")
			merr = multierror.Append(merr, err)
		}

		_ = tarRC.Close()
	}

	return merr.ErrorOrNil()

}

func writeTar(_ context.Context, r io.Reader, outdir string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return fmt.Errorf("error while reading tar: %s", err)
		case header == nil:
			continue
		}

		target := filepath.Join(outdir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); os.IsNotExist(err) {
				if err := os.MkdirAll(target, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", target, err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to stat directory %s: %w", target, err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			_, err = io.Copy(f, tr)
			_ = f.Close()

			if err != nil {
				return fmt.Errorf("failed to write file contents for %s: %w", target, err)
			}
		}
	}
}
