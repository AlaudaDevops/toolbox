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

package middleware

import (
	"net/http"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// JiraClientKey is the key used to store the Jira client in the context
	JiraClientKey = "jira_client"
	// UserKey is the key used to store user information in the context
	UserKey = "user"
	// ProjectKey is the key use to store project information in the context
	ProjectKey = "project"
)

// AuthMiddleware creates a middleware that validates Jira authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we'll use session-based authentication
		// In a production environment, you might want to use JWT tokens or similar

		// Check if user is authenticated (has valid session)
		session := getSession(c)
		if session == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		logger.Info("Creating Jira client in middleware",
			zap.String("base_url", session.BaseURL),
			zap.String("username", session.Username))

		// Create Jira client with stored credentials
		jiraClient, err := jira.NewClient(
			session.BaseURL,
			session.Username,
			session.Password,
			session.Project,
		)
		if err != nil {
			logger.Error("Failed to create Jira client", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to initialize Jira client",
			})
			c.Abort()
			return
		}

		// Test the connection
		if err := jiraClient.TestConnection(c.Request.Context()); err != nil {
			logger.Error("Jira connection test failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Jira credentials",
			})
			c.Abort()
			return
		}

		// Store the Jira client in the context
		c.Set(JiraClientKey, jiraClient)
		c.Set(UserKey, session.Username)
		c.Set(ProjectKey, session.Project)

		c.Next()
	}
}

// SessionData represents user session data
type SessionData struct {
	Username string `json:"username"`
	BaseURL  string `json:"base_url"`
	Password string `json:"password"`
	Project  string `json:"project"`
}

// getSession retrieves session data from the request
// In a real implementation, this would use proper session management
func getSession(c *gin.Context) *SessionData {
	// For simplicity, we'll use headers for authentication
	// In production, use proper session management or JWT tokens

	username := c.GetHeader("X-Jira-Username")
	password := c.GetHeader("X-Jira-Password")
	baseURL := c.GetHeader("X-Jira-BaseURL")
	project := c.GetHeader("X-Jira-Project")

	if username == "" || password == "" || baseURL == "" {
		return nil
	}

	if project == "" {
		project = "DEVOPS" // Default project
	}

	return &SessionData{
		Username: username,
		Password: password,
		BaseURL:  baseURL,
		Project:  project,
	}
}

// GetJiraClient retrieves the Jira client from the context
func GetJiraClient(c *gin.Context) (*jira.Client, bool) {
	client, exists := c.Get(JiraClientKey)
	if !exists {
		return nil, false
	}

	jiraClient, ok := client.(*jira.Client)
	return jiraClient, ok
}

// GetUser retrieves the current user from the context
func GetUser(c *gin.Context) (string, bool) {
	user, exists := c.Get(UserKey)
	if !exists {
		return "", false
	}

	username, ok := user.(string)
	return username, ok
}

// GetProject retrieves the current project from the context
func GetProject(c *gin.Context) (string, bool) {
	project, exists := c.Get(ProjectKey)
	if !exists {
		return "", false
	}

	projectName, ok := project.(string)
	return projectName, ok
}



