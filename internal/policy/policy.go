// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package policy

import (
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

	ForEach          cel.Program
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

type PipelineValues struct {
	Target           v1beta1.Target
	DownloaderParams []v1beta1.ParameterSetting
	ProfileParams    []v1beta1.ParameterSetting
}

func (c CompiledPolicy) Extract(report map[string]any) ([]PipelineValues, error) {
	var activations []map[string]any
	if c.ForEach != nil {
		each, err := evalForEach(c.ForEach, report)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate for each expression: %w", err)
		}

		for _, e := range each {
			activations = append(activations, map[string]any{
				"report": report,
				"each":   e,
			})
		}

	} else {
		activations = append(activations, map[string]any{"report": report})
	}

	values := make([]PipelineValues, len(activations))
	for i, a := range activations {
		target, err := evalTarget(c.Target, a)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate target: %w", err)
		}

		vals := PipelineValues{
			Target: target,
		}

		if c.ProfileParams != nil {
			vals.ProfileParams, err = evalParameters(c.ProfileParams, a)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate profile parameters: %w", err)
			}
		}

		if c.DownloaderParams != nil {
			vals.DownloaderParams, err = evalParameters(c.DownloaderParams, a)
			if err != nil {
				return nil, fmt.Errorf("failed to evalulate downloader params: %w", err)
			}
		}
		values[i] = vals
	}
	return values, nil
}

func evalForEach(p cel.Program, report map[string]any) ([]any, error) {
	val, _, err := p.Eval(map[string]any{
		"report": report,
	})
	if err != nil {
		return nil, err
	}

	switch v := val.Value().(type) {
	case []any:
		return v, nil
	case []ref.Val:
		var result []any
		for _, i := range v {
			result = append(result, i.Value())
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid type returned for 'forEach' , expected list but got %T", v)
	}
}

func evalParameters(p cel.Program, activation map[string]any) ([]v1beta1.ParameterSetting, error) {
	val, _, err := p.Eval(activation)
	if err != nil {
		return nil, err
	}

	var settings []v1beta1.ParameterSetting
	native, err := val.ConvertToNative(reflect.TypeFor[map[string]string]())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters from %v: %w", val.Value(), err)
	}

	m, ok := native.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("failed to marshal parameters, got unexpected type %T", native)
	}

	for k, v := range m {
		settings = append(settings, v1beta1.ParameterSetting{
			Name:  k,
			Value: v,
		})
	}
	return settings, nil
}

func evalTarget(p cel.Program, activation map[string]any) (v1beta1.Target, error) {
	val, _, err := p.Eval(activation)
	if err != nil {
		return v1beta1.Target{}, err
	}

	native, err := val.ConvertToNative(reflect.TypeFor[map[string]string]())
	if err != nil {
		return v1beta1.Target{}, fmt.Errorf("failed to marshal target from value %s: %w", val.Value(), err)
	}

	m, ok := native.(map[string]string)
	if !ok {
		return v1beta1.Target{}, fmt.Errorf("failed to marshal target, got unexpected type %T", native)
	}

	id, idFound := m["identifier"]
	if !idFound {
		return v1beta1.Target{}, fmt.Errorf("target must container identifier")
	}

	target := v1beta1.Target{
		Identifier: id,
	}

	if ver, verFound := m["version"]; verFound {
		target.Version = ver
	}
	return target, nil
}
