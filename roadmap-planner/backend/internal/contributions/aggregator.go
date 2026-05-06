/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"fmt"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
	"go.uber.org/zap"
)

// Aggregator (re)builds the member_week_metrics rollup table from the
// raw snapshot + PR + review tables.
//
// Status (B1): the SQL plumbing is staked out, but we deliberately keep
// the body conservative — only the simplest counts (PRs merged in week,
// PRs reviewed in week) are computed. Jira-side counts and latency p50
// land in B2 once the snapshot writer is emitting sprint_id.
//
// The intended cadence is: every successful Collect() in the Jira side
// triggers a Rebuild for the affected window. Manual rebuild is also
// exposed via the API for operators.
type Aggregator struct {
	store  storage.Store
	logger *zap.Logger
}

func NewAggregator(store storage.Store) *Aggregator {
	return &Aggregator{
		store:  store,
		logger: logger.WithComponent("contributions-aggregator"),
	}
}

// Rebuild deletes existing rollup rows in [from, to) and recomputes them
// from raw tables. Cheap because the rollup table is small; we use it as
// our cache-coherency strategy rather than incremental updates.
//
// from / to should be week boundaries (Monday 00:00 UTC); MondayOf
// rounds for callers that need help.
func (a *Aggregator) Rebuild(ctx context.Context, from, to time.Time) error {
	if !to.After(from) {
		return fmt.Errorf("rebuild: to must be after from (got %s, %s)", from, to)
	}
	a.logger.Info("rebuilding rollups", zap.Time("from", from), zap.Time("to", to))

	db := a.store.DB()
	if _, err := db.ExecContext(ctx,
		`DELETE FROM member_week_metrics WHERE week_start >= ? AND week_start < ?`,
		from, to); err != nil {
		return fmt.Errorf("clear window: %w", err)
	}

	// PRs merged, by author × week. Pillar/component empty for now —
	// requires the repos table to be populated, which B2 fixes.
	prSQL := `
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_merged)
		SELECT
		    pr.author_id AS member_id,
		    -- Truncate merged_at to its Monday 00:00 UTC.
		    DATE(merged_at, 'weekday 1', '-7 days') AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(*) AS prs_merged
		FROM pull_requests pr
		WHERE pr.author_id IS NOT NULL
		  AND pr.merged_at IS NOT NULL
		  AND pr.merged_at >= ?
		  AND pr.merged_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_merged = excluded.prs_merged`

	if _, err := db.ExecContext(ctx, prSQL, from, to); err != nil {
		return fmt.Errorf("aggregate PRs: %w", err)
	}

	// PRs reviewed, by reviewer × week.
	reviewSQL := `
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_reviewed)
		SELECT
		    rv.reviewer_id AS member_id,
		    DATE(submitted_at, 'weekday 1', '-7 days') AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(DISTINCT pr_id) AS prs_reviewed
		FROM pr_reviews rv
		WHERE rv.reviewer_id IS NOT NULL
		  AND rv.submitted_at >= ?
		  AND rv.submitted_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_reviewed = excluded.prs_reviewed`

	if _, err := db.ExecContext(ctx, reviewSQL, from, to); err != nil {
		return fmt.Errorf("aggregate reviews: %w", err)
	}

	// TODO(B2): jira_issues_done, jira_points_done — needs the latest
	// run_id snapshot per (assignee_id, resolved_at) to avoid double-
	// counting an issue that appears in N consecutive snapshots.
	// TODO(B2): review_latency_p50_hours — derive from
	// pull_requests.first_review_at - pull_requests.created_at, p50 by
	// (reviewer_id, week).

	return nil
}

// MondayOf returns the Monday 00:00 UTC of t's week.
func MondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	d := t.UTC().AddDate(0, 0, -(wd - 1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}
