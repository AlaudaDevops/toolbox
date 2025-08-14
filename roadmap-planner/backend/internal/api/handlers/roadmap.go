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

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/api/middleware"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RoadmapHandler handles roadmap-related requests
type RoadmapHandler struct {
	logger *zap.Logger
}

// NewRoadmapHandler creates a new RoadmapHandler
func NewRoadmapHandler() *RoadmapHandler {
	return &RoadmapHandler{
		logger: logger.WithComponent("roadmap-handler"),
	}
}

// GetRoadmap returns the complete roadmap data
func (h *RoadmapHandler) GetRoadmap(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	// Fetch pillars with their milestones and epics
	pillars, err := jiraClient.GetPillars(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to fetch pillars", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch roadmap data",
		})
		return
	}

	// Generate quarters for the roadmap
	quarters := models.GenerateQuarters()

	roadmapData := models.RoadmapData{
		Pillars:  pillars,
		Quarters: quarters,
	}

	c.JSON(http.StatusOK, roadmapData)
}

// GetPillars returns all pillars
func (h *RoadmapHandler) GetPillars(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	pillars, err := jiraClient.GetPillars(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to fetch pillars", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch pillars",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pillars": pillars,
	})
}

// CreateMilestone creates a new milestone
func (h *RoadmapHandler) CreateMilestone(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	var req models.CreateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	h.logger.Sugar().Debugw("create milestone request payload", "payload", req)

	milestone, err := jiraClient.CreateMilestone(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create milestone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create milestone",
		})
		return
	}

	c.JSON(http.StatusCreated, milestone)
}

// UpdateMilestone updates a milestone's name and quarter
func (h *RoadmapHandler) UpdateMilestone(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	milestoneID := c.Param("id")
	if milestoneID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Milestone ID is required",
		})
		return
	}

	var req models.UpdateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	err := jiraClient.UpdateMilestone(c.Request.Context(), milestoneID, req)
	if err != nil {
		h.logger.Error("Failed to update milestone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update milestone",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Milestone updated successfully",
	})
}

// CreateEpic creates a new epic
func (h *RoadmapHandler) CreateEpic(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	var req models.CreateEpicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	epic, err := jiraClient.CreateEpic(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create epic", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create epic",
		})
		return
	}

	c.JSON(http.StatusCreated, epic)
}

// UpdateEpicMilestone moves an epic to a different milestone
func (h *RoadmapHandler) UpdateEpicMilestone(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	epicID := c.Param("id")
	if epicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Epic ID is required",
		})
		return
	}

	var req models.UpdateEpicMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	err := jiraClient.UpdateEpicMilestone(c.Request.Context(), epicID, req.MilestoneID)
	if err != nil {
		h.logger.Error("Failed to update epic milestone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update epic milestone",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Epic milestone updated successfully",
	})
}

// GetComponentVersions returns available versions for a component
func (h *RoadmapHandler) GetComponentVersions(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}

	component := c.Param("name")
	if component == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Component name is required",
		})
		return
	}

	versions, err := jiraClient.GetComponentVersions(c.Request.Context(), component)
	if err != nil {
		h.logger.Error("Failed to fetch component versions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch component versions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// GetAssignableUsers returns users that can be assigned to issues
func (h *RoadmapHandler) GetAssignableUsers(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}
	project, _ := middleware.GetProject(c)

	query, _ := c.GetQuery("query")
	issueKey, _ := c.GetQuery("issueKey")

	// Validate that issueKey is provided
	if issueKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "issueKey parameter is required",
		})
		return
	}

	h.logger.Info("Querying assignable users",
		zap.String("query", query),
		zap.String("project", project),
		zap.String("issueKey", issueKey))

	users, err := jiraClient.GetAssignableUsers(c.Request.Context(), query, project, issueKey)
	if err != nil {
		h.logger.Error("Failed to fetch assignable users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignable users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// Health returns the health status of the API
func (h *RoadmapHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "roadmap-planner",
	})
}
