/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"net/http"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MetricsHandler handles metrics-related HTTP requests
type MetricsHandler struct {
	logger  *zap.Logger
	config  *config.Config
	service *metrics.Service
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(cfg *config.Config, svc *metrics.Service) *MetricsHandler {
	return &MetricsHandler{
		logger:  logger.WithComponent("metrics-handler"),
		config:  cfg,
		service: svc,
	}
}

// ListMetrics returns all available metrics
// GET /api/metrics
func (h *MetricsHandler) ListMetrics(c *gin.Context) {
	available := h.service.ListAvailableMetrics()

	c.JSON(http.StatusOK, gin.H{
		"metrics": available,
	})
}

// GetMetric returns a specific metric with optional filters
// GET /api/metrics/:name
func (h *MetricsHandler) GetMetric(c *gin.Context) {
	name := c.Param("name")

	// Parse query filters
	filters := models.MetricFilters{
		Components: c.QueryArray("component"),
		Pillars:    c.QueryArray("pillar"),
		Quarters:   c.QueryArray("quarter"),
	}

	// Parse time range
	timeRange := h.parseTimeRange(c)

	results, err := h.service.CalculateMetric(c.Request.Context(), name, filters, timeRange)
	if err != nil {
		h.logger.Error("Failed to calculate metric",
			zap.String("metric", name),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate metric: " + err.Error(),
		})
		return
	}

	// If single result, return directly; otherwise return array
	if len(results) == 1 {
		c.JSON(http.StatusOK, results[0])
	} else {
		c.JSON(http.StatusOK, gin.H{
			"name":    name,
			"results": results,
		})
	}
}

// GetSummary returns all metrics aggregated
// GET /api/metrics/summary
func (h *MetricsHandler) GetSummary(c *gin.Context) {
	filters := models.MetricFilters{
		Components: c.QueryArray("component"),
		Pillars:    c.QueryArray("pillar"),
		Quarters:   c.QueryArray("quarter"),
	}

	summary, err := h.service.CalculateAllMetrics(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to calculate summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate metrics summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetCollectorStatus returns the current status of the data collector
// GET /api/metrics/status
func (h *MetricsHandler) GetCollectorStatus(c *gin.Context) {
	collector := h.service.Collector()
	if collector == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "disabled",
			"message": "Metrics collector is not enabled",
		})
		return
	}

	lastCollected := collector.LastCollected()
	status := "healthy"
	if lastCollected.IsZero() {
		status = "initializing"
	} else if time.Since(lastCollected) > 10*time.Minute {
		status = "stale"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         status,
		"last_collected": lastCollected,
		"releases_count": collector.ReleaseCount(),
		"epics_count":    collector.EpicCount(),
		"issues_count":   collector.IssuesCount(),
	})
}

// parseTimeRange parses the from/to query parameters into a TimeRange
func (h *MetricsHandler) parseTimeRange(c *gin.Context) models.TimeRange {
	var timeRange models.TimeRange

	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			timeRange.Start = t
		} else if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			timeRange.Start = t
		}
	}

	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			timeRange.End = t
		} else if t, err := time.Parse("2006-01-02", toStr); err == nil {
			timeRange.End = t
		}
	}

	// Default to now if end is not specified
	if timeRange.End.IsZero() {
		timeRange.End = time.Now()
	}

	// Default to 1 year ago if start is not specified
	if timeRange.Start.IsZero() {
		timeRange.Start = timeRange.End.AddDate(-1, 0, 0)
	}

	return timeRange
}
