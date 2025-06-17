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

// Package pipeline provides a comprehensive pipeline for dependency updates and PR creation
package pipeline

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/notice"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/pr"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/scanner"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
	"github.com/sirupsen/logrus"
)

// Pipeline orchestrates the dependency update and PR creation process
type Pipeline struct {
	// config holds pipeline configuration
	config *Config
}

// Config holds configuration for the pipeline
type Config struct {
	config.DependaBotConfig `json:",inline" yaml:",inline"`

	// ProjectPath is the path to the project
	ProjectPath string `json:"projectPath" yaml:"projectPath"`
}

// NewPipeline creates a new dependency update pipeline
func NewPipeline(config *Config) *Pipeline {
	return &Pipeline{
		config: config,
	}
}

// Run executes the dependency update and PR creation pipeline
func (p *Pipeline) Run() error {
	logrus.Infof("Starting dependency update pipeline for project: %s", p.config.ProjectPath)

	logrus.Info("Running Security Scanning...")

	scannerInstance, err := scanner.NewScanner(p.config.ProjectPath, p.config.Scanner)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	defer func() {
		if cleanupErr := scannerInstance.Cleanup(); cleanupErr != nil {
			logrus.Warnf("Warning: failed to cleanup scanner resources: %v", cleanupErr)
		}
	}()

	vulnerabilities, err := scannerInstance.Scan()
	if err != nil {
		return fmt.Errorf("failed to run security scan: %w", err)
	}

	logrus.Info("Processing scan results...")
	if len(vulnerabilities) == 0 {
		logrus.Info("No vulnerabilities found in scan results")
		return nil
	}

	logrus.Infof("Found %d vulnerabilities to update", len(vulnerabilities))

	logrus.Info("Updating vulnerable packages...")
	updater := updater.New(p.config.ProjectPath)
	updateSummary, err := updater.UpdatePackages(vulnerabilities)
	if err != nil {
		// Even if some updates failed, we might still have successful ones
		logrus.Warnf("Warning: Some updates failed: %v", err)
	}

	fixedVulns := updateSummary.FixedVulns()
	if len(fixedVulns) == 0 {
		logrus.Info("No packages were successfully updated")
		return fmt.Errorf("no packages were successfully updated")
	}

	logrus.Debugf("Successfully updated %d packages", len(fixedVulns))
	logrus.Debugf("PR Description:\n%s", pr.GeneratePRBody(updateSummary))
	if !p.config.PR.NeedCreatePR() {
		logrus.Info("Auto PR creation is disabled, skipping Git and PR operations")
		return nil
	}

	logrus.Info("Creating branch and committing changes...")
	branchName, err := p.commitChanges(updateSummary)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	logrus.Info("Creating Pull Request...")
	prCreator, err := pr.NewPRCreator(p.config.Git, p.config.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to initialize PR creator: %w", err)
	}

	prInfo, err := prCreator.CreatePR(&p.config.Repo, branchName, pr.PRCreateOption{
		Labels:        p.config.PR.Labels,
		Assignees:     p.config.PR.Assignees,
		UpdateSummary: updateSummary,
	})
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	logrus.Info("✅ Pipeline completed successfully!")
	logrus.Debugf("   - Updated %d packages", len(fixedVulns))
	logrus.Debugf("   - Branch: %s", branchName)
	logrus.Debugf("   - Target: %s", p.config.Repo.Branch)

	// Send notification if configured
	if notice.IsNotificationEnabled(p.config.Notice) {
		logrus.Info("Sending notification...")
		if err := p.sendNotification(p.config.Repo.URL, updateSummary, prInfo); err != nil {
			// Don't fail the entire pipeline if notification fails
			logrus.Warnf("Warning: Failed to send notification: %v", err)
		} else {
			logrus.Info("✅ Notification sent successfully")
		}
	}

	logrus.Info("✅ Pipeline completed successfully!")
	return nil
}

// sendNotification sends a notification about the vulnerability updates
func (p *Pipeline) sendNotification(repoURL string, updateSummary types.VulnFixResults, prInfo types.PRInfo) error {
	notifier, err := notice.NewNotifier(p.config.Notice)
	if err != nil {
		return fmt.Errorf("failed to create notifier: %w", err)
	}

	if notifier == nil {
		// No notifier configured
		return nil
	}

	return notifier.Notify(repoURL, updateSummary, prInfo)
}

func (p *Pipeline) commitChanges(updateSummary types.VulnFixResults) (newBranchName string, err error) {
	gitOperator := git.NewGitOperator(p.config.ProjectPath)
	hasChanges, err := gitOperator.HasChanges()
	if err != nil {
		return "", fmt.Errorf("failed to check Git changes: %w", err)
	}

	if !hasChanges {
		logrus.Info("No Git changes detected, skipping branch creation and PR")
		return "", nil
	}

	commitID, err := gitOperator.GetCommitID()
	if err != nil {
		return "", fmt.Errorf("failed to get commit ID: %w", err)
	}

	branchName := p.generateBranchName(commitID)
	if err := gitOperator.CreateBranch(branchName); err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	commitMessage := generateCommitMessage(updateSummary)
	if err := gitOperator.CommitChanges(commitMessage); err != nil {
		return "", fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := gitOperator.PushBranch(); err != nil {
		return "", fmt.Errorf("failed to push branch: %w", err)
	}

	return branchName, nil
}

// generateBranchName generates a unique branch name
func (p *Pipeline) generateBranchName(baseCommitID string) string {
	return fmt.Sprintf("dependabot/security-updates-%s", baseCommitID[0:7])
}
