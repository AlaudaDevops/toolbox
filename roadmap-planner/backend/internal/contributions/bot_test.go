/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

func TestBotSet(t *testing.T) {
	cases := []struct {
		name  string
		in    []string
		probe map[string]bool
	}{
		{
			name: "empty set rejects everything",
			in:   nil,
			probe: map[string]bool{
				"alaudabot":         false,
				"anybody":           false,
				"copilot[bot]":      false,
			},
		},
		{
			name: "exact + case-folded match",
			in:   []string{"alaudabot", "Edge-Katanomi-App2[bot]", "  copilot  "},
			probe: map[string]bool{
				"alaudabot":               true,
				"ALAUDABOT":               true,
				"edge-katanomi-app2[bot]": true,
				"copilot":                 true,
				"copilot-pull-request-reviewer[bot]": false,
				"daniel":                              false,
				"":                                    false,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBotSet(tc.in)
			for k, want := range tc.probe {
				if got := b.Contains(k); got != want {
					t.Errorf("Contains(%q) = %v, want %v", k, got, want)
				}
			}
		})
	}
}

// TestConsolidateBots seeds PRs + reviews authored by both a human and
// a bot login, runs the one-shot consolidation pass, and verifies all
// four side-effects: is_bot tagging, reviewer_id rewrite, author_id
// rewrite, and first_human_review_at population.
func TestConsolidateBots(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "bot.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, m := range []storage.Member{
		{ID: "bot", DisplayName: "Bot", Active: true},
		{ID: "daniel", DisplayName: "Daniel", GitHubLogin: "danielfbm", Active: true},
	} {
		if err := store.UpsertMember(ctx, m); err != nil {
			t.Fatalf("upsert %s: %v", m.ID, err)
		}
	}

	created := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	botReview := created.Add(1 * time.Hour)
	humanReview := created.Add(4 * time.Hour)

	if err := store.UpsertPullRequests(ctx, []storage.PullRequest{
		// Human-authored PR, will get both a bot review and a human review.
		{
			ID: "alaudadevops/x#1", Source: "github", RepoID: "alaudadevops/x", Number: 1,
			Title: "human PR", State: "merged",
			AuthorID: "daniel", AuthorLogin: "danielfbm",
			CreatedAt: created, MergedAt: ptrTime(humanReview.Add(2 * time.Hour)),
			FirstReviewAt: ptrTime(botReview), // pre-W2: filled with the earliest review including bots
			FetchedAt:     created,
		},
		// Bot-authored PR (renovate-style). Pre-W2 author_id is empty.
		{
			ID: "alaudadevops/x#2", Source: "github", RepoID: "alaudadevops/x", Number: 2,
			Title: "renovate PR", State: "merged",
			AuthorID: "", AuthorLogin: "alaudabot",
			CreatedAt: created, MergedAt: ptrTime(created.Add(30 * time.Minute)),
			FetchedAt: created,
		},
	}); err != nil {
		t.Fatalf("upsert prs: %v", err)
	}
	if err := store.UpsertPRReviews(ctx, []storage.PRReview{
		{ID: "alaudadevops/x#1/r1", PRID: "alaudadevops/x#1", Source: "github",
			ReviewerID: "", ReviewerLogin: "alaudabot", State: "commented", SubmittedAt: botReview},
		{ID: "alaudadevops/x#1/r2", PRID: "alaudadevops/x#1", Source: "github",
			ReviewerID: "daniel", ReviewerLogin: "danielfbm", State: "approved", SubmittedAt: humanReview},
	}); err != nil {
		t.Fatalf("upsert reviews: %v", err)
	}

	bots := NewBotSet([]string{"alaudabot"})
	res, err := ConsolidateBots(ctx, store, bots)
	if err != nil {
		t.Fatalf("consolidate: %v", err)
	}
	if res.ReviewsMarkedBot != 1 {
		t.Fatalf("reviews_marked_bot = %d, want 1", res.ReviewsMarkedBot)
	}
	if res.ReviewsAuthorRew != 1 {
		t.Fatalf("reviewer_id_rewritten = %d, want 1", res.ReviewsAuthorRew)
	}
	if res.PRsAuthorRewrite != 1 {
		t.Fatalf("pr_author_id_rewritten = %d, want 1 (the renovate PR)", res.PRsAuthorRewrite)
	}

	// Verify via SQL.
	row := store.DB().QueryRow(`SELECT is_bot, reviewer_id FROM pr_reviews WHERE id = 'alaudadevops/x#1/r1'`)
	var isBot int
	var revID string
	if err := row.Scan(&isBot, &revID); err != nil {
		t.Fatalf("scan bot review: %v", err)
	}
	if isBot != 1 || revID != "bot" {
		t.Fatalf("bot review row: is_bot=%d reviewer_id=%q, want 1 / bot", isBot, revID)
	}
	row = store.DB().QueryRow(`SELECT author_id FROM pull_requests WHERE id = 'alaudadevops/x#2'`)
	var authorID string
	if err := row.Scan(&authorID); err != nil {
		t.Fatalf("scan bot PR: %v", err)
	}
	if authorID != "bot" {
		t.Fatalf("bot-authored PR author_id=%q, want bot", authorID)
	}

	// first_human_review_at should be the human's review time (not the
	// earlier bot review).
	row = store.DB().QueryRow(`SELECT first_human_review_at FROM pull_requests WHERE id = 'alaudadevops/x#1'`)
	var firstHuman *time.Time
	if err := row.Scan(&firstHuman); err != nil {
		t.Fatalf("scan first_human_review_at: %v", err)
	}
	if firstHuman == nil || !firstHuman.Equal(humanReview) {
		t.Fatalf("first_human_review_at = %v, want %v", firstHuman, humanReview)
	}

	// Idempotency: re-running consolidates nothing further.
	res2, err := ConsolidateBots(ctx, store, bots)
	if err != nil {
		t.Fatalf("consolidate again: %v", err)
	}
	if res2.ReviewsMarkedBot != 0 || res2.ReviewsAuthorRew != 0 || res2.PRsAuthorRewrite != 0 {
		t.Fatalf("second run not idempotent: %+v", res2)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
