// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package httpserver

import "errors"

var (
	ErrUnauthenticated = errors.New("unable to authenticate user")
	ErrUnauthorized    = errors.New("unable to authorize user")
)

// APIResponse is the standard response from any [Server] endpoint
type APIResponse[T any] struct {
	Code     int    `json:"code" yaml:"code"`
	Response T      `json:"response,omitempty,omitzero" yaml:"response,omitempty,omitzero"`
	Message  string `json:"message,omitempty,omitzero" yaml:"message,omitempty,omitzero"`
}
