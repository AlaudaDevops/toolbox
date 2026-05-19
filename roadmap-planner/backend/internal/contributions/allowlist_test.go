/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"reflect"
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
)

func TestBuildAllowlist(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.TeamAnalytics
		want []string // nil means Allowlist.Enabled() should be false
	}{
		{
			name: "empty config = disabled",
			cfg:  config.TeamAnalytics{},
			want: nil,
		},
		{
			name: "prefills only — bot is auto-injected",
			cfg: config.TeamAnalytics{
				GitHubLoginPrefills: map[string]string{
					"daniel":  "danielfbm",
					"jtcheng": "chengjingtao",
				},
				GitLabUsernamePrefills: map[string]string{
					"daniel": "daniel",
					"qliu":   "qingliu",
				},
			},
			want: []string{"bot", "daniel", "jtcheng", "qliu"},
		},
		{
			name: "denylist removes prefilled entries",
			cfg: config.TeamAnalytics{
				GitHubLoginPrefills: map[string]string{
					"daniel": "danielfbm",
					"zhwang": "zonghang",
				},
				MemberDenylist: []string{"zhwang"},
			},
			want: []string{"bot", "daniel"},
		},
		{
			name: "case is folded and whitespace trimmed",
			cfg: config.TeamAnalytics{
				GitHubLoginPrefills: map[string]string{
					"Daniel":   "danielfbm",
					"ZhWang  ": "zonghang",
				},
				MemberDenylist: []string{"  ZHWANG"},
			},
			want: []string{"bot", "daniel"},
		},
		{
			name: "denylist alone keeps allowlist disabled",
			cfg: config.TeamAnalytics{
				MemberDenylist: []string{"zhwang"},
			},
			// Denylist with no prefills configured: we can't derive a
			// human-meaningful allowlist, so we leave the filter off
			// rather than collapse to {bot}.
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildAllowlist(tc.cfg)
			if tc.want == nil {
				if got.Enabled() {
					t.Fatalf("expected disabled allowlist, got %v", got.IDs())
				}
				if got.Size() != 0 {
					t.Fatalf("expected size 0, got %d", got.Size())
				}
				// Disabled allowlist must accept any id (preserve old behaviour).
				if !got.Contains("anyone") {
					t.Fatalf("disabled allowlist must Contains(anyone)")
				}
				return
			}
			if !got.Enabled() {
				t.Fatalf("expected enabled allowlist for %v", tc.want)
			}
			if !reflect.DeepEqual(got.IDs(), tc.want) {
				t.Fatalf("IDs() = %v, want %v", got.IDs(), tc.want)
			}
			for _, id := range tc.want {
				if !got.Contains(id) {
					t.Fatalf("Contains(%q) = false, want true", id)
				}
			}
			if got.Contains("definitely-not-in-the-set") {
				t.Fatalf("Contains(unknown) = true, want false")
			}
		})
	}
}
