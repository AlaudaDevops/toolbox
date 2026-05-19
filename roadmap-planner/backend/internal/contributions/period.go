/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// Period names accepted by `?period=` on the contributions endpoints.
const (
	PeriodWeek           = "week"
	PeriodReleaseQuarter = "release_quarter"
)

// QuarterResolver maps a Jira key (or fallback timestamp) to a release
// quarter label like "2026Q1". The DEVOPS project tags quarters as a
// prefix on Milestone issue summaries — the resolver looks the row up
// in `quarter_assignments` first; on miss it falls back to the
// calendar quarter of the supplied timestamp.
//
// The Milestone → quarter_label propagation pass that authoritatively
// populates `quarter_assignments` is a follow-up to W7; until it
// lands, every resolve goes through the calendar-quarter fallback,
// and the `source` field on the returned label reflects that.
type QuarterResolver struct {
	store storage.Store
	// cache memoises one process-cycle of lookups so repeated
	// resolution of the same issue inside one read doesn't hit the
	// store every time. Cleared each call to ResolveCached when the
	// caller opens a fresh handler context.
	cache map[string]string
}

// NewQuarterResolver constructs a resolver against the given store.
// The cache is per-resolver — callers that hold the resolver across
// requests should expect a small, bounded memory footprint (one entry
// per Jira key seen since startup).
func NewQuarterResolver(store storage.Store) *QuarterResolver {
	return &QuarterResolver{store: store, cache: map[string]string{}}
}

// Resolve returns the release quarter label for the supplied Jira
// key. When the key is empty, or when no row exists in
// `quarter_assignments`, the fallback timestamp is used to derive a
// calendar quarter.
//
// Source: "milestone" when the row exists in the table, "fallback"
// otherwise. A zero time means "no fallback either" — the resolver
// returns ("", "none", nil), and the caller should bucket the item
// under a "Dangling" or "Unassigned" label.
func (r *QuarterResolver) Resolve(ctx context.Context, jiraKey string, fallback time.Time) (label, source string, err error) {
	if jiraKey != "" {
		if cached, ok := r.cache[jiraKey]; ok {
			return cached, "milestone", nil
		}
		row, fetchErr := r.fetchOne(ctx, jiraKey)
		if fetchErr != nil {
			return "", "", fetchErr
		}
		if row != "" {
			r.cache[jiraKey] = row
			return row, "milestone", nil
		}
	}
	if fallback.IsZero() {
		return "", "none", nil
	}
	return CalendarQuarter(fallback), "fallback", nil
}

// CalendarQuarter returns a "<YYYY>Q<1-4>" label derived from t's
// calendar quarter (Jan-Mar=Q1, Apr-Jun=Q2, Jul-Sep=Q3, Oct-Dec=Q4).
// Used by Resolve as the fallback when no Milestone assignment
// exists.
func CalendarQuarter(t time.Time) string {
	q := int(t.Month()-1)/3 + 1
	return fmt.Sprintf("%dQ%d", t.Year(), q)
}

func (r *QuarterResolver) fetchOne(ctx context.Context, jiraKey string) (string, error) {
	d, ok := r.store.(interface {
		Dialect() storage.Dialect
		DB() *sql.DB
	})
	if !ok {
		return "", nil
	}
	dialect := d.Dialect()
	q := rebindSimple(dialect, `SELECT quarter_label FROM quarter_assignments WHERE issue_key = ?`)
	var label sql.NullString
	if err := d.DB().QueryRowContext(ctx, q, jiraKey).Scan(&label); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if !label.Valid {
		return "", nil
	}
	return label.String, nil
}
