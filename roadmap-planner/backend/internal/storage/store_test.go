/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestSQLiteRoundTrip exercises the full surface area of the SQLite
// store: open → migrate → write a member, a collection run, a snapshot,
// a PR, a review, and a rollup row → read them back. It is the one test
// that catches schema/code drift; if this passes, the rest of the
// dashboards have a working datastore underneath.
func TestSQLiteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenSQLite(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	// ---- members ----
	mem := Member{
		ID:            "alice",
		DisplayName:   "Alice Tan",
		Email:         "alice@alauda.io",
		JiraAccountID: "abc-123",
		GitHubLogin:   "alicetan",
		Active:        true,
	}
	if err := store.UpsertMember(ctx, mem); err != nil {
		t.Fatalf("upsert member: %v", err)
	}
	mems, err := store.ListMembers(ctx)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(mems) != 1 || mems[0].ID != "alice" || !mems[0].Active {
		t.Fatalf("members round-trip failed: %+v", mems)
	}

	// Upsert again with a new display name — should update, not insert.
	mem.DisplayName = "Alice Tan (updated)"
	if err := store.UpsertMember(ctx, mem); err != nil {
		t.Fatalf("re-upsert member: %v", err)
	}
	mems, _ = store.ListMembers(ctx)
	if len(mems) != 1 || mems[0].DisplayName != "Alice Tan (updated)" {
		t.Fatalf("upsert did not update: %+v", mems)
	}

	// ---- collection run + issue snapshots ----
	run := CollectionRun{
		ID:          "run-1",
		CapturedAt:  now,
		Source:      "jira",
		DurationMs:  4321,
		RecordCount: 2,
	}
	if err := store.WriteCollectionRun(ctx, run); err != nil {
		t.Fatalf("write run: %v", err)
	}

	resolved := now
	issues := []IssueSnapshot{
		{
			IssueKey: "DEVOPS-1", IssueType: "Epic", Status: "Done",
			AssigneeID: "alice", PillarID: "essentials",
			Components:  []string{"argo-cd", "tekton"},
			Versions:    []string{"argo-cd-2.9.0"},
			SprintID:    "Sprint 26.05",
			StoryPoints: 3,
			CreatedAt:   now.Add(-72 * time.Hour),
			ResolvedAt:  &resolved,
		},
		{
			IssueKey: "DEVOPS-2", IssueType: "Bug", Status: "In Progress",
			AssigneeID: "alice", PillarID: "ci-cd",
			Components: []string{"jenkins"},
			Versions:   nil,
			CreatedAt:  now.Add(-24 * time.Hour),
		},
	}
	if err := store.WriteIssueSnapshots(ctx, run.ID, issues); err != nil {
		t.Fatalf("write snapshots: %v", err)
	}

	latest, err := store.LatestCollectionRun(ctx, "jira")
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if latest == nil || latest.ID != "run-1" || latest.RecordCount != 2 {
		t.Fatalf("latest run mismatch: %+v", latest)
	}

	// ---- PRs and reviews ----
	merged := now
	first := now.Add(-2 * time.Hour)
	prs := []PullRequest{
		{
			ID: "alaudadevops/toolbox#173", RepoID: "alaudadevops/toolbox",
			Number: 173, Title: "ci: smoke test", State: "merged",
			AuthorID: "alice", HeadBranch: "DEVOPS-1-smoke", BaseBranch: "main",
			Additions: 42, Deletions: 8, ChangedFiles: 3,
			JiraKey:       "DEVOPS-1",
			CreatedAt:     now.Add(-12 * time.Hour),
			FirstReviewAt: &first,
			MergedAt:      &merged,
			FetchedAt:     now,
		},
	}
	if err := store.UpsertPullRequests(ctx, prs); err != nil {
		t.Fatalf("upsert PRs: %v", err)
	}
	// re-upsert: must not fail
	if err := store.UpsertPullRequests(ctx, prs); err != nil {
		t.Fatalf("re-upsert PRs: %v", err)
	}

	reviews := []PRReview{
		{
			ID: "alaudadevops/toolbox#173/r1", PRID: "alaudadevops/toolbox#173",
			ReviewerID: "alice", State: "approved",
			SubmittedAt: now.Add(-1 * time.Hour),
		},
	}
	if err := store.UpsertPRReviews(ctx, reviews); err != nil {
		t.Fatalf("upsert reviews: %v", err)
	}

	// ---- rollup read path ----
	// We insert directly here because the aggregator is not yet wired in.
	weekStart := mondayOf(now)
	if _, err := store.DB().ExecContext(ctx, `
		INSERT INTO member_week_metrics
			(member_id, week_start, pillar_id, component,
			 jira_issues_done, jira_points_done, prs_merged, prs_reviewed, review_latency_p50_hours)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"alice", weekStart, "essentials", "argo-cd",
		1, 3.0, 1, 1, 1.5,
	); err != nil {
		t.Fatalf("insert rollup: %v", err)
	}

	rows, err := store.MemberWeekMetrics(ctx, MemberWeekQuery{
		From: weekStart.Add(-7 * 24 * time.Hour),
		To:   weekStart.Add(7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("read rollup: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 rollup, got %d", len(rows))
	}
	r := rows[0]
	if r.MemberID != "alice" || r.PRsMerged != 1 || r.JiraIssuesDone != 1 {
		t.Fatalf("rollup mismatch: %+v", r)
	}
	if r.ReviewLatencyP50Hours == nil || *r.ReviewLatencyP50Hours != 1.5 {
		t.Fatalf("review latency mismatch: %+v", r)
	}
}

func mondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	d := t.AddDate(0, 0, -(wd - 1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}
