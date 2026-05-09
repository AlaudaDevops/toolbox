/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package gitlab

import "testing"

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
		{"/lgtm cancel", "commented", true},          // not an approval, but a real comment
		{"LGTM", "commented", true},                  // bare text without slash counts as a comment
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
