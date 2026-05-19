/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

func TestParseGroupSpec(t *testing.T) {
	cases := []struct {
		in           string
		wantOK       bool
		wantGroup    string
		wantSubgroup bool
		wantExact    string
	}{
		{"devops/**", true, "devops", true, ""},
		{"devops/*", true, "devops", false, ""},
		{"devops/edge", true, "", false, "devops/edge"},
		{"devops/sub/proj", true, "", false, "devops/sub/proj"},
		{"devops", false, "", false, ""},
		{"", false, "", false, ""},
		{"   ", false, "", false, ""},
		{"/devops/**", false, "", false, ""},
		{"/**", false, "", false, ""},
	}
	for _, tc := range cases {
		got, ok := ParseGroupSpec(tc.in)
		if ok != tc.wantOK {
			t.Errorf("ParseGroupSpec(%q) ok=%v, want %v", tc.in, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if got.Group != tc.wantGroup || got.IncludeSubgroups != tc.wantSubgroup || got.Exact != tc.wantExact {
			t.Errorf("ParseGroupSpec(%q) = %+v, want group=%q sub=%v exact=%q",
				tc.in, got, tc.wantGroup, tc.wantSubgroup, tc.wantExact)
		}
	}
}

func TestClassifyNote(t *testing.T) {
	cases := []struct {
		body      string
		wantState string
		wantOK    bool
	}{
		{"/lgtm", "approved", true},
		{"/lgtm\n", "approved", true},
		{"  /lgtm  ", "approved", true},
		{"prefix\n/lgtm\nsuffix", "approved", true}, // /lgtm anywhere on its own line
		{"/lgtm cancel", "commented", true},         // not an approval, but a real comment
		{"LGTM", "commented", true},                 // bare text without slash counts as a comment
		{"", "", false},
		{"   ", "", false},
		{"/retest", "", false},
		{"/hold", "", false},
		{"/cherry-pick release-1.0", "", false},
		{"/retest\n/hold", "", false},
		{"/retest\nlooks fine to me", "commented", true}, // procedural + comment → comment wins
		{"normal review comment", "commented", true},
	}
	for _, tc := range cases {
		state, ok := classifyNote(tc.body)
		if ok != tc.wantOK {
			t.Errorf("classifyNote(%q) ok=%v, want %v", tc.body, ok, tc.wantOK)
			continue
		}
		if state != tc.wantState {
			t.Errorf("classifyNote(%q) state=%q, want %q", tc.body, state, tc.wantState)
		}
	}
}

func TestDefaultLinkerExtractsEpic(t *testing.T) {
	l := DefaultLinker("DEVOPS")
	cases := []struct {
		mr   MergeRequest
		want string
	}{
		{mr(t, "DEVOPS-123-frobnicate", "branch carries it"), "DEVOPS-123"},
		{mr(t, "feature/frobnicate", "DEVOPS-456: title carries it"), "DEVOPS-456"},
		{mr(t, "feature/frobnicate", "[DEVOPS-789] title also fine"), "DEVOPS-789"},
		{mr(t, "release-1.0", "Bump deps"), ""},
		{mr(t, "devops-321-lower", "ok"), "DEVOPS-321"}, // case-insensitive match returns upper
	}
	for _, tc := range cases {
		got := l.Link(tc.mr)
		if got != tc.want {
			t.Errorf("Link(%+v) = %q, want %q", tc.mr, got, tc.want)
		}
	}
}

func mr(_ *testing.T, branch, title string) MergeRequest {
	return MergeRequest{SourceBranch: branch, Title: title}
}

// TestSyncerHydrateDiffDefault pins the W6 (2026-05-19) decision that
// NewSyncer ships with diff hydration on. Operators who hit GitLab
// rate-limit budget can still disable via `gitlab.hydrate_diff: false`;
// this test guards the default so it doesn't silently flip back.
func TestSyncerHydrateDiffDefault(t *testing.T) {
	s := NewSyncer(nil, nil, nil, nil, 0)
	if !s.HydrateDiff {
		t.Fatalf("NewSyncer.HydrateDiff = false, want true (W6 default)")
	}
}

func TestMergeRequestProjectPath(t *testing.T) {
	cases := []struct {
		name string
		mr   MergeRequest
		want string
	}{
		{
			name: "references.full takes priority",
			mr: MergeRequest{
				References: struct {
					Full string `json:"full"`
				}{Full: "group/sub/proj!42"},
				WebURL: "https://gitlab.example/x/y/-/merge_requests/42",
			},
			want: "group/sub/proj",
		},
		{
			name: "falls back to web_url",
			mr: MergeRequest{
				WebURL: "https://gitlab.example.com/group/sub/proj/-/merge_requests/7",
			},
			want: "group/sub/proj",
		},
		{
			name: "empty when nothing parses",
			mr:   MergeRequest{WebURL: "not a url"},
			want: "",
		},
	}
	for _, tc := range cases {
		if got := tc.mr.ProjectPath(); got != tc.want {
			t.Errorf("%s: ProjectPath() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestSyncPassBMemberSweep verifies Pass B (W1): with no group specs
// resolved, only allowlisted members produce MRs, and a re-run with the
// same data leaves the row count unchanged (dedup against existing).
//
// The fake GitLab serves:
//   - `/api/v4/merge_requests?author_username=daniel` → 2 MRs in
//     `container-platform/edge` and `alaudadevops/toolbox`.
//   - `/api/v4/merge_requests?author_username=stranger` → 1 MR (must
//     NOT be ingested — stranger is not allowlisted).
//   - Empty notes for every MR so we don't fight with note ordering.
func TestSyncPassBMemberSweep(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "gl.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "daniel", DisplayName: "Daniel", GitLabUsername: "daniel", Active: true,
	}); err != nil {
		t.Fatalf("upsert daniel: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "stranger", DisplayName: "Stranger", GitLabUsername: "stranger", Active: true,
	}); err != nil {
		t.Fatalf("upsert stranger: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	merged := now.Add(-24 * time.Hour)
	mkMR := func(iid int, fullRef string, author string) MergeRequest {
		webHost := "https://gitlab.example"
		projPath := fullRef[:strings.Index(fullRef, "!")]
		out := MergeRequest{
			IID: iid, ProjectID: int64(iid * 1000), Title: "fix", State: "merged",
			WebURL:    webHost + "/" + projPath + "/-/merge_requests/" + itoa(iid),
			CreatedAt: merged.Add(-time.Hour), MergedAt: &merged,
		}
		out.Author.Username = author
		out.References.Full = fullRef
		return out
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse(r.RequestURI)
		q := u.Query()
		var out []MergeRequest
		switch q.Get("author_username") {
		case "daniel":
			out = []MergeRequest{
				mkMR(1, "container-platform/edge!1", "daniel"),
				mkMR(2, "alaudadevops/toolbox!2", "daniel"),
			}
		case "stranger":
			out = []MergeRequest{mkMR(3, "ops/some-thing!3", "stranger")}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		// Per-project notes (always empty for this test). Path is
		// /api/v4/projects/<pid>/merge_requests/<iid>/notes.
		if strings.HasSuffix(r.URL.Path, "/notes") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := New(srv.URL, "fake-token", nil)
	syncer := NewSyncer(client, store, nil, DefaultLinker("DEVOPS"), 30)
	syncer.MemberInstanceSweep = true
	syncer.AllowedMemberIDs = map[string]struct{}{"daniel": {}}

	if err := syncer.Sync(ctx); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	gotIDs := listPRIDs(t, store)
	sort.Strings(gotIDs)
	wantIDs := []string{"alaudadevops/toolbox!2", "container-platform/edge!1"}
	if !equalSlices(gotIDs, wantIDs) {
		t.Fatalf("after first sync got %v, want %v", gotIDs, wantIDs)
	}

	// Re-run: same data must dedupe via the `id` PK; no growth.
	if err := syncer.Sync(ctx); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	got2 := listPRIDs(t, store)
	if len(got2) != len(gotIDs) {
		t.Fatalf("re-sync grew PR count from %d to %d; expected dedup", len(gotIDs), len(got2))
	}

	// And with the sweep disabled, no MRs land at all (Pass B is the
	// only ingest path in this fixture — no group specs are configured).
	syncer2 := NewSyncer(client, store, nil, DefaultLinker("DEVOPS"), 30)
	syncer2.MemberInstanceSweep = false
	syncer2.AllowedMemberIDs = map[string]struct{}{"daniel": {}}
	// Clear the store first.
	if _, err := store.DB().ExecContext(ctx, `DELETE FROM pull_requests`); err != nil {
		t.Fatalf("clear PRs: %v", err)
	}
	if err := syncer2.Sync(ctx); err != nil {
		t.Fatalf("sweep-off sync: %v", err)
	}
	if got := listPRIDs(t, store); len(got) != 0 {
		t.Fatalf("sweep disabled but got PRs: %v", got)
	}
}

// listPRIDs returns every pull_requests.id sorted ascending — the
// storage interface doesn't expose a list method, so we read straight
// from the DB handle for the test.
func listPRIDs(t *testing.T, store storage.Store) []string {
	t.Helper()
	rows, err := store.DB().Query(`SELECT id FROM pull_requests ORDER BY id`)
	if err != nil {
		t.Fatalf("query PR ids: %v", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	return out
}

func itoa(n int) string {
	// Tiny helper to avoid pulling strconv into the test fixtures.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{'0' + byte(n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func equalSlices(a, b []string) bool {
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
