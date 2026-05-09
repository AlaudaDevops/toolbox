/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
)

// TestApplyGitHubLoginPrefills covers the four behaviors that matter:
//   - empty github_login on the member is filled from config (apply)
//   - non-empty github_login (e.g. operator-edited via the drawer)
//     is left alone (preserve)
//   - members listed in the config but missing from the DB are
//     skipped without error (no panic, no insert)
//   - whitespace and casing are normalized before write
func TestApplyGitHubLoginPrefills(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "prefills.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// alice: empty github_login → should be filled.
	// bob:   already linked manually → must NOT be overwritten.
	// (carol intentionally not seeded → must be skipped, no error.)
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "alice", DisplayName: "Alice Tan", Active: true,
	}); err != nil {
		t.Fatalf("alice: %v", err)
	}
	if err := store.UpsertMember(ctx, storage.Member{
		ID: "bob", DisplayName: "Bob Lee", Active: true,
	}); err != nil {
		t.Fatalf("bob: %v", err)
	}
	// Lay down bob's existing link via the literal-overwrite path.
	if err := store.SetMemberIdentity(ctx, "bob", storage.MemberIdentity{
		DisplayName: "Bob Lee", GitHubLogin: "bob-on-github",
	}); err != nil {
		t.Fatalf("bob set: %v", err)
	}

	prefills := map[string]string{
		"alice": "  AliceTan  ", // exercises whitespace + case normalization
		"bob":   "bob-shouldnt-clobber",
		"carol": "carol-on-github",
		"":      "ignore-empty-key",
		"dan":   "",
	}
	applied, configured, err := ApplyGitHubLoginPrefills(ctx, store, prefills)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if configured != 5 {
		t.Fatalf("configured=%d, want 5", configured)
	}
	if applied != 1 {
		t.Fatalf("applied=%d, want 1 (only alice)", applied)
	}

	got, err := store.ListMembers(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	by := map[string]storage.Member{}
	for _, m := range got {
		by[m.ID] = m
	}
	if by["alice"].GitHubLogin != "alicetan" {
		t.Fatalf("alice.GitHubLogin = %q, want alicetan", by["alice"].GitHubLogin)
	}
	if by["bob"].GitHubLogin != "bob-on-github" {
		t.Fatalf("bob.GitHubLogin = %q, want bob-on-github (operator edit must win)", by["bob"].GitHubLogin)
	}
	if _, ok := by["carol"]; ok {
		t.Fatalf("carol should not exist — prefills must not insert members")
	}

	// Re-running is a no-op: alice now has a value, the rest were
	// already skipped.
	applied2, _, err := ApplyGitHubLoginPrefills(ctx, store, prefills)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if applied2 != 0 {
		t.Fatalf("applied2=%d, want 0 on re-run", applied2)
	}
}

// TestApplyGitHubLoginPrefills_EmptyMap is the no-op fast path —
// callers with no config shouldn't pay for a member-list query.
func TestApplyGitHubLoginPrefills_EmptyMap(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "prefills.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	applied, configured, err := ApplyGitHubLoginPrefills(ctx, store, nil)
	if err != nil {
		t.Fatalf("nil map: %v", err)
	}
	if applied != 0 || configured != 0 {
		t.Fatalf("applied=%d configured=%d, want both 0", applied, configured)
	}
}
