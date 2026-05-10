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
// issue snapshots in the window — useful as a debugging signal but not
// authoritative for pillar attribution. Pillars is the set of pillars
// the member is associated with, computed by the same matcher chain the
// throughput-by-pillar bucket aggregator uses (PillarsForComponents
// then PillarsForVersions fallback). Computing pillars server-side
// keeps the team table in sync with the chart even when an issue routes
// to its pillar via the Phase 1 fixVersion-prefix fallback rather than
// a Jira component match.
type MemberSummary struct {
	MemberID         string   `json:"member_id"`
	WeekTotals       []Bucket `json:"week_totals"`
	JiraIssuesDone   int      `json:"jira_issues_done"`
	JiraPointsDone   float64  `json:"jira_points_done"`
	PRsMerged        int      `json:"prs_merged"`
	PRsOpened        int      `json:"prs_opened"`
	PRsReviewed      int      `json:"prs_reviewed"`
	ReviewLatencyP50 float64  `json:"review_latency_p50_hours,omitempty"`
	Components       []string `json:"components,omitempty"`
	Pillars          []string `json:"pillars,omitempty"`
}

// Bucket is one weekly aggregation point.
type Bucket struct {
	WeekStart time.Time `json:"week_start"`
	JiraDone  int       `json:"jira_done"`
	Points    float64   `json:"points"`
	PRsMerged int       `json:"prs_merged"`
	PRsOpened int       `json:"prs_opened"`
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
		ms.PRsOpened += r.PRsOpened
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
		bk.Points += r.JiraPointsDone
		bk.PRsMerged += r.PRsMerged
		bk.PRsOpened += r.PRsOpened
		bk.Reviews += r.PRsReviewed
	}

	// Components + pillars per member — one extra query that walks each
	// member's deduped issue set and applies the same pillar matcher
	// chain as the throughput-by-pillar bucket aggregator. The rollup
	// table doesn't carry per-member pillar lists, so this is the
	// cheapest way to surface a directly-readable pillars[] to the
	// frontend without a schema change.
	componentsByMember, pillarsByMember, err := s.attributionByMember(ctx, q)
	if err != nil {
		// Non-fatal: empty lists just mean the frontend falls back to
		// the operator-set pillar. Log path: caller logs.
		componentsByMember = map[string][]string{}
		pillarsByMember = map[string][]string{}
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
		ms.Pillars = pillarsByMember[id]
		out = append(out, *ms)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MemberID < out[j].MemberID })
	return out, nil
}

// attributionByMember walks every member's deduped issue set in
// [From, To) and returns two parallel maps: the flat set of Jira
// components touched (descending by frequency, then alphabetic) and
// the union of pillars matched. Pillar matching mirrors the
// throughput-by-pillar bucket aggregator's chain — try
// PillarsForComponents on the issue's components, fall back to
// PillarsForVersions on its fixVersions — so the team table and
// the chart agree even when an issue routes to a pillar via the
// Phase 1 fixVersion-prefix fallback rather than a component hit.
//
// Latest-snapshot-per-issue de-dup runs in SQL (one row per issue_key,
// the most recently captured one), so an issue contributes its
// component/version values exactly once even across multiple
// collection cycles.
func (s *Service) attributionByMember(ctx context.Context, q storage.MemberWeekQuery) (map[string][]string, map[string][]string, error) {
	d, ok := s.store.(interface {
		Dialect() storage.Dialect
		DB() *sql.DB
	})
	if !ok {
		return nil, nil, nil
	}
	dialect := d.Dialect()
	db := d.DB()

	sqlStr := rebindSimple(dialect, `
		SELECT assignee_id, components, versions
		FROM (
		  SELECT s.assignee_id, s.components, s.versions, s.issue_key,
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
		return nil, nil, err
	}
	defer rows.Close()
	pm := s.PillarMap()
	compTally := map[string]map[string]int{}
	pillarSet := map[string]map[string]struct{}{}
	for rows.Next() {
		var memberID string
		var rawComps, rawVers sql.NullString
		if err := rows.Scan(&memberID, &rawComps, &rawVers); err != nil {
			return nil, nil, err
		}
		var comps, vers []string
		if rawComps.Valid && rawComps.String != "" && rawComps.String != "null" {
			_ = json.Unmarshal([]byte(rawComps.String), &comps)
		}
		if rawVers.Valid && rawVers.String != "" && rawVers.String != "null" {
			_ = json.Unmarshal([]byte(rawVers.String), &vers)
		}
		mp, ok := compTally[memberID]
		if !ok {
			mp = map[string]int{}
			compTally[memberID] = mp
		}
		for _, c := range comps {
			if c != "" {
				mp[c]++
			}
		}
		// Shared chain: components-first, fixVersions as the Phase 1
		// fallback. An issue contributes to every pillar it matches
		// (multi-pillar components / shared version_prefixes are
		// honored), and contributes nothing if neither matcher hits.
		matched := pm.PillarsFor(comps, vers)
		if len(matched) == 0 {
			continue
		}
		pset, ok := pillarSet[memberID]
		if !ok {
			pset = map[string]struct{}{}
			pillarSet[memberID] = pset
		}
		for _, p := range matched {
			pset[p] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	componentsOut := make(map[string][]string, len(compTally))
	for mid, counts := range compTally {
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
		componentsOut[mid] = list
	}
	pillarsOut := make(map[string][]string, len(pillarSet))
	for mid, set := range pillarSet {
		list := make([]string, 0, len(set))
		for p := range set {
			list = append(list, p)
		}
		sort.Strings(list)
		pillarsOut[mid] = list
	}
	return componentsOut, pillarsOut, nil
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
