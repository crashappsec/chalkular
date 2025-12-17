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
//	"fmt"
//	"strings"
//
//	"github.com/crashappsec/chalkular/api/webserver"
//	v1 "k8s.io/api/authentication/v1"
//	authzv1 "k8s.io/api/authorization/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//)
//
//func (s *Server) authorizeRequest(ctx context.Context, authorization string, ra *authzv1.ResourceAttributes) error {
//	if !strings.HasPrefix(authorization, "Bearer ") {
//		return fmt.Errorf("%w: invalid authorization header", webserver.ErrUnauthenticated)
//	}
//
//	token := strings.TrimPrefix(authorization, "Bearer ")
//
//	user, err := s.authenticateToken(ctx, token)
//	if err != nil {
//		return fmt.Errorf("%w: invalid bearer token", webserver.ErrUnauthenticated)
//	}
//
//	allowed, err := s.authorizeUser(ctx, user, ra)
//	if err != nil {
//		return err
//	}
//
//	if !allowed {
//		return fmt.Errorf("%w: user does not sufficient permissions", webserver.ErrUnauthorized)
//	}
//
//	return nil
//
//}
//
//func (s *Server) authenticateToken(ctx context.Context, token string) (string, error) {
//	review, err := s.kubeClient.AuthenticationV1().TokenReviews().Create(ctx, &v1.TokenReview{
//		Spec: v1.TokenReviewSpec{
//			Token: token,
//		},
//	}, metav1.CreateOptions{})
//
//	if err != nil {
//		return "", err
//	}
//
//	return review.Spec.Token, nil
//
//}
//
//func (s *Server) authorizeUser(ctx context.Context, user string, ra *authzv1.ResourceAttributes) (bool, error) {
//	sar := &authzv1.SubjectAccessReview{
//		Spec: authzv1.SubjectAccessReviewSpec{
//			ResourceAttributes: ra,
//			User:               user,
//		},
//	}
//
//	sarResult, err := s.kubeClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
//	if err != nil {
//		return false, err
//	}
//
//	return sarResult.Status.Allowed, nil
//}
