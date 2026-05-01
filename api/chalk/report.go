// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package chalk

// Key is a string key for a item
// instead a chalk report or chalk mark
type Key = string

const (
	// KeyActionID is the chalk report key
	// for the action ID value.
	KeyActionID Key = "_ACTION_ID"

	// KeyChalks is the chalk report key
	// that will contain a list of chalk marks
	// created from the operation.
	KeyChalks Key = "_CHALKS"
)
