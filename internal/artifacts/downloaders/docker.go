// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func DownloadDockerImage(ctx context.Context, ref name.Reference, img v1.Image, path string) error {
	l := log.FromContext(ctx)
	l.Info("writing docker image to tarball", "ref", ref.String(), "path", path)
	return tarball.WriteToFile(path, ref, img)
}
