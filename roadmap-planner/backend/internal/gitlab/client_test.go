/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestListMRCommits_ReturnsAuthoredDates mirrors the GitHub PR-commits
// test: verifies the MR-commits endpoint is decoded into MRCommit{}
// with the authored_date field preserved as time.Time, and that
// callers can compute pull_requests.first_commit_at via min().
func TestListMRCommits_ReturnsAuthoredDates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v4/projects/123/merge_requests/77/commits"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("per_page = %q, want 100", got)
		}
		w.Header().Set("Content-Type", "application/json")
		// Three commits — middle one has the earliest authored_date.
		_, _ = w.Write([]byte(`[
			{"id": "aaaa", "authored_date": "2026-04-10T09:00:00.000+00:00"},
			{"id": "bbbb", "authored_date": "2026-04-08T14:30:00.000+00:00"},
			{"id": "cccc", "authored_date": "2026-04-12T11:00:00.000+00:00"}
		]`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	commits, err := c.ListMRCommits(context.Background(), 123, 77)
	if err != nil {
		t.Fatalf("ListMRCommits: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("len(commits)=%d, want 3", len(commits))
	}

	var firstCommit time.Time
	for _, cm := range commits {
		if firstCommit.IsZero() || cm.AuthoredDate.Before(firstCommit) {
			firstCommit = cm.AuthoredDate
		}
	}
	want := time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)
	if !firstCommit.Equal(want) {
		t.Fatalf("min(authored_date) = %s, want %s", firstCommit, want)
	}
}
