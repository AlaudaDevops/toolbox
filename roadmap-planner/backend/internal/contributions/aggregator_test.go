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

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
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

// TestAggregatorAllowlist seeds three GitHub PRs by three different
// authors, configures an allowlist that names only one of them, and
// verifies the rollup keeps just that author. The other two authors
// have GitHub-login prefills (so they're real Jira members) but one
// is in the denylist and the other isn't in either prefill map — both
// have to disappear from `member_week_metrics`.
//
// Then it flips the allowlist off and re-runs Rebuild to confirm the
// no-filter sentinel preserves the pre-W1 shape (all three return).
func TestAggregatorAllowlist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "agg-allow.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Three Jira members, each linked to a distinct GitHub login.
	for _, m := range []storage.Member{
		{ID: "daniel", DisplayName: "Daniel", GitHubLogin: "danielfbm", Active: true},
		{ID: "zhwang", DisplayName: "ZH Wang", GitHubLogin: "zhwang", Active: true},
		{ID: "stranger", DisplayName: "External Author", GitHubLogin: "stranger", Active: true},
	} {
		if err := store.UpsertMember(ctx, m); err != nil {
			t.Fatalf("upsert %s: %v", m.ID, err)
		}
	}

	// Three merged PRs in the same week, one per author.
	now := time.Now().UTC().Truncate(time.Second)
	week := MondayOf(now).Add(36 * time.Hour)
	for i, login := range []string{"danielfbm", "zhwang", "stranger"} {
		merged := week.Add(time.Duration(i) * time.Hour)
		if err := store.UpsertPullRequests(ctx, []storage.PullRequest{{
			ID:          "alaudadevops/repo#" + login,
			Source:      "github",
			RepoID:      "alaudadevops/repo",
			Number:      i + 1,
			Title:       "PR by " + login,
			State:       "merged",
			AuthorLogin: login,
			CreatedAt:   merged.Add(-24 * time.Hour),
			MergedAt:    &merged,
			FetchedAt:   now,
		}}); err != nil {
			t.Fatalf("upsert PR %s: %v", login, err)
		}
	}

	agg := NewAggregator(store)
	// Allowlist via the prefill maps: daniel and zhwang are configured,
	// stranger is not — and zhwang is explicitly denied. After the
	// filter the only survivor is daniel (+ the bot synthetic, which
	// has no PRs in this fixture so it contributes nothing).
	agg.SetAllowlist(BuildAllowlist(config.TeamAnalytics{
		GitHubLoginPrefills: map[string]string{
			"daniel": "danielfbm",
			"zhwang": "zhwang",
		},
		MemberDenylist: []string{"zhwang"},
	}))
	rebuildFrom := MondayOf(week).Add(-7 * 24 * time.Hour)
	rebuildTo := MondayOf(week).Add(14 * 24 * time.Hour)
	if err := agg.Rebuild(ctx, rebuildFrom, rebuildTo); err != nil {
		t.Fatalf("rebuild filtered: %v", err)
	}
	rows, err := store.MemberWeekMetrics(ctx, storage.MemberWeekQuery{From: rebuildFrom, To: rebuildTo})
	if err != nil {
		t.Fatalf("read filtered: %v", err)
	}
	if len(rows) != 1 || rows[0].MemberID != "daniel" {
		t.Fatalf("filtered rollup = %+v, want exactly daniel", rows)
	}

	// Flip filter off and confirm all three rows come back.
	agg.SetAllowlist(Allowlist{})
	if err := agg.Rebuild(ctx, rebuildFrom, rebuildTo); err != nil {
		t.Fatalf("rebuild unfiltered: %v", err)
	}
	rows, err = store.MemberWeekMetrics(ctx, storage.MemberWeekQuery{From: rebuildFrom, To: rebuildTo})
	if err != nil {
		t.Fatalf("read unfiltered: %v", err)
	}
	got := map[string]bool{}
	for _, r := range rows {
		got[r.MemberID] = true
	}
	for _, id := range []string{"daniel", "zhwang", "stranger"} {
		if !got[id] {
			t.Fatalf("unfiltered rollup missing %s; got %+v", id, got)
		}
	}
}

// TestAggregatorReviewLatency exercises the W8 (2026-05-19) per-reviewer
// p50 of `first_human_review_at - created_at`.
//
// Seeds three PRs created in the same week, all reviewed by `alice` as
// the first human reviewer. Latencies (in hours): 1, 4, 9 — p50 = 4.
// One bot review is also seeded to confirm it's ignored (is_bot = 1).
func TestAggregatorReviewLatency(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "lat.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, m := range []storage.Member{
		{ID: "alice", DisplayName: "Alice", GitHubLogin: "alice", Active: true},
		{ID: "bot", DisplayName: "Bot", Active: true},
	} {
		if err := store.UpsertMember(ctx, m); err != nil {
			t.Fatalf("upsert %s: %v", m.ID, err)
		}
	}

	now := time.Now().UTC().Truncate(time.Hour)
	created := MondayOf(now).Add(36 * time.Hour)
	week := MondayOf(created)
	prs := []storage.PullRequest{}
	reviews := []storage.PRReview{}
	for i, latHours := range []int{1, 4, 9} {
		prID := "alaudadevops/x#" + itoa(i+1)
		first := created.Add(time.Duration(latHours) * time.Hour)
		prs = append(prs, storage.PullRequest{
			ID: prID, Source: "github", RepoID: "alaudadevops/x", Number: i + 1,
			Title: "PR", State: "merged",
			AuthorLogin:        "danielfbm",
			CreatedAt:          created,
			FirstReviewAt:      &first,
			FirstHumanReviewAt: &first,
			FetchedAt:          created,
		})
		reviews = append(reviews, storage.PRReview{
			ID: prID + "/r1", PRID: prID, Source: "github",
			ReviewerID: "alice", ReviewerLogin: "alice", State: "approved",
			SubmittedAt: first,
		})
	}
	// Bot review on the first PR — should be ignored.
	reviews = append(reviews, storage.PRReview{
		ID: "alaudadevops/x#1/r-bot", PRID: "alaudadevops/x#1", Source: "github",
		ReviewerID: "bot", ReviewerLogin: "renovate", State: "commented",
		SubmittedAt: created.Add(30 * time.Minute), IsBot: true,
	})
	if err := store.UpsertPullRequests(ctx, prs); err != nil {
		t.Fatalf("upsert prs: %v", err)
	}
	if err := store.UpsertPRReviews(ctx, reviews); err != nil {
		t.Fatalf("upsert reviews: %v", err)
	}

	agg := NewAggregator(store)
	if err := agg.Rebuild(ctx, week.Add(-7*24*time.Hour), week.Add(14*24*time.Hour)); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	rows, err := store.MemberWeekMetrics(ctx, storage.MemberWeekQuery{
		From:      week.Add(-7 * 24 * time.Hour),
		To:        week.Add(14 * 24 * time.Hour),
		MemberIDs: []string{"alice"},
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(rows), rows)
	}
	r := rows[0]
	if r.ReviewLatencyP50Hours == nil {
		t.Fatalf("review_latency_p50_hours nil; want 4.0")
	}
	if *r.ReviewLatencyP50Hours != 4.0 {
		t.Errorf("review_latency_p50_hours = %v, want 4.0", *r.ReviewLatencyP50Hours)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	out := []byte{}
	for n > 0 {
		out = append([]byte{'0' + byte(n%10)}, out...)
		n /= 10
	}
	if neg {
		out = append([]byte{'-'}, out...)
	}
	return string(out)
}
