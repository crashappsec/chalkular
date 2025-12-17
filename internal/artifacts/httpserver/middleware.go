// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/apis/apiserver"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/authenticatorfactory"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createAuthClients(config *rest.Config, httpClient *http.Client) (authenticator.Request, authorizer.Authorizer, error) {
	authenticationV1Client, err := authenticationv1.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, nil, err
	}
	authorizationV1Client, err := authorizationv1.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, nil, err
	}

	authenticatorConfig := authenticatorfactory.DelegatingAuthenticatorConfig{
		Anonymous:                &apiserver.AnonymousAuthConfig{Enabled: false}, // Require authentication.
		CacheTTL:                 1 * time.Minute,
		TokenAccessReviewClient:  authenticationV1Client,
		TokenAccessReviewTimeout: 10 * time.Second,
		WebhookRetryBackoff: &wait.Backoff{
			Duration: 500 * time.Millisecond,
			Factor:   1.5,
			Jitter:   0.2,
			Steps:    5,
		},
	}
	delegatingAuthenticator, _, err := authenticatorConfig.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	authorizerConfig := authorizerfactory.DelegatingAuthorizerConfig{
		SubjectAccessReviewClient: authorizationV1Client,
		AllowCacheTTL:             5 * time.Minute,
		DenyCacheTTL:              30 * time.Second,
		WebhookRetryBackoff: &wait.Backoff{
			Duration: 500 * time.Millisecond,
			Factor:   1.5,
			Jitter:   0.2,
			Steps:    5,
		},
	}
	delegatingAuthorizer, err := authorizerConfig.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create authorizer: %w", err)
	}
	return delegatingAuthenticator, delegatingAuthorizer, nil
}

// authorizationMiddleware provides a [gin.HandleFunc] middleware for authentication and authorization.
// The middleware uses TokenReviews for authentication and SubjectAccessReviews for authorization with the kube-apiserver.
// The caller must provide the [authorizer.AttributesRecord] to be used for authorization. The User field of the
// AttributesRecord will be set by the middleware based on the authentication result.
// This is adapted from [sigs.k8s.io/controller-runtime/pkg/metrics/filters.WithAuthenticationAndAuthorization]
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.22.4/pkg/metrics/filters/filters.go
func authorizationMiddleware(authN authenticator.Request, authZ authorizer.Authorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := log.FromContext(c)
		res, ok, err := authN.AuthenticateRequest(c.Request)
		if err != nil {
			l.Error(err, "failed to authenticate request")
			errorResponse(c, http.StatusInternalServerError, "Authentication failed")
			return
		}
		if !ok {
			l.V(4).Info("Authentication failed")
			errorResponse(c, http.StatusUnauthorized, "Unauthorized")
			return
		}

		attrs := authorizer.AttributesRecord{
			User: res.User,
			Verb: strings.ToLower(c.Request.Method),
			Path: c.Request.URL.Path,
		}

		authorized, reason, err := authZ.Authorize(c, attrs)
		if err != nil {
			msg := fmt.Sprintf("Authorization for user %s failed", attrs.User.GetName())
			l.Error(err, msg)
			errorResponse(c, http.StatusInternalServerError, msg)
			return
		}
		if authorized != authorizer.DecisionAllow {
			msg := fmt.Sprintf("Authorization denied for user %s", attrs.User.GetName())
			l.Info(fmt.Sprintf("%s: %s", msg, reason))
			errorResponse(c, http.StatusForbidden, msg)
			return
		}
	}
}
