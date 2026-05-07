/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// genericStore is the single Store implementation, parameterised by a
// Dialect (SQLite or Postgres). It speaks `?`-placeholder SQL and lets
// rebind() shift to `$1`-style at exec time when the dialect requires.
//
// The schema itself is portable — see migrations/0001_init.sql. The only
// dialect-specific SQL lives in the aggregator (week-start expression),
// which routes through dialect.WeekStart.
type genericStore struct {
	db *sql.DB
	d  Dialect
}

func (s *genericStore) Close() error    { return s.db.Close() }
func (s *genericStore) DB() *sql.DB     { return s.db }
func (s *genericStore) Dialect() Dialect { return s.d }

func (s *genericStore) Migrate(ctx context.Context) error {
	return runMigrations(ctx, s.db, s.d)
}

// ----------------------------------------------------------------------
// collection_runs
// ----------------------------------------------------------------------

func (s *genericStore) WriteCollectionRun(ctx context.Context, r CollectionRun) error {
	q := rebind(s.d, `
		INSERT INTO collection_runs (id, captured_at, source, duration_ms, record_count, error)
		VALUES (?, ?, ?, ?, ?, ?)`)
	_, err := s.db.ExecContext(ctx, q,
		r.ID, r.CapturedAt, r.Source, r.DurationMs, r.RecordCount, nullable(r.Error))
	return err
}

func (s *genericStore) LatestCollectionRun(ctx context.Context, source string) (*CollectionRun, error) {
	q := rebind(s.d, `
		SELECT id, captured_at, source, duration_ms, record_count, COALESCE(error, '')
		FROM collection_runs
		WHERE source = ?
		ORDER BY captured_at DESC
		LIMIT 1`)
	row := s.db.QueryRowContext(ctx, q, source)
	var r CollectionRun
	if err := row.Scan(&r.ID, &r.CapturedAt, &r.Source, &r.DurationMs, &r.RecordCount, &r.Error); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// ----------------------------------------------------------------------
// issue_snapshots
// ----------------------------------------------------------------------

func (s *genericStore) WriteIssueSnapshots(ctx context.Context, runID string, issues []IssueSnapshot) error {
	if len(issues) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := rebind(s.d, `
		INSERT INTO issue_snapshots (
			run_id, issue_key, issue_type, status, assignee_id, pillar_id,
			components, versions, sprint_id, story_points, created_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, it := range issues {
		comps, _ := json.Marshal(it.Components)
		vers, _ := json.Marshal(it.Versions)
		_, err := stmt.ExecContext(ctx,
			runID, it.IssueKey, it.IssueType, it.Status,
			nullable(it.AssigneeID), nullable(it.PillarID),
			string(comps), string(vers),
			nullable(it.SprintID), it.StoryPoints,
			it.CreatedAt, it.ResolvedAt,
		)
		if err != nil {
			return fmt.Errorf("insert issue %s: %w", it.IssueKey, err)
		}
	}
	return tx.Commit()
}

// ----------------------------------------------------------------------
// pull_requests / pr_reviews — UPSERT on natural key
// ----------------------------------------------------------------------

func (s *genericStore) UpsertPullRequests(ctx context.Context, prs []PullRequest) error {
	if len(prs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := rebind(s.d, `
		INSERT INTO pull_requests (
			id, repo_id, number, title, state, author_id, github_author_login,
			head_branch, base_branch, additions, deletions, changed_files,
			epic_key, created_at, first_review_at, merged_at, closed_at, fetched_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			state = excluded.state,
			author_id = excluded.author_id,
			github_author_login = excluded.github_author_login,
			additions = excluded.additions,
			deletions = excluded.deletions,
			changed_files = excluded.changed_files,
			epic_key = excluded.epic_key,
			first_review_at = excluded.first_review_at,
			merged_at = excluded.merged_at,
			closed_at = excluded.closed_at,
			fetched_at = excluded.fetched_at`)
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, p := range prs {
		_, err := stmt.ExecContext(ctx,
			p.ID, p.RepoID, p.Number, p.Title, p.State, nullable(p.AuthorID), nullable(p.GitHubAuthorLogin),
			nullable(p.HeadBranch), nullable(p.BaseBranch), p.Additions, p.Deletions, p.ChangedFiles,
			nullable(p.EpicKey), p.CreatedAt, p.FirstReviewAt, p.MergedAt, p.ClosedAt, p.FetchedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert pr %s: %w", p.ID, err)
		}
	}
	return tx.Commit()
}

func (s *genericStore) UpsertPRReviews(ctx context.Context, reviews []PRReview) error {
	if len(reviews) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := rebind(s.d, `
		INSERT INTO pr_reviews (id, pr_id, reviewer_id, github_reviewer_login, state, submitted_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reviewer_id = excluded.reviewer_id,
			github_reviewer_login = excluded.github_reviewer_login,
			state = excluded.state,
			submitted_at = excluded.submitted_at`)
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range reviews {
		_, err := stmt.ExecContext(ctx, r.ID, r.PRID, nullable(r.ReviewerID),
			nullable(r.GitHubReviewerLogin), r.State, r.SubmittedAt)
		if err != nil {
			return fmt.Errorf("upsert review %s: %w", r.ID, err)
		}
	}
	return tx.Commit()
}

// ----------------------------------------------------------------------
// members
// ----------------------------------------------------------------------

func (s *genericStore) UpsertMember(ctx context.Context, m Member) error {
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	active := 0
	if m.Active {
		active = 1
	}
	q := rebind(s.d, `
		INSERT INTO members (id, display_name, email, jira_account_id, github_login, pillar_id, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			email = excluded.email,
			jira_account_id = excluded.jira_account_id,
			github_login = excluded.github_login,
			pillar_id = excluded.pillar_id,
			active = excluded.active,
			updated_at = excluded.updated_at`)
	_, err := s.db.ExecContext(ctx, q,
		m.ID, m.DisplayName, nullable(m.Email),
		nullable(m.JiraAccountID), nullable(m.GitHubLogin),
		nullable(m.PillarID), active, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *genericStore) ListMembers(ctx context.Context) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, display_name, COALESCE(email, ''), COALESCE(jira_account_id, ''),
		       COALESCE(github_login, ''), COALESCE(pillar_id, ''), active,
		       created_at, updated_at
		FROM members
		ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		var active int
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.Email, &m.JiraAccountID,
			&m.GitHubLogin, &m.PillarID, &active, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.Active = active != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------
// member_week_metrics — read path
// ----------------------------------------------------------------------

func (s *genericStore) MemberWeekMetrics(ctx context.Context, q MemberWeekQuery) ([]MemberWeekRow, error) {
	conds := []string{"week_start >= ?", "week_start <= ?"}
	args := []interface{}{q.From, q.To}

	if len(q.MemberIDs) > 0 {
		ph := make([]string, len(q.MemberIDs))
		for i, id := range q.MemberIDs {
			ph[i] = "?"
			args = append(args, id)
		}
		conds = append(conds, "member_id IN ("+strings.Join(ph, ",")+")")
	}
	if len(q.PillarIDs) > 0 {
		ph := make([]string, len(q.PillarIDs))
		for i, id := range q.PillarIDs {
			ph[i] = "?"
			args = append(args, id)
		}
		conds = append(conds, "pillar_id IN ("+strings.Join(ph, ",")+")")
	}
	if q.Component != "" {
		conds = append(conds, "component = ?")
		args = append(args, q.Component)
	}

	query := rebind(s.d, `
		SELECT member_id, week_start, pillar_id, component,
		       jira_issues_done, jira_points_done, prs_merged, prs_reviewed,
		       review_latency_p50_hours
		FROM member_week_metrics
		WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY week_start, member_id`)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberWeekRow
	for rows.Next() {
		var r MemberWeekRow
		var lat sql.NullFloat64
		if err := rows.Scan(&r.MemberID, &r.WeekStart, &r.PillarID, &r.Component,
			&r.JiraIssuesDone, &r.JiraPointsDone, &r.PRsMerged, &r.PRsReviewed, &lat); err != nil {
			return nil, err
		}
		if lat.Valid {
			v := lat.Float64
			r.ReviewLatencyP50Hours = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// nullable converts an empty Go string to a NULL on the wire.
func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
