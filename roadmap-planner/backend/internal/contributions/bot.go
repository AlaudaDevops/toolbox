/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// BotSet is the W2 closed list of logins that should be folded into the
// synthetic `bot` member. Lookup is exact + case-folded; suffix or
// prefix patterns are NOT honored — per maintainer guidance, the
// allowlist is explicit and new bots are added to config.
//
// Use NewBotSet to build the set from a flat config slice. The zero
// value is a no-op (Contains always returns false), which is the
// upgrade path for installs that haven't curated the list yet.
type BotSet struct {
	set map[string]struct{}
}

// NewBotSet returns a set populated from the configured list. Empty
// input yields a usable, no-op set.
func NewBotSet(logins []string) BotSet {
	out := make(map[string]struct{}, len(logins))
	for _, l := range logins {
		k := normalizeBotLogin(l)
		if k == "" {
			continue
		}
		out[k] = struct{}{}
	}
	return BotSet{set: out}
}

// Contains reports whether the given raw login should be folded into
// the synthetic `bot` member. Lookup is case-folded so callers don't
// need to normalize before calling.
func (b BotSet) Contains(login string) bool {
	if len(b.set) == 0 {
		return false
	}
	_, ok := b.set[normalizeBotLogin(login)]
	return ok
}

// Size returns the number of distinct configured bot logins. Useful
// for startup telemetry.
func (b BotSet) Size() int { return len(b.set) }

func normalizeBotLogin(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ConsolidateBots is the one-shot startup pass that retroactively
// applies the W2 bot-consolidation rules to rows ingested before this
// feature shipped. It walks `pr_reviews` and `pull_requests`, sets
// `is_bot` / `first_human_review_at` columns, and rewrites bot
// `author_id` / `reviewer_id` to the synthetic `bot` member id.
//
// Behavior is idempotent — every step uses a deterministic predicate
// over the configured bot list, so re-running on already-consolidated
// data is a no-op. Callers should run this once on startup *after*
// the migration applies the new columns and *before* the aggregator
// rebuilds, so the network-density panel reads the fresh values.
//
// Returns a summary of how many rows changed per step, for the
// startup log.
type ConsolidateBotsResult struct {
	ReviewsMarkedBot int
	PRsAuthorRewrite int
	ReviewsAuthorRew int
	PRsFirstHumanSet int
}

func ConsolidateBots(ctx context.Context, store storage.Store, bots BotSet) (ConsolidateBotsResult, error) {
	var out ConsolidateBotsResult
	if bots.Size() == 0 {
		return out, nil
	}
	d, ok := store.(interface{ Dialect() storage.Dialect })
	if !ok {
		return out, fmt.Errorf("ConsolidateBots: store does not expose a Dialect")
	}
	dialect := d.Dialect()
	db := store.DB()

	// Build IN-clause args ONCE; reused across the four statements.
	logins := make([]any, 0, bots.Size())
	for k := range bots.set {
		logins = append(logins, k)
	}
	if len(logins) == 0 {
		return out, nil
	}
	placeholders := strings.Repeat("?, ", len(logins))
	placeholders = strings.TrimSuffix(placeholders, ", ")

	rebind := func(q string) string {
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

	// 1. Tag every existing review row by a configured bot login. This
	//    is the safe one to lead with: pure metadata flip, no FK touch.
	res, err := db.ExecContext(ctx,
		rebind(fmt.Sprintf(
			`UPDATE pr_reviews SET is_bot = 1 WHERE is_bot = 0 AND LOWER(reviewer_login) IN (%s)`,
			placeholders)),
		logins...)
	if err != nil {
		return out, fmt.Errorf("backfill is_bot on pr_reviews: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		out.ReviewsMarkedBot = int(n)
	}

	// 2. Reassign bot reviewer_id to the synthetic `bot` member id.
	//    Skip rows already pointing at `bot` (idempotency on rerun).
	res, err = db.ExecContext(ctx,
		rebind(fmt.Sprintf(
			`UPDATE pr_reviews SET reviewer_id = 'bot' WHERE (reviewer_id IS NULL OR reviewer_id <> 'bot') AND LOWER(reviewer_login) IN (%s)`,
			placeholders)),
		logins...)
	if err != nil {
		return out, fmt.Errorf("backfill reviewer_id on pr_reviews: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		out.ReviewsAuthorRew = int(n)
	}

	// 3. Reassign bot author_id on pull_requests.
	res, err = db.ExecContext(ctx,
		rebind(fmt.Sprintf(
			`UPDATE pull_requests SET author_id = 'bot' WHERE (author_id IS NULL OR author_id <> 'bot') AND LOWER(author_login) IN (%s)`,
			placeholders)),
		logins...)
	if err != nil {
		return out, fmt.Errorf("backfill author_id on pull_requests: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		out.PRsAuthorRewrite = int(n)
	}

	// 4. Populate first_human_review_at on pull_requests — `MIN(submitted_at)`
	//    over non-bot reviews. UPDATE-FROM-subquery flavor; runs once and
	//    is idempotent because subsequent passes either match the same
	//    minimum or NULL when no human reviews exist.
	res, err = db.ExecContext(ctx, rebind(`
		UPDATE pull_requests
		SET first_human_review_at = (
		    SELECT MIN(rv.submitted_at)
		    FROM pr_reviews rv
		    WHERE rv.pr_id = pull_requests.id AND rv.is_bot = 0
		)`))
	if err != nil {
		return out, fmt.Errorf("backfill first_human_review_at: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		out.PRsFirstHumanSet = int(n)
	}

	return out, nil
}
