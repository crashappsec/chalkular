// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package policy

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
)

// CompiledPolicy holds the compiled CEL programs for a single ChalkReportPolicy.
type CompiledPolicy struct {
	ObservedGeneration int64
	MatchCondition     cel.Program
	Target             cel.Program

	// Optional //

	DownloaderParams cel.Program
	ProfileParams    cel.Program
}

func (c CompiledPolicy) Matches(report map[string]any) (bool, error) {
	policyMatch, _, err := c.MatchCondition.Eval(map[string]any{
		"report": report,
	})
	if err != nil {
		return false, err
	}
	matched, ok := policyMatch.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expected match condition to result in boolean, got %s", policyMatch.Type().TypeName())
	}
	return matched, nil
}

func (c CompiledPolicy) ExtractTargets(report map[string]any) ([]v1beta1.Target, error) {
	activation := map[string]any{
		"report": report,
	}
	return evalToListOfStruct[v1beta1.Target](c.Target, activation)

}

func (c CompiledPolicy) ExtractParameters(report map[string]any) (profile, downloader []v1beta1.ParameterSetting, err error) {
	activation := map[string]any{
		"report": report,
	}
	if c.ProfileParams != nil {
		pParams, err := evalProgramToStringMap(c.ProfileParams, activation)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to eval profile params: %w", err)
		}
		for k, v := range pParams {
			profile = append(profile, v1beta1.ParameterSetting{
				Name:  k,
				Value: v,
			})
		}
	}

	if c.DownloaderParams != nil {
		dParams, err := evalProgramToStringMap(c.DownloaderParams, activation)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to eval downloader params: %w", err)
		}
		for k, v := range dParams {
			downloader = append(downloader, v1beta1.ParameterSetting{
				Name:  k,
				Value: v,
			})
		}
	}

	return profile, downloader, nil

}

func evalProgramToStringMap(p cel.Program, activation map[string]any) (map[string]string, error) {
	val, _, err := p.Eval(activation)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string)
	m, ok := val.Value().(map[ref.Val]ref.Val)
	if !ok {
		return nil, fmt.Errorf("invalid type for cel expression, expected string map got %s", val.Type().TypeName())
	}
	for k, v := range m {
		out[fmt.Sprint(k.Value())] = fmt.Sprint(v.Value())
	}
	return out, nil
}

// evalToListOfStruct will evaluate a CEL expression to a list of structs `T`.
// It also supports the CEL expression evalutation to one `T` and creating a singleton
// list
func evalToListOfStruct[T any](p cel.Program, activation map[string]any) ([]T, error) {
	val, _, err := p.Eval(activation)
	if err != nil {
		return nil, err
	}
	fmt.Println("here", val.Value())

	var items []any
	switch v := val.Value().(type) {
	case []ref.Val:
		items = make([]any, len(v))
		for i, item := range v {
			native, err := item.ConvertToNative(reflect.TypeFor[map[string]any]())
			if err != nil {
				return nil, fmt.Errorf("converting list item %d: %w", i, err)
			}
			items[i] = native
		}

	case map[ref.Val]ref.Val: // case for singleton list
		native, err := val.ConvertToNative(reflect.TypeFor[map[string]any]())
		if err != nil {
			return nil, fmt.Errorf("unsupported expression type %T: %w", val.Value(), err)
		}
		items = []any{native}
	default:
		return nil, fmt.Errorf("unsupported expression type %T: %w", val.Value(), err)
	}

	b, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal CEL expression: %w", err)
	}

	var result []T
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("failed to marshal CEl expression into type %T: %w", result, err)
	}
	return result, nil
}
