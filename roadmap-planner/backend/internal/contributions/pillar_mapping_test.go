/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions_test

import (
	"sort"
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/contributions"
)

func TestPillarMap_RepoMatchSinglePillar(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {Repos: []string{"alaudadevops/tektoncd-*"}},
		},
	})
	got := pm.PillarsForRepo("AlaudaDevops/tektoncd-cli")
	if !equalUnordered(got, []string{"CI/CD"}) {
		t.Fatalf("want [CI/CD], got %v", got)
	}
	if got := pm.PillarsForRepo("AlaudaDevops/unrelated"); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestPillarMap_RepoMatchMultiPillar(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD":           {Repos: []string{"alaudadevops/helm*"}},
			"Tool Deployment": {Repos: []string{"alaudadevops/helm*"}},
		},
	})
	got := pm.PillarsForRepo("alaudadevops/helm-charts")
	if !equalUnordered(got, []string{"CI/CD", "Tool Deployment"}) {
		t.Fatalf("want [CI/CD, Tool Deployment], got %v", got)
	}
}

func TestPillarMap_ComponentMatch(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD":            {Components: []string{"Tekton"}},
			"Tool Integration": {Components: []string{"Connectors"}},
		},
	})
	got := pm.PillarsForComponents([]string{"tekton", "Connectors"})
	if !equalUnordered(got, []string{"CI/CD", "Tool Integration"}) {
		t.Fatalf("want both, got %v", got)
	}
	if got := pm.PillarsForComponents([]string{"unknown"}); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestPillarMap_Empty(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{})
	if pm == nil {
		t.Fatal("want non-nil")
	}
	if pm.Configured() {
		t.Fatal("want not configured")
	}
	if got := pm.PillarsForRepo("any/repo"); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
	if got := pm.PillarsForComponents([]string{"any"}); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
	if got := pm.PillarsForVersions([]string{"any-v1.0"}); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestPillarMap_VersionPrefixMatch(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD":            {VersionPrefixes: []string{"tektoncd-operator", "katanomi-operator"}},
			"Tool Deployment":  {VersionPrefixes: []string{"gitlab-ce-operator", "harbor-ce-operator", "sonarqube-ce-operator"}},
			"Tool Integration": {VersionPrefixes: []string{"connectors-operator"}},
		},
	})
	if !pm.Configured() {
		t.Fatal("want configured")
	}
	cases := []struct {
		name     string
		versions []string
		want     []string
	}{
		{"single prefix match", []string{"tektoncd-operator-v4.11.0"}, []string{"CI/CD"}},
		{"case insensitive", []string{"Tektoncd-Operator-V4.11.0"}, []string{"CI/CD"}},
		{"multi-version fan-out", []string{"gitlab-ce-operator-v18.8.0", "harbor-ce-operator-v2.14.3", "sonarqube-ce-operator-v2026.1.2", "nexus-ce-operator-v3.76.12"}, []string{"Tool Deployment"}},
		{"connectors", []string{"connectors-operator-v1.11.0"}, []string{"Tool Integration"}},
		{"no match", []string{"thanos-v1.0.0"}, nil},
		{"meta-only is not a prefix match", []string{"v4.x"}, nil},
		{"empty input", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pm.PillarsForVersions(tc.versions)
			if !equalUnordered(got, tc.want) {
				t.Fatalf("PillarsForVersions(%v) = %v, want %v", tc.versions, got, tc.want)
			}
		})
	}
}

func TestPillarMap_VersionPrefixGlob(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"NFR": {VersionPrefixes: []string{"thanos-*"}},
		},
	})
	if got := pm.PillarsForVersions([]string{"thanos-v1.0.0"}); !equalUnordered(got, []string{"NFR"}) {
		t.Fatalf("want [NFR], got %v", got)
	}
	if got := pm.PillarsForVersions([]string{"thanos"}); len(got) != 0 {
		t.Fatalf("bare 'thanos' should not match 'thanos-*'; got %v", got)
	}
}

func TestPillarMap_VersionPrefixMultiPillar(t *testing.T) {
	// Same prefix declared under two pillars credits both (mirrors the
	// repo-glob multi-pillar behavior).
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD":           {VersionPrefixes: []string{"shared-operator"}},
			"Tool Deployment": {VersionPrefixes: []string{"shared-operator"}},
		},
	})
	got := pm.PillarsForVersions([]string{"shared-operator-v1.0.0"})
	if !equalUnordered(got, []string{"CI/CD", "Tool Deployment"}) {
		t.Fatalf("want both, got %v", got)
	}
}

func TestPillarMap_ComponentBeforeVersion(t *testing.T) {
	// Sanity: PillarsForComponents and PillarsForVersions are independent;
	// the fallback ordering is enforced at the caller (PillarThroughput),
	// which we exercise in the route-level test below.
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {Components: []string{"Tekton"}, VersionPrefixes: []string{"tektoncd-operator"}},
		},
	})
	if got := pm.PillarsForComponents([]string{"Tekton"}); !equalUnordered(got, []string{"CI/CD"}) {
		t.Fatalf("component path: want [CI/CD], got %v", got)
	}
	if got := pm.PillarsForVersions([]string{"tektoncd-operator-v4.11.0"}); !equalUnordered(got, []string{"CI/CD"}) {
		t.Fatalf("version path: want [CI/CD], got %v", got)
	}
}

func TestPillarMap_OrderIsStableAlpha(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"Z": {Repos: []string{"o/z"}},
			"A": {Repos: []string{"o/a"}},
			"M": {Repos: []string{"o/m"}},
		},
	})
	got := pm.Order()
	want := []string{"A", "M", "Z"}
	if len(got) != len(want) {
		t.Fatalf("len(order) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPillarMap_PillarsFor_ComponentsWin(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD":           {Components: []string{"Tekton"}, VersionPrefixes: []string{"tektoncd-operator"}},
			"Tool Deployment": {VersionPrefixes: []string{"harbor-ce-operator"}},
		},
	})
	// Components match → version fallback should NOT run, even if
	// versions would also match a different pillar.
	got := pm.PillarsFor([]string{"Tekton"}, []string{"harbor-ce-operator-v2.14.3"})
	if !equalUnordered(got, []string{"CI/CD"}) {
		t.Fatalf("components-win failed: %v", got)
	}
}

func TestPillarMap_PillarsFor_VersionFallback(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {Components: []string{"Tekton"}, VersionPrefixes: []string{"tektoncd-operator"}},
		},
	})
	got := pm.PillarsFor(nil, []string{"tektoncd-operator-v4.11.0"})
	if !equalUnordered(got, []string{"CI/CD"}) {
		t.Fatalf("version-fallback failed: %v", got)
	}
	// Empty components AND empty versions → empty result.
	if got := pm.PillarsFor(nil, nil); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestPillarMap_ComponentsFor_DirectComponentsWin(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {Components: []string{"tektoncd-operator"}, VersionPrefixes: []string{"tektoncd-operator"}},
		},
	})
	got := pm.ComponentsFor([]string{"Tekton", " ", "Tekton", "Triggers"}, []string{"tektoncd-operator-v4.11.0"})
	// Verbatim case preserved, dedup case-insensitive, empties dropped,
	// and the version fallback does NOT run when direct components are
	// present.
	if !equalUnordered(got, []string{"Tekton", "Triggers"}) {
		t.Fatalf("direct-components-win failed: %v", got)
	}
}

func TestPillarMap_ComponentsFor_LongestPrefixWins(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {
				Components:      []string{"tekton", "tekton-cli", "tektoncd-operator"},
				VersionPrefixes: []string{"tektoncd-operator", "tekton-cli"},
			},
		},
	})
	cases := []struct {
		name     string
		versions []string
		want     []string
	}{
		{"longest match — full operator", []string{"tektoncd-operator-v4.11.0"}, []string{"tektoncd-operator"}},
		{"longest match — cli", []string{"tekton-cli-v1.0.0"}, []string{"tekton-cli"}},
		{"multi-version dedupes", []string{"tektoncd-operator-v4.11.0", "tektoncd-operator-v4.10.0"}, []string{"tektoncd-operator"}},
		{"no version match", []string{"unknown-v1.0.0"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pm.ComponentsFor(nil, tc.versions)
			if !equalUnordered(got, tc.want) {
				t.Fatalf("ComponentsFor(nil, %v) = %v, want %v", tc.versions, got, tc.want)
			}
		})
	}
}

func TestPillarMap_ComponentsFor_PillarMatchedByVersionPrefixOnly(t *testing.T) {
	// Pillar matches the version (via versionPrefixes) but its
	// components[] doesn't contain anything that's a prefix of the
	// version. Expected: no derived component (we don't invent one,
	// and we don't return the whole components[] list).
	pm := contributions.NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {
				Components:      []string{"tekton"},
				VersionPrefixes: []string{"tektoncd-operator"},
			},
		},
	})
	got := pm.ComponentsFor(nil, []string{"tektoncd-operator-v4.11.0"})
	// "tekton" is NOT a versionMatches() prefix of "tektoncd-operator-v4.11.0"
	// (would need "tekton-" or "tektonv" boundary). So we return nothing
	// rather than guessing.
	if len(got) != 0 {
		t.Fatalf("want empty (no longest-prefix candidate), got %v", got)
	}
}

func TestPillarMap_ComponentsFor_Empty(t *testing.T) {
	pm := contributions.NewPillarMap(config.TeamAnalytics{})
	if got := pm.ComponentsFor(nil, nil); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
	if got := pm.ComponentsFor([]string{"X"}, nil); !equalUnordered(got, []string{"X"}) {
		t.Fatalf("direct comps should pass through even without config, got %v", got)
	}
}

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
