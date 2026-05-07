/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

// Package storage is the persistence layer for roadmap-planner team analytics.
//
// The package owns the schema (migrations/) and exposes one interface,
// Store, with a single in-tree SQLite implementation. The interface is
// shaped so a Postgres implementation can be added later without touching
// callers — we keep SQL portable (no SQLite-isms in the application
// queries) and use database/sql throughout.
//
// See docs/team-analytics/PROPOSAL.md §6 for schema rationale.
package storage

import (
	"context"
	"database/sql"
	"time"
)

// Store is the persistence boundary for team-analytics data.
//
// Two responsibilities:
//   - durable history (snapshots written every collection cycle, immutable)
//   - rollups (member_week_metrics — derived, droppable, queried by the API)
//
// All methods must be safe for concurrent use; implementations rely on the
// underlying *sql.DB pool for that.
type Store interface {
	// Lifecycle.
	Close() error
	Migrate(ctx context.Context) error
	DB() *sql.DB

	// Collection runs — every Collect() call ends with one of these.
	WriteCollectionRun(ctx context.Context, r CollectionRun) error
	LatestCollectionRun(ctx context.Context, source string) (*CollectionRun, error)

	// Issue snapshots (one row per issue per run).
	WriteIssueSnapshots(ctx context.Context, runID string, issues []IssueSnapshot) error

	// GitHub.
	UpsertPullRequests(ctx context.Context, prs []PullRequest) error
	UpsertPRReviews(ctx context.Context, reviews []PRReview) error

	// Members.
	UpsertMember(ctx context.Context, m Member) error
	ListMembers(ctx context.Context) ([]Member, error)

	// Read paths used by the contributions service.
	MemberWeekMetrics(ctx context.Context, q MemberWeekQuery) ([]MemberWeekRow, error)
}

// ----------------------------------------------------------------------
// Domain shapes — kept narrow on purpose, mapped 1:1 to the schema.
// ----------------------------------------------------------------------

// CollectionRun is the audit record for a single fetch cycle.
type CollectionRun struct {
	ID          string
	CapturedAt  time.Time
	Source      string // "jira" | "github"
	DurationMs  int64
	RecordCount int
	Error       string
}

// IssueSnapshot is the durable shape of a Jira issue at a point in time.
// It's deliberately denormalised so a single SELECT serves the dashboards.
type IssueSnapshot struct {
	IssueKey    string
	IssueType   string
	Status      string
	AssigneeID  string // FK to members.id, may be empty
	PillarID    string
	Components  []string // serialised JSON in storage
	Versions    []string
	SprintID    string
	StoryPoints float64
	CreatedAt   time.Time
	ResolvedAt  *time.Time
}

// PullRequest is a GitHub PR record. Linked to an Epic via EpicKey when
// the configured Linker can resolve one (branch regex, PR title, etc.).
type PullRequest struct {
	ID            string // "org/name#number"
	RepoID        string
	Number        int
	Title         string
	State         string // "open" | "merged" | "closed"
	AuthorID      string // FK to members.id, empty if no match
	HeadBranch    string
	BaseBranch    string
	Additions     int
	Deletions     int
	ChangedFiles  int
	EpicKey       string
	CreatedAt     time.Time
	FirstReviewAt *time.Time
	MergedAt      *time.Time
	ClosedAt      *time.Time
	FetchedAt     time.Time
}

// PRReview is one review event on a PR.
type PRReview struct {
	ID          string
	PRID        string
	ReviewerID  string
	State       string // approved | changes_requested | commented
	SubmittedAt time.Time
}

// Member is the join entity across Jira and GitHub.
//
// JSON tags use snake_case to match what the frontend (and any future
// API consumer) expects; without them the default marshaller emits
// PascalCase field names and the Team dashboard breaks because
// `m.display_name` is undefined.
type Member struct {
	ID            string    `json:"id"`
	DisplayName   string    `json:"display_name"`
	Email         string    `json:"email,omitempty"`
	JiraAccountID string    `json:"jira_account_id,omitempty"`
	GitHubLogin   string    `json:"github_login,omitempty"`
	PillarID      string    `json:"pillar_id,omitempty"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ----------------------------------------------------------------------
// Query parameters / result shapes
// ----------------------------------------------------------------------

// MemberWeekQuery filters the rollup read.
type MemberWeekQuery struct {
	From      time.Time
	To        time.Time
	MemberIDs []string
	PillarIDs []string
	Component string
}

// MemberWeekRow is one row of the contributions feed used by the Team
// dashboard. Mirrors member_week_metrics 1:1.
type MemberWeekRow struct {
	MemberID              string    `json:"member_id"`
	WeekStart             time.Time `json:"week_start"`
	PillarID              string    `json:"pillar_id"`
	Component             string    `json:"component"`
	JiraIssuesDone        int       `json:"jira_issues_done"`
	JiraPointsDone        float64   `json:"jira_points_done"`
	PRsMerged             int       `json:"prs_merged"`
	PRsReviewed           int       `json:"prs_reviewed"`
	ReviewLatencyP50Hours *float64  `json:"review_latency_p50_hours,omitempty"`
}
