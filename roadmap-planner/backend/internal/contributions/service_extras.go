/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Service extras — supplementary aggregations for the new Team Analytics
// views (pillar throughput, review network density, member-profile
// extras). Kept in their own file so service.go stays focused on the
// canonical TeamOverview / MemberDetail rollup reads.
//
// The queries here go straight to the raw tables (issue_snapshots,
// pull_requests, pr_reviews, members) rather than through the rollup,
// because the questions they answer aren't pre-aggregated:
//
//   - PillarThroughput needs members.pillar joined onto pull_requests.
//   - NetworkDensity needs PR-level latency / orphan distribution and
//     a cross-pillar review classification.
//   - MemberProfile (components_touched, current sprint) reads
//     issue_snapshots directly; the rollup throws away component
//     information.

package contributions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// ----------------------------------------------------------------------
// /api/contributions/network — Review-network density panel
// ----------------------------------------------------------------------

// NetworkDensity is the aggregate review health summary used by the
// Team Overview "Review network density" panel.
//
// All percentages are 0–100 (already multiplied), all latencies are in
// hours. Counts are absolute. Empty (zero-value) is a valid response —
// the dashboard renders "—" for unset values rather than spinning.
type NetworkDensity struct {
	From                       time.Time `json:"from"`
	To                         time.Time `json:"to"`
	PRsConsidered              int       `json:"prs_considered"`
	PRsMerged                  int       `json:"prs_merged"`
	OrphanPct                  float64   `json:"orphan_pct"`
	FirstReviewUnder24hPct     float64   `json:"first_review_under_24h_pct"`
	FirstReviewLatencyP50Hours float64   `json:"first_review_p50_hours"`
	FirstReviewLatencyP90Hours float64   `json:"first_review_p90_hours"`
	CrossPillarReviewPct       float64   `json:"cross_pillar_review_pct"`
}

// NetworkDensity returns the review-density rollup for the [From, To)
// window of q. Filters on q.MemberIDs / q.PillarIDs are honoured if
// set (a pillar slice asks "what does the network look like for this
// pillar's authors").
func (s *Service) NetworkDensity(ctx context.Context, q storage.MemberWeekQuery) (*NetworkDensity, error) {
	d, ok := s.store.(interface {
		Dialect() storage.Dialect
		DB() *sql.DB
	})
	if !ok {
		return nil, fmt.Errorf("network: store does not expose Dialect/DB")
	}
	dialect := d.Dialect()
	db := d.DB()
	out := &NetworkDensity{From: q.From, To: q.To}

	// 1. PR-level distribution: latency to first review + orphan count.
	//
	// We bound by created_at (i.e., "PRs opened in the window") not
	// merged_at — orphan rate measures *received review during life*,
	// which is anchored on creation. Drafts are excluded because they
	// shouldn't count toward orphan rate; we approximate by skipping
	// PRs that closed without a merge AND without a review.
	hours := hoursBetween(dialect, "first_review_at", "created_at")
	prSQL := rebindSimple(dialect, fmt.Sprintf(`
		SELECT
		  COUNT(*) AS total,
		  SUM(CASE WHEN merged_at IS NOT NULL THEN 1 ELSE 0 END) AS merged,
		  SUM(CASE WHEN first_review_at IS NULL THEN 1 ELSE 0 END) AS orphans,
		  SUM(CASE WHEN first_review_at IS NOT NULL AND %s <= 24.0
		           THEN 1 ELSE 0 END) AS under24h
		FROM pull_requests
		WHERE created_at >= ? AND created_at < ?`, hours))

	var total, merged, orphans, under24h sql.NullInt64
	if err := db.QueryRowContext(ctx, prSQL, q.From, q.To).Scan(&total, &merged, &orphans, &under24h); err != nil {
		return nil, fmt.Errorf("network pr-distribution: %w", err)
	}
	out.PRsConsidered = int(total.Int64)
	out.PRsMerged = int(merged.Int64)
	if total.Int64 > 0 {
		out.OrphanPct = float64(orphans.Int64) / float64(total.Int64) * 100
		out.FirstReviewUnder24hPct = float64(under24h.Int64) / float64(total.Int64) * 100
	}

	// 2. Latency p50 / p90.
	//
	// Pull every (latency_hours) value into Go and sort — the dataset
	// is at most a few thousand rows in our deployments and this keeps
	// the query portable (Postgres has percentile_cont, SQLite does
	// not, and we want one query for both).
	latSQL := rebindSimple(dialect, fmt.Sprintf(`
		SELECT %s
		FROM pull_requests
		WHERE created_at >= ? AND created_at < ?
		  AND first_review_at IS NOT NULL`, hours))
	rows, err := db.QueryContext(ctx, latSQL, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("network latencies: %w", err)
	}
	defer rows.Close()
	var lats []float64
	for rows.Next() {
		var v sql.NullFloat64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid && v.Float64 >= 0 {
			lats = append(lats, v.Float64)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(lats) > 0 {
		sort.Float64s(lats)
		out.FirstReviewLatencyP50Hours = round1(percentile(lats, 0.50))
		out.FirstReviewLatencyP90Hours = round1(percentile(lats, 0.90))
	}

	// 3. Cross-pillar review %.
	//
	// A review is "cross-pillar" when the reviewer's pillar differs
	// from the author's pillar AND both pillars are populated. Reviews
	// where either side has no pillar are excluded from the
	// denominator — we cannot answer the question without the data.
	xSQL := rebindSimple(dialect, `
		SELECT
		  COUNT(*) AS total,
		  SUM(CASE WHEN ma.pillar_id <> mr.pillar_id THEN 1 ELSE 0 END) AS xpillar
		FROM pr_reviews rv
		JOIN pull_requests pr ON pr.id = rv.pr_id
		JOIN members ma ON ma.id = COALESCE(NULLIF(pr.author_id, ''), '')
		JOIN members mr ON mr.id = COALESCE(NULLIF(rv.reviewer_id, ''), '')
		WHERE rv.submitted_at >= ? AND rv.submitted_at < ?
		  AND ma.pillar_id IS NOT NULL AND ma.pillar_id <> ''
		  AND mr.pillar_id IS NOT NULL AND mr.pillar_id <> ''`)
	var xTotal, xCross sql.NullInt64
	if err := db.QueryRowContext(ctx, xSQL, q.From, q.To).Scan(&xTotal, &xCross); err != nil {
		return nil, fmt.Errorf("network cross-pillar: %w", err)
	}
	if xTotal.Int64 > 0 {
		out.CrossPillarReviewPct = float64(xCross.Int64) / float64(xTotal.Int64) * 100
	}
	return out, nil
}

// ----------------------------------------------------------------------
// /api/contributions/pillars — Pillar throughput stack chart
// ----------------------------------------------------------------------

// PillarBucket is one (pillar, week) cell in the stack chart.
type PillarBucket struct {
	Pillar    string    `json:"pillar"`
	WeekStart time.Time `json:"week_start"`
	PRsMerged int       `json:"prs_merged"`
	JiraDone  int       `json:"jira_done"`
}

// PillarThroughput returns weekly PR-merged / Jira-done counts per
// pillar, restricted to members with a non-empty `pillar_id`. Members
// without a pillar are surfaced under the synthetic "Unassigned" key
// so the dashboard can hint that pillars need to be set.
func (s *Service) PillarThroughput(ctx context.Context, q storage.MemberWeekQuery) ([]PillarBucket, error) {
	rows, err := s.store.MemberWeekMetrics(ctx, q)
	if err != nil {
		return nil, err
	}
	members, err := s.store.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	pillarOf := map[string]string{}
	for _, m := range members {
		p := m.PillarID
		if p == "" {
			p = "Unassigned"
		}
		pillarOf[m.ID] = p
	}
	type key struct {
		Pillar string
		Week   time.Time
	}
	agg := map[key]*PillarBucket{}
	for _, r := range rows {
		p := pillarOf[r.MemberID]
		if p == "" {
			p = "Unassigned"
		}
		k := key{Pillar: p, Week: r.WeekStart}
		b, ok := agg[k]
		if !ok {
			b = &PillarBucket{Pillar: p, WeekStart: r.WeekStart}
			agg[k] = b
		}
		b.PRsMerged += r.PRsMerged
		b.JiraDone += r.JiraIssuesDone
	}
	out := make([]PillarBucket, 0, len(agg))
	for _, b := range agg {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].WeekStart.Equal(out[j].WeekStart) {
			return out[i].WeekStart.Before(out[j].WeekStart)
		}
		return out[i].Pillar < out[j].Pillar
	})
	return out, nil
}

// ----------------------------------------------------------------------
// MemberDetail extras — components touched + current sprint
// ----------------------------------------------------------------------

// ComponentBucket is one row of the "components touched" bar chart on
// the member profile.
type ComponentBucket struct {
	Component string `json:"component"`
	Issues    int    `json:"issues"`
}

// SprintStats is the "this sprint" KPI cluster on the member profile.
//
// We use the *most recently mentioned* sprint id across the member's
// open or recently-resolved issues as "the current sprint". This is a
// pragmatic heuristic — there is no canonical "current sprint" in
// Jira's data model that we can read without an extra API. The name
// is whatever the latest snapshot recorded.
type SprintStats struct {
	Name      string `json:"name,omitempty"`
	WIP       int    `json:"wip"`
	Done      int    `json:"done"`
	PRsOpen   int    `json:"prs_open"`
	PRsMerged int    `json:"prs_merged"`
}

// MemberExtras packages everything the profile page wants beyond the
// rollup numbers already returned by MemberDetail.
type MemberExtras struct {
	ComponentsTouched []ComponentBucket `json:"components_touched"`
	Sprint            *SprintStats      `json:"sprint,omitempty"`
}

// MemberExtras computes the components-touched bars and current-sprint
// stats for one member, scoped to the [From, To) window. Returns an
// empty (non-nil) ComponentsTouched on no-data; Sprint is nil if the
// member has no sprint association.
func (s *Service) MemberExtras(ctx context.Context, memberID string, q storage.MemberWeekQuery) (*MemberExtras, error) {
	d, ok := s.store.(interface {
		Dialect() storage.Dialect
		DB() *sql.DB
	})
	if !ok {
		return nil, fmt.Errorf("member-extras: store does not expose Dialect/DB")
	}
	dialect := d.Dialect()
	db := d.DB()
	out := &MemberExtras{ComponentsTouched: []ComponentBucket{}}

	// Components touched — pull every snapshot row for this member in
	// the window, decode the JSON components array, and tally. We use
	// only the latest snapshot per issue (ROW_NUMBER() over captured_at)
	// so an issue isn't counted N times across collection cycles.
	compSQL := rebindSimple(dialect, `
		SELECT components
		FROM (
		  SELECT s.components,
		         ROW_NUMBER() OVER (PARTITION BY s.issue_key ORDER BY r.captured_at DESC) AS rn
		  FROM issue_snapshots s
		  JOIN collection_runs r ON s.run_id = r.id
		  WHERE s.assignee_id = ?
		    AND s.created_at < ?
		    AND (s.resolved_at IS NULL OR s.resolved_at >= ?)
		) ranked
		WHERE rn = 1`)
	rows, err := db.QueryContext(ctx, compSQL, memberID, q.To, q.From)
	if err != nil {
		return nil, fmt.Errorf("components: %w", err)
	}
	defer rows.Close()
	tally := map[string]int{}
	for rows.Next() {
		var raw sql.NullString
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if !raw.Valid || raw.String == "" || raw.String == "null" {
			continue
		}
		var comps []string
		if err := json.Unmarshal([]byte(raw.String), &comps); err != nil {
			continue // malformed; skip
		}
		for _, c := range comps {
			if c != "" {
				tally[c]++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for c, n := range tally {
		out.ComponentsTouched = append(out.ComponentsTouched, ComponentBucket{Component: c, Issues: n})
	}
	sort.Slice(out.ComponentsTouched, func(i, j int) bool {
		if out.ComponentsTouched[i].Issues != out.ComponentsTouched[j].Issues {
			return out.ComponentsTouched[i].Issues > out.ComponentsTouched[j].Issues
		}
		return out.ComponentsTouched[i].Component < out.ComponentsTouched[j].Component
	})
	if len(out.ComponentsTouched) > 8 {
		out.ComponentsTouched = out.ComponentsTouched[:8]
	}

	// Current sprint — pick the most-recently-captured snapshot row
	// with a non-empty sprint_id for this member. That gives us the
	// sprint id + name; we then count WIP / Done by joining all rows
	// with the same sprint id.
	sprintSQL := rebindSimple(dialect, `
		SELECT s.sprint_id, MAX(r.captured_at)
		FROM issue_snapshots s
		JOIN collection_runs r ON s.run_id = r.id
		WHERE s.assignee_id = ? AND s.sprint_id IS NOT NULL AND s.sprint_id <> ''
		GROUP BY s.sprint_id
		ORDER BY MAX(r.captured_at) DESC
		LIMIT 1`)
	var sprintID sql.NullString
	var lastSeen sql.NullTime
	err = db.QueryRowContext(ctx, sprintSQL, memberID).Scan(&sprintID, &lastSeen)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("sprint pick: %w", err)
	}
	if sprintID.Valid && sprintID.String != "" {
		stats, err := s.sprintCounts(ctx, db, dialect, memberID, sprintID.String)
		if err != nil {
			return nil, err
		}
		out.Sprint = stats
	}
	return out, nil
}

// sprintCounts gathers (wip, done, prs_open, prs_merged) for a member
// scoped to one sprint. WIP / Done come from the *latest* snapshot
// per issue under that sprint id; PR counts come from pull_requests
// linked via epic_key (best-effort).
func (s *Service) sprintCounts(ctx context.Context, db *sql.DB, dialect storage.Dialect, memberID, sprintID string) (*SprintStats, error) {
	out := &SprintStats{Name: sprintID} // default name = id; we overwrite below if we see something better

	q := rebindSimple(dialect, `
		SELECT s.status, s.resolved_at
		FROM (
		  SELECT s.status, s.resolved_at, s.issue_key, s.sprint_id, s.assignee_id,
		         ROW_NUMBER() OVER (PARTITION BY s.issue_key ORDER BY r.captured_at DESC) AS rn
		  FROM issue_snapshots s
		  JOIN collection_runs r ON s.run_id = r.id
		  WHERE s.sprint_id = ? AND s.assignee_id = ?
		) s
		WHERE s.rn = 1`)
	rows, err := db.QueryContext(ctx, q, sprintID, memberID)
	if err != nil {
		return nil, fmt.Errorf("sprint counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var resolved sql.NullTime
		if err := rows.Scan(&status, &resolved); err != nil {
			return nil, err
		}
		if resolved.Valid {
			out.Done++
		} else {
			out.WIP++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// PR counts: look at pull_requests opened by this member that
	// reference any issue assigned to the sprint via epic_key. This
	// only works for projects where the linker resolves an epic key
	// onto every PR — when it doesn't, the counts stay at zero.
	prQ := rebindSimple(dialect, `
		SELECT
		  SUM(CASE WHEN merged_at IS NOT NULL THEN 1 ELSE 0 END),
		  SUM(CASE WHEN merged_at IS NULL AND closed_at IS NULL THEN 1 ELSE 0 END)
		FROM pull_requests
		WHERE author_id = ?
		  AND epic_key IN (
		    SELECT DISTINCT issue_key FROM issue_snapshots
		    WHERE sprint_id = ? AND assignee_id = ?
		  )`)
	var merged, open sql.NullInt64
	if err := db.QueryRowContext(ctx, prQ, memberID, sprintID, memberID).Scan(&merged, &open); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("sprint pr counts: %w", err)
	}
	out.PRsMerged = int(merged.Int64)
	out.PRsOpen = int(open.Int64)
	return out, nil
}

// ----------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------

// rebindSimple is a duplicate of storage.rebind — kept here to avoid
// exporting the unexported helper from storage. The aggregator does
// the same trick.
func rebindSimple(d storage.Dialect, q string) string {
	if d.Placeholder(1) == "?" {
		return q
	}
	var b []byte
	n := 1
	for i := 0; i < len(q); i++ {
		if q[i] == '?' {
			b = append(b, []byte(d.Placeholder(n))...)
			n++
		} else {
			b = append(b, q[i])
		}
	}
	return string(b)
}

// hoursBetween returns a SQL expression evaluating to the number of
// hours between `endCol` and `startCol` (decimal). Dialect-aware:
//
//   - SQLite: julianday(end) - julianday(start) yields fractional days;
//     multiply by 24.
//   - Postgres: extract(epoch from end - start) yields seconds; divide
//     by 3600.
//
// Both cols may be expressions or column names; the caller is on the
// hook for safe quoting if interpolating untrusted strings (we don't —
// only fixed column names go through here).
func hoursBetween(d storage.Dialect, endCol, startCol string) string {
	if d.Name() == "postgres" {
		return fmt.Sprintf("EXTRACT(EPOCH FROM (%s - %s)) / 3600.0", endCol, startCol)
	}
	return fmt.Sprintf("(julianday(%s) - julianday(%s)) * 24.0", endCol, startCol)
}

// percentile returns the p-th percentile (0..1) of a sorted ascending
// slice using nearest-rank rounded down. Returns 0 for empty input.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(p * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
