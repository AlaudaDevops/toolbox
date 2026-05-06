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

	// modernc.org/sqlite is a pure-Go SQLite driver — no CGO. Important so
	// the existing multi-stage Alpine Docker build stays slim and the
	// binary stays statically linked.
	_ "modernc.org/sqlite"
)

// OpenSQLite returns a Store backed by a SQLite database at path.
//
// The driver registers itself as "sqlite". We set a small set of pragmas
// that are essentially mandatory for any concurrent use:
//   - journal_mode=WAL — single-writer-many-readers; the collector goroutine
//     writes while HTTP handlers read.
//   - synchronous=NORMAL — durability is not as important to us as
//     throughput; any single lost cycle is reconstructable from Jira.
//   - busy_timeout=5000 — back off short lock collisions instead of failing.
//   - foreign_keys=ON — we declare FKs in the schema; let the DB enforce.
func OpenSQLite(path string) (Store, error) {
	if path == "" {
		return nil, fmt.Errorf("storage path is empty")
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}
	// Even with WAL, having more than one writable conn breaks SQLite's
	// single-writer model under contention. Cap to one writer; the pool
	// auto-grows for read-only ops.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

type sqliteStore struct {
	db *sql.DB
}

func (s *sqliteStore) Close() error  { return s.db.Close() }
func (s *sqliteStore) DB() *sql.DB   { return s.db }

func (s *sqliteStore) Migrate(ctx context.Context) error {
	return runMigrations(ctx, s.db)
}

// ----------------------------------------------------------------------
// collection_runs
// ----------------------------------------------------------------------

func (s *sqliteStore) WriteCollectionRun(ctx context.Context, r CollectionRun) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collection_runs (id, captured_at, source, duration_ms, record_count, error)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.CapturedAt, r.Source, r.DurationMs, r.RecordCount, nullable(r.Error))
	return err
}

func (s *sqliteStore) LatestCollectionRun(ctx context.Context, source string) (*CollectionRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, captured_at, source, duration_ms, record_count, COALESCE(error, '')
		FROM collection_runs
		WHERE source = ?
		ORDER BY captured_at DESC
		LIMIT 1`, source)
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

func (s *sqliteStore) WriteIssueSnapshots(ctx context.Context, runID string, issues []IssueSnapshot) error {
	if len(issues) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO issue_snapshots (
			run_id, issue_key, issue_type, status, assignee_id, pillar_id,
			components, versions, sprint_id, story_points, created_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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

func (s *sqliteStore) UpsertPullRequests(ctx context.Context, prs []PullRequest) error {
	if len(prs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pull_requests (
			id, repo_id, number, title, state, author_id,
			head_branch, base_branch, additions, deletions, changed_files,
			epic_key, created_at, first_review_at, merged_at, closed_at, fetched_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			state = excluded.state,
			author_id = excluded.author_id,
			additions = excluded.additions,
			deletions = excluded.deletions,
			changed_files = excluded.changed_files,
			epic_key = excluded.epic_key,
			first_review_at = excluded.first_review_at,
			merged_at = excluded.merged_at,
			closed_at = excluded.closed_at,
			fetched_at = excluded.fetched_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, p := range prs {
		_, err := stmt.ExecContext(ctx,
			p.ID, p.RepoID, p.Number, p.Title, p.State, nullable(p.AuthorID),
			nullable(p.HeadBranch), nullable(p.BaseBranch), p.Additions, p.Deletions, p.ChangedFiles,
			nullable(p.EpicKey), p.CreatedAt, p.FirstReviewAt, p.MergedAt, p.ClosedAt, p.FetchedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert pr %s: %w", p.ID, err)
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) UpsertPRReviews(ctx context.Context, reviews []PRReview) error {
	if len(reviews) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pr_reviews (id, pr_id, reviewer_id, state, submitted_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			state = excluded.state,
			submitted_at = excluded.submitted_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range reviews {
		_, err := stmt.ExecContext(ctx, r.ID, r.PRID, nullable(r.ReviewerID), r.State, r.SubmittedAt)
		if err != nil {
			return fmt.Errorf("upsert review %s: %w", r.ID, err)
		}
	}
	return tx.Commit()
}

// ----------------------------------------------------------------------
// members
// ----------------------------------------------------------------------

func (s *sqliteStore) UpsertMember(ctx context.Context, m Member) error {
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	active := 0
	if m.Active {
		active = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO members (id, display_name, email, jira_account_id, github_login, pillar_id, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			email = excluded.email,
			jira_account_id = excluded.jira_account_id,
			github_login = excluded.github_login,
			pillar_id = excluded.pillar_id,
			active = excluded.active,
			updated_at = excluded.updated_at`,
		m.ID, m.DisplayName, nullable(m.Email),
		nullable(m.JiraAccountID), nullable(m.GitHubLogin),
		nullable(m.PillarID), active, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *sqliteStore) ListMembers(ctx context.Context) ([]Member, error) {
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

func (s *sqliteStore) MemberWeekMetrics(ctx context.Context, q MemberWeekQuery) ([]MemberWeekRow, error) {
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

	query := `
		SELECT member_id, week_start, pillar_id, component,
		       jira_issues_done, jira_points_done, prs_merged, prs_reviewed,
		       review_latency_p50_hours
		FROM member_week_metrics
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY week_start, member_id`
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
