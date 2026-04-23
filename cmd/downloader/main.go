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

	"github.com/crashappsec/chalkular/internal/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	l := zap.New(zap.UseFlagOptions(&opts)).
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	ctrl.SetLogger(l)

	ctx := ctrl.LoggerInto(context.Background(), l)

	image := os.Getenv(v1beta1.EnvVarTargetIdentifier)
	platform := os.Getenv(v1beta1.EnvVarTargetVersion)

	l = l.WithValues("image", image, "platform", platform)

	var nameOpts []name.Option
	if insecure := os.Getenv("OCULAR_PARAM_INSECURE_REGISTRY"); insecure != "" {
		nameOpts = append(nameOpts, name.Insecure)
	}

	ref, err := name.ParseReference(image, nameOpts...)
	if err != nil {
		l.Error(err, "failed to parse image reference", "image", image)
		os.Exit(1)
	}

	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}

	if platform != "" {
		p, err := v1.ParsePlatform(platform)
		if err != nil {
			l.Info("failed to parse platform, skipping", "platform", platform)
		} else {
			l = l.WithValues("platform", p.String())
			remoteOpts = append(remoteOpts, remote.WithPlatform(*p))
		}
	}

	img, err := remote.Image(ref, remoteOpts...)
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
