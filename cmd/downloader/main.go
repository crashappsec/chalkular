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
	"os"
	"strings"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/crashappsec/chalkular/internal/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	version   = "unknown"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// artifactManifest is a [v1.Mainfiest] but
// additionally with an artifact type
type artifactManifest struct {
	v1.Manifest  `json:",inline"`
	ArtifactType string `json:"artifactType,omitempty"`
}

func main() {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	l := zap.New(zap.UseFlagOptions(&opts)).
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	ctrl.SetLogger(l)

	ctx := ctrl.LoggerInto(context.Background(), l)

	imageRepo := os.Getenv(v1beta1.EnvVarTargetIdentifier)
	tag := os.Getenv(v1beta1.EnvVarTargetVersion)

	l = l.WithValues("image", imageRepo, "tag", tag)
	l.Info("downloading image")

	var nameOpts []name.Option
	if insecure := os.Getenv("OCULAR_PARAM_INSECURE_REGISTRY"); insecure != "" {
		nameOpts = append(nameOpts, name.Insecure)
	}

	imageRef := imageRepo + ":" + tag
	if strings.HasPrefix(tag, "@sha256:") {
		imageRef = imageRepo + tag
	}

	ref, err := name.ParseReference(imageRef, nameOpts...)
	if err != nil {
		l.Error(err, "failed to parse image reference", "image", imageRef)
		os.Exit(1)
	}

	// key chain order:
	// 1. use k8schain (if successfully built)
	// 2. IRSA/EKS metadata endpoint
	// 3. GKE metadata endpoint
	// 4. default (i.e. DOCKER_CONFIG)
	var keychains []authn.Keychain
	if k8sKeychain, err := k8schain.NewInCluster(ctx, k8schain.Options{}); err != nil {
		l.Info("failed to build k8s auth keychain, skipping", "error-message", err.Error())
	} else {
		keychains = append(keychains, k8sKeychain)
	}

	keychains = append(keychains,
		authn.NewKeychainFromHelper(ecr.NewECRHelper()),
		google.Keychain,
		authn.DefaultKeychain,
	)
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(authn.NewMultiKeychain(keychains...)),
	}

	platform := os.Getenv("OCULAR_PARAM_PLATFORM")
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
		l.Error(err, "unable to retieve image info", "image", imageRef)
		os.Exit(1)
	}

	var manifest artifactManifest
	rawManifest, err := img.RawManifest()
	if err != nil {
		l.Error(err, "unable to retieve image manifest", "image", imageRef)
		os.Exit(1)
	}

	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		l.Error(err, "invalid JSON for image manifest")
		os.Exit(1)
	}

	if manifest.MediaType != "application/vnd.oci.image.manifest.v1+json" {
		l.Error(err, "unsupported OCI media type given, "+
			"only 'application/vnd.oci.image.manifest.v1+json' is supported currently",
			"mediaType", manifest.MediaType)
		os.Exit(1)
	}

	var downloadErr error
	// if artifact type is set, we download based on that
	switch manifest.ArtifactType {
	// Custom docker "build context" artifact type
	// is a tar.gz of context available to docker at build time
	case "application/vnd.crashoverride.chalk.build-context.v1":
		downloadErr = downloaders.DownloadBuildContext(ctx, ref, img, os.Getenv("OCULAR_TARGET_DIR"))
	// standard docker images wont have artifact type
	// so we can default to normal image
	default:
		downloadErr = downloaders.DownloadDockerImage(ctx, ref, img, "./target.tar")
	}

	if downloadErr != nil {
		l.Error(downloadErr, "unable to download image", "image", imageRef, "artifactType", manifest.ArtifactType)
		os.Exit(1)
	}

	l.Info("download complete", "artifactType", manifest.ArtifactType, "mediaType", manifest.MediaType)

}
