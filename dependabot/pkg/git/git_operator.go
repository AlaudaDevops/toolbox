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
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
)

// DefaultGitOperator implements GitOperator interface using system git commands
type DefaultGitOperator struct {
	// workingDir is the working directory for git operations
	workingDir string
}

// NewGitOperator creates a new Git operator
func NewGitOperator(workingDir string) *DefaultGitOperator {
	return &DefaultGitOperator{
		workingDir: workingDir,
	}
}

// GetRepo retrieves the repository information from the current working directory
// It returns the repository URL and parses it into a Repository struct
func (g *DefaultGitOperator) GetRepoURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get repo: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func (g *DefaultGitOperator) DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "not found") {
			// ignore if branch not found
			return nil
		}
		return fmt.Errorf("failed to delete branch %s: %w, output: %s", branchName, err, string(output))
	}
	return nil
}

// CreateBranch creates a new branch from the current branch
func (g *DefaultGitOperator) CreateBranch(newBranchName string) error {
	err := g.DeleteBranch(newBranchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", newBranchName, err)
	}

	cmd := exec.Command("git", "checkout", "-b", newBranchName)
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w, output: %s", newBranchName, err, string(output))
	}

	logrus.Debugf("Created and switched to branch: %s", newBranchName)
	return nil
}

// GetCommitID returns the commit ID of the current branch
func (g *DefaultGitOperator) GetCommitID() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = g.workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit ID: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CommitChanges commits all changes with the given message
func (g *DefaultGitOperator) CommitChanges(commitMessage string) error {
	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = g.workingDir

	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %w, output: %s", err, string(output))
	}

	// Commit changes
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = g.workingDir

	output, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w, output: %s", err, string(output))
	}

	return nil
}

// PushBranch pushes the current branch to remote origin
func (g *DefaultGitOperator) PushBranch() error {
	// Get current branch name
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Push current branch to origin
	cmd := exec.Command("git", "push", "-u", "origin", "-f", currentBranch)
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %w, output: %s", currentBranch, err, string(output))
	}

	logrus.Debugf("Pushed branch %s to origin", currentBranch)
	return nil
}

// GetCurrentBranch returns the current branch name
func (g *DefaultGitOperator) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = g.workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// HasChanges returns true if there are uncommitted changes
func (g *DefaultGitOperator) HasChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.workingDir

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	// If output is not empty, there are changes
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// Ensure DefaultGitOperator implements GitOperator interface
var _ updater.GitOperator = (*DefaultGitOperator)(nil)
