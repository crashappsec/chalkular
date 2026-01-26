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
	"flag"
	"os"
	"regexp"

	"github.com/crashappsec/chalkular/internal/artifacts/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	version   = "unknown"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	l := zap.New(zap.UseFlagOptions(&opts)).
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	ctrl.SetLogger(l)

	ctx := ctrl.LoggerInto(context.Background(), l)

	imageRegistry := os.Getenv(v1beta1.EnvVarTargetIdentifier)
	imageVersion := os.Getenv(v1beta1.EnvVarTargetVersion)

	l = l.WithValues("imageVersion", imageVersion, "imageRegistry", imageRegistry)

	image := imageRegistry + ":" + imageVersion
	if matched, err := regexp.Match(`^sha256:[a-fA-F0-9]{64}$`, []byte(imageVersion)); err != nil {
		l.Error(err, "failed to match image version, assuming tag",
			"imageVersion", imageVersion, "imageRegistry", imageRegistry)
	} else if matched {
		image = imageRegistry + "@" + imageRegistry
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		l.Error(err, "failed to parse image reference", "image", image)
		os.Exit(1)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		l.Error(err, "unable to retieve image info", "image", image)
		os.Exit(1)
	}

	mediaType, err := img.MediaType()
	if err != nil {
		l.Error(err, "unable to determine media type for image", "image", image)
		os.Exit(1)
	}

	var downloadErr error
	switch mediaType {
	case // standard docker images
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json":
		downloadErr = downloaders.DownloadDockerImage(ctx, ref, img, "./target.tar")
	case // custom git upload
		"application/git.chalk.v1beta+tgz":
		// TODO(bryce): handle custom git upload
	}

	if downloadErr != nil {
		l.Error(downloadErr, "unable to download image", "image", image, "mediaType", mediaType)
		os.Exit(1)
	}

	l.Info("download complete")

}
