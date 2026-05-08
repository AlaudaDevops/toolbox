/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// pageOneRepos / pageTwoRepos mock two pages of /orgs/AlaudaDevops/repos.
// Names chosen so a sorted assertion pins down exactly which repos
// survive the archived/fork filter.
var pageOneRepos = []orgRepo{
	{Name: "alpha", Archived: false, Fork: false},        // keep
	{Name: "beta-archived", Archived: true, Fork: false}, // drop (archived)
	{Name: "gamma-fork", Archived: false, Fork: true},    // drop (fork)
}
var pageTwoRepos = []orgRepo{
	{Name: "delta", Archived: false, Fork: false},      // keep
	{Name: "epsilon-both", Archived: true, Fork: true}, // drop (both flags)
}

// newOrgReposServer returns an httptest server that serves two pages of
// /orgs/{org}/repos and counts the number of API hits via *hits.
func newOrgReposServer(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if !strings.HasPrefix(r.URL.Path, "/orgs/") || !strings.HasSuffix(r.URL.Path, "/repos") {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "5000")

		var body []orgRepo
		switch page {
		case "1", "":
			// Pad to 100 entries so the client knows another page might exist.
			body = padTo100(pageOneRepos)
		case "2":
			body = pageTwoRepos
		default:
			body = []orgRepo{}
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// padTo100 returns a copy of in extended to 100 elements with synthetic
// "filler-N" repos that all get filtered out (we mark them archived).
// The wildcard expansion only stops paging once a short page comes back,
// so the first page must be full-length to force a second request.
func padTo100(in []orgRepo) []orgRepo {
	out := make([]orgRepo, 0, 100)
	out = append(out, in...)
	for i := len(out); i < 100; i++ {
		out = append(out, orgRepo{Name: "filler", Archived: true})
	}
	return out
}

// TestResolveRepos_WildcardFiltersArchivedAndForks walks the two-page
// fixture and asserts the resolver returns only active, non-fork repos
// in the expected order.
func TestResolveRepos_WildcardFiltersArchivedAndForks(t *testing.T) {
	var hits int32
	srv := newOrgReposServer(t, &hits)
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	s := &Syncer{
		client:        c,
		repos:         []RepoConfig{{Owner: "AlaudaDevops", Name: "*"}},
		WildcardTTL:   time.Hour,
		wildcardCache: newWildcardCache(),
		nowFn:         time.Now,
	}

	got, err := s.resolveRepos(context.Background())
	if err != nil {
		t.Fatalf("resolveRepos: %v", err)
	}
	names := repoNames(got)
	sort.Strings(names)
	want := []string{"AlaudaDevops/alpha", "AlaudaDevops/delta"}
	if !equalStringSlices(names, want) {
		t.Fatalf("resolved repos = %v, want %v", names, want)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("api hits = %d, want 2 (two pages)", got)
	}
}

// TestResolveRepos_WildcardCachedWithinTTL asserts the second resolve
// inside the TTL window does not hit the API, and a third resolve
// after the TTL expires triggers a refresh.
func TestResolveRepos_WildcardCachedWithinTTL(t *testing.T) {
	var hits int32
	srv := newOrgReposServer(t, &hits)
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	s := &Syncer{
		client:        c,
		repos:         []RepoConfig{{Owner: "AlaudaDevops", Name: "*"}},
		WildcardTTL:   time.Hour,
		wildcardCache: newWildcardCache(),
		nowFn:         func() time.Time { return now },
	}

	if _, err := s.resolveRepos(context.Background()); err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	firstHits := atomic.LoadInt32(&hits)

	// Second resolve inside the TTL window -> cache hit, no new requests.
	if _, err := s.resolveRepos(context.Background()); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != firstHits {
		t.Fatalf("after TTL-cached resolve: hits=%d, want %d (no new API calls)", got, firstHits)
	}

	// Advance the clock past the TTL -> next resolve should refresh.
	now = now.Add(2 * time.Hour)
	if _, err := s.resolveRepos(context.Background()); err != nil {
		t.Fatalf("third resolve: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got <= firstHits {
		t.Fatalf("after TTL-expiry resolve: hits=%d, want > %d (cache refresh)", got, firstHits)
	}
}

// TestResolveRepos_ExplicitEntryWinsAlongsideWildcard asserts that when
// an operator writes both `OWNER/*` and `OWNER/foo:my-component`, the
// explicit entry retains its component label even though `OWNER/*`
// would also include `foo`.
func TestResolveRepos_ExplicitEntryWinsAlongsideWildcard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "5000")
		page := r.URL.Query().Get("page")
		if page == "1" || page == "" {
			_ = json.NewEncoder(w).Encode([]orgRepo{
				{Name: "alpha"}, {Name: "beta"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode([]orgRepo{})
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "tok", srv.Client())
	s := &Syncer{
		client: c,
		repos: []RepoConfig{
			{Owner: "AlaudaDevops", Name: "alpha", Component: "my-component"},
			{Owner: "AlaudaDevops", Name: "*"},
		},
		WildcardTTL:   time.Hour,
		wildcardCache: newWildcardCache(),
		nowFn:         time.Now,
	}

	got, err := s.resolveRepos(context.Background())
	if err != nil {
		t.Fatalf("resolveRepos: %v", err)
	}

	// We expect 3 entries: explicit alpha (with label) + wildcard alpha
	// (no label) + wildcard beta (no label). Explicit entries are not
	// deduplicated against wildcard expansions.
	var explicitHit, wildcardHit bool
	for _, r := range got {
		if r.Owner == "AlaudaDevops" && r.Name == "alpha" {
			if r.Component == "my-component" {
				explicitHit = true
			} else {
				wildcardHit = true
			}
		}
	}
	if !explicitHit {
		t.Fatal("explicit AlaudaDevops/alpha:my-component was lost in resolution")
	}
	if !wildcardHit {
		t.Fatal("wildcard AlaudaDevops/alpha (no component) was lost in resolution")
	}
	if len(got) != 3 {
		t.Fatalf("resolved %d repos, want 3 (explicit alpha + wildcard alpha + wildcard beta); got=%v",
			len(got), repoNames(got))
	}
}

// TestParseWildcardOwner pins down the helper used by main.go's
// parseRepos to detect wildcard specs.
func TestParseWildcardOwner(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"AlaudaDevops/*", "AlaudaDevops"},
		{"AlaudaDevops/toolbox", ""},
		{"AlaudaDevops/", ""},
		{"/*", ""},
		{"AlaudaDevops/*/sub", ""},
	}
	for _, c := range cases {
		if got := parseWildcardOwner(c.in); got != c.want {
			t.Errorf("parseWildcardOwner(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func repoNames(repos []RepoConfig) []string {
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		out = append(out, r.FullName())
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
