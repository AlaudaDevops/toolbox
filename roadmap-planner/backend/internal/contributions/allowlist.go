/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"sort"
	"strings"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
)

// BotMemberID is the synthetic member every recognised bot login is
// reassigned to. The Jira sync auto-creates this row with empty logins;
// we keep it permanently in the allowlist so that W2's "merge bot
// contributions into one fictional Bot user" surfaces a single dashboard
// row instead of being filtered away.
const BotMemberID = "bot"

// Allowlist is the W1 "who counts" set — the operator-curated subset of
// Jira member ids whose contributions land in the rollup table and in
// every contributions-API response. An empty Allowlist is the sentinel
// for "no filter configured"; callers must respect Enabled() before
// applying it as a WHERE clause, otherwise an empty set would silently
// drop the whole dashboard.
//
// The set is derived rather than configured: the operator declares who
// the team is via the `github_login_prefills` and `gitlab_username_prefills`
// maps, and W1 reuses that declaration. The escape hatch is the
// `member_denylist` field — entries there override the prefills (a
// person can be configured as a contributor and still excluded).
//
// Allowlist values are Jira member ids (slugified email), lowercased,
// trimmed — the same key shape used in the prefill maps. The membership
// check is also case-folded so callers don't need to normalise IDs
// before lookup.
type Allowlist struct {
	set map[string]struct{}
}

// BuildAllowlist computes the effective allowlist from the team-analytics
// config block. The set is the union of:
//   - the keys of GitHubLoginPrefills
//   - the keys of GitLabUsernamePrefills
//   - the synthetic BotMemberID (W2 keeps bot rollups visible)
//
// minus everything in MemberDenylist.
//
// If both prefill maps are empty AND the denylist is empty, the returned
// Allowlist is empty — meaning "no filter configured, preserve today's
// behaviour" (the aggregator and handlers fall back to the pre-W1 path
// when the allowlist is empty). This gives operators a clean upgrade:
// adopting W1 is opt-in via the existing prefill curation.
func BuildAllowlist(cfg config.TeamAnalytics) Allowlist {
	prefilled := len(cfg.GitHubLoginPrefills) > 0 || len(cfg.GitLabUsernamePrefills) > 0
	if !prefilled && len(cfg.MemberDenylist) == 0 {
		return Allowlist{}
	}
	deny := make(map[string]struct{}, len(cfg.MemberDenylist))
	for _, id := range cfg.MemberDenylist {
		k := normalize(id)
		if k == "" {
			continue
		}
		deny[k] = struct{}{}
	}
	out := make(map[string]struct{}, len(cfg.GitHubLoginPrefills)+len(cfg.GitLabUsernamePrefills)+1)
	add := func(id string) {
		k := normalize(id)
		if k == "" {
			return
		}
		if _, denied := deny[k]; denied {
			return
		}
		out[k] = struct{}{}
	}
	for id := range cfg.GitHubLoginPrefills {
		add(id)
	}
	for id := range cfg.GitLabUsernamePrefills {
		add(id)
	}
	if prefilled {
		// Only inject `bot` when we're already filtering — otherwise an
		// installation with only a denylist would have its allowlist
		// shrink to `{bot}` and silently drop every human contributor.
		add(BotMemberID)
	}
	return Allowlist{set: out}
}

// Enabled reports whether the allowlist should be applied as a filter.
// An empty set means "no filter configured" — callers should preserve
// the pre-W1 behaviour rather than reject everything.
func (a Allowlist) Enabled() bool { return len(a.set) > 0 }

// Size returns the number of member ids in the set.
func (a Allowlist) Size() int { return len(a.set) }

// Contains reports whether the given member id is in the allowlist.
// Empty allowlist returns true for any id (matches the "no filter
// configured" semantics — see Enabled).
func (a Allowlist) Contains(memberID string) bool {
	if !a.Enabled() {
		return true
	}
	_, ok := a.set[normalize(memberID)]
	return ok
}

// IDs returns the allowlisted member ids in deterministic order.
// Returns nil when the allowlist is empty so callers can branch on it.
func (a Allowlist) IDs() []string {
	if !a.Enabled() {
		return nil
	}
	out := make([]string, 0, len(a.set))
	for id := range a.set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// normalize matches the resolution rules used by the Jira sync when it
// derives member ids — lowercase + trim — so a denylist entry of
// "ZhWang " correctly filters out the `zhwang` member.
func normalize(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
