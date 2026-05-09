/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/contributions"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ContributionsHandler exposes /api/contributions/* endpoints.
//
// The handler is intentionally thin — it parses query params, calls the
// contributions.Service, and returns JSON. All logic lives one layer
// down. This makes it easy to keep the API stable while we evolve the
// rollup pipeline (B2/B3).
type ContributionsHandler struct {
	store      storage.Store
	service    *contributions.Service
	aggregator *contributions.Aggregator
}

func NewContributionsHandler(store storage.Store, service *contributions.Service, aggregator *contributions.Aggregator) *ContributionsHandler {
	return &ContributionsHandler{store: store, service: service, aggregator: aggregator}
}

// ListMembers — GET /api/contributions/members?include_inactive=1
//
// Inactive Jira users (deactivated accounts) are hidden by default — the
// team-overview UI only shows people who can still receive work.
func (h *ContributionsHandler) ListMembers(c *gin.Context) {
	members, err := h.store.ListMembers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !truthy(c.Query("include_inactive")) {
		members = filterActive(members)
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// TeamOverview — GET /api/contributions/team?from=&to=&pillar=&component=&include_inactive=
//
// Inactive members are dropped from the rollup table by default for the
// same reason as ListMembers.
func (h *ContributionsHandler) TeamOverview(c *gin.Context) {
	q, err := h.parseQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.TeamOverview(c.Request.Context(), q)
	if err != nil {
		logger.Error("team overview failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !truthy(c.Query("include_inactive")) {
		members, mErr := h.store.ListMembers(c.Request.Context())
		if mErr == nil {
			active := map[string]bool{}
			for _, m := range members {
				if m.Active {
					active[m.ID] = true
				}
			}
			filtered := out[:0]
			for _, ms := range out {
				if active[ms.MemberID] {
					filtered = append(filtered, ms)
				}
			}
			out = filtered
		}
	}
	c.JSON(http.StatusOK, gin.H{"members": out, "from": q.From, "to": q.To})
}

// MemberDetail — GET /api/contributions/members/:id?from=&to=
//
// We do *not* hide inactive members here: the URL is a stable handle to
// a person and the operator may want to see history even after the
// account is gone.
func (h *ContributionsHandler) MemberDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "member id required"})
		return
	}
	q, err := h.parseQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.service.MemberDetail(c.Request.Context(), id, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Enrich with the directory entry so the profile page has the
	// identity fields (display name, email, github_login, …) without a
	// second round-trip.
	var info *storage.Member
	if members, mErr := h.store.ListMembers(c.Request.Context()); mErr == nil {
		for i := range members {
			if members[i].ID == id {
				info = &members[i]
				break
			}
		}
	}
	resp := gin.H{
		"member_id":                summary.MemberID,
		"week_totals":              summary.WeekTotals,
		"jira_issues_done":         summary.JiraIssuesDone,
		"jira_points_done":         summary.JiraPointsDone,
		"prs_merged":               summary.PRsMerged,
		"prs_reviewed":             summary.PRsReviewed,
		"review_latency_p50_hours": summary.ReviewLatencyP50,
	}
	if info != nil {
		resp["info"] = info
	}
	// Best-effort: components-touched + current sprint. We surface the
	// error in logs but do not fail the whole response — the profile
	// page is useful even without these extras.
	if extras, xErr := h.service.MemberExtras(c.Request.Context(), id, q); xErr == nil {
		resp["components_touched"] = extras.ComponentsTouched
		if extras.Sprint != nil {
			resp["sprint"] = extras.Sprint
		}
	} else {
		logger.Warn("member extras failed", zap.String("member_id", id), zap.Error(xErr))
	}
	c.JSON(http.StatusOK, resp)
}

// NetworkDensity — GET /api/contributions/network?from=&to=
//
// Aggregate review-health stats for the Team Overview "Review network
// density" panel: orphan rate, first-review p50/p90, cross-pillar
// review percentage.
func (h *ContributionsHandler) NetworkDensity(c *gin.Context) {
	q, err := h.parseQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.service.NetworkDensity(c.Request.Context(), q)
	if err != nil {
		logger.Error("network density failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// PillarThroughput — GET /api/contributions/pillars?from=&to=
//
// Returns weekly PR-merged + Jira-done counts grouped by pillar. Pillar
// attribution is per-PR (via repo→pillars) and per-issue (via Jira
// component → pillars), driven by the team_analytics.pillars config.
// PRs / issues that don't match any configured pillar fall under the
// synthetic "Unassigned" key. The `pillars` field in the response gives
// the configured display order so the frontend can render zero-stack
// pillars even when they had no activity.
func (h *ContributionsHandler) PillarThroughput(c *gin.Context) {
	q, err := h.parseQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.service.PillarThroughput(c.Request.Context(), q)
	if err != nil {
		logger.Error("pillar throughput failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	pm := h.service.PillarMap()
	c.JSON(http.StatusOK, gin.H{
		"buckets":    rows,
		"from":       q.From,
		"to":         q.To,
		"pillars":    pm.PublicConfig(),
		"order":      pm.Order(),
		"configured": pm.Configured(),
	})
}

// UpdateMember — PATCH /api/contributions/members/:id
//
// The Jira-→-GitHub identity link is intentionally manual: the user
// explicitly set `github_login` on a member to wire up PR / review
// counts. Auto-matching by email was dropped because the success rate
// depended on every engineer making their GitHub email public. After
// the upsert we kick off an aggregator rebuild so the dashboards
// recompute against the new linkage on the next read.
//
// Only the writable fields are accepted. Use *string to distinguish
// "not provided" from "explicitly emptied" so an operator can clear a
// stale github_login by sending {"github_login": ""}.
func (h *ContributionsHandler) UpdateMember(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "member id required"})
		return
	}
	var req struct {
		DisplayName    *string `json:"display_name"`
		GitHubLogin    *string `json:"github_login"`
		GitLabUsername *string `json:"gitlab_username"`
		PillarID       *string `json:"pillar_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	members, err := h.store.ListMembers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var existing *storage.Member
	for i := range members {
		if members[i].ID == id {
			existing = &members[i]
			break
		}
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}

	if req.DisplayName != nil {
		existing.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.GitHubLogin != nil {
		existing.GitHubLogin = strings.ToLower(strings.TrimSpace(*req.GitHubLogin))
	}
	if req.GitLabUsername != nil {
		existing.GitLabUsername = strings.ToLower(strings.TrimSpace(*req.GitLabUsername))
	}
	if req.PillarID != nil {
		existing.PillarID = strings.TrimSpace(*req.PillarID)
	}

	// Literal-overwrite — UpsertMember's COALESCE semantics (which
	// protect operator edits from being clobbered by the Jira sync)
	// would otherwise turn an explicit clear ("github_login": "") into
	// a no-op. SetMemberIdentity bypasses that and writes verbatim.
	if err := h.store.SetMemberIdentity(c.Request.Context(), id, storage.MemberIdentity{
		DisplayName:    existing.DisplayName,
		GitHubLogin:    existing.GitHubLogin,
		GitLabUsername: existing.GitLabUsername,
		PillarID:       existing.PillarID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Async rebuild — don't block the HTTP response on a 30-90s SQL
	// pass. The dashboards will see fresh numbers on the next refresh.
	if h.aggregator != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := h.aggregator.RebuildRecent(ctx, 0); err != nil {
				logger.Error("aggregator rebuild after member PATCH failed",
					zap.String("member_id", id), zap.Error(err))
			}
		}()
	}

	c.JSON(http.StatusOK, *existing)
}

// CollectorStatus — GET /api/contributions/status
//
// Surfaces "when did the GitHub side last sync" so the UI can warn when
// data is stale.
func (h *ContributionsHandler) CollectorStatus(c *gin.Context) {
	out := gin.H{}
	for _, src := range []string{"jira", "github", "gitlab"} {
		run, err := h.store.LatestCollectionRun(c.Request.Context(), src)
		if err != nil {
			out[src] = gin.H{"error": err.Error()}
			continue
		}
		if run == nil {
			out[src] = gin.H{"status": "never_run"}
			continue
		}
		out[src] = gin.H{
			"last_run_at":  run.CapturedAt,
			"duration_ms":  run.DurationMs,
			"record_count": run.RecordCount,
			"error":        run.Error,
		}
	}
	c.JSON(http.StatusOK, out)
}

// parseQuery normalises ?from=&to=&pillar=&component=&member=.
//
// Default window is the last 12 weeks (84 days), which matches the
// prototype and is the most common dashboard view.
func (h *ContributionsHandler) parseQuery(c *gin.Context) (storage.MemberWeekQuery, error) {
	now := time.Now().UTC()
	q := storage.MemberWeekQuery{
		From: contributions.MondayOf(now.AddDate(0, 0, -84)),
		To:   contributions.MondayOf(now.AddDate(0, 0, 7)),
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return q, err
		}
		q.From = t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return q, err
		}
		q.To = t
	}
	if v := c.Query("pillar"); v != "" {
		q.PillarIDs = splitCSV(v)
	}
	if v := c.Query("member"); v != "" {
		q.MemberIDs = splitCSV(v)
	}
	if v := c.Query("component"); v != "" {
		q.Component = v
	}
	return q, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// truthy is permissive: 1, true, yes, on (case-insensitive). Empty is false.
func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// filterActive returns only members whose Active flag is true.
func filterActive(in []storage.Member) []storage.Member {
	out := in[:0]
	for _, m := range in {
		if m.Active {
			out = append(out, m)
		}
	}
	return out
}
