/*
Copyright 2024 The AlaudaDevops Authors.

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

// ApplyGitHubLoginPrefills walks an operator-curated map (Jira id →
// GitHub login) and writes each entry onto the matching member iff
// that member's `github_login` is currently empty. Intended to run
// once at startup, after the first Jira sync has populated the
// member table — re-running is harmless (no-op for already-linked
// members), so wiring it after every sync would also be safe.
//
// The "only when empty" rule is the design: a manual edit through the
// team-overview drawer (PATCH /api/contributions/members/:id) is the
// operator's source of truth and must never be silently overwritten
// by a future restart. New mappings the operator wants applied either
// land via the drawer in-line, or via a config update + a fresh
// startup pass.
//
// Returns (applied, configured) — number of members actually updated
// vs. number of entries in the prefills map. Members listed in the
// config but not present in the DB (e.g. the Jira sync hasn't picked
// them up yet) are silently skipped; the operator can re-run by
// restarting the pod after the Jira sync completes.
//
// Errors writing a single entry are logged via the returned error
// (joined into a wrapped error) but the function does NOT abort:
// other prefills in the same batch still apply. That keeps a typo'd
// gh login from blocking a clean batch of 12 valid entries.
func ApplyGitHubLoginPrefills(ctx context.Context, store storage.Store, prefills map[string]string) (applied int, configured int, err error) {
	if len(prefills) == 0 {
		return 0, 0, nil
	}
	configured = len(prefills)
	members, mErr := store.ListMembers(ctx)
	if mErr != nil {
		return 0, configured, fmt.Errorf("prefills: list members: %w", mErr)
	}
	by := make(map[string]storage.Member, len(members))
	for _, m := range members {
		by[m.ID] = m
	}
	var failures []string
	for jiraID, ghLogin := range prefills {
		jid := strings.TrimSpace(jiraID)
		gh := strings.ToLower(strings.TrimSpace(ghLogin))
		if jid == "" || gh == "" {
			continue
		}
		m, ok := by[jid]
		if !ok {
			continue
		}
		if m.GitHubLogin != "" {
			continue
		}
		// SetMemberIdentity writes literally — round-trip the other
		// identity fields so we only flip github_login.
		ident := storage.MemberIdentity{
			DisplayName:    m.DisplayName,
			GitHubLogin:    gh,
			GitLabUsername: m.GitLabUsername,
			PillarID:       m.PillarID,
		}
		if sErr := store.SetMemberIdentity(ctx, m.ID, ident); sErr != nil {
			failures = append(failures, fmt.Sprintf("%s→%s: %v", jid, gh, sErr))
			continue
		}
		applied++
	}
	if len(failures) > 0 {
		return applied, configured, fmt.Errorf("prefills: %d entry write(s) failed: %s", len(failures), strings.Join(failures, "; "))
	}
	return applied, configured, nil
}

// ApplyGitLabUsernamePrefills mirrors ApplyGitHubLoginPrefills for the
// GitLab side. Walks an operator-curated map (Jira id → GitLab
// username) and writes each entry onto the matching member iff that
// member's `gitlab_username` is currently empty. Same one-shot-on-empty
// semantics — manual UI edits always win.
func ApplyGitLabUsernamePrefills(ctx context.Context, store storage.Store, prefills map[string]string) (applied int, configured int, err error) {
	if len(prefills) == 0 {
		return 0, 0, nil
	}
	configured = len(prefills)
	members, mErr := store.ListMembers(ctx)
	if mErr != nil {
		return 0, configured, fmt.Errorf("gitlab prefills: list members: %w", mErr)
	}
	by := make(map[string]storage.Member, len(members))
	for _, m := range members {
		by[m.ID] = m
	}
	var failures []string
	for jiraID, glUser := range prefills {
		jid := strings.TrimSpace(jiraID)
		gl := strings.ToLower(strings.TrimSpace(glUser))
		if jid == "" || gl == "" {
			continue
		}
		m, ok := by[jid]
		if !ok {
			continue
		}
		if m.GitLabUsername != "" {
			continue
		}
		ident := storage.MemberIdentity{
			DisplayName:    m.DisplayName,
			GitHubLogin:    m.GitHubLogin,
			GitLabUsername: gl,
			PillarID:       m.PillarID,
		}
		if sErr := store.SetMemberIdentity(ctx, m.ID, ident); sErr != nil {
			failures = append(failures, fmt.Sprintf("%s→%s: %v", jid, gl, sErr))
			continue
		}
		applied++
	}
	if len(failures) > 0 {
		return applied, configured, fmt.Errorf("gitlab prefills: %d entry write(s) failed: %s", len(failures), strings.Join(failures, "; "))
	}
	return applied, configured, nil
}
