/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
//
// SetAllowlist (W1) gates which members get rolled up; when disabled,
// Rebuild reverts to the pre-W1 "everyone with a Jira id" behaviour so
// the upgrade is opt-in.
type Aggregator struct {
	store     storage.Store
	logger    *zap.Logger
	allowlist Allowlist
}

func NewAggregator(store storage.Store) *Aggregator {
	return &Aggregator{
		store:  store,
		logger: logger.WithComponent("contributions-aggregator"),
	}
}

// SetAllowlist installs the W1 "who counts" filter. Pass a zero-value
// Allowlist (Enabled() == false) to disable the filter and preserve the
// pre-W1 rollup shape — that's the safe rollback knob.
func (a *Aggregator) SetAllowlist(al Allowlist) { a.allowlist = al }

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

	// W1 allowlist: when enabled, the four INSERTs below all carry an
	// `AND <member_expr> IN (?, …)` clause. The fragment is composed
	// per-INSERT (the member expression differs: COALESCE(m.id,
	// pr.author_id) for the PR passes, COALESCE(m.id, rv.reviewer_id)
	// for the review pass, assignee_id for the Jira pass). The bound
	// arguments are the same for every INSERT and get appended to each
	// param slice.
	allowArgs := a.allowArgs()

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
		WHERE COALESCE(m.id, pr.author_id) IS NOT NULL%s
		  AND pr.merged_at IS NOT NULL
		  AND pr.merged_at >= ?
		  AND pr.merged_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_merged = excluded.prs_merged`, dialect.WeekStart("merged_at"), a.allowFragment("COALESCE(m.id, pr.author_id)")))

	prArgs := append([]any{}, allowArgs...)
	prArgs = append(prArgs, from, to)
	if _, err := db.ExecContext(ctx, prSQL, prArgs...); err != nil {
		return fmt.Errorf("aggregate PRs: %w", err)
	}

	// PRs/MRs opened, by author × week. Mirrors the merged pass but
	// buckets on created_at — the dashboard's Inflow vs Outflow panel
	// reads opened-vs-merged together to spot review-queue buildup.
	openedSQL := rebind(fmt.Sprintf(`
		INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_opened)
		SELECT
		    COALESCE(m.id, pr.author_id) AS member_id,
		    %s AS week_start,
		    '' AS pillar_id,
		    '' AS component,
		    COUNT(*) AS prs_opened
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
		WHERE COALESCE(m.id, pr.author_id) IS NOT NULL%s
		  AND pr.created_at >= ?
		  AND pr.created_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_opened = excluded.prs_opened`, dialect.WeekStart("pr.created_at"), a.allowFragment("COALESCE(m.id, pr.author_id)")))

	openedArgs := append([]any{}, allowArgs...)
	openedArgs = append(openedArgs, from, to)
	if _, err := db.ExecContext(ctx, openedSQL, openedArgs...); err != nil {
		return fmt.Errorf("aggregate opened PRs: %w", err)
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
		WHERE COALESCE(m.id, rv.reviewer_id) IS NOT NULL%s
		  AND rv.submitted_at >= ?
		  AND rv.submitted_at <  ?
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    prs_reviewed = excluded.prs_reviewed`, dialect.WeekStart("submitted_at"), a.allowFragment("COALESCE(m.id, rv.reviewer_id)")))

	rvArgs := append([]any{}, allowArgs...)
	rvArgs = append(rvArgs, from, to)
	if _, err := db.ExecContext(ctx, reviewSQL, rvArgs...); err != nil {
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
		  AND assignee_id IS NOT NULL%s
		GROUP BY 1, 2
		ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
		    jira_issues_done = excluded.jira_issues_done,
		    jira_points_done = excluded.jira_points_done`, dialect.WeekStart("resolved_at"), a.allowFragment("assignee_id")))

	jiraArgs := []any{from, to}
	jiraArgs = append(jiraArgs, allowArgs...)
	if _, err := db.ExecContext(ctx, jiraSQL, jiraArgs...); err != nil {
		return fmt.Errorf("aggregate jira: %w", err)
	}

	// Review latency p50, by reviewer × week (W8 2026-05-19).
	//
	// For each non-bot review row that matches `pr.first_human_review_at`
	// (i.e., the first *human* touch on the PR), we record the latency
	// in hours from the PR's `created_at`. The week is the week the PR
	// was opened — the metric answers "when reviewer X picks up a PR,
	// how long after creation does that first response take?" rather
	// than "how long does the review process last", which keeps the
	// number comparable across reviewers regardless of how long the PR
	// sat before they happened to look at it.
	//
	// p50 is computed in Go (portable across SQLite + Postgres). The
	// dataset per (reviewer, week) is small (single-digit reviews for
	// most engineers per week), so the in-memory sort is cheap.
	latSQL := rebind(fmt.Sprintf(`
		SELECT
		  rv.reviewer_id AS member_id,
		  %s AS week_start,
		  %s AS latency_hours
		FROM pr_reviews rv
		JOIN pull_requests pr ON pr.id = rv.pr_id
		WHERE rv.is_bot = 0
		  AND rv.reviewer_id IS NOT NULL
		  AND rv.reviewer_id <> ''
		  AND pr.first_human_review_at IS NOT NULL
		  AND rv.submitted_at = pr.first_human_review_at
		  AND pr.created_at >= ?
		  AND pr.created_at <  ?`,
		dialect.WeekStart("pr.created_at"),
		hoursBetween(dialect, "pr.first_human_review_at", "pr.created_at"),
	))
	latArgs := []any{from, to}
	latRows, err := db.QueryContext(ctx, latSQL, latArgs...)
	if err != nil {
		return fmt.Errorf("aggregate review latency: %w", err)
	}
	type latKey struct {
		member string
		week   time.Time
	}
	pools := map[latKey][]float64{}
	for latRows.Next() {
		var memberID string
		var weekStartRaw any
		var hours float64
		if err := latRows.Scan(&memberID, &weekStartRaw, &hours); err != nil {
			latRows.Close()
			return fmt.Errorf("scan review latency: %w", err)
		}
		weekStart, err := scanTimeAny(weekStartRaw)
		if err != nil {
			latRows.Close()
			return fmt.Errorf("parse review-latency week_start: %w", err)
		}
		if hours < 0 {
			continue
		}
		pools[latKey{memberID, weekStart}] = append(pools[latKey{memberID, weekStart}], hours)
	}
	latRows.Close()
	if err := latRows.Err(); err != nil {
		return fmt.Errorf("iterate review latency: %w", err)
	}
	if len(pools) > 0 {
		upsertSQL := rebind(`
			INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, review_latency_p50_hours)
			VALUES (?, ?, '', '', ?)
			ON CONFLICT(member_id, week_start, pillar_id, component) DO UPDATE SET
			    review_latency_p50_hours = excluded.review_latency_p50_hours`)
		for k, vs := range pools {
			sort.Float64s(vs)
			p50 := round1(percentile(vs, 0.50))
			// Bind week_start as the same `YYYY-MM-DD` string the other
			// INSERTs in Rebuild emit via `dialect.WeekStart(...)`, so the
			// composite primary key matches and the ON CONFLICT branch
			// fires instead of creating a parallel row.
			weekStr := k.week.UTC().Format("2006-01-02")
			if _, err := db.ExecContext(ctx, upsertSQL, k.member, weekStr, p50); err != nil {
				return fmt.Errorf("upsert review latency for %s/%s: %w", k.member, k.week, err)
			}
		}
	}

	// W1 cleanup: every rebuild leaves the four INSERTs above blocked
	// on the allowlist, but historical rollup rows for now-denied
	// members linger outside the rebuild window. A simple sweep keeps
	// the table consistent with the configured set on every rebuild.
	// No-op when the allowlist is disabled (rebinds away to a no-arg
	// query that matches nothing).
	if a.allowlist.Enabled() {
		ids := a.allowlist.IDs()
		placeholders := make([]string, len(ids))
		args := make([]any, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args[i] = id
		}
		cleanup := rebind(fmt.Sprintf(
			`DELETE FROM member_week_metrics WHERE member_id NOT IN (%s)`,
			strings.Join(placeholders, ", "),
		))
		res, err := db.ExecContext(ctx, cleanup, args...)
		if err != nil {
			return fmt.Errorf("cleanup non-allowlisted rollups: %w", err)
		}
		var removed int64
		if res != nil {
			removed, _ = res.RowsAffected()
		}
		a.logger.Info("allowlist cleanup",
			zap.Int("allowlist_size", len(ids)),
			zap.Int64("rows_removed", removed))
	}

	return nil
}

// allowArgs returns the parameter slice for the IN clause emitted by
// allowFragment. It's separated so callers can append it onto each
// INSERT's existing param slice (the order of bound args has to match
// the order of '?' placeholders in the final, rebound SQL).
func (a *Aggregator) allowArgs() []any {
	if !a.allowlist.Enabled() {
		return nil
	}
	ids := a.allowlist.IDs()
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// allowFragment returns the SQL fragment that filters by the W1
// allowlist on the given member-id expression. The fragment is the
// empty string when the allowlist is disabled, so the surrounding
// query keeps its pre-W1 shape. When enabled, the fragment starts
// with " AND " so it can be appended straight after an existing
// `WHERE x IS NOT NULL` predicate.
//
// Bound args for the '?' placeholders come from allowArgs and are
// appended to the INSERT's existing param slice in source order.
func (a *Aggregator) allowFragment(memberExpr string) string {
	if !a.allowlist.Enabled() {
		return ""
	}
	placeholders := strings.Repeat("?, ", a.allowlist.Size())
	placeholders = strings.TrimSuffix(placeholders, ", ")
	return fmt.Sprintf("\n\t\t  AND %s IN (%s)", memberExpr, placeholders)
}

// RebuildRecent is a convenience wrapper for the auto-trigger after a
// successful sync: it rebuilds the rollup over a recent-history window
// large enough to cover both the backfill default and any out-of-band
// updates we may have absorbed (e.g., a back-dated `resolved` change).
//
// Default window is the last 400 days (W7 2026-05-19, was 187), capped
// at 730. Pass days=0 to use the default. The bump matches the
// 365-day Storage.BackfillDays so the first rebuild after a fresh
// deploy fully populates the year of rollup rows.
func (a *Aggregator) RebuildRecent(ctx context.Context, days int) error {
	if days <= 0 {
		days = 400
	}
	if days > 730 {
		days = 730
	}
	now := time.Now().UTC()
	return a.Rebuild(ctx, MondayOf(now.AddDate(0, 0, -days)), MondayOf(now.AddDate(0, 0, 7)))
}

// scanTimeAny converts a driver value emitted by `dialect.WeekStart(...)`
// to a time.Time. SQLite returns the strftime result as TEXT
// ("YYYY-MM-DD"); Postgres returns date_trunc as time.Time directly.
// Empty / nil inputs produce the zero time without erroring.
func scanTimeAny(v any) (time.Time, error) {
	switch t := v.(type) {
	case nil:
		return time.Time{}, nil
	case time.Time:
		return t.UTC(), nil
	case string:
		return parseDateString(t)
	case []byte:
		return parseDateString(string(t))
	}
	return time.Time{}, fmt.Errorf("unsupported time scan type %T", v)
}

func parseDateString(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("week_start %q: no layout matched", s)
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
