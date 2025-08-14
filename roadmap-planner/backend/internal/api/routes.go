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
	roadmapHandler := handlers.NewRoadmapHandler()

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
			// Roadmap routes
			protected.GET("/roadmap", roadmapHandler.GetRoadmap)
			protected.GET("/pillars", roadmapHandler.GetPillars)

			// Milestone routes
			protected.POST("/milestones", roadmapHandler.CreateMilestone)
			protected.PUT("/milestones/:id", roadmapHandler.UpdateMilestone)

			// Epic routes
			protected.POST("/epics", roadmapHandler.CreateEpic)
			protected.PUT("/epics/:id/milestone", roadmapHandler.UpdateEpicMilestone)

			// Component routes
			protected.GET("/components/:name/versions", roadmapHandler.GetComponentVersions)

			// User routes
			protected.GET("/users/assignable", roadmapHandler.GetAssignableUsers)
		}
	}

	return router
}
