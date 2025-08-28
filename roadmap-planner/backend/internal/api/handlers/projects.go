/*
Copyright 2025 The AlaudaDevops Authors.

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
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProjectsHandler handles project-related requests
type ProjectsHandler struct {
	logger *zap.Logger
	config *config.Config
}

// NewProjectsHandler creates a new ProjectsHandler
func NewProjectsHandler(cfg *config.Config) *ProjectsHandler {
	return &ProjectsHandler{
		logger: logger.WithComponent("projects-handler"),
		config: cfg,
	}
}

// ListProjects returns list of projects with query option
func (h *ProjectsHandler) ListProjects(c *gin.Context) {
	jiraClient, ok := middleware.GetJiraClient(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Jira client not available",
		})
		return
	}
	projects, err := jiraClient.ListProjects(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to fetch project list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch project list",
		})
		return
	}
	c.JSON(http.StatusOK, projects)
}
