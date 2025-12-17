// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package analyze

import (
	"context"
	"fmt"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/crashappsec/chalkular/api/v1beta1/artifacts"
	ocularV1beta1 "github.com/crashappsec/ocular/api/v1beta1"
	"github.com/crashappsec/ocular/pkg/generated/clientset"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/hashicorp/go-multierror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type eventBus = chan artifacts.AnalysisRequest

type Scheduler struct {
	eventBus  eventBus
	ocularCS  *clientset.Clientset
	mgrClient client.Client
}

func NewScheduler(mgrClient client.Client, cfg *rest.Config) (*Scheduler, error) {
	e := make(chan artifacts.AnalysisRequest)

	ocularCS, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	scheduler := &Scheduler{
		eventBus:  e,
		ocularCS:  ocularCS,
		mgrClient: mgrClient,
	}

	return scheduler, nil
}

func (s *Scheduler) GetClient() *Client {
	return &Client{
		eventBus: s.eventBus,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	l := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-s.eventBus:
			if err := s.scheduleAnalysis(
				ctx, req.ImageRef, req.Namespace,
			); err != nil {
				l.Error(err, "unable to create new pipeline",
					"imageRef", req.ImageRef,
					"namespace", req.Namespace,
				)

			}
		}
	}
}

// scheduleAnalysis creates and submits pipelines for scanning the given artifact
// in the given namespace.
func (s *Scheduler) scheduleAnalysis(ctx context.Context, imageRef, namespace string) error {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("unable to parse artifact URI: %w", err)
	}

	pipelines, err := s.createPipelinesForArtifact(ctx, ref, namespace)
	if err != nil {
		return fmt.Errorf("unable to retrieve profile for artifact: %w", err)
	}

	var merr *multierror.Error
	for _, pipeline := range pipelines {
		_, err = s.ocularCS.ApiV1beta1().Pipelines(namespace).
			Create(ctx, pipeline, metav1.CreateOptions{})
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("unable to create pipeline in namespace %s: %w", namespace, err))
		}
	}

	return merr.ErrorOrNil()
}

func (s *Scheduler) createPipelinesForArtifact(ctx context.Context, artifact name.Reference, namespace string) ([]*ocularV1beta1.Pipeline, error) {
	l := log.FromContext(ctx)

	mappings := &chalkularv1beta1.ArtifactMediaTypeMappingList{}
	err := s.mgrClient.List(ctx, mappings, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("unable to list artifact media type mappings: %w", err)
	}
	desc, err := remote.Get(artifact)
	if err != nil {
		return nil, err
	}

	var pipelines []*ocularV1beta1.Pipeline
	for _, mapping := range mappings.Items {
		if !mapping.Status.Profile.Available {
			l.Info(fmt.Sprintf("skipping mapping %s, profile unavailable", mapping.Name))
			continue
		}
		for _, mediaType := range mapping.Spec.MediaTypes {
			if types.MediaType(mediaType) != desc.MediaType {
				continue
			}
			pipeline := &ocularV1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "chalkular-",
					Namespace:    namespace,
				},
				Spec: ocularV1beta1.PipelineSpec{
					DownloaderRef: v1.ObjectReference{
						// FIXME(bryce): we somehow need to manage a "chalkular-artifact" downloader in every namespace
						// currently, we assume it exists and will probably install it manually into every namespace
						// we want to support chalkular artifacts in, but this is not ideal.
						// Maybe whenever a mapping is created in a namespace, we check if the chalkular-artifact downloader exists,
						// and create it if not
						Name: "chalkular-artifacts",
					},
					ProfileRef: v1.ObjectReference{
						Name: mapping.Status.Profile.Name,
					},
					TTLSecondsMaxLifetime:    mapping.Spec.TTLSecondsMaxLifetime,
					TTLSecondsAfterFinished:  mapping.Spec.TTLSecondsAfterFinished,
					ScanServiceAccountName:   mapping.Spec.ScanServiceAccountName,
					UploadServiceAccountName: mapping.Spec.UploadServiceAccountName,
				},
			}
			pipelines = append(pipelines, pipeline)
			break
		}
	}
	return pipelines, nil
}
