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

package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/api/handlers"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/api/middleware"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// NewRouter creates a new Gin router with all routes configured
func NewRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.CORSMiddleware(cfg))

	// Create handlers
	authHandler := handlers.NewAuthHandler()
	roadmapHandler := handlers.NewRoadmapHandler(cfg)
	projectsHandler := handlers.NewProjectsHandler(cfg)

	// Health check endpoint (no auth required)
	router.GET("/health", roadmapHandler.Health)

	// API routes
	api := router.Group("/api")
	{
		// Authentication routes (no auth middleware)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/status", authHandler.Status)
		}

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Projects routes (keep for future use)
			protected.GET("/projects", projectsHandler.ListProjects)

			// Roadmap routes
			protected.GET("/basic", roadmapHandler.GetBasicData)

			// Filtering APIs
			protected.GET("/milestones", roadmapHandler.GetMilestones)
			protected.GET("/epics", roadmapHandler.GetEpics)

			// Milestone routes
			protected.POST("/milestones", roadmapHandler.CreateMilestone)
			protected.PUT("/milestones/:id", roadmapHandler.UpdateMilestone)

			// Epic routes
			protected.POST("/epics", roadmapHandler.CreateEpic)
			protected.PUT("/epics/:id", roadmapHandler.UpdateEpic)
			protected.PUT("/epics/:id/milestone", roadmapHandler.UpdateEpicMilestone)

			// Component routes
			protected.GET("/components/:name/versions", roadmapHandler.GetComponentVersions)

			// User routes
			protected.GET("/users/assignable", roadmapHandler.GetAssignableUsers)
		}
	}

	// Serve static files from embedded frontend (production) or filesystem (development)
	staticPath := cfg.Server.StaticFilesPath
	if staticPath == "" {
		staticPath = "../frontend/build" // Default path for development
	}

	// Check if static files directory exists
	if _, err := os.Stat(staticPath); err == nil {
		// Serve static assets
		router.Static("/static", filepath.Join(staticPath, "static"))

		// Serve other static files (manifest.json, etc.)
		router.StaticFile("/manifest.json", filepath.Join(staticPath, "manifest.json"))
		router.StaticFile("/asset-manifest.json", filepath.Join(staticPath, "asset-manifest.json"))

		// Serve index.html for all non-API routes (SPA routing)
		router.NoRoute(func(c *gin.Context) {
			// Don't serve index.html for API routes
			if c.Request.URL.Path != "/" &&
				c.Request.URL.Path != "/health" &&
				!gin.IsDebugging() {
				// For API routes, return 404
				if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
					c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
					return
				}
			}
			c.File(filepath.Join(staticPath, "index.html"))
		})
	}

	return router
}
