/*
Copyright 2026 The AlaudaDevops Authors.

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

func TestCalendarQuarter(t *testing.T) {
	cases := []struct {
		t    string
		want string
	}{
		{"2026-01-05T00:00:00Z", "2026Q1"},
		{"2026-03-31T23:59:59Z", "2026Q1"},
		{"2026-04-01T00:00:00Z", "2026Q2"},
		{"2026-06-30T23:59:59Z", "2026Q2"},
		{"2026-07-01T00:00:00Z", "2026Q3"},
		{"2026-09-30T23:59:59Z", "2026Q3"},
		{"2026-10-01T00:00:00Z", "2026Q4"},
		{"2026-12-31T23:59:59Z", "2026Q4"},
		{"2025-12-31T23:59:59Z", "2025Q4"},
	}
	for _, tc := range cases {
		parsed, err := time.Parse(time.RFC3339, tc.t)
		if err != nil {
			t.Fatalf("parse %s: %v", tc.t, err)
		}
		if got := CalendarQuarter(parsed); got != tc.want {
			t.Errorf("CalendarQuarter(%s) = %s, want %s", tc.t, got, tc.want)
		}
	}
}

func TestFoldWeeksByCalendarQuarter(t *testing.T) {
	mk := func(year int, month time.Month, day int, prs int) Bucket {
		return Bucket{
			WeekStart: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
			PRsMerged: prs,
		}
	}
	in := []Bucket{
		mk(2026, time.January, 5, 2),
		mk(2026, time.February, 16, 3),
		mk(2026, time.April, 6, 5),
		mk(2025, time.December, 29, 1),
	}
	out := FoldWeeksByCalendarQuarter(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 quarters, got %d: %+v", len(out), out)
	}
	if out[0].Period != "2025Q4" || out[0].PRsMerged != 1 {
		t.Fatalf("bucket[0] = %+v, want 2025Q4/1", out[0])
	}
	if out[1].Period != "2026Q1" || out[1].PRsMerged != 5 {
		t.Fatalf("bucket[1] = %+v, want 2026Q1/5", out[1])
	}
	if out[2].Period != "2026Q2" || out[2].PRsMerged != 5 {
		t.Fatalf("bucket[2] = %+v, want 2026Q2/5", out[2])
	}
}

// TestQuarterResolverMilestoneAndFallback exercises the W7 quarter
// resolver: a row in `quarter_assignments` wins, an empty key falls
// back to the calendar quarter of the supplied timestamp, and a
// completely dangling input returns ("", "none").
func TestQuarterResolverMilestoneAndFallback(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "qa.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := store.DB().ExecContext(ctx,
		`INSERT INTO quarter_assignments(issue_key, quarter_label, source) VALUES (?, ?, ?)`,
		"DEVOPS-42014", "2026Q1", "milestone"); err != nil {
		t.Fatalf("seed quarter_assignments: %v", err)
	}

	r := NewQuarterResolver(store)

	// Milestone hit.
	if label, src, err := r.Resolve(ctx, "DEVOPS-42014", time.Time{}); err != nil {
		t.Fatalf("resolve milestone: %v", err)
	} else if label != "2026Q1" || src != "milestone" {
		t.Errorf("milestone resolve = (%s, %s), want (2026Q1, milestone)", label, src)
	}

	// Unknown key with timestamp falls back to calendar quarter.
	ts := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	if label, src, err := r.Resolve(ctx, "DEVOPS-99999", ts); err != nil {
		t.Fatalf("resolve fallback: %v", err)
	} else if label != "2026Q3" || src != "fallback" {
		t.Errorf("fallback resolve = (%s, %s), want (2026Q3, fallback)", label, src)
	}

	// Empty key + zero time: nothing to bucket against.
	if label, src, err := r.Resolve(ctx, "", time.Time{}); err != nil {
		t.Fatalf("resolve none: %v", err)
	} else if label != "" || src != "none" {
		t.Errorf("none resolve = (%s, %s), want (, none)", label, src)
	}
}
