// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package chalk

//
//import (
//	"context"
//	"encoding/json"
//	"errors"
//	"net/http"
//
//	"github.com/crashappsec/chalkular/api/webserver"
//	"github.com/google/go-containerregistry/pkg/v1/types"
//	authzv1 "k8s.io/api/authorization/v1"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//
//	ocularcs "github.com/crashappsec/ocular/pkg/generated/clientset"
//)
//
//type Server struct {
//	mgrClient  client.Client
//	kubeClient *kubernetes.Clientset
//	ocularCS   *ocularcs.Clientset
//	httpMux    *http.ServeMux
//
//	mediaTypeToProfiles map[types.MediaType]string
//}
//
//func NewServer(c client.Client, cfg *rest.Config) (*Server, error) {
//	kubeClient, err := kubernetes.NewForConfig(cfg)
//	if err != nil {
//		return nil, err
//	}
//
//	ocularCS, err := ocularcs.NewForConfig(cfg)
//	if err != nil {
//		return nil, err
//	}
//	mux := http.NewServeMux()
//	s := &Server{
//		httpMux:    mux,
//		mgrClient:  c,
//		ocularCS:   ocularCS,
//		kubeClient: kubeClient,
//		mediaTypeToProfiles: map[types.MediaType]string{
//			types.OCIImageIndex:      "docker",
//			types.OCIManifestSchema1: "docker",
//		},
//	}
//	mux.HandleFunc("POST /chalk/scan-artifacts/container-registry", s.scanContainerArtifact)
//
//	return s, nil
//}
//func errorResponse(rw http.ResponseWriter, code int, message string) {
//
//	err := json.NewEncoder(rw).Encode(webserver.APIResponse[struct{}]{
//		Code:    code,
//		Message: message,
//	})
//	if err != nil {
//		rw.WriteHeader(http.StatusInternalServerError)
//	}
//	rw.WriteHeader(code)
//}
//
//func successResponse[T any](rw http.ResponseWriter, response T) {
//	code := http.StatusOK
//	err := json.NewEncoder(rw).Encode(webserver.APIResponse[T]{
//		Code:     code,
//		Response: response,
//	})
//	if err != nil {
//		rw.WriteHeader(http.StatusInternalServerError)
//	} else {
//		rw.WriteHeader(code)
//	}
//}
//
//func (s *Server) scanContainerArtifact(rw http.ResponseWriter, req *http.Request) {
//	ctx := req.Context()
//
//	// var artifactRequest v1beta1.ContainerRegistryArtifactRequest
//	var artifactRequest any
//	if err := json.NewDecoder(req.Body).Decode(&artifactRequest); err != nil {
//		errorResponse(rw, http.StatusBadRequest, "unable to parse payload, bad request")
//		return
//	}
//
//	err := s.authorizeRequest(ctx, req.Header.Get("Authorization"),
//		&authzv1.ResourceAttributes{
//			Namespace: "test",
//			Verb:      "create",
//			Group:     "ocular.crashoverride.run",
//			Version:   "v1beta1",
//			Resource:  "pipelines",
//		})
//
//	if err != nil && errors.Is(err, webserver.ErrUnauthenticated) {
//		errorResponse(rw, http.StatusUnauthorized, err.Error())
//		return
//	} else if err != nil && errors.Is(err, webserver.ErrUnauthorized) {
//		errorResponse(rw, http.StatusForbidden, err.Error())
//		return
//	} else if err != nil {
//		errorResponse(rw, http.StatusInternalServerError, "unable to authorize request")
//		return
//	}
//
//	// determine what the media type is
//
//	pipelineName, err := s.triggerPipeline(ctx, "", "test")
//	if err != nil {
//		errorResponse(rw, http.StatusInternalServerError, "unable to start pipeline")
//		return
//	}
//
//	successResponse(rw, webserver.PipelineCreatedResponse{
//		Name:      pipelineName,
//		Namespace: "test",
//
//		Artifact: webserver.PipelineArtifact{
//			URI:       "test",
//			MediaType: "",
//		},
//	})
//}
//
//func (s *Server) Handle(rw http.ResponseWriter, req *http.Request) {
//	s.httpMux.ServeHTTP(rw, req)
//}
//
//func optionsFromArtifactRequest(req any) []pipelineOptions {
//	var opts []pipelineOptions
//
//	// if req.Options.ScanServiceAccoutnName != "" {
//	// opts = append(opts, pipelineWithScannerSA(req.Options.ScanServiceAccoutnName))
//	// }
//	// if req.Options.UploadServiceAccountName != "" {
//	// opts = append(opts, pipelineWithScannerSA(req.Options.UploadServiceAccountName))
//	// }
//	return opts
//}
//
//func (s *Server) NeedLeaderElection() bool {
//	return true
//}
//
//func (s *Server) Start(ctx context.Context) error {
//	return nil
//}
