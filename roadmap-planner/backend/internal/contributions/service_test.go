/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// TestTeamOverview_PillarsViaVersionPrefixFallback writes one member
// whose only issue carries a Jira component that does NOT match any
// configured pillar, but whose fixVersion DOES match a pillar via
// `version_prefixes`. This is the case that hit the team-table pillar
// filter in the dev pod: the chart aggregator routed bozhou's points
// to CI/CD via the version_prefix `tektoncd-operator`, but the team
// table dropped him from the CI/CD filter because the frontend was
// re-deriving pillars from components alone. Pillars[] must include
// the version-prefix-matched pillar so the team table and chart
// agree.
func TestTeamOverview_PillarsViaVersionPrefixFallback(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "team.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "bozhou", DisplayName: "Bo Zhou", Active: true,
	}); err != nil {
		t.Fatalf("member: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "alice", DisplayName: "Alice Tan", Active: true,
	}); err != nil {
		t.Fatalf("member: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	resolved := MondayOf(now).Add(36 * time.Hour)
	weekStart := MondayOf(resolved)
	run := storage.CollectionRun{
		ID: "jira-1", CapturedAt: now.Add(-1 * time.Hour),
		Source: "jira", DurationMs: 1, RecordCount: 3,
	}
	if err := store.WriteCollectionRun(ctx, run); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := store.WriteIssueSnapshots(ctx, run.ID, []storage.IssueSnapshot{
		{
			// Component-only path: matches CI/CD via PillarsForComponents.
			IssueKey: "DEVOPS-1", IssueType: "Story", Status: "Done",
			AssigneeID: "alice", StoryPoints: 1,
			Components: []string{"Tekton"},
			CreatedAt:  resolved.Add(-72 * time.Hour),
			ResolvedAt: &resolved,
		},
		{
			// Version-prefix-only path: literal Jira component is
			// `tektoncd-operator`, which is NOT in CI/CD's components
			// list — only its `version_prefixes` covers it. Without
			// the fallback this issue's pillar attribution is lost.
			IssueKey: "DEVOPS-2", IssueType: "Story", Status: "Done",
			AssigneeID: "bozhou", StoryPoints: 2,
			Components: []string{"tektoncd-operator"},
			Versions:   []string{"tektoncd-operator-v0.78.0"},
			CreatedAt:  resolved.Add(-72 * time.Hour),
			ResolvedAt: &resolved,
		},
		{
			// Bare component, no fixVersion → genuinely unattributed.
			IssueKey: "DEVOPS-3", IssueType: "Story", Status: "Done",
			AssigneeID: "bozhou", StoryPoints: 1,
			Components: []string{"unmapped"},
			CreatedAt:  resolved.Add(-72 * time.Hour),
			ResolvedAt: &resolved,
		},
	}); err != nil {
		t.Fatalf("snap: %v", err)
	}

	// Run the aggregator so MemberWeekMetrics has rows.
	agg := NewAggregator(store)
	if err := agg.Rebuild(ctx, weekStart.Add(-7*24*time.Hour), weekStart.Add(14*24*time.Hour)); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	svc := NewService(store)
	pm := NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {
				Components:      []string{"Tekton"},
				VersionPrefixes: []string{"tektoncd-operator", "katanomi-operator"},
			},
		},
	})
	svc.SetPillarMap(pm)

	out, err := svc.TeamOverview(ctx, storage.MemberWeekQuery{
		From: weekStart.Add(-7 * 24 * time.Hour),
		To:   weekStart.Add(14 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("team overview: %v", err)
	}
	by := map[string]MemberSummary{}
	for _, ms := range out {
		by[ms.MemberID] = ms
	}
	alice, ok := by["alice"]
	if !ok {
		t.Fatalf("alice missing from rollup; got %+v", by)
	}
	if !reflect.DeepEqual(sorted(alice.Pillars), []string{"CI/CD"}) {
		t.Fatalf("alice.Pillars = %v, want [CI/CD]", alice.Pillars)
	}
	bozhou, ok := by["bozhou"]
	if !ok {
		t.Fatalf("bozhou missing from rollup; got %+v", by)
	}
	if !reflect.DeepEqual(sorted(bozhou.Pillars), []string{"CI/CD"}) {
		t.Fatalf("bozhou.Pillars = %v, want [CI/CD] (matched via version_prefix fallback)", bozhou.Pillars)
	}
	// The unmapped component AND missing fixVersion on DEVOPS-3 must
	// not poison bozhou's pillar set — pillars is a union, not an
	// intersection.
	if got := sorted(bozhou.Components); !contains(got, "tektoncd-operator") || !contains(got, "unmapped") {
		t.Fatalf("bozhou.Components = %v, want both tektoncd-operator and unmapped", got)
	}
}

// TestTeamOverview_PillarsEmpty verifies that a member whose issues
// match no pillar at all gets an empty Pillars slice (not nil panic,
// not "Unassigned" — that's a chart-bucket label, not a pillar).
func TestTeamOverview_PillarsEmpty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "team.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "carol", DisplayName: "Carol", Active: true,
	}); err != nil {
		t.Fatalf("member: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	resolved := MondayOf(now).Add(36 * time.Hour)
	weekStart := MondayOf(resolved)
	run := storage.CollectionRun{
		ID: "jira-1", CapturedAt: now.Add(-1 * time.Hour),
		Source: "jira", DurationMs: 1, RecordCount: 1,
	}
	if err := store.WriteCollectionRun(ctx, run); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := store.WriteIssueSnapshots(ctx, run.ID, []storage.IssueSnapshot{
		{
			IssueKey: "DEVOPS-9", IssueType: "Story", Status: "Done",
			AssigneeID: "carol", StoryPoints: 1,
			Components: []string{"random-thing"},
			Versions:   []string{"unknown-1.0"},
			CreatedAt:  resolved.Add(-72 * time.Hour),
			ResolvedAt: &resolved,
		},
	}); err != nil {
		t.Fatalf("snap: %v", err)
	}
	agg := NewAggregator(store)
	if err := agg.Rebuild(ctx, weekStart.Add(-7*24*time.Hour), weekStart.Add(14*24*time.Hour)); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	svc := NewService(store)
	pm := NewPillarMap(config.TeamAnalytics{
		Pillars: map[string]config.PillarMapping{
			"CI/CD": {Components: []string{"Tekton"}, VersionPrefixes: []string{"tektoncd-operator"}},
		},
	})
	svc.SetPillarMap(pm)

	out, err := svc.TeamOverview(ctx, storage.MemberWeekQuery{
		From: weekStart.Add(-7 * 24 * time.Hour),
		To:   weekStart.Add(14 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("team overview: %v", err)
	}
	if len(out) != 1 || out[0].MemberID != "carol" {
		t.Fatalf("expected carol-only rollup, got %+v", out)
	}
	if len(out[0].Pillars) != 0 {
		t.Fatalf("carol.Pillars = %v, want empty (no matcher hits)", out[0].Pillars)
	}
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}
