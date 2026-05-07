/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// TestAggregatorJiraRollup writes two snapshots of the same Jira issue
// (a backfill snapshot + an incremental snapshot) and verifies the
// rollup counts the issue exactly once, using the latest snapshot's
// resolved_at. This exercises the window-function de-duplication that
// is the trickiest part of the query.
func TestAggregatorJiraRollup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "agg.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := store.UpsertMember(ctx, storage.Member{
		ID: "alice", DisplayName: "Alice Tan", Active: true,
	}); err != nil {
		t.Fatalf("upsert member: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	// Pin resolved_at to a known week so the rollup-window query hits.
	resolved := MondayOf(now).Add(36 * time.Hour) // mid-week
	weekStart := MondayOf(resolved)

	// First "backfill" run.
	earlyRun := storage.CollectionRun{
		ID: "jira-backfill-1", CapturedAt: now.Add(-2 * time.Hour),
		Source: "jira", DurationMs: 100, RecordCount: 1,
	}
	if err := store.WriteCollectionRun(ctx, earlyRun); err != nil {
		t.Fatalf("run1: %v", err)
	}
	if err := store.WriteIssueSnapshots(ctx, earlyRun.ID, []storage.IssueSnapshot{
		{
			IssueKey: "DEVOPS-1", IssueType: "Story", Status: "In Progress",
			AssigneeID: "alice", StoryPoints: 3,
			CreatedAt: resolved.Add(-72 * time.Hour),
		},
	}); err != nil {
		t.Fatalf("snap1: %v", err)
	}

	// Later "incremental" run — same issue, but now resolved with
	// updated story points. The aggregator must use *this* row.
	laterRun := storage.CollectionRun{
		ID: "jira-2", CapturedAt: now.Add(-1 * time.Hour),
		Source: "jira", DurationMs: 100, RecordCount: 1,
	}
	if err := store.WriteCollectionRun(ctx, laterRun); err != nil {
		t.Fatalf("run2: %v", err)
	}
	if err := store.WriteIssueSnapshots(ctx, laterRun.ID, []storage.IssueSnapshot{
		{
			IssueKey: "DEVOPS-1", IssueType: "Story", Status: "Done",
			AssigneeID: "alice", StoryPoints: 5,
			CreatedAt:  resolved.Add(-72 * time.Hour),
			ResolvedAt: &resolved,
		},
	}); err != nil {
		t.Fatalf("snap2: %v", err)
	}

	// Run the aggregator.
	agg := NewAggregator(store)
	if err := agg.Rebuild(ctx, weekStart.Add(-7*24*time.Hour), weekStart.Add(14*24*time.Hour)); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	rows, err := store.MemberWeekMetrics(ctx, storage.MemberWeekQuery{
		From: weekStart.Add(-7 * 24 * time.Hour),
		To:   weekStart.Add(14 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 rollup row, got %d: %+v", len(rows), rows)
	}
	r := rows[0]
	if r.MemberID != "alice" {
		t.Fatalf("member=%q want alice", r.MemberID)
	}
	if r.JiraIssuesDone != 1 {
		t.Fatalf("issues_done=%d want 1 (must dedupe across runs)", r.JiraIssuesDone)
	}
	if r.JiraPointsDone != 5 {
		t.Fatalf("points_done=%v want 5 (must use latest run's story points)", r.JiraPointsDone)
	}
	if !r.WeekStart.Equal(weekStart) {
		t.Fatalf("week_start=%s want %s", r.WeekStart, weekStart)
	}
}
