/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

// Package jirasync orchestrates Jira → storage ingestion for the
// team-analytics module.
//
// Responsibilities:
//   - On first run: 3-pass backfill (resolved, open-in-window, expanded
//     changelog) per PROPOSAL.md §6.5.
//   - On subsequent runs: incremental fetch of issues updated since the
//     previous run, written as a fresh snapshot row (we never mutate
//     historic rows).
//   - Member upsert: any assignee seen turns into a row in `members`,
//     keyed on a stable slug derived from email/accountID/name.
//
// The package depends on internal/jira (for the API surface) and
// internal/storage (for persistence). It does not reach into the metrics
// collector — the existing collector's in-memory cache is intentionally
// unaffected, so the roadmap UI stays unchanged when team analytics is
// turned on.
package jirasync

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
	"go.uber.org/zap"
)

// Config holds the knobs the operator sets in config.yaml.
type Config struct {
	Project          string
	BackfillDays     int    // first-run window
	StoryPointsField string // optional Jira customfield id, e.g. "customfield_10016"
	SprintField      string // optional, e.g. "customfield_10020"

	// PageSize for JQL searches. Default 200 — Jira caps at 100 for some
	// instances; 200 still works on most.
	PageSize int

	// MaxBackfillIssues is a safety net — if the first-run fetch returns
	// more issues than this, we abort. Default 0 = unlimited.
	MaxBackfillIssues int
}

// Searcher is the slice of *jira.Client this package needs. Defining it
// as an interface here lets the round-trip test substitute a fake
// without spinning up a real Jira.
type Searcher interface {
	SearchSnapshots(ctx context.Context, opts jira.SnapshotSearchOpts) ([]jira.SnapshotIssue, error)
}

// Syncer drives one sync cycle (backfill or incremental).
type Syncer struct {
	client Searcher
	store  storage.Store
	cfg    Config
	logger *zap.Logger
}

// NewSyncer builds a Syncer with sane defaults.
func NewSyncer(client Searcher, store storage.Store, cfg Config) *Syncer {
	if cfg.BackfillDays <= 0 {
		cfg.BackfillDays = 180
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 200
	}
	return &Syncer{
		client: client,
		store:  store,
		cfg:    cfg,
		logger: logger.WithComponent("jira-syncer"),
	}
}

// Run executes one cycle. Auto-detects backfill vs incremental from
// collection_runs. Returns the *Result so callers can log / surface to
// the UI.
type Result struct {
	Mode         string // "backfill" | "incremental"
	IssuesSeen   int
	IssuesWritten int
	MembersSeen  int
	DurationMs   int64
}

func (s *Syncer) Run(ctx context.Context) (*Result, error) {
	last, err := s.store.LatestCollectionRun(ctx, "jira")
	if err != nil {
		return nil, fmt.Errorf("read last run: %w", err)
	}
	if last == nil {
		return s.runBackfill(ctx)
	}
	return s.runIncremental(ctx, last.CapturedAt)
}

// runBackfill executes the 3-pass JQL set described in PROPOSAL.md §6.5.
//
// We expand the changelog only on Pass 1 (resolved issues), where cycle
// time is computable. Pass 2 covers open issues that started in window
// — for them, "current state" + creation timestamp is enough.
func (s *Syncer) runBackfill(ctx context.Context) (*Result, error) {
	start := time.Now()
	runID := fmt.Sprintf("jira-backfill-%d", start.UnixNano())
	cutoff := start.AddDate(0, 0, -s.cfg.BackfillDays)
	jqlDate := cutoff.Format("2006-01-02")

	s.logger.Info("jira backfill starting",
		zap.String("project", s.cfg.Project),
		zap.Time("since", cutoff),
		zap.Int("days", s.cfg.BackfillDays))

	pass1JQL := fmt.Sprintf(`project = %s AND resolved >= "%s"`, s.cfg.Project, jqlDate)
	pass2JQL := fmt.Sprintf(`project = %s AND created >= "%s" AND resolution is EMPTY`, s.cfg.Project, jqlDate)

	pass1, err := s.searchPaged(ctx, pass1JQL, true /* expand changelog */)
	if err != nil {
		return nil, fmt.Errorf("backfill pass 1 (resolved): %w", err)
	}
	pass2, err := s.searchPaged(ctx, pass2JQL, false)
	if err != nil {
		return nil, fmt.Errorf("backfill pass 2 (open in window): %w", err)
	}

	all := append(pass1, pass2...)
	if s.cfg.MaxBackfillIssues > 0 && len(all) > s.cfg.MaxBackfillIssues {
		return nil, fmt.Errorf("backfill aborted: %d issues exceeds MaxBackfillIssues=%d", len(all), s.cfg.MaxBackfillIssues)
	}

	written, members, err := s.writeBatch(ctx, runID, all)
	if err != nil {
		return nil, err
	}
	if err := s.store.WriteCollectionRun(ctx, storage.CollectionRun{
		ID:          runID,
		CapturedAt:  start,
		Source:      "jira",
		DurationMs:  time.Since(start).Milliseconds(),
		RecordCount: written,
	}); err != nil {
		return nil, fmt.Errorf("write collection run: %w", err)
	}

	s.logger.Info("jira backfill done",
		zap.Int("issues", written),
		zap.Int("members", members),
		zap.Duration("duration", time.Since(start)))
	return &Result{
		Mode: "backfill", IssuesSeen: len(all), IssuesWritten: written,
		MembersSeen: members, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// runIncremental fetches issues touched since the previous run.
//
// We deliberately re-fetch via JQL `updated > since` rather than relying
// on Jira webhooks: the webhook story is per-instance complicated, and
// the JQL fetch is idempotent — re-running a window twice writes two
// snapshot rows but doesn't corrupt history (snapshots are versioned by
// run_id).
func (s *Syncer) runIncremental(ctx context.Context, since time.Time) (*Result, error) {
	start := time.Now()
	runID := fmt.Sprintf("jira-%d", start.UnixNano())
	// Subtract 1h overlap to absorb clock skew, like the github sync.
	cutoff := since.Add(-1 * time.Hour)
	jqlDate := cutoff.Format("2006-01-02 15:04")

	jql := fmt.Sprintf(`project = %s AND updated >= "%s"`, s.cfg.Project, jqlDate)

	s.logger.Info("jira incremental starting",
		zap.String("project", s.cfg.Project),
		zap.Time("since", cutoff))

	issues, err := s.searchPaged(ctx, jql, false)
	if err != nil {
		return nil, fmt.Errorf("incremental search: %w", err)
	}

	written, members, err := s.writeBatch(ctx, runID, issues)
	if err != nil {
		return nil, err
	}
	if err := s.store.WriteCollectionRun(ctx, storage.CollectionRun{
		ID:          runID,
		CapturedAt:  start,
		Source:      "jira",
		DurationMs:  time.Since(start).Milliseconds(),
		RecordCount: written,
	}); err != nil {
		return nil, fmt.Errorf("write collection run: %w", err)
	}

	s.logger.Info("jira incremental done",
		zap.Int("issues", written),
		zap.Int("members", members),
		zap.Duration("duration", time.Since(start)))
	return &Result{
		Mode: "incremental", IssuesSeen: len(issues), IssuesWritten: written,
		MembersSeen: members, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (s *Syncer) searchPaged(ctx context.Context, jql string, expand bool) ([]jira.SnapshotIssue, error) {
	return s.client.SearchSnapshots(ctx, jira.SnapshotSearchOpts{
		JQL:              jql,
		PageSize:         s.cfg.PageSize,
		ExpandChangelog:  expand,
		StoryPointsField: s.cfg.StoryPointsField,
		SprintField:      s.cfg.SprintField,
	})
}

// writeBatch upserts all assignees we saw as members and writes one
// issue_snapshots row per issue under runID. Returns (issuesWritten,
// distinctMembersUpserted, err).
func (s *Syncer) writeBatch(ctx context.Context, runID string, issues []jira.SnapshotIssue) (int, int, error) {
	if len(issues) == 0 {
		return 0, 0, nil
	}

	// Member upsert pass first — so issue_snapshots can reference members
	// without a missing FK (we don't enforce FKs in SQLite for this; PG
	// would complain).
	memberSeen := map[string]bool{}
	for _, it := range issues {
		if it.Assignee == nil {
			continue
		}
		mid := MemberIDFromUser(it.Assignee)
		if mid == "" || memberSeen[mid] {
			continue
		}
		memberSeen[mid] = true
		if err := s.store.UpsertMember(ctx, storage.Member{
			ID:            mid,
			DisplayName:   coalesce(it.Assignee.DisplayName, it.Assignee.Name, mid),
			Email:         strings.ToLower(it.Assignee.EmailAddress),
			JiraAccountID: it.Assignee.AccountID,
			Active:        true,
		}); err != nil {
			s.logger.Warn("upsert member failed", zap.String("id", mid), zap.Error(err))
		}
	}

	// Convert to storage shape.
	snaps := make([]storage.IssueSnapshot, 0, len(issues))
	for _, it := range issues {
		snap := storage.IssueSnapshot{
			IssueKey:    it.Key,
			IssueType:   it.IssueType,
			Status:      it.Status,
			Components:  it.Components,
			Versions:    it.Versions,
			SprintID:    it.SprintID,
			StoryPoints: it.StoryPoints,
			CreatedAt:   it.CreatedAt,
			ResolvedAt:  it.ResolvedAt,
		}
		if it.Assignee != nil {
			snap.AssigneeID = MemberIDFromUser(it.Assignee)
		}
		snaps = append(snaps, snap)
	}
	if err := s.store.WriteIssueSnapshots(ctx, runID, snaps); err != nil {
		return 0, 0, fmt.Errorf("write snapshots: %w", err)
	}
	return len(snaps), len(memberSeen), nil
}

// MemberIDFromUser produces a stable internal id for a Jira user.
//
// Priority order:
//   - email (lowercased + sanitised) — most stable across Cloud/Server,
//     and aligns with GitHub's notion of identity.
//   - accountID — Cloud's globally-unique id, opaque but immutable.
//   - name (username) — Server fallback when no accountID.
//   - displayName-hash — last-resort, salted so we never produce empty.
//
// The function is exported so tests can assert against it and the future
// member-merge UI can reuse the slug strategy.
func MemberIDFromUser(u *models.User) string {
	if u == nil {
		return ""
	}
	if u.EmailAddress != "" {
		return slugifyEmail(u.EmailAddress)
	}
	if u.AccountID != "" {
		return "acct-" + sanitise(u.AccountID)
	}
	if u.Name != "" {
		return sanitise(u.Name)
	}
	if u.DisplayName != "" {
		h := sha1.Sum([]byte(u.DisplayName))
		return "u-" + hex.EncodeToString(h[:6])
	}
	return ""
}

func slugifyEmail(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	at := strings.IndexByte(s, '@')
	if at <= 0 {
		return sanitise(s)
	}
	return sanitise(s[:at])
}

// sanitise keeps [a-z0-9-_.], replaces other runes with '-', and trims
// leading/trailing dashes.
func sanitise(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func coalesce(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
