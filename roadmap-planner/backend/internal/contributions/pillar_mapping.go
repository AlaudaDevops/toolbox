/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Pillar attribution map.
//
// /api/contributions/pillars rolls weekly PR-merged + Jira-done counts up
// by pillar. The mapping that says "this PR belongs to CI/CD, that issue
// also belongs to Tool Deployment" lives in config (TeamAnalytics.Pillars)
// rather than in Jira, because the DEVOPS Jira project's pillar issues
// don't carry a component label and we want predictable attribution
// without waiting on a Jira data fix.
//
// Multi-attribution: a single PR or issue can land under N pillars.
// Each pillar gets +1 in its weekly bucket — there is no proportional
// split. Per-member totals are unaffected (they aggregate from
// member_week_metrics which is a flat per-member rollup).

package contributions

import (
	"path"
	"sort"
	"strings"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
)

// PillarMap is the in-memory form of config.TeamAnalytics. It pre-compiles
// the lookups for hot-path queries.
type PillarMap struct {
	// repo glob → list of pillars that claim it (lowercased glob keys).
	repoGlobs []repoGlob
	// component name (lowercased) → pillars.
	componentToPillars map[string][]string
	// stable pillar order (config map is unordered) — preserves the YAML
	// declaration order for chart legends. Empty when the user did not
	// configure any pillars.
	order []string
}

type repoGlob struct {
	pattern string // lowercased glob
	pillars []string
}

// NewPillarMap builds the attribution map from the parsed config block.
// Returns a non-nil but empty *PillarMap when the config is empty so
// callers can use it without nil-guarding.
func NewPillarMap(cfg config.TeamAnalytics) *PillarMap {
	pm := &PillarMap{
		componentToPillars: map[string][]string{},
	}
	if len(cfg.Pillars) == 0 {
		return pm
	}
	// Stable order: alphabetical for now. Viper does not preserve YAML
	// map order on unmarshal, so the chart legend is alphabetical too.
	for name := range cfg.Pillars {
		pm.order = append(pm.order, name)
	}
	sort.Strings(pm.order)
	for _, name := range pm.order {
		mp := cfg.Pillars[name]
		for _, r := range mp.Repos {
			r = strings.ToLower(strings.TrimSpace(r))
			if r == "" {
				continue
			}
			pm.repoGlobs = append(pm.repoGlobs, repoGlob{pattern: r, pillars: []string{name}})
		}
		for _, c := range mp.Components {
			c = strings.ToLower(strings.TrimSpace(c))
			if c == "" {
				continue
			}
			pm.componentToPillars[c] = append(pm.componentToPillars[c], name)
		}
	}
	pm.repoGlobs = compactRepoGlobs(pm.repoGlobs)
	return pm
}

// Configured reports whether any pillar mapping is loaded.
func (p *PillarMap) Configured() bool {
	return p != nil && (len(p.repoGlobs) > 0 || len(p.componentToPillars) > 0)
}

// Order returns the configured pillar names in display order. The
// caller can use it to render the chart legend so empty pillars still
// appear as zero-stacks.
func (p *PillarMap) Order() []string {
	if p == nil {
		return nil
	}
	out := make([]string, len(p.order))
	copy(out, p.order)
	return out
}

// PillarConfigEntry is the wire-friendly view of one pillar's mapping
// (used in the /pillars response so the frontend can derive a member's
// pillar set from the same component list the backend uses).
type PillarConfigEntry struct {
	Name       string   `json:"name"`
	Components []string `json:"components,omitempty"`
}

// PublicConfig returns the configured pillars in display order, with
// component lists. Repo globs are intentionally omitted — the frontend
// does not need them, and they would only encourage drift between
// frontend and backend matching logic.
func (p *PillarMap) PublicConfig() []PillarConfigEntry {
	if p == nil {
		return nil
	}
	out := make([]PillarConfigEntry, 0, len(p.order))
	for _, name := range p.order {
		entry := PillarConfigEntry{Name: name}
		for comp, pillars := range p.componentToPillars {
			for _, pl := range pillars {
				if pl == name {
					entry.Components = append(entry.Components, comp)
					break
				}
			}
		}
		sort.Strings(entry.Components)
		out = append(out, entry)
	}
	return out
}

// PillarsForRepo returns every pillar that claims `owner/name`. Empty
// slice means "no mapping" — the caller decides whether to bucket those
// PRs under "Unassigned" or skip them.
func (p *PillarMap) PillarsForRepo(fullName string) []string {
	if p == nil || len(p.repoGlobs) == 0 {
		return nil
	}
	low := strings.ToLower(strings.TrimSpace(fullName))
	if low == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, g := range p.repoGlobs {
		ok, err := path.Match(g.pattern, low)
		if err != nil || !ok {
			continue
		}
		for _, pl := range g.pillars {
			if !seen[pl] {
				seen[pl] = true
				out = append(out, pl)
			}
		}
	}
	return out
}

// PillarsForComponents returns every pillar that claims any of the
// listed Jira component names. Comparison is case-insensitive.
func (p *PillarMap) PillarsForComponents(components []string) []string {
	if p == nil || len(p.componentToPillars) == 0 || len(components) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range components {
		key := strings.ToLower(strings.TrimSpace(c))
		if key == "" {
			continue
		}
		for _, pl := range p.componentToPillars[key] {
			if !seen[pl] {
				seen[pl] = true
				out = append(out, pl)
			}
		}
	}
	return out
}

// compactRepoGlobs merges duplicate pattern entries so we don't double-count
// when the same glob appears under multiple pillars.
func compactRepoGlobs(in []repoGlob) []repoGlob {
	if len(in) <= 1 {
		return in
	}
	idx := map[string]int{}
	var out []repoGlob
	for _, g := range in {
		if at, ok := idx[g.pattern]; ok {
			seen := map[string]bool{}
			for _, p := range out[at].pillars {
				seen[p] = true
			}
			for _, p := range g.pillars {
				if !seen[p] {
					out[at].pillars = append(out[at].pillars, p)
				}
			}
			continue
		}
		idx[g.pattern] = len(out)
		out = append(out, g)
	}
	return out
}
