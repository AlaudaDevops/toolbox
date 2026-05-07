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
