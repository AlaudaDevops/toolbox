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

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	logger *zap.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		logger: logger.WithComponent("auth-handler"),
	}
}

// Login handles user authentication with Jira
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.AuthRequest
	h.logger.Info("Received login request")

	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		h.logger.Error("Failed to bind login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	h.logger.Debug("Processing login request",
		zap.String("base_url", req.BaseURL),
		zap.String("username", req.Username))

	// Create Jira client to test credentials
	jiraClient, err := jira.NewClient(req.BaseURL, req.Username, req.Password, "DEVOPS")
	if err != nil {
		h.logger.Error("Failed to create Jira client", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initialize Jira client",
		})
		return
	}

	// Test the connection
	if err := jiraClient.TestConnection(c.Request.Context()); err != nil {
		h.logger.Error("Jira authentication failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid Jira credentials",
		})
		return
	}

	// In a real implementation, you would create a session or JWT token here
	// For simplicity, we'll just return success
	c.JSON(http.StatusOK, gin.H{
		"message": "Authentication successful",
		"user":    req.Username,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real implementation, you would invalidate the session or token here
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// Status returns the current authentication status
func (h *AuthHandler) Status(c *gin.Context) {
	// Check if user is authenticated by trying to get user from context
	username := c.GetHeader("X-Jira-Username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"authenticated": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user":          username,
	})
}
