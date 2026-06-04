/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"reflect"
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
	"go.uber.org/zap"
)

// TestParseVersionName guards the invalid-component fix: names that do
// not carry a component-X.Y.Z pattern must return an empty component
// instead of the legacy whole-name fallback that polluted every
// per-component metric with buckets like "0.3" or "v2.1".
func TestParseVersionName(t *testing.T) {
	cases := []struct {
		name                string
		wantComponent       string
		major, minor, patch int
	}{
		// Valid names — component extracted.
		{"argo-cd-2.9.0", "argo-cd", 2, 9, 0},
		{"tektoncd-operator-v4.6.3", "tektoncd-operator", 4, 6, 3},
		{"harbor 1.2.3", "harbor", 1, 2, 3},
		{"connectors-operator-1.2.3-rc1", "connectors-operator", 1, 2, 3},
		// Legacy / invalid names — no component.
		{"0.3", "", 0, 0, 0},
		{"v2.1", "", 0, 0, 0},
		{"1.0", "", 0, 0, 0},
		{"Sprint 2024", "", 0, 0, 0},
		{"", "", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			component, major, minor, patch := parseVersionName(tc.name)
			if component != tc.wantComponent || major != tc.major || minor != tc.minor || patch != tc.patch {
				t.Errorf("parseVersionName(%q) = (%q, %d, %d, %d), want (%q, %d, %d, %d)",
					tc.name, component, major, minor, patch,
					tc.wantComponent, tc.major, tc.minor, tc.patch)
			}
		})
	}
}

// TestDropInvalidComponentReleases covers both invalid classes: empty
// component (unparsable version name) and metrics.exclude_plugins (D6 —
// v3-era plugins).
func TestDropInvalidComponentReleases(t *testing.T) {
	c := &Collector{
		config: &config.Metrics{
			ExcludePlugins: []string{"katanomi", "knative", "jenkins", "tekton-operator"},
		},
		logger: zap.NewNop(),
	}
	in := []models.EnrichedRelease{
		{Name: "tektoncd-operator-v4.6.3", Component: "tektoncd-operator"},
		{Name: "0.3", Component: ""},                                // unparsable → dropped
		{Name: "katanomi-v3.1.0", Component: "katanomi"},            // D6 → dropped
		{Name: "tekton-operator-v3.20.0", Component: "tekton-operator"}, // D6 → dropped
		{Name: "argo-cd-2.9.0", Component: "argo-cd"},
	}
	got := c.dropInvalidComponentReleases(in)
	want := []string{"tektoncd-operator", "argo-cd"}
	gotComponents := make([]string, 0, len(got))
	for _, r := range got {
		gotComponents = append(gotComponents, r.Component)
	}
	if !reflect.DeepEqual(gotComponents, want) {
		t.Errorf("kept components = %v, want %v", gotComponents, want)
	}
}

// TestDropExcludedComponents covers the issue-side path: D6 plugins are
// removed from issue component lists while everything else is kept.
func TestDropExcludedComponents(t *testing.T) {
	c := &Collector{config: &config.Metrics{ExcludePlugins: []string{"katanomi", "jenkins"}}}
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"mixed", []string{"katanomi", "tektoncd-operator", "jenkins"}, []string{"tektoncd-operator"}},
		{"all excluded", []string{"katanomi"}, []string{}},
		{"none excluded", []string{"argo-cd"}, []string{"argo-cd"}},
		{"empty", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.dropExcludedComponents(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dropExcludedComponents(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
