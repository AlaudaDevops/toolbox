/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Package contributions exposes member/team/pillar rollups to the API
// layer. It sits between the storage tier (raw rows) and the HTTP
// handlers (JSON shapes the frontend consumes).
//
// The two responsibilities are intentionally split:
//
//   - Aggregator (aggregator.go) — writes member_week_metrics from
//     issue_snapshots + pull_requests + pr_reviews. Runs after every
//     storage cycle. Stub in B1; real impl in B2.
//
//   - Service (this file) — read-only. Queries the rollup table and
//     post-processes for the API.
package contributions

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// Service is the read-only contributions API surface.
type Service struct {
	store    storage.Store
	pillarMP *PillarMap
}

func NewService(store storage.Store) *Service {
	return &Service{store: store, pillarMP: &PillarMap{}}
}

// SetPillarMap installs the pillar attribution map. Called from main.go
// after config load. Safe to call once at startup; not thread-safe to
// swap at runtime.
func (s *Service) SetPillarMap(pm *PillarMap) {
	if pm == nil {
		pm = &PillarMap{}
	}
	s.pillarMP = pm
}

// PillarMap returns the configured attribution map (never nil).
func (s *Service) PillarMap() *PillarMap {
	if s.pillarMP == nil {
		return &PillarMap{}
	}
	return s.pillarMP
}

// MemberSummary is one row of the team-overview dashboard.
//
// Components is the *flat* set of Jira components seen on this member's
// issue snapshots in the window. The frontend uses it to derive the
// member's pillar by matching against BasicPillar.component (same logic
// the roadmap tab uses), so the team-analytics view stays consistent
// with the roadmap source-of-truth instead of relying on a free-form
// operator-set members.pillar_id.
type MemberSummary struct {
	MemberID         string   `json:"member_id"`
	WeekTotals       []Bucket `json:"week_totals"`
	JiraIssuesDone   int      `json:"jira_issues_done"`
	JiraPointsDone   float64  `json:"jira_points_done"`
	PRsMerged        int      `json:"prs_merged"`
	PRsReviewed      int      `json:"prs_reviewed"`
	ReviewLatencyP50 float64  `json:"review_latency_p50_hours,omitempty"`
	Components       []string `json:"components,omitempty"`
}

// Bucket is one weekly aggregation point.
type Bucket struct {
	WeekStart time.Time `json:"week_start"`
	JiraDone  int       `json:"jira_done"`
	PRsMerged int       `json:"prs_merged"`
	Reviews   int       `json:"reviews"`
}

// TeamOverview returns one MemberSummary per member, summed across the
// requested window. Filters mirror MemberWeekQuery so any pillar or
// component slice flows through unchanged.
func (s *Service) TeamOverview(ctx context.Context, q storage.MemberWeekQuery) ([]MemberSummary, error) {
	rows, err := s.store.MemberWeekMetrics(ctx, q)
	if err != nil {
		return nil, err
	}
	byMember := map[string]*MemberSummary{}
	bucketByMember := map[string]map[time.Time]*Bucket{}

	for _, r := range rows {
		ms, ok := byMember[r.MemberID]
		if !ok {
			ms = &MemberSummary{MemberID: r.MemberID}
			byMember[r.MemberID] = ms
			bucketByMember[r.MemberID] = map[time.Time]*Bucket{}
		}
		ms.JiraIssuesDone += r.JiraIssuesDone
		ms.JiraPointsDone += r.JiraPointsDone
		ms.PRsMerged += r.PRsMerged
		ms.PRsReviewed += r.PRsReviewed
		// We pick the latest non-nil latency seen as a representative;
		// proper aggregation across weeks happens in the Aggregator.
		if r.ReviewLatencyP50Hours != nil {
			ms.ReviewLatencyP50 = *r.ReviewLatencyP50Hours
		}

		buckets := bucketByMember[r.MemberID]
		bk, ok := buckets[r.WeekStart]
		if !ok {
			bk = &Bucket{WeekStart: r.WeekStart}
			buckets[r.WeekStart] = bk
		}
		bk.JiraDone += r.JiraIssuesDone
		bk.PRsMerged += r.PRsMerged
		bk.Reviews += r.PRsReviewed
	}

	// Components per member — one extra query, joined client-side onto
	// each MemberSummary. The aggregator's rollup table doesn't carry
	// component lists (only the {member, week, pillar, component} primary
	// key with one row per component-tag), so this is the cheapest way
	// to surface a per-member component list to the frontend without
	// schema changes.
	componentsByMember, err := s.componentsByMember(ctx, q)
	if err != nil {
		// Non-fatal: an empty components list just means the frontend
		// falls back to the operator-set pillar. Log path: caller logs.
		componentsByMember = map[string][]string{}
	}

	out := make([]MemberSummary, 0, len(byMember))
	for id, ms := range byMember {
		bks := make([]Bucket, 0, len(bucketByMember[id]))
		for _, b := range bucketByMember[id] {
			bks = append(bks, *b)
		}
		sort.Slice(bks, func(i, j int) bool { return bks[i].WeekStart.Before(bks[j].WeekStart) })
		ms.WeekTotals = bks
		ms.Components = componentsByMember[id]
		out = append(out, *ms)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MemberID < out[j].MemberID })
	return out, nil
}

// componentsByMember tallies the distinct components each member has
// touched in the [From, To) window. Latest-snapshot-per-issue de-dup
// (so one issue contributes once even across multiple collection cycles),
// then a flat union of components across the member's deduped issue
// set. Order is descending by frequency, then alphabetic — useful when
// the frontend wants "primary component" without an extra count map.
func (s *Service) componentsByMember(ctx context.Context, q storage.MemberWeekQuery) (map[string][]string, error) {
	d, ok := s.store.(interface {
		Dialect() storage.Dialect
		DB() *sql.DB
	})
	if !ok {
		return nil, nil
	}
	dialect := d.Dialect()
	db := d.DB()

	sqlStr := rebindSimple(dialect, `
		SELECT assignee_id, components
		FROM (
		  SELECT s.assignee_id, s.components, s.issue_key,
		         ROW_NUMBER() OVER (PARTITION BY s.issue_key ORDER BY r.captured_at DESC) AS rn
		  FROM issue_snapshots s
		  JOIN collection_runs r ON s.run_id = r.id
		  WHERE s.assignee_id IS NOT NULL AND s.assignee_id <> ''
		    AND s.created_at < ?
		    AND (s.resolved_at IS NULL OR s.resolved_at >= ?)
		) ranked
		WHERE rn = 1`)
	rows, err := db.QueryContext(ctx, sqlStr, q.To, q.From)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tally := map[string]map[string]int{}
	for rows.Next() {
		var memberID string
		var raw sql.NullString
		if err := rows.Scan(&memberID, &raw); err != nil {
			return nil, err
		}
		if !raw.Valid || raw.String == "" || raw.String == "null" {
			continue
		}
		var comps []string
		if err := json.Unmarshal([]byte(raw.String), &comps); err != nil {
			continue
		}
		mp, ok := tally[memberID]
		if !ok {
			mp = map[string]int{}
			tally[memberID] = mp
		}
		for _, c := range comps {
			if c != "" {
				mp[c]++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(tally))
	for mid, counts := range tally {
		list := make([]string, 0, len(counts))
		for c := range counts {
			list = append(list, c)
		}
		sort.Slice(list, func(i, j int) bool {
			if counts[list[i]] != counts[list[j]] {
				return counts[list[i]] > counts[list[j]]
			}
			return list[i] < list[j]
		})
		out[mid] = list
	}
	return out, nil
}

// MemberDetail returns the per-week breakdown for one member.
func (s *Service) MemberDetail(ctx context.Context, memberID string, q storage.MemberWeekQuery) (*MemberSummary, error) {
	q.MemberIDs = []string{memberID}
	all, err := s.TeamOverview(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return &MemberSummary{MemberID: memberID}, nil
	}
	return &all[0], nil
}
