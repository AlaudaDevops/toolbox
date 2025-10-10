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

package git

import "github.com/sirupsen/logrus"

// PullRequest represents a pull request across different platforms
type PullRequest struct {
	Number int       // Pull request number
	Title  string    // Pull request title
	State  string    // Pull request state (open, closed)
	Merged bool      // Whether the PR was merged (only meaningful when State is "closed")
	Author string    // Pull request author username
	Body   string    // Pull request body/description
	URL    string    // Web URL to the pull request
	Head   Reference // Head reference (source branch)
	Base   Reference // Base reference (target branch)
}

// Reference represents a git reference (branch/commit)
type Reference struct {
	Branch string // Branch name
	SHA    string // Commit SHA hash
}

// User represents a user across different platforms
type User struct {
	Login      string // User login name
	Permission string // User permission level (admin, write, read)
}

// Review represents a code review
type Review struct {
	User        User   // Reviewer information
	State       string // Review state: APPROVED, CHANGES_REQUESTED, COMMENTED
	Body        string // Review comment body
	SubmittedAt string // Review submission time (ISO 8601 format)
}

// Issue represents a repository issue
type Issue struct {
	Number    int    // Issue number
	Title     string // Issue title
	State     string // Issue state (open, closed)
	Author    string // Issue author username
	Body      string // Issue body/description
	URL       string // Web URL to the issue
	CreatedAt string // Issue creation time (ISO 8601 format)
}

// IssueSearchOptions defines filters for locating issues
type IssueSearchOptions struct {
	Title  string   // Exact or quoted title match
	Author string   // Issue author username
	State  string   // Desired issue state (open, closed, all)
	Labels []string // Labels the issue must contain
	Sort   string   // Sort field (created, updated)
	Order  string   // Sort order (asc, desc)
}

// Comment represents a comment on a pull request
type Comment struct {
	ID   int64  // Unique comment identifier
	User User   // Comment author information
	Body string // Comment text content
	URL  string // URL to the comment
}

// CheckRun represents a CI/CD check run
type CheckRun struct {
	Name         string // Check run name
	Status       string // Check run status: queued, in_progress, completed
	Conclusion   string // Check run conclusion: success, failure, neutral, cancelled, skipped, timed_out, action_required
	URL          string // URL to the check run details
	AppSlug      string // App slug that created this check (e.g., "github-actions")
	CheckSuiteID int64  // Check suite ID (for GitHub Actions)
}

// Commit represents a git commit
type Commit struct {
	SHA     string // Commit SHA hash
	Message string // Commit message
	Author  string // Commit author
}

//go:generate mockgen -package=git -destination=../../testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/git_client.go github.com/AlaudaDevops/toolbox/pr-cli/pkg/git GitClient

// GitClient defines the interface that all git platform clients must implement
type GitClient interface {
	// GetPR retrieves pull request information
	GetPR() (*PullRequest, error)
	// CheckPRStatus verifies if the PR is in the expected state
	CheckPRStatus(expectedState string) error

	// PostComment posts a comment to the pull request
	PostComment(message string) error
	// UpdatePRBody updates the pull request description/body
	UpdatePRBody(body string) error
	// GetComments retrieves all comments from the pull request
	GetComments() ([]Comment, error)
	// GetIssue retrieves an issue by number
	GetIssue(issueNumber int) (*Issue, error)
	// UpdateIssueBody updates the specified issue body/description
	UpdateIssueBody(issueNumber int, body string) error
	// FindIssue locates a single issue matching the provided options
	FindIssue(opts IssueSearchOptions) (*Issue, error)

	// GetReviews retrieves all reviews from the pull request
	GetReviews() ([]Review, error)
	// ApprovePR creates an approval review for the pull request
	ApprovePR(message string) error
	// DismissApprove dismisses the current user's approval review
	DismissApprove(message string) error

	// GetRequestedReviewers retrieves current requested reviewers for the pull request
	GetRequestedReviewers() ([]string, error)
	// AssignReviewers assigns users as reviewers to the pull request
	AssignReviewers(reviewers []string) error
	// RemoveReviewers removes users from pull request reviewers
	RemoveReviewers(reviewers []string) error

	// GetUserPermission gets the user's permission level for the repository
	GetUserPermission(username string) (string, error)
	// CheckUserPermissions validates if user has required permissions
	CheckUserPermissions(username string, requiredPerms []string) (bool, string, error)

	// GetLGTMVotes retrieves and validates LGTM votes using provided comments for optimization
	// If ignoreUserRemove is provided, it will ignore that user's latest /remove-lgtm comment
	GetLGTMVotes(comments []Comment, requiredPerms []string, debugMode bool, ignoreUserRemove ...string) (int, map[string]string, error)

	// MergePR merges the pull request with the specified method
	MergePR(method string) error
	// ClosePR closes the pull request without merging
	ClosePR() error
	// RebasePR updates the PR branch with the base branch
	RebasePR() error
	// GetAvailableMergeMethods retrieves the available merge methods for the pull request
	GetAvailableMergeMethods() ([]string, error)

	// CheckRunsStatus checks if all check runs are successful
	CheckRunsStatus() (bool, []CheckRun, error)

	// AddLabels adds labels to the pull request
	AddLabels(labels []string) error
	// RemoveLabels removes labels from the pull request
	RemoveLabels(labels []string) error
	// GetLabels retrieves current labels of the pull request
	GetLabels() ([]string, error)

	// CreateBranch creates a new branch from the specified base branch
	CreateBranch(branchName, baseBranch string) error
	// GetCommits retrieves commits from a pull request
	GetCommits() ([]Commit, error)
	// CreatePR creates a new pull request
	CreatePR(title, body, head, base string) (*PullRequest, error)
	// CherryPickCommit cherry-picks a commit to a branch
	CherryPickCommit(commitSHA, targetBranch string) error
	// BranchExists checks if a branch exists in the repository
	BranchExists(branchName string) (bool, error)
}

// Config holds the configuration for creating a Git client
type Config struct {
	Platform      string // "github" or "gitlab"
	Token         string
	CommentToken  string   // Token specifically for posting comments (optional, falls back to Token)
	BaseURL       string   // API base URL
	Owner         string   // Repository owner/namespace
	Repo          string   // Repository name
	PRNum         int      // Pull Request number
	PRSender      string   // PR author
	CommentSender string   // Comment author
	SelfCheckName string   // Name of the tool's own check run to exclude from status checks
	RobotAccounts []string // Robot/bot account usernames
}

//go:generate mockgen -package=git -destination=../../testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/client_factory.go github.com/AlaudaDevops/toolbox/pr-cli/pkg/git ClientFactory

// ClientFactory defines the interface for creating platform-specific clients
type ClientFactory interface {
	// CreateClient creates a new git client for the specified platform
	CreateClient(logger *logrus.Logger, config *Config) (GitClient, error)
}
