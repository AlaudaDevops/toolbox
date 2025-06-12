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

// Package updater provides language-agnostic vulnerability package updating functionality.
//
// Architecture Design Principles:
//
//  1. Simplified Interface Design:
//     The LanguageUpdater interface is intentionally minimal with only two methods:
//     - UpdatePackages: Core functionality to update vulnerable packages
//     - GetLanguageType: Returns the language this updater handles
//
//  2. Language-Specific Internal Processing:
//     Each language implementation receives trivy vulnerability data and handles
//     all language-specific logic internally, including:
//     - Project structure validation (e.g., finding go.mod, requirements.txt)
//     - Package version normalization (e.g., Go's "v" prefix handling)
//     - Package manager command execution (e.g., "go get", "pip install")
//     - Post-update tasks (e.g., "go mod tidy")
//
//  3. Trivy Result Integration:
//     The design leverages trivy's standardized JSON output format. All languages
//     use the same trivy vulnerability structure, making the interface consistent
//     and reducing the need for language-specific data transformations at the
//     interface level.
//
// 4. Flexibility and Extensibility:
//   - Easy to add new languages by implementing the simple LanguageUpdater interface
//   - Language detection is handled separately and can be enhanced independently
//   - Each language updater can evolve its internal logic without affecting others
//   - Testing is simplified with mock implementations
//
// 5. Separation of Concerns:
//   - Interface focuses on the core "update packages" responsibility
//   - Language detection is handled by a separate LanguageDetector interface
//   - Project validation and package management details are encapsulated within
//     each language implementation
//
// This design provides better maintainability, testability, and extensibility
// compared to a more complex interface with many methods.
package updater

import (
	"github.com/alauda-devops/toolbox/dependabot/pkg/scanner"
)

// LanguageType represents different programming languages
type LanguageType string

const (
	// LanguageGo represents Go programming language
	LanguageGo LanguageType = "go"
	// LanguagePython represents Python programming language (for future implementation)
	LanguagePython LanguageType = "python"
	// LanguageNode represents Node.js/JavaScript (for future implementation)
	LanguageNode LanguageType = "node"
	// LanguageUnknown represents unknown programming language
	LanguageUnknown LanguageType = "unknown"
)

// LanguageUpdater defines the simplified interface for language-specific package updaters
// Each language implementation receives trivy vulnerabilities and handles the update process internally
type LanguageUpdater interface {
	// UpdatePackages is the core method that updates vulnerable packages for the specific language
	// It receives vulnerabilities from trivy and handles all language-specific logic internally
	UpdatePackages(vulnerabilities []scanner.Vulnerability) error

	// GetLanguageType returns the language type this updater handles
	GetLanguageType() LanguageType
}

// PackageUpdate represents a single package update
type PackageUpdate struct {
	scanner.Vulnerability `json:",inline" yaml:",inline"`

	// Success indicates whether this package update was successful
	Success bool
	// Error contains the error message if update failed
	Error string
}

// UpdateSummary represents the comprehensive result of an update operation
type UpdateSummary struct {
	// ProjectPath is the path to the project
	ProjectPath string
	// TotalPackages is the total number of packages that needed updates
	TotalPackages int
	// SuccessfulUpdates contains successfully updated packages
	SuccessfulUpdates []PackageUpdate
	// FailedUpdates contains packages that failed to update
	FailedUpdates []PackageUpdate
	// Summary provides a human-readable summary
	Summary string
	// Timestamp is when the update operation completed
	Timestamp string
}

// GitOperator defines the interface for Git operations
type GitOperator interface {
	// CreateBranch creates a new branch from the current branch
	CreateBranch(branchName string) error

	// CommitChanges commits all changes with the given message
	CommitChanges(commitMessage string) error

	// PushBranch pushes the current branch to remote origin
	PushBranch() error

	// GetCurrentBranch returns the current branch name
	GetCurrentBranch() (string, error)

	// HasChanges returns true if there are uncommitted changes
	HasChanges() (bool, error)
}

// PRRequest represents a pull request creation request
type PRRequest struct {
	// Title is the pull request title
	Title string
	// Body is the pull request body/description
	Body string
	// SourceBranch is the branch containing changes
	SourceBranch string
	// TargetBranch is the branch to merge into (usually main/master)
	TargetBranch string
	// Labels are labels to add to the pull request
	Labels []string
	// Assignees are users to assign to the pull request
	Assignees []string
}
