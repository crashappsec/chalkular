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
	"sync"

	chalkularv1beta1 "github.com/crashappsec/chalkular/api/v1beta1"
	"github.com/google/cel-go/cel"
	"k8s.io/utils/lru"
)

// Compiler compiles and caches the CEL expressions for a
// [v1beta1.ChalkReportPolicy] keyed by "<uid>"
type Compiler struct {
	env   *cel.Env
	cache *lru.Cache
	mu    sync.Mutex
}

func NewCompiler(cacheSize int) (*Compiler, error) {
	env, err := cel.NewEnv(
		cel.Variable("chalkmark", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("report", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating CEL env: %w", err)
	}

	cache := lru.New(cacheSize)

	return &Compiler{env: env, cache: cache}, nil
}

func (c *Compiler) Clear() {
	c.cache.Clear()
}

// Get returns a compiled policy, compiling and caching it on first access.
func (c *Compiler) Get(policyResource *chalkularv1beta1.ChalkReportPolicy) (*CompiledPolicy, error) {
	key := cacheKey(policyResource)
	c.mu.Lock()
	defer c.mu.Unlock()

	if hit, ok := c.cache.Get(key); ok {
		policy, ok := hit.(*CompiledPolicy)
		if !ok {
			return nil, fmt.Errorf("invalid type in cache, got %s", reflect.TypeOf(hit))
		}
		if policy.ObservedGeneration == policyResource.Generation {
			return policy, nil
		}
	}

	compiled, err := c.compile(policyResource)
	if err != nil {
		return nil, err
	}

	c.cache.Add(key, compiled)
	return compiled, nil
}

// Remove removes the compiled policy from the cache
func (c *Compiler) Remove(policyDef *chalkularv1beta1.ChalkReportPolicy) error {
	key := cacheKey(policyDef)
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Remove(key)
	return nil
}

func (c *Compiler) compile(policy *chalkularv1beta1.ChalkReportPolicy) (*CompiledPolicy, error) {
	s := policy.Spec

	match, err := c.program(s.MatchCondition)
	if err != nil {
		return nil, fmt.Errorf("matchCondition: %w", err)
	}

	target, err := c.program(s.Extraction.Target)
	if err != nil {
		return nil, fmt.Errorf("extraction.target: %w", err)
	}

	compiled := &CompiledPolicy{
		ObservedGeneration: policy.Generation,
		MatchCondition:     match,
		Target:             target,
	}

	if s.Extraction.DownloaderParams != nil {
		compiled.DownloaderParams, err = c.program(*s.Extraction.DownloaderParams)
		if err != nil {
			return nil, fmt.Errorf("extraction.downloaderParams: %w", err)
		}
	}

	if s.Extraction.ProfileParams != nil {
		compiled.ProfileParams, err = c.program(*s.Extraction.ProfileParams)
		if err != nil {
			return nil, fmt.Errorf("extraction.scannerParams: %w", err)
		}
	}

	return compiled, nil
}

func (c *Compiler) program(expr string) (cel.Program, error) {
	ast, issues := c.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prog, err := c.env.Program(ast)
	if err != nil {
		return nil, err
	}

	return prog, nil
}

func cacheKey(p *chalkularv1beta1.ChalkReportPolicy) string {
	return string(p.UID)
}
