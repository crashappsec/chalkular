// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"path"

	v1beta1 "github.com/crashappsec/chalkular/api/v1beta1/httpserver"
	"github.com/crashappsec/chalkular/internal/reports"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Server struct {
	opts            Options
	engine          *gin.Engine
	schedulerClient *reports.SchedulerClient
}

type Options struct {
	BindAddress string
	Secure      bool
	CertDir     string
	CertName    string
	KeyName     string
	TlSOpts     []func(*tls.Config)

	DevelopmentMode bool
}

func NewServer(config *rest.Config, httpClient *http.Client, client *reports.SchedulerClient, opts Options) (*Server, error) {
	if opts.DevelopmentMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(gin.Logger(), gin.Recovery())

	s := &Server{
		opts:            opts,
		schedulerClient: client,
	}

	authN, authZ, err := createAuthClients(config, httpClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create auth clients: %w", err)
	}

	engine.GET("/health", health())

	apiV1beta1 := engine.Group("/api/v1beta1", authorizationMiddleware(authN, authZ))
	{
		apiV1beta1.POST("/report", scheduleReport(s.schedulerClient))
	}

	s.engine = engine
	return s, nil

}

func (s *Server) Start(ctx context.Context) error {
	l := log.FromContext(ctx)

	cfg := &tls.Config{
		NextProtos: []string{"h2"},
	}

	for _, op := range s.opts.TlSOpts {
		op(cfg)
	}

	srv := &http.Server{
		Addr:      s.opts.BindAddress,
		Handler:   s.engine,
		TLSConfig: cfg,
	}

	l.Info("starting artifacts HTTP server", "address", s.opts.BindAddress, "secure", s.opts.Secure)

	go func() {
		l.Info("starting http server go routine")
		// service connections
		var serverErr error
		if s.opts.Secure {
			serverErr = srv.ListenAndServeTLS(
				path.Join(s.opts.CertDir, s.opts.CertName),
				path.Join(s.opts.CertDir, s.opts.KeyName))
		} else {
			serverErr = srv.ListenAndServe()
		}
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			l.Error(serverErr, "error shutting down server")
		}
		l.Info("server exited")
	}()

	<-ctx.Done()
	l.Info("shutting down artifacts HTTP server")
	if err := srv.Close(); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}
	return ctx.Err()
}

func errorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, v1beta1.APIResponse[struct{}]{
		Code:    code,
		Message: message,
	})
}

func health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"health": "ok",
		})
	}
}
