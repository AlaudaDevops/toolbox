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
//
// Dialect-aware: the week-start expression and placeholder syntax both
// route through the underlying storage.Dialect, so the same code runs on
// SQLite and Postgres.
func (a *Aggregator) Rebuild(ctx context.Context, from, to time.Time) error {
	if !to.After(from) {
		return fmt.Errorf("rebuild: to must be after from (got %s, %s)", from, to)
	}
	a.logger.Info("rebuilding rollups", zap.Time("from", from), zap.Time("to", to))

	d, ok := a.store.(interface{ Dialect() storage.Dialect })
	if !ok {
		return fmt.Errorf("rebuild: store does not expose a Dialect")
	}
	dialect := d.Dialect()
	db := a.store.DB()

	rebind := func(q string) string {
		// Local copy of storage.rebind via the dialect's placeholder.
		// Kept here to avoid exporting rebind from the storage package.
		if dialect.Placeholder(1) == "?" {
			return q
		}
		var b []byte
		n := 1
		for i := 0; i < len(q); i++ {
			if q[i] == '?' {
				b = append(b, []byte(dialect.Placeholder(n))...)
				n++
			} else {
				b = append(b, q[i])
			}
		}
		return string(b)
	}

	if _, err := db.ExecContext(ctx, rebind(
		`DELETE FROM member_week_metrics WHERE week_start >= ? AND week_start < ?`),
		from, to); err != nil {
		return fmt.Errorf("clear window: %w", err)
	}

	// PRs/MRs merged, by author × week. Pillar/component empty for now —
	// requires the repos table to be populated, which B2 fixes.
	//
	// Resolution priority for member_id:
	//   1. members row matched on raw login (LEFT JOIN below) — reflects
	//      the *current* identity link, so editing a member's
	//      github_login or gitlab_username via PATCH retroactively
	//      re-links history on the next rebuild. The JOIN branches on
	//      pr.source so a GitHub PR matches members.github_login and a
	//      GitLab MR matches members.gitlab_username.
	//   2. pr.author_id frozen at write time — backward-compat fallback
	//      for rows ingested before migration 0002 (the raw login column
	//      is NULL on those).
	prSQL := rebind(fmt.Sprintf(`
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_merged)
		SELECT
		    COALESCE(m.id, pr.author_id) AS member_id,
		    %s AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(*) AS prs_merged
		FROM pull_requests pr
		LEFT JOIN members m
		       ON (pr.source = 'github'
		            AND m.github_login IS NOT NULL
		            AND m.github_login <> ''
		            AND LOWER(m.github_login) = pr.author_login)
		       OR (pr.source = 'gitlab'
		            AND m.gitlab_username IS NOT NULL
		            AND m.gitlab_username <> ''
		            AND LOWER(m.gitlab_username) = pr.author_login)
		WHERE COALESCE(m.id, pr.author_id) IS NOT NULL
		  AND pr.merged_at IS NOT NULL
		  AND pr.merged_at >= ?
		  AND pr.merged_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_merged = excluded.prs_merged`, dialect.WeekStart("merged_at")))

	if _, err := db.ExecContext(ctx, prSQL, from, to); err != nil {
		return fmt.Errorf("aggregate PRs: %w", err)
	}

	// PRs/MRs reviewed, by reviewer × week. Same resolution priority as
	// the merge aggregation above — the JOIN branches on rv.source so
	// editing either github_login or gitlab_username retroactively
	// populates review counts on the next rebuild.
	reviewSQL := rebind(fmt.Sprintf(`
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_reviewed)
		SELECT
		    COALESCE(m.id, rv.reviewer_id) AS member_id,
		    %s AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(DISTINCT pr_id) AS prs_reviewed
		FROM pr_reviews rv
		LEFT JOIN members m
		       ON (rv.source = 'github'
		            AND m.github_login IS NOT NULL
		            AND m.github_login <> ''
		            AND LOWER(m.github_login) = rv.reviewer_login)
		       OR (rv.source = 'gitlab'
		            AND m.gitlab_username IS NOT NULL
		            AND m.gitlab_username <> ''
		            AND LOWER(m.gitlab_username) = rv.reviewer_login)
		WHERE COALESCE(m.id, rv.reviewer_id) IS NOT NULL
		  AND rv.submitted_at >= ?
		  AND rv.submitted_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_reviewed = excluded.prs_reviewed`, dialect.WeekStart("submitted_at")))

	if _, err := db.ExecContext(ctx, reviewSQL, from, to); err != nil {
		return fmt.Errorf("aggregate reviews: %w", err)
	}

	// Jira issues done + story points, by assignee × week.
	//
	// Two subtleties this query handles:
	//
	//   1. An issue can appear in many `issue_snapshots` rows (one per
	//      collection cycle). The window function picks the *latest*
	//      snapshot per `issue_key` so we count each issue exactly once,
	//      using its most-recent state.
	//
	//   2. We only count issues with a non-null `resolved_at` falling in
	//      the [from, to) window. `resolved_at` is set by Jira on the
	//      transition into a Done-class status; using it instead of a
	//      status-name allowlist keeps the aggregator agnostic to each
	//      project's workflow naming.
	//
	// "Latest" needs a temporal order on runs — `collection_runs.captured_at`
	// is the canonical clock, not `issue_snapshots.run_id` (which is a
	// string).
	jiraSQL := rebind(fmt.Sprintf(`
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, jira_issues_done, jira_points_done)
		SELECT
		    assignee_id,
		    %s AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(*),
		    COALESCE(SUM(story_points), 0)
		FROM (
		    SELECT s.assignee_id, s.resolved_at, s.story_points, s.issue_key,
		           ROW_NUMBER() OVER (PARTITION BY s.issue_key ORDER BY r.captured_at DESC) AS rn
		    FROM issue_snapshots s
		    JOIN collection_runs r ON s.run_id = r.id
		    WHERE s.resolved_at IS NOT NULL
		      AND s.resolved_at >= ?
		      AND s.resolved_at <  ?
		) ranked
		WHERE rn = 1
		  AND assignee_id IS NOT NULL
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    jira_issues_done = excluded.jira_issues_done,
		    jira_points_done = excluded.jira_points_done`, dialect.WeekStart("resolved_at")))

	if _, err := db.ExecContext(ctx, jiraSQL, from, to); err != nil {
		return fmt.Errorf("aggregate jira: %w", err)
	}

	// TODO(B3): review_latency_p50_hours — derive from
	// pull_requests.first_review_at - pull_requests.created_at, p50 by
	// (reviewer_id, week). p50 across reviews per author per week.

	return nil
}

// RebuildRecent is a convenience wrapper for the auto-trigger after a
// successful sync: it rebuilds the rollup over a recent-history window
// large enough to cover both the backfill default and any out-of-band
// updates we may have absorbed (e.g., a back-dated `resolved` change).
//
// Default window is the last (storageBackfillDays + 7) days, capped at
// 365. Pass days=0 to use the default.
func (a *Aggregator) RebuildRecent(ctx context.Context, days int) error {
	if days <= 0 {
		days = 187
	}
	if days > 365 {
		days = 365
	}
	now := time.Now().UTC()
	return a.Rebuild(ctx, MondayOf(now.AddDate(0, 0, -days)), MondayOf(now.AddDate(0, 0, 7)))
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
