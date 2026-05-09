/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultWildcardTTL is how long a resolved group-spec expansion stays
// cached. Operators rarely add or archive projects within a 24h window,
// so a daily refresh keeps the API budget tiny while still picking up
// new projects in bounded time.
const DefaultWildcardTTL = 24 * time.Hour

// GroupSpec describes one entry in the operator-curated `gitlab.groups`
// list. Three forms are supported:
//
//   - "group/sub/proj"  — exact project lookup, no wildcard.
//   - "group/*"         — direct children of `group` only (no recursion
//                         into subgroups).
//   - "group/**"        — entire subtree under `group`, recursively.
//
// The "**" form is the common case for an org-wide "everything under
// devops" sweep; "*" is for narrower per-team groups.
type GroupSpec struct {
	Group            string // top-level path, e.g. "devops"
	IncludeSubgroups bool   // true for "**", false for "*" or exact
	Exact            string // non-empty → exact project path, no listing
}

// ParseGroupSpec parses one entry from the `gitlab.groups` config list.
// Returns ok=false on malformed inputs (caller logs and skips).
func ParseGroupSpec(raw string) (GroupSpec, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "/") {
		return GroupSpec{}, false
	}
	if strings.HasSuffix(raw, "/**") {
		group := strings.TrimSuffix(raw, "/**")
		if group == "" {
			return GroupSpec{}, false
		}
		return GroupSpec{Group: group, IncludeSubgroups: true}, true
	}
	if strings.HasSuffix(raw, "/*") {
		group := strings.TrimSuffix(raw, "/*")
		if group == "" {
			return GroupSpec{}, false
		}
		return GroupSpec{Group: group, IncludeSubgroups: false}, true
	}
	// Exact project path: must contain at least one '/'.
	if !strings.Contains(raw, "/") {
		return GroupSpec{}, false
	}
	return GroupSpec{Exact: raw}, true
}

// IsGlob reports whether the spec needs an API listing (vs. a single
// project lookup).
func (g GroupSpec) IsGlob() bool { return g.Exact == "" }

// wildcardCacheEntry holds a resolved expansion plus its mint time.
type wildcardCacheEntry struct {
	projects []Project
	resolved time.Time
}

type wildcardCache struct {
	mu      sync.Mutex
	entries map[string]wildcardCacheEntry
}

func newWildcardCache() *wildcardCache {
	return &wildcardCache{entries: make(map[string]wildcardCacheEntry)}
}

func (w *wildcardCache) get(key string, ttl time.Duration, now time.Time) ([]Project, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e, ok := w.entries[key]
	if !ok {
		return nil, false
	}
	if now.Sub(e.resolved) > ttl {
		return nil, false
	}
	out := make([]Project, len(e.projects))
	copy(out, e.projects)
	return out, true
}

func (w *wildcardCache) put(key string, projects []Project, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	stored := make([]Project, len(projects))
	copy(stored, projects)
	w.entries[key] = wildcardCacheEntry{projects: stored, resolved: now}
}

// cacheKey distinguishes the two listing modes for the same group.
func (g GroupSpec) cacheKey() string {
	suffix := "/*"
	if g.IncludeSubgroups {
		suffix = "/**"
	}
	return g.Group + suffix
}

// expandSpec returns the cached or freshly fetched project list for one
// spec. Errors are returned per-spec so the syncer can keep going.
func (s *Syncer) expandSpec(ctx context.Context, spec GroupSpec, now time.Time) ([]Project, error) {
	if !spec.IsGlob() {
		// Exact-project form: cache by full path.
		if cached, ok := s.wildcardCache.get(spec.Exact, s.WildcardTTL, now); ok {
			return cached, nil
		}
		p, err := s.client.GetProject(ctx, spec.Exact)
		if err != nil {
			return nil, fmt.Errorf("get project %s: %w", spec.Exact, err)
		}
		out := []Project{*p}
		s.wildcardCache.put(spec.Exact, out, now)
		return out, nil
	}
	if cached, ok := s.wildcardCache.get(spec.cacheKey(), s.WildcardTTL, now); ok {
		return cached, nil
	}
	projects, err := s.client.ListGroupProjects(ctx, spec.Group, ListGroupProjectsOptions{
		IncludeSubgroups: spec.IncludeSubgroups,
		IncludeArchived:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("list group %s: %w", spec.Group, err)
	}
	s.wildcardCache.put(spec.cacheKey(), projects, now)
	return projects, nil
}
