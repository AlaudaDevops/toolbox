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

package github

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/google/go-github/v74/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Pre-compiled regular expressions for LGTM commands to avoid repeated compilation
var (
	lgtmRegexp       = regexp.MustCompile(`^/lgtm\b`)
	removeLgtmRegexp = regexp.MustCompile(`^/remove-lgtm\b`)
	lgtmCancelRegexp = regexp.MustCompile(`^/lgtm cancel\b`)
)

// Client implements the GitClient interface for GitHub
type Client struct {
	logger        *logrus.Logger
	client        *github.Client  // GitHub API client
	ctx           context.Context // Request context
	owner         string          // Repository owner
	repo          string          // Repository name
	prNum         int             // Pull request number
	prSender      string          // Pull request author
	commentSender string          // Comment author
	selfCheckName string          // Name of the tool's own check run to exclude
}

// Factory implements ClientFactory for GitHub
type Factory struct{}

// CreateClient creates a new GitHub client
func (f *Factory) CreateClient(logger *logrus.Logger, config *git.Config) (git.GitClient, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Set custom base URL if provided
	if config.BaseURL != "" && config.BaseURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(config.BaseURL, config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to set GitHub enterprise URL: %w", err)
		}
	}

	return &Client{
		logger:        logger,
		client:        client,
		ctx:           ctx,
		owner:         config.Owner,
		repo:          config.Repo,
		prNum:         config.PRNum,
		prSender:      config.PRSender,
		commentSender: config.CommentSender,
		selfCheckName: config.SelfCheckName,
	}, nil
}

// GetPR retrieves the pull request information
func (c *Client) GetPR() (*git.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(c.ctx, c.owner, c.repo, c.prNum)
	if err != nil {
		return nil, err
	}

	return &git.PullRequest{
		Number: pr.GetNumber(),
		Title:  pr.GetTitle(),
		State:  pr.GetState(),
		Merged: pr.GetMerged(),
		Author: pr.GetUser().GetLogin(),
		Head: git.Reference{
			Branch: pr.GetHead().GetRef(),
			SHA:    pr.GetHead().GetSHA(),
		},
		Base: git.Reference{
			Branch: pr.GetBase().GetRef(),
			SHA:    pr.GetBase().GetSHA(),
		},
	}, nil
}

// CheckPRStatus verifies if the PR is in the expected state
func (c *Client) CheckPRStatus(expectedState string) error {
	pr, err := c.GetPR()
	if err != nil {
		return fmt.Errorf("failed to get PR: %w", err)
	}

	if pr.State != expectedState {
		return fmt.Errorf("PR #%d is not %s (current state: %s)", c.prNum, expectedState, pr.State)
	}

	return nil
}

// PostComment posts a comment to the pull request
func (c *Client) PostComment(message string) error {
	comment := &github.IssueComment{
		Body: github.Ptr(message),
	}

	_, _, err := c.client.Issues.CreateComment(c.ctx, c.owner, c.repo, c.prNum, comment)
	return err
}

// GetComments retrieves all comments from the pull request with pagination
func (c *Client) GetComments() ([]git.Comment, error) {
	var allComments []*github.IssueComment

	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100}, // GitHub max per page
	}

	for {
		comments, resp, err := c.client.Issues.ListComments(c.ctx, c.owner, c.repo, c.prNum, opts)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	result := make([]git.Comment, len(allComments))
	for i, comment := range allComments {
		result[i] = git.Comment{
			User: git.User{
				Login: comment.GetUser().GetLogin(),
			},
			Body: comment.GetBody(),
			URL:  comment.GetHTMLURL(),
		}
	}

	return result, nil
}

// GetReviews retrieves all reviews from the pull request with pagination
func (c *Client) GetReviews() ([]git.Review, error) {
	var allReviews []*github.PullRequestReview

	opts := &github.ListOptions{PerPage: 100} // GitHub max per page

	for {
		reviews, resp, err := c.client.PullRequests.ListReviews(c.ctx, c.owner, c.repo, c.prNum, opts)
		if err != nil {
			return nil, err
		}

		allReviews = append(allReviews, reviews...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	result := make([]git.Review, len(allReviews))
	for i, review := range allReviews {
		submittedAt := ""
		if review.GetSubmittedAt().IsZero() == false {
			submittedAt = review.GetSubmittedAt().Format("2006-01-02T15:04:05Z07:00") // ISO 8601 format
		}

		result[i] = git.Review{
			User: git.User{
				Login: review.GetUser().GetLogin(),
			},
			State:       review.GetState(),
			Body:        review.GetBody(),
			SubmittedAt: submittedAt,
		}
	}

	return result, nil
}

// GetUserPermission gets the user's permission level for the repository
func (c *Client) GetUserPermission(username string) (string, error) {
	perm, _, err := c.client.Repositories.GetPermissionLevel(c.ctx, c.owner, c.repo, username)
	if err != nil {
		return "", err
	}

	return perm.GetPermission(), nil
}

// CheckUserPermissions validates if user has required permissions
func (c *Client) CheckUserPermissions(username string, requiredPerms []string) (bool, string, error) {
	perm, err := c.GetUserPermission(username)
	if err != nil {
		return false, "", err
	}

	if slices.Contains(requiredPerms, perm) {
		return true, perm, nil
	}

	return false, perm, nil
}

// GetRequestedReviewers retrieves current requested reviewers for the PR
func (c *Client) GetRequestedReviewers() ([]string, error) {
	pr, _, err := c.client.PullRequests.Get(c.ctx, c.owner, c.repo, c.prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var reviewers []string
	for _, reviewer := range pr.RequestedReviewers {
		reviewers = append(reviewers, reviewer.GetLogin())
	}

	return reviewers, nil
}

// AssignReviewers assigns users as reviewers to the PR (one by one to ensure all are assigned)
func (c *Client) AssignReviewers(reviewers []string) error {
	// Clean usernames (remove @ prefix if present)
	cleanReviewers := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		cleanReviewers[i] = strings.TrimPrefix(reviewer, "@")
	}

	// Check if PR sender is trying to assign themselves
	if slices.Contains(cleanReviewers, c.prSender) {
		return fmt.Errorf("PR author cannot assign themselves as reviewer")
	}

	// Debug: Log what we're about to send
	c.logger.Debugf("Requesting reviewers one by one: %v\n", cleanReviewers)

	var assignedReviewers []string
	var failedReviewers []string

	// GitHub API has issues when assigning multiple reviewers at once,
	// so we use individual assignment as a fallback solution
	for _, reviewer := range cleanReviewers {
		c.logger.Debugf("Assigning individual reviewer: %s\n", reviewer)

		reviewerRequest := github.ReviewersRequest{
			Reviewers: []string{reviewer},
		}

		response, _, err := c.client.PullRequests.RequestReviewers(c.ctx, c.owner, c.repo, c.prNum, reviewerRequest)
		if err != nil {
			c.logger.Debugf("Failed to assign reviewer %s: %v\n", reviewer, err)
			failedReviewers = append(failedReviewers, reviewer)
			continue
		}

		// Check if the reviewer was successfully assigned
		if response != nil && len(response.RequestedReviewers) > 0 {
			for _, assignedReviewer := range response.RequestedReviewers {
				if assignedReviewer.GetLogin() == reviewer {
					assignedReviewers = append(assignedReviewers, reviewer)
					c.logger.Debugf("Successfully assigned reviewer: %s\n", reviewer)
					break
				}
			}
		} else {
			// If no response or empty, still consider it potentially successful
			// (GitHub API sometimes doesn't return the full list)
			assignedReviewers = append(assignedReviewers, reviewer)
			c.logger.Debugf("Reviewer assignment request sent for: %s\n", reviewer)
		}
	}

	c.logger.Debugf("Assignment summary - Assigned: %v, Failed: %v\n", assignedReviewers, failedReviewers)

	// If some assignments failed, return an error with details
	if len(failedReviewers) > 0 {
		return fmt.Errorf("failed to assign some reviewers: %v (successfully assigned: %v)", failedReviewers, assignedReviewers)
	}

	return nil
}

// RemoveReviewers removes users from PR reviewers
func (c *Client) RemoveReviewers(reviewers []string) error {
	// Clean usernames (remove @ prefix if present)
	cleanReviewers := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		cleanReviewers[i] = strings.TrimPrefix(reviewer, "@")
	}

	reviewerRequest := github.ReviewersRequest{
		Reviewers: cleanReviewers,
	}

	_, err := c.client.PullRequests.RemoveReviewers(c.ctx, c.owner, c.repo, c.prNum, reviewerRequest)
	return err
}

// GetLGTMVotes retrieves and validates LGTM votes from comments and reviews
func (c *Client) GetLGTMVotes(requiredPerms []string, debugMode bool) (int, map[string]string, error) {
	lgtmUsers := make(map[string]string)
	userLatestReviews := make(map[string]*git.Review) // Track latest review for each user

	// Get reviews for approvals
	reviews, err := c.GetReviews()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	// Find the latest review for each user
	for i := range reviews {
		review := &reviews[i] // Get address of slice element directly
		user := review.User.Login
		if user == c.prSender { // Skip self-approvals
			continue
		}

		// If this is the first review from this user, or if it's newer than existing one
		if existingReview, exists := userLatestReviews[user]; !exists || review.SubmittedAt > existingReview.SubmittedAt {
			userLatestReviews[user] = review // Assign pointer directly
		}
	}

	// Process latest reviews to determine current approval status
	for user, latestReview := range userLatestReviews {
		if latestReview.State == "APPROVED" {
			lgtmUsers[user] = ""
			c.logger.Debugf("Found APPROVED from user: %s (latest review)", user)
		}
		// Note: We don't add users with CHANGES_REQUESTED or COMMENTED states
	}

	// Get comments for /lgtm commands
	comments, err := c.GetComments()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// Process LGTM commands from comments
	for _, comment := range comments {
		user := comment.User.Login
		body := strings.TrimSpace(comment.Body)
		// c.logger.Debugf("Processing comment from user: %s, body: %s", user, body)

		switch {
		case removeLgtmRegexp.MatchString(body):
			// Only remove if user hasn't approved via latest review
			if latestReview, exists := userLatestReviews[user]; !exists || latestReview.State != "APPROVED" {
				delete(lgtmUsers, user)
				c.logger.Debugf("Found `/remove-lgtm` from user: %s", user)
			} else {
				c.logger.Debugf("Ignoring `/remove-lgtm` from user: %s (has latest APPROVED review)", user)
			}
		case lgtmCancelRegexp.MatchString(body):
			// Only remove if user hasn't approved via latest review
			if latestReview, exists := userLatestReviews[user]; !exists || latestReview.State != "APPROVED" {
				delete(lgtmUsers, user)
				c.logger.Debugf("Found `/lgtm cancel` from user: %s", user)
			} else {
				c.logger.Debugf("Ignoring `/lgtm cancel` from user: %s (has latest APPROVED review)", user)
			}
		case lgtmRegexp.MatchString(body):
			// This regex should be placed at the end because `/lgtm cancel` needs to be matched first
			if user == c.prSender {
				if !debugMode {
					c.logger.Debugf("Skipping LGTM from PR author %s (not allowed)", user)
					continue
				}
				c.logger.Debugf("Debug mode: allowing PR author %s to give LGTM to their own PR", user)
			}
			// Only add if user hasn't already approved via latest review
			if latestReview, exists := userLatestReviews[user]; !exists || latestReview.State != "APPROVED" {
				lgtmUsers[user] = ""
				c.logger.Debugf("Found /lgtm from user: %s", user)
			} else {
				c.logger.Debugf("Ignoring /lgtm from user: %s (already has latest APPROVED review)", user)
			}
		default:
			continue
		}
	}
	c.logger.Debugf("Collected LGTM users: %v", lgtmUsers)

	// Validate permissions for each user
	validVotes := 0
	for user := range lgtmUsers {
		hasPermission, perm, err := c.CheckUserPermissions(user, requiredPerms)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to check permissions for %s: %w", user, err)
		}
		lgtmUsers[user] = perm
		if hasPermission {
			validVotes++
		}
	}

	return validVotes, lgtmUsers, nil
}

// ApprovePR creates a review approval for the PR
func (c *Client) ApprovePR(message string) error {
	review := &github.PullRequestReviewRequest{
		Event: github.Ptr("APPROVE"),
		Body:  github.Ptr(message),
	}

	_, _, err := c.client.PullRequests.CreateReview(c.ctx, c.owner, c.repo, c.prNum, review)
	return err
}

// DismissApprove dismisses the current user's approval review
func (c *Client) DismissApprove(message string) error {
	// Get all reviews to find the current user's approval
	reviews, _, err := c.client.PullRequests.ListReviews(c.ctx, c.owner, c.repo, c.prNum, nil)
	if err != nil {
		return fmt.Errorf("failed to get reviews: %w", err)
	}

	// Find the current user's latest APPROVED review
	var latestApprovalID int64 = 0
	currentUser := c.commentSender // The user who is dismissing their own review

	for _, review := range reviews {
		if review.GetUser().GetLogin() == currentUser && review.GetState() == "APPROVED" {
			if review.GetID() > latestApprovalID {
				latestApprovalID = review.GetID()
			}
		}
	}

	if latestApprovalID == 0 {
		return fmt.Errorf("no approval review found for user %s to dismiss", currentUser)
	}

	// Dismiss the review
	dismissRequest := &github.PullRequestReviewDismissalRequest{
		Message: github.Ptr(message),
	}

	_, _, err = c.client.PullRequests.DismissReview(c.ctx, c.owner, c.repo, c.prNum, latestApprovalID, dismissRequest)
	return err
}

// MergePR merges the pull request with the specified method
func (c *Client) MergePR(method string) error {
	options := &github.PullRequestOptions{
		MergeMethod: method,
	}

	_, _, err := c.client.PullRequests.Merge(c.ctx, c.owner, c.repo, c.prNum, "", options)
	return err
}

// RebasePR updates the PR branch with the base branch
func (c *Client) RebasePR() error {
	_, _, err := c.client.PullRequests.UpdateBranch(c.ctx, c.owner, c.repo, c.prNum, nil)
	return err
}

// CheckRunsStatus checks if all check runs are successful with pagination
func (c *Client) CheckRunsStatus() (bool, []git.CheckRun, error) {
	pr, err := c.GetPR()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var allCheckRuns []*github.CheckRun

	opts := &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100}, // GitHub max per page
	}

	for {
		checkRuns, resp, err := c.client.Checks.ListCheckRunsForRef(c.ctx, c.owner, c.repo, pr.Head.SHA, opts)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get check runs: %w", err)
		}

		allCheckRuns = append(allCheckRuns, checkRuns.CheckRuns...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Debugf("Check run: selfCheckName: %q", c.selfCheckName)

	var failedChecks []git.CheckRun
	for _, check := range allCheckRuns {
		c.logger.Debugf("Check run: %q, Status: %q, Conclusion: %q, URL: %q",
			check.GetName(), check.GetStatus(), check.GetConclusion(), check.GetHTMLURL())
		if check.GetStatus() == "completed" &&
			check.GetConclusion() != "success" &&
			check.GetConclusion() != "skipped" {
			failedChecks = append(failedChecks, git.CheckRun{
				Name:       check.GetName(),
				Status:     check.GetStatus(),
				Conclusion: check.GetConclusion(),
				URL:        check.GetHTMLURL(),
			})
		} else if check.GetStatus() != "completed" &&
			(c.selfCheckName == "" || !strings.HasSuffix(strings.TrimSpace(check.GetName()), "/ "+c.selfCheckName)) {
			failedChecks = append(failedChecks, git.CheckRun{
				Name:   check.GetName(),
				Status: check.GetStatus(),
				URL:    check.GetHTMLURL(),
			})
		}
	}

	return len(failedChecks) == 0, failedChecks, nil
}

// AddLabels adds labels to the pull request
func (c *Client) AddLabels(labels []string) error {
	if len(labels) == 0 {
		return fmt.Errorf("no labels specified")
	}

	c.logger.Debugf("Adding labels to PR #%d: %v", c.prNum, labels)

	// Get current labels first
	currentLabels, err := c.GetLabels()
	if err != nil {
		return fmt.Errorf("failed to get current labels: %w", err)
	}

	// Create combined label list (current + new)
	labelSet := make(map[string]bool)
	for _, label := range currentLabels {
		labelSet[label] = true
	}
	for _, label := range labels {
		labelSet[label] = true
	}

	// Convert back to slice
	var allLabels []string
	for label := range labelSet {
		allLabels = append(allLabels, label)
	}

	// Update labels on the issue (PRs are issues in GitHub API)
	_, _, err = c.client.Issues.ReplaceLabelsForIssue(c.ctx, c.owner, c.repo, c.prNum, allLabels)
	return err
}

// RemoveLabels removes labels from the pull request
func (c *Client) RemoveLabels(labels []string) error {
	if len(labels) == 0 {
		return fmt.Errorf("no labels specified")
	}

	c.logger.Debugf("Removing labels from PR #%d: %v", c.prNum, labels)

	// Get current labels first
	currentLabels, err := c.GetLabels()
	if err != nil {
		return fmt.Errorf("failed to get current labels: %w", err)
	}

	// Create set of labels to remove for efficient lookup
	removeSet := make(map[string]bool)
	for _, label := range labels {
		removeSet[label] = true
	}

	// Filter out labels to be removed
	var remainingLabels []string
	for _, label := range currentLabels {
		if !removeSet[label] {
			remainingLabels = append(remainingLabels, label)
		}
	}

	// Update labels on the issue (PRs are issues in GitHub API)
	_, _, err = c.client.Issues.ReplaceLabelsForIssue(c.ctx, c.owner, c.repo, c.prNum, remainingLabels)
	return err
}

// GetLabels retrieves current labels of the pull request
func (c *Client) GetLabels() ([]string, error) {
	issue, _, err := c.client.Issues.Get(c.ctx, c.owner, c.repo, c.prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	var labels []string
	for _, label := range issue.Labels {
		labels = append(labels, label.GetName())
	}

	return labels, nil
}

// CreateBranch creates a new branch from the specified base branch
func (c *Client) CreateBranch(branchName, baseBranch string) error {
	c.logger.Debugf("Creating branch %s from %s", branchName, baseBranch)

	// Get the SHA of the base branch
	baseRef, _, err := c.client.Git.GetRef(c.ctx, c.owner, c.repo, "heads/"+baseBranch)
	if err != nil {
		return fmt.Errorf("failed to get base branch %s: %w", baseBranch, err)
	}

	// Create the new branch
	newRef := &github.Reference{
		Ref:    github.Ptr("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}

	_, _, err = c.client.Git.CreateRef(c.ctx, c.owner, c.repo, newRef)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	return nil
}

// GetCommits retrieves commits from a pull request
func (c *Client) GetCommits() ([]git.Commit, error) {
	c.logger.Debugf("Getting commits for PR #%d", c.prNum)

	opts := &github.ListOptions{}
	var allCommits []git.Commit

	for {
		commits, resp, err := c.client.PullRequests.ListCommits(c.ctx, c.owner, c.repo, c.prNum, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get commits: %w", err)
		}

		for _, commit := range commits {
			allCommits = append(allCommits, git.Commit{
				SHA:     commit.GetSHA(),
				Message: commit.GetCommit().GetMessage(),
				Author:  commit.GetCommit().GetAuthor().GetName(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allCommits, nil
}

// CreatePR creates a new pull request
func (c *Client) CreatePR(title, body, head, base string) (*git.PullRequest, error) {
	c.logger.Debugf("Creating PR: %s -> %s", head, base)

	newPR := &github.NewPullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
	}

	pr, _, err := c.client.PullRequests.Create(c.ctx, c.owner, c.repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	return &git.PullRequest{
		Number: pr.GetNumber(),
		Title:  pr.GetTitle(),
		State:  pr.GetState(),
		Merged: pr.GetMerged(),
		Author: pr.GetUser().GetLogin(),
		Head: git.Reference{
			Branch: pr.GetHead().GetRef(),
			SHA:    pr.GetHead().GetSHA(),
		},
		Base: git.Reference{
			Branch: pr.GetBase().GetRef(),
			SHA:    pr.GetBase().GetSHA(),
		},
	}, nil
}

// CherryPickCommit cherry-picks a commit to a branch by applying only the changes from that commit
func (c *Client) CherryPickCommit(commitSHA, targetBranch string) error {
	c.logger.Debugf("Cherry-picking commit %s to branch %s", commitSHA, targetBranch)

	// Get the commit to cherry-pick
	commit, _, err := c.client.Git.GetCommit(c.ctx, c.owner, c.repo, commitSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit %s: %w", commitSHA, err)
	}

	// Handle different commit types
	var parentSHA string
	if len(commit.Parents) == 0 {
		return fmt.Errorf("cannot cherry-pick initial commit %s (no parents)", commitSHA)
	} else if len(commit.Parents) == 1 {
		// Regular commit - use the single parent
		parentSHA = commit.Parents[0].GetSHA()
	} else {
		// Merge commit - use the first parent (the branch that was merged into)
		// This will cherry-pick all changes introduced by the merge
		parentSHA = commit.Parents[0].GetSHA()
		c.logger.Infof("Cherry-picking merge commit %s (using first parent %s)", commitSHA, parentSHA)
	}

	// Get the target branch reference
	targetRef, _, err := c.client.Git.GetRef(c.ctx, c.owner, c.repo, "heads/"+targetBranch)
	if err != nil {
		return fmt.Errorf("failed to get target branch %s: %w", targetBranch, err)
	}

	// Get the tree of the target branch (where we'll apply the changes)
	targetTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, targetRef.Object.GetSHA(), false)
	if err != nil {
		return fmt.Errorf("failed to get target tree: %w", err)
	}

	// Get the parent commit tree
	parentTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, parentSHA, false)
	if err != nil {
		return fmt.Errorf("failed to get parent tree: %w", err)
	}

	// Get the commit tree (the state after the changes)
	commitTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, commit.Tree.GetSHA(), false)
	if err != nil {
		return fmt.Errorf("failed to get commit tree: %w", err)
	}

	// Build a new tree by applying the changes from the commit to the target tree
	newTreeEntries, err := c.applyCommitChanges(targetTree, parentTree, commitTree)
	if err != nil {
		return fmt.Errorf("failed to apply commit changes: %w", err)
	}

	// Create the new tree
	newTree, _, err := c.client.Git.CreateTree(c.ctx, c.owner, c.repo, targetRef.Object.GetSHA(), newTreeEntries)
	if err != nil {
		return fmt.Errorf("failed to create new tree: %w", err)
	}

	// Create the cherry-pick commit
	newCommit := &github.Commit{
		Message: commit.Message,
		Tree:    &github.Tree{SHA: newTree.SHA},
		Parents: []*github.Commit{
			{SHA: targetRef.Object.SHA},
		},
	}

	createdCommit, _, err := c.client.Git.CreateCommit(c.ctx, c.owner, c.repo, newCommit, nil)
	if err != nil {
		return fmt.Errorf("failed to create cherry-pick commit: %w", err)
	}

	// Update the target branch to point to the new commit
	targetRef.Object.SHA = createdCommit.SHA
	_, _, err = c.client.Git.UpdateRef(c.ctx, c.owner, c.repo, targetRef, false)
	if err != nil {
		return fmt.Errorf("failed to update target branch: %w", err)
	}

	c.logger.Debugf("Cherry-pick successful, created commit: %s", createdCommit.GetSHA())
	return nil
}

// applyCommitChanges calculates the changes from parentTree to commitTree and applies them to targetTree
func (c *Client) applyCommitChanges(targetTree, parentTree, commitTree *github.Tree) ([]*github.TreeEntry, error) {
	// Create maps for easier lookup
	targetEntries := make(map[string]*github.TreeEntry)
	for _, entry := range targetTree.Entries {
		targetEntries[entry.GetPath()] = entry
	}

	parentEntries := make(map[string]*github.TreeEntry)
	for _, entry := range parentTree.Entries {
		parentEntries[entry.GetPath()] = entry
	}

	commitEntries := make(map[string]*github.TreeEntry)
	for _, entry := range commitTree.Entries {
		commitEntries[entry.GetPath()] = entry
	}

	// Start with all entries from target tree
	resultEntries := make([]*github.TreeEntry, 0, len(targetEntries))
	processedPaths := make(map[string]bool)

	// Process changes: files that were modified or added in the commit
	for path, commitEntry := range commitEntries {
		parentEntry := parentEntries[path]

		// If file was added or modified in the commit
		if parentEntry == nil || commitEntry.GetSHA() != parentEntry.GetSHA() {
			// Apply the change to our result
			resultEntries = append(resultEntries, &github.TreeEntry{
				Path: github.Ptr(path),
				Mode: commitEntry.Mode,
				Type: commitEntry.Type,
				SHA:  commitEntry.SHA,
			})
			processedPaths[path] = true
		}
	}

	// Add remaining files from target tree that weren't modified
	for path, targetEntry := range targetEntries {
		if !processedPaths[path] {
			// Check if this file was deleted in the commit (exists in parent but not in commit)
			_, existsInParent := parentEntries[path]
			_, existsInCommit := commitEntries[path]

			if existsInParent && !existsInCommit {
				// File was deleted in commit, so delete it from result too
				continue
			}

			// Keep the file from target tree
			resultEntries = append(resultEntries, &github.TreeEntry{
				Path: targetEntry.Path,
				Mode: targetEntry.Mode,
				Type: targetEntry.Type,
				SHA:  targetEntry.SHA,
			})
		}
	}

	return resultEntries, nil
}
