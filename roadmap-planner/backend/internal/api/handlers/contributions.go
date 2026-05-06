/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package handlers

import (
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
	store   storage.Store
	service *contributions.Service
}

func NewContributionsHandler(store storage.Store, service *contributions.Service) *ContributionsHandler {
	return &ContributionsHandler{store: store, service: service}
}

// ListMembers — GET /api/contributions/members
func (h *ContributionsHandler) ListMembers(c *gin.Context) {
	members, err := h.store.ListMembers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// TeamOverview — GET /api/contributions/team?from=&to=&pillar=&component=
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
	c.JSON(http.StatusOK, gin.H{"members": out, "from": q.From, "to": q.To})
}

// MemberDetail — GET /api/contributions/members/:id?from=&to=
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
	out, err := h.service.MemberDetail(c.Request.Context(), id, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// CollectorStatus — GET /api/contributions/status
//
// Surfaces "when did the GitHub side last sync" so the UI can warn when
// data is stale.
func (h *ContributionsHandler) CollectorStatus(c *gin.Context) {
	out := gin.H{}
	for _, src := range []string{"jira", "github"} {
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
