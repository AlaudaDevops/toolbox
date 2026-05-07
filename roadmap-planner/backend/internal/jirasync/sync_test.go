/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package jirasync

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// fakeSearcher returns canned SnapshotIssue lists per JQL pattern. It
// records the JQLs it was called with so we can assert the backfill
// flow runs the right two queries.
type fakeSearcher struct {
	byPattern  map[string][]jira.SnapshotIssue
	calledWith []string
}

func (f *fakeSearcher) SearchSnapshots(_ context.Context, opts jira.SnapshotSearchOpts) ([]jira.SnapshotIssue, error) {
	f.calledWith = append(f.calledWith, opts.JQL)
	for pat, list := range f.byPattern {
		if strings.Contains(opts.JQL, pat) {
			return list, nil
		}
	}
	return nil, nil
}

// TestBackfillThenIncremental drives both code paths against a real
// SQLite store. Validates: schema accepts our writes; member ids stay
// stable across passes; backfill runs both JQLs (resolved + open);
// incremental fetches only the "updated >=" query.
func TestBackfillThenIncremental(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "jira.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	resolved := now.Add(-48 * time.Hour)

	// Pass 1 hits — resolved in window. Includes one with story points
	// and a sprint, to exercise the snapshot extras.
	resolvedIssues := []jira.SnapshotIssue{
		{
			ID: "10001", Key: "DEVOPS-1",
			IssueType: "Bug", Status: "Done",
			Components: []string{"argo-cd"},
			Versions:   []string{"argo-cd-2.9.0"},
			Assignee: &models.User{
				AccountID: "acct-abc", Name: "atan", DisplayName: "Alice Tan",
				EmailAddress: "alice@alauda.io",
			},
			StoryPoints: 3,
			SprintID:    "42",
			SprintName:  "Sprint 26.05",
			CreatedAt:   now.Add(-72 * time.Hour),
			ResolvedAt:  &resolved,
		},
		{
			ID: "10002", Key: "DEVOPS-2",
			IssueType: "Story", Status: "Done",
			// Same assignee — must reuse the same member id.
			Assignee: &models.User{
				AccountID: "acct-abc", Name: "atan", DisplayName: "Alice Tan",
				EmailAddress: "alice@alauda.io",
			},
			StoryPoints: 5,
			CreatedAt:   now.Add(-100 * time.Hour),
			ResolvedAt:  &resolved,
		},
	}

	// Pass 2 hits — open in window, different assignee.
	openIssues := []jira.SnapshotIssue{
		{
			ID: "10003", Key: "DEVOPS-3",
			IssueType: "Task", Status: "In Progress",
			Assignee: &models.User{
				AccountID: "acct-xyz", Name: "bohan", DisplayName: "Bo Han",
				EmailAddress: "bohan@alauda.io",
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	fake := &fakeSearcher{byPattern: map[string][]jira.SnapshotIssue{
		`resolved >=`:         resolvedIssues,
		`resolution is EMPTY`: openIssues,
	}}

	syncer := NewSyncer(fake, store, Config{
		Project:      "DEVOPS",
		BackfillDays: 30,
	})

	// ---- backfill ----
	res, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if res.Mode != "backfill" {
		t.Fatalf("mode=%q want backfill", res.Mode)
	}
	if res.IssuesWritten != 3 {
		t.Fatalf("written=%d want 3", res.IssuesWritten)
	}
	if res.MembersSeen != 2 {
		t.Fatalf("members=%d want 2 (alice + bohan)", res.MembersSeen)
	}
	if len(fake.calledWith) != 2 {
		t.Fatalf("expected 2 search calls, got %d: %v", len(fake.calledWith), fake.calledWith)
	}

	// Members should be persisted with email-derived slugs.
	mems, _ := store.ListMembers(context.Background())
	if len(mems) != 2 {
		t.Fatalf("members in store = %d, want 2", len(mems))
	}
	want := map[string]bool{"alice": true, "bohan": true}
	for _, m := range mems {
		if !want[m.ID] {
			t.Fatalf("unexpected member id %q (members=%+v)", m.ID, mems)
		}
		if m.JiraAccountID == "" {
			t.Fatalf("member %s missing JiraAccountID", m.ID)
		}
	}

	// ---- incremental ----
	fake.calledWith = nil
	fake.byPattern = map[string][]jira.SnapshotIssue{
		`updated >=`: {
			// One issue mutated since the previous run.
			{
				ID: "10003", Key: "DEVOPS-3",
				IssueType: "Task", Status: "Done",
				Assignee: &models.User{
					AccountID: "acct-xyz", Name: "bohan", DisplayName: "Bo Han",
					EmailAddress: "bohan@alauda.io",
				},
				CreatedAt:  now.Add(-24 * time.Hour),
				ResolvedAt: &now,
			},
		},
	}
	res, err = syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("incremental: %v", err)
	}
	if res.Mode != "incremental" {
		t.Fatalf("mode=%q want incremental", res.Mode)
	}
	if res.IssuesWritten != 1 {
		t.Fatalf("incremental written=%d want 1", res.IssuesWritten)
	}
	if len(fake.calledWith) != 1 {
		t.Fatalf("expected 1 incremental call, got %d", len(fake.calledWith))
	}
	if !strings.Contains(fake.calledWith[0], "updated >=") {
		t.Fatalf("expected incremental JQL to use updated>=, got %q", fake.calledWith[0])
	}
}

// TestMemberIDFromUser locks down the slug priority so any future
// change is deliberate and visible in diffs.
func TestMemberIDFromUser(t *testing.T) {
	cases := []struct {
		name string
		in   *models.User
		want string
	}{
		{"nil", nil, ""},
		{"email wins", &models.User{EmailAddress: "Alice.Tan@alauda.io", AccountID: "acct-abc", Name: "atan"}, "alice.tan"},
		{"account fallback", &models.User{AccountID: "acct-abc", Name: "atan"}, "acct-acct-abc"},
		{"name fallback", &models.User{Name: "Bohan Z"}, "bohan-z"},
		{"display last", &models.User{DisplayName: "Carlos Méndez"}, "u-"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MemberIDFromUser(tc.in)
			if tc.want == "u-" {
				if !strings.HasPrefix(got, "u-") {
					t.Fatalf("got %q, want prefix u-", got)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
