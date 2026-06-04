/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestListPRCommits_ReturnsAuthorDates verifies that the PR-commits
// endpoint is decoded into PRCommit{} with the commit.author.date
// field preserved as time.Time. Callers (sync + Lead Time calculator)
// rely on this to compute pull_requests.first_commit_at as the
// minimum across all returned commits.
func TestListPRCommits_ReturnsAuthorDates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/repos/AlaudaDevops/tektoncd-operator/pulls/42/commits"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("per_page = %q, want 100", got)
		}
		w.Header().Set("X-RateLimit-Remaining", "5000")
		w.Header().Set("Content-Type", "application/json")
		// Three commits in graph order (base → head). Middle commit
		// has an *earlier* author.date than the first to exercise the
		// "caller picks min" contract.
		_, _ = w.Write([]byte(`[
			{
				"sha": "aaaa",
				"commit": {"author": {"date": "2026-04-10T09:00:00Z"}}
			},
			{
				"sha": "bbbb",
				"commit": {"author": {"date": "2026-04-08T14:30:00Z"}}
			},
			{
				"sha": "cccc",
				"commit": {"author": {"date": "2026-04-12T11:00:00Z"}}
			}
		]`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	commits, err := c.ListPRCommits(context.Background(), "AlaudaDevops", "tektoncd-operator", 42)
	if err != nil {
		t.Fatalf("ListPRCommits: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("len(commits)=%d, want 3", len(commits))
	}
	if commits[0].SHA != "aaaa" || commits[1].SHA != "bbbb" || commits[2].SHA != "cccc" {
		t.Fatalf("commit SHAs out of order: %+v", commits)
	}

	// Simulate the caller's min(author.date) logic.
	var firstCommit time.Time
	for _, cm := range commits {
		if firstCommit.IsZero() || cm.Commit.Author.Date.Before(firstCommit) {
			firstCommit = cm.Commit.Author.Date
		}
	}
	want := time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)
	if !firstCommit.Equal(want) {
		t.Fatalf("min(author.date) = %s, want %s", firstCommit, want)
	}
}
