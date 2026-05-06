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
	"sort"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// Service is the read-only contributions API surface.
type Service struct {
	store storage.Store
}

func NewService(store storage.Store) *Service {
	return &Service{store: store}
}

// MemberSummary is one row of the team-overview dashboard.
type MemberSummary struct {
	MemberID         string  `json:"member_id"`
	WeekTotals       []Bucket `json:"week_totals"`
	JiraIssuesDone   int     `json:"jira_issues_done"`
	JiraPointsDone   float64 `json:"jira_points_done"`
	PRsMerged        int     `json:"prs_merged"`
	PRsReviewed      int     `json:"prs_reviewed"`
	ReviewLatencyP50 float64 `json:"review_latency_p50_hours,omitempty"`
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

	out := make([]MemberSummary, 0, len(byMember))
	for id, ms := range byMember {
		bks := make([]Bucket, 0, len(bucketByMember[id]))
		for _, b := range bucketByMember[id] {
			bks = append(bks, *b)
		}
		sort.Slice(bks, func(i, j int) bool { return bks[i].WeekStart.Before(bks[j].WeekStart) })
		ms.WeekTotals = bks
		out = append(out, *ms)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MemberID < out[j].MemberID })
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
