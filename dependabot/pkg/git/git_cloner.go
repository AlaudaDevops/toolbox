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

// Package git provides Git operations for dependency updates
package git

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// GitCloner handles automatic repository cloning
type GitCloner struct {
	// repoURL is the repository URL to clone
	repoURL string
	// branch is the specific branch to clone
	branch string
	// tempDir is the temporary directory for cloned repository
	tempDir string
	// clonedPath is the full path to the cloned repository
	clonedPath string
}

// NewGitClonerWithBranch creates a new git cloner for specific branch
func NewGitCloner(repoURL, branch string) *GitCloner {
	return &GitCloner{
		repoURL: repoURL,
		branch:  branch,
	}
}

// CloneRepository clones the repository and returns the path to the cloned directory
func (g *GitCloner) CloneRepository() (string, error) {
	// Validate repository URL
	if err := g.validateRepoURL(); err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Create temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "dependabot-clone-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	g.tempDir = tempDir

	// Extract repository name from URL for directory naming
	repoName := g.extractRepoName()
	g.clonedPath = filepath.Join(tempDir, repoName)

	logrus.Debugf("Cloning repository: %s", g.repoURL)
	logrus.Debugf("Clone destination: %s", g.clonedPath)

	// Prepare git clone command
	args := []string{
		"clone",
		"--depth", "1",
	}

	// Add branch specification if provided
	if g.branch != "" {
		args = append(args, "--branch", g.branch)
		logrus.Debugf("Cloning specific branch: %s", g.branch)
	}

	args = append(args, g.repoURL, g.clonedPath)

	// Log the command being executed
	logrus.Debugf("Executing: git %s", strings.Join(args, " "))

	// Execute git clone command
	cmd := exec.Command("git", args...)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp directory on error
		g.Cleanup()
		return "", fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	// Print git output for user information
	if len(output) > 0 {
		logrus.Debugf("Git clone output:%s", string(output))
	}

	// Verify that the cloned directory exists
	if _, err := os.Stat(g.clonedPath); os.IsNotExist(err) {
		g.Cleanup()
		return "", fmt.Errorf("cloned repository directory was not created: %s", g.clonedPath)
	}

	logrus.Info("✅ Repository cloned successfully")
	logrus.Debugf("Cloned to: %s", g.clonedPath)

	return g.clonedPath, nil
}

// Cleanup removes the temporary directory and all its contents
func (g *GitCloner) Cleanup() error {
	if g.tempDir == "" {
		return nil
	}

	logrus.Debugf("Cleaning up cloned repository: %s", g.tempDir)

	err := os.RemoveAll(g.tempDir)
	if err != nil {
		return fmt.Errorf("failed to cleanup temp directory %s: %w", g.tempDir, err)
	}

	g.tempDir = ""
	g.clonedPath = ""
	return nil
}

// GetClonedPath returns the path to the cloned repository
func (g *GitCloner) GetClonedPath() string {
	return g.clonedPath
}

// GetTempDir returns the temporary directory path (useful for testing)
func (g *GitCloner) GetTempDir() string {
	return g.tempDir
}

// validateRepoURL validates the repository URL format
func (g *GitCloner) validateRepoURL() error {
	if g.repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Check if it's a valid URL or SSH format
	if strings.HasPrefix(g.repoURL, "http://") || strings.HasPrefix(g.repoURL, "https://") {
		// HTTP/HTTPS URL validation
		_, err := url.Parse(g.repoURL)
		if err != nil {
			return fmt.Errorf("invalid HTTP/HTTPS URL: %w", err)
		}
	} else if strings.Contains(g.repoURL, "@") && strings.Contains(g.repoURL, ":") {
		// SSH format validation (basic check)
		// Format: git@github.com:user/repo.git
		if !strings.Contains(g.repoURL, ".git") {
			return fmt.Errorf("SSH URL should end with .git")
		}
	} else {
		return fmt.Errorf("URL should be HTTP/HTTPS or SSH format")
	}

	return nil
}

// extractRepoName extracts repository name from URL for directory naming
func (g *GitCloner) extractRepoName() string {
	// Remove .git suffix if present
	repoURL := strings.TrimSuffix(g.repoURL, ".git")

	// Extract the last part of the path
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		repoName := parts[len(parts)-1]
		// Remove any special characters that might cause issues
		repoName = strings.ReplaceAll(repoName, ":", "_")
		return repoName
	}

	return "cloned-repo"
}

// CheckGitInstalled checks if git CLI is available
func CheckGitInstalled() error {
	cmd := exec.Command("git", "version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git CLI is not installed or not available in PATH: %wOutput: %s", err, string(output))
	}

	logrus.Debugf("Git CLI is available:%s", string(output))
	return nil
}
