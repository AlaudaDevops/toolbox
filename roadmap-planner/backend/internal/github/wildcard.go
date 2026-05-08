/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultWildcardTTL is how long a resolved OWNER/* expansion stays
// cached before the next Syncer cycle re-fetches it. Operators rarely
// add or archive repos within a 24 h window, so a daily refresh keeps
// the API budget tiny while still picking up new repos in bounded time.
const DefaultWildcardTTL = 24 * time.Hour

// orgRepo is the slim subset of the /orgs/{org}/repos response we use
// to filter wildcard results. Field tags mirror GitHub's JSON.
type orgRepo struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Fork     bool   `json:"fork"`
}

// listOrgRepos pages through GET /orgs/{org}/repos and returns every
// non-archived, non-fork repo. Both filters are togglable from the
// caller — the Syncer threads its IncludeArchived / IncludeForks flags
// through here.
//
// We follow the GitHub convention of `per_page=100` and walk pages by
// number until a short page is returned. The Client's `do` helper is
// already paginating-friendly (see client.go ListPullRequests).
func (c *Client) listOrgRepos(ctx context.Context, org string, includeArchived, includeForks bool) ([]orgRepo, error) {
	out := make([]orgRepo, 0, 64)
	const perPage = 100
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("type", "all")
		q.Set("page", strconv.Itoa(page))

		path := fmt.Sprintf("/orgs/%s/repos?%s", url.PathEscape(org), q.Encode())
		var batch []orgRepo
		if err := c.do(ctx, "GET", path, nil, &batch); err != nil {
			return nil, fmt.Errorf("list org repos %s page %d: %w", org, page, err)
		}
		for _, r := range batch {
			if !includeArchived && r.Archived {
				continue
			}
			if !includeForks && r.Fork {
				continue
			}
			out = append(out, r)
		}
		if len(batch) < perPage {
			break
		}
	}
	return out, nil
}

// wildcardCacheEntry holds a resolved expansion plus its mint time.
type wildcardCacheEntry struct {
	repos    []RepoConfig
	resolved time.Time
}

// wildcardCache is a tiny per-Syncer cache, keyed by owner. We don't
// share across Syncer instances — there's at most one process-wide
// Syncer in production, and tests want fresh state per case.
type wildcardCache struct {
	mu      sync.Mutex
	entries map[string]wildcardCacheEntry
}

func newWildcardCache() *wildcardCache {
	return &wildcardCache{entries: make(map[string]wildcardCacheEntry)}
}

func (w *wildcardCache) get(owner string, ttl time.Duration, now time.Time) ([]RepoConfig, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e, ok := w.entries[owner]
	if !ok {
		return nil, false
	}
	if now.Sub(e.resolved) > ttl {
		return nil, false
	}
	// Return a copy so callers can't mutate the cache slice.
	out := make([]RepoConfig, len(e.repos))
	copy(out, e.repos)
	return out, true
}

func (w *wildcardCache) put(owner string, repos []RepoConfig, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	stored := make([]RepoConfig, len(repos))
	copy(stored, repos)
	w.entries[owner] = wildcardCacheEntry{repos: stored, resolved: now}
}

// isWildcardSpec reports whether a RepoConfig is an OWNER/* spec that
// needs to be expanded against the org-repos API. We only support
// owner-level wildcards for now — path globs (foo/bar-*) are left for
// a future iteration because GitHub's REST has no name-prefix filter.
func isWildcardSpec(r RepoConfig) bool {
	return r.Owner != "" && r.Name == "*"
}

// resolveRepos returns the effective list of repos to sync: explicit
// entries are kept verbatim (component labels and all), and OWNER/*
// entries are expanded via the GitHub org-repos endpoint. Explicit
// entries are NOT deduplicated against wildcard expansions — operators
// who write `OWNER/foo:my-component` alongside `OWNER/*` expect their
// label to win, and a duplicate row downstream is harmless because
// every PR is keyed by `owner/name#number` regardless of how many
// RepoConfigs name the same repo.
//
// Failures on a single wildcard owner do not abort the whole sync; we
// log via the caller's error path instead. (The Syncer.Sync loop is
// already tolerant of partial repo-level failures — same contract.)
func (s *Syncer) resolveRepos(ctx context.Context) ([]RepoConfig, error) {
	if len(s.repos) == 0 {
		return nil, nil
	}
	now := s.nowFn()
	out := make([]RepoConfig, 0, len(s.repos))
	for _, r := range s.repos {
		if !isWildcardSpec(r) {
			out = append(out, r)
			continue
		}
		expanded, err := s.expandWildcard(ctx, r.Owner, now)
		if err != nil {
			return out, fmt.Errorf("expand wildcard %s/*: %w", r.Owner, err)
		}
		out = append(out, expanded...)
	}
	return out, nil
}

// expandWildcard returns the cached (or freshly fetched) expansion of
// OWNER/*. Component labels default to empty for wildcard matches —
// callers who want a specific label add an explicit
// `OWNER/NAME:component` entry alongside the wildcard.
func (s *Syncer) expandWildcard(ctx context.Context, owner string, now time.Time) ([]RepoConfig, error) {
	if cached, ok := s.wildcardCache.get(owner, s.WildcardTTL, now); ok {
		return cached, nil
	}
	repos, err := s.client.listOrgRepos(ctx, owner, s.IncludeArchived, s.IncludeForks)
	if err != nil {
		return nil, err
	}
	expanded := make([]RepoConfig, 0, len(repos))
	for _, r := range repos {
		expanded = append(expanded, RepoConfig{Owner: owner, Name: r.Name})
	}
	s.wildcardCache.put(owner, expanded, now)
	return expanded, nil
}

// HasWildcardSpecs reports whether the syncer's configured repo list
// contains any OWNER/* entries. Mostly useful for log lines and tests.
func (s *Syncer) HasWildcardSpecs() bool {
	for _, r := range s.repos {
		if isWildcardSpec(r) {
			return true
		}
	}
	return false
}

// parseWildcardOwner extracts the owner from "OWNER/*", returning ""
// if the spec is not a wildcard. Exposed for parseRepos in main.go.
func parseWildcardOwner(spec string) string {
	owner, name, ok := strings.Cut(spec, "/")
	if !ok || owner == "" || name != "*" {
		return ""
	}
	return owner
}

// IsWildcardSpec is the exported analogue of isWildcardSpec, intended
// for callers (e.g. cmd/server) that have a raw `owner/name` string
// rather than a constructed RepoConfig.
func IsWildcardSpec(spec string) bool { return parseWildcardOwner(spec) != "" }
