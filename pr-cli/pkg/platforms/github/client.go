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
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/lgtm"
	"github.com/google/go-github/v74/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Client implements the GitClient interface for GitHub
type Client struct {
	*logrus.Logger
	client        *github.Client  // GitHub API client for general operations
	commentClient *github.Client  // GitHub API client for comment operations (may use different token)
	ctx           context.Context // Request context
	owner         string          // Repository owner
	repo          string          // Repository name
	prNum         int             // Pull request number
	prSender      string          // Pull request author
	commentSender string          // Comment author
	selfCheckName string          // Name of the tool's own check run to exclude
	robotAccounts []string        // Robot/bot account usernames
}

// Factory implements ClientFactory for GitHub
type Factory struct{}

// createGitHubClient creates a GitHub client with the specified token
func createGitHubClient(ctx context.Context, token, baseURL string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Set custom base URL if provided
	if baseURL != "" && baseURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to set GitHub enterprise URL: %w", err)
		}
	}

	return client, nil
}

// CreateClient creates a new GitHub client
func (f *Factory) CreateClient(logger *logrus.Logger, config *git.Config) (git.GitClient, error) {
	ctx := context.Background()

	// Create primary client with main token
	client, err := createGitHubClient(ctx, config.Token, config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create main GitHub client: %w", err)
	}

	// Create comment client - only if CommentToken is different from main Token
	var commentClient *github.Client
	if config.CommentToken != "" && config.CommentToken != config.Token {
		logger.Debugf("Using separate comment token for posting comments")
		commentClient, err = createGitHubClient(ctx, config.CommentToken, config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create comment GitHub client: %w", err)
		}
	} else {
		logger.Debugf("Using main token for posting comments")
		commentClient = client
	}

	return &Client{
		Logger:        logger,
		client:        client,
		commentClient: commentClient,
		ctx:           ctx,
		owner:         config.Owner,
		repo:          config.Repo,
		prNum:         config.PRNum,
		prSender:      config.PRSender,
		commentSender: config.CommentSender,
		selfCheckName: config.SelfCheckName,
		robotAccounts: config.RobotAccounts,
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
		Body:   pr.GetBody(),
		URL:    pr.GetHTMLURL(),
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

	_, _, err := c.commentClient.Issues.CreateComment(c.ctx, c.owner, c.repo, c.prNum, comment)
	return err
}

// GetIssue retrieves an issue by number
func (c *Client) GetIssue(issueNumber int) (*git.Issue, error) {
	issue, _, err := c.client.Issues.Get(c.ctx, c.owner, c.repo, issueNumber)
	if err != nil {
		return nil, err
	}
	return convertGitHubIssue(issue), nil
}

// UpdateIssueBody updates an issue body/description
func (c *Client) UpdateIssueBody(issueNumber int, body string) error {
	request := &github.IssueRequest{
		Body: github.Ptr(body),
	}
	_, _, err := c.client.Issues.Edit(c.ctx, c.owner, c.repo, issueNumber, request)
	return err
}

const (
	defaultListPerPage  = 50
	maxIssueListPages   = 10
	defaultIssueState   = "open"
	defaultIssueSort    = "created"
	defaultIssueOrder   = "asc"
	fallbackErrorPrefix = "github list fallback"
)

// FindIssue locates an issue using search options
func (c *Client) FindIssue(opts git.IssueSearchOptions) (*git.Issue, error) {
	issue, searchErr := c.findIssueUsingSearch(opts)
	if searchErr == nil {
		return issue, nil
	}

	c.Debugf("issue search fallback triggered: %v", searchErr)

	listIssue, listErr := c.findIssueUsingList(opts)
	if listErr == nil {
		return listIssue, nil
	}

	if searchErr == nil {
		return nil, listErr
	}

	return nil, errors.Join(searchErr, listErr)
}

func (c *Client) findIssueUsingSearch(opts git.IssueSearchOptions) (*git.Issue, error) {
	queryParts := []string{
		fmt.Sprintf("repo:%s/%s", c.owner, c.repo),
		"type:issue",
	}
	if opts.State != "" {
		queryParts = append(queryParts, fmt.Sprintf("state:%s", opts.State))
	}
	// Note: We intentionally skip author in the search query to avoid GitHub API errors with bot accounts.
	// Bot usernames like "renovate[bot]" cannot be searched directly using "author:renovate" as it results
	// in a 422 error (user not found). Instead, we search without author filter and then apply author
	// filtering on the results using normalizeLogin(), which handles bot username variations correctly.
	if opts.Title != "" {
		queryParts = append(queryParts, fmt.Sprintf("in:title \"%s\"", opts.Title))
	}
	for _, label := range opts.Labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		queryParts = append(queryParts, fmt.Sprintf("label:\"%s\"", label))
	}

	query := strings.Join(queryParts, " ")
	searchOptions := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	if opts.Sort != "" {
		searchOptions.Sort = opts.Sort
	}
	if opts.Order != "" {
		searchOptions.Order = opts.Order
	}

	c.Debugf("searching issues query=%q sort=%s order=%s per_page=%d", query, searchOptions.Sort, searchOptions.Order, searchOptions.PerPage)

	results, _, err := c.client.Search.Issues(c.ctx, query, searchOptions)
	if err != nil {
		return nil, err
	}
	if results.GetTotal() == 0 || len(results.Issues) == 0 {
		return nil, fmt.Errorf("issue not found for query: %s", query)
	}

	c.Debugf("search results total=%d returned=%d", results.GetTotal(), len(results.Issues))

	// Filter results by all criteria (title, author, labels, state).
	// This is especially important for author filtering, as issueMatchesFilters() uses normalizeLogin()
	// which correctly handles bot account name variations (e.g., "renovate" matches "renovate[bot]").
	for _, issue := range results.Issues {
		if !issueMatchesFilters(issue, opts) {
			continue
		}
		match := convertGitHubIssue(issue)
		if match != nil {
			c.Debugf("search matched issue #%d title=%q author=%q", match.Number, match.Title, match.Author)
		}
		return match, nil
	}

	return nil, fmt.Errorf("issue not found matching filters: title=%q author=%q", opts.Title, opts.Author)
}

func (c *Client) findIssueUsingList(opts git.IssueSearchOptions) (*git.Issue, error) {
	listOpts := &github.IssueListByRepoOptions{
		State:     firstNonEmpty(opts.State, defaultIssueState),
		Sort:      firstNonEmpty(opts.Sort, defaultIssueSort),
		Direction: normalizeOrder(opts.Order),
		ListOptions: github.ListOptions{
			PerPage: defaultListPerPage,
			Page:    1,
		},
	}

	if len(opts.Labels) > 0 {
		listOpts.Labels = append([]string{}, opts.Labels...)
	}

	c.Debugf("listing issues fallback title=%q author=%q state=%q labels=%v sort=%s order=%s", opts.Title, opts.Author, listOpts.State, listOpts.Labels, listOpts.Sort, listOpts.Direction)

	totalInspected := 0
	for page := 1; page <= maxIssueListPages; page++ {
		listOpts.ListOptions.Page = page
		issues, resp, err := c.client.Issues.ListByRepo(c.ctx, c.owner, c.repo, listOpts)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fallbackErrorPrefix, err)
		}
		totalInspected += len(issues)
		nextPage := 0
		if resp != nil {
			nextPage = resp.NextPage
		}
		c.Debugf("list fallback page=%d count=%d next_page=%d", page, len(issues), nextPage)

		for _, issue := range issues {
			if issueMatchesFilters(issue, opts) {
				match := convertGitHubIssue(issue)
				if match != nil {
					c.Debugf("list fallback matched issue #%d title=%q author=%q", match.Number, match.Title, match.Author)
				}
				return match, nil
			}
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
	}

	c.Debugf("list fallback completed inspected=%d max_pages=%d", totalInspected, maxIssueListPages)
	return nil, fmt.Errorf("%s: issue not found for filters title=%q author=%q", fallbackErrorPrefix, opts.Title, opts.Author)
}

func normalizeLogin(login string) string {
	normalized := strings.TrimSpace(strings.ToLower(login))
	if normalized == "" {
		return ""
	}
	normalized = strings.TrimSuffix(normalized, "[bot]")
	normalized = strings.TrimSuffix(normalized, "-bot")
	return strings.TrimSpace(normalized)
}

func issueMatchesFilters(issue *github.Issue, opts git.IssueSearchOptions) bool {
	if issue == nil {
		return false
	}
	if issue.IsPullRequest() {
		return false
	}

	if opts.Title != "" {
		title := strings.TrimSpace(issue.GetTitle())
		if title == "" {
			return false
		}
		actualTitle := strings.ToLower(title)
		targetTitle := strings.ToLower(strings.TrimSpace(opts.Title))
		if targetTitle == "" {
			return false
		}
		if !strings.Contains(actualTitle, targetTitle) {
			return false
		}
	}

	if strings.TrimSpace(opts.Author) != "" {
		expectedAuthor := normalizeLogin(opts.Author)
		if expectedAuthor == "" {
			return false
		}

		actualAuthor := ""
		if issue.GetUser() != nil {
			actualAuthor = normalizeLogin(issue.GetUser().GetLogin())
		}
		if actualAuthor != expectedAuthor {
			return false
		}
	}

	if len(opts.Labels) > 0 {
		labelSet := make(map[string]struct{}, len(issue.Labels))
		for _, label := range issue.Labels {
			labelName := strings.ToLower(strings.TrimSpace(label.GetName()))
			if labelName == "" {
				continue
			}
			labelSet[labelName] = struct{}{}
		}

		for _, required := range opts.Labels {
			requiredName := strings.ToLower(strings.TrimSpace(required))
			if requiredName == "" {
				continue
			}
			if _, ok := labelSet[requiredName]; !ok {
				return false
			}
		}
	}

	if opts.State != "" {
		if !strings.EqualFold(issue.GetState(), opts.State) {
			return false
		}
	}

	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeOrder(value string) string {
	order := strings.ToLower(strings.TrimSpace(value))
	if order == "" {
		return defaultIssueOrder
	}
	if order != "asc" && order != "desc" {
		return defaultIssueOrder
	}
	return order
}

// UpdatePRBody updates the pull request body/description
func (c *Client) UpdatePRBody(body string) error {
	request := &github.IssueRequest{
		Body: github.Ptr(body),
	}
	_, _, err := c.client.Issues.Edit(c.ctx, c.owner, c.repo, c.prNum, request)
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
			ID: comment.GetID(),
			User: git.User{
				Login: comment.GetUser().GetLogin(),
			},
			Body: comment.GetBody(),
			URL:  comment.GetHTMLURL(),
		}
	}

	return result, nil
}

func convertGitHubIssue(issue *github.Issue) *git.Issue {
	if issue == nil {
		return nil
	}
	author := ""
	if issue.GetUser() != nil {
		author = issue.GetUser().GetLogin()
	}
	createdAt := ""
	if created := issue.GetCreatedAt(); !created.IsZero() {
		createdAt = created.Format(time.RFC3339)
	}
	return &git.Issue{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		State:     issue.GetState(),
		Author:    author,
		Body:      issue.GetBody(),
		URL:       issue.GetHTMLURL(),
		CreatedAt: createdAt,
	}
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
		if !review.GetSubmittedAt().IsZero() {
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
	c.Debugf("Requesting reviewers one by one: %v\n", cleanReviewers)

	var assignedReviewers []string
	var failedReviewers []string

	// GitHub API has issues when assigning multiple reviewers at once,
	// so we use individual assignment as a fallback solution
	for _, reviewer := range cleanReviewers {
		c.Debugf("Assigning individual reviewer: %s\n", reviewer)

		reviewerRequest := github.ReviewersRequest{
			Reviewers: []string{reviewer},
		}

		response, _, err := c.client.PullRequests.RequestReviewers(c.ctx, c.owner, c.repo, c.prNum, reviewerRequest)
		if err != nil {
			c.Debugf("Failed to assign reviewer %s: %v\n", reviewer, err)
			failedReviewers = append(failedReviewers, reviewer)
			continue
		}

		// Check if the reviewer was successfully assigned
		if response != nil && len(response.RequestedReviewers) > 0 {
			for _, assignedReviewer := range response.RequestedReviewers {
				if strings.EqualFold(assignedReviewer.GetLogin(), reviewer) {
					assignedReviewers = append(assignedReviewers, reviewer)
					c.Debugf("Successfully assigned reviewer: %s\n", reviewer)
					break
				}
			}
		} else {
			// If no response or empty, still consider it potentially successful
			// (GitHub API sometimes doesn't return the full list)
			assignedReviewers = append(assignedReviewers, reviewer)
			c.Debugf("Reviewer assignment request sent for: %s\n", reviewer)
		}
	}

	c.Debugf("Assignment summary - Assigned: %v, Failed: %v\n", assignedReviewers, failedReviewers)

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

// GetLGTMVotes retrieves and validates LGTM votes using provided or fetched comments
func (c *Client) GetLGTMVotes(comments []git.Comment, requiredPerms []string, debugMode bool, ignoreUserRemove ...string) (int, map[string]string, error) {
	ignoreUser := strings.ToLower(c.getIgnoreUser(ignoreUserRemove))
	lgtmUsers := make(map[string]string)

	// 1. Process review votes
	userLatestReviews, err := c.processReviewVotes(lgtmUsers)
	if err != nil {
		return 0, nil, err
	}

	// 2. Process comment votes
	if err := c.processCommentVotesWithComments(comments, lgtmUsers, userLatestReviews, debugMode, ignoreUser); err != nil {
		return 0, nil, err
	}

	c.logCollectedUsers(lgtmUsers, ignoreUser)

	// 3. Validate permissions and count valid votes
	return c.validatePermissionsAndCount(lgtmUsers, requiredPerms)
}

// getIgnoreUser extracts the ignore user from optional parameters
func (c *Client) getIgnoreUser(ignoreUserRemove []string) string {
	if len(ignoreUserRemove) > 0 {
		return ignoreUserRemove[0]
	}
	return ""
}

// processReviewVotes processes review approvals and returns the latest reviews map
func (c *Client) processReviewVotes(lgtmUsers map[string]string) (map[string]*git.Review, error) {
	reviews, err := c.GetReviews()
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	userLatestReviews := c.findLatestReviews(reviews)
	c.addApprovedReviews(lgtmUsers, userLatestReviews)

	return userLatestReviews, nil
}

// findLatestReviews finds the latest review for each user
func (c *Client) findLatestReviews(reviews []git.Review) map[string]*git.Review {
	userLatestReviews := make(map[string]*git.Review)

	for i := range reviews {
		review := &reviews[i]
		user := strings.ToLower(review.User.Login)

		if strings.EqualFold(user, c.prSender) { // Skip self-approvals
			continue
		}

		if existingReview, exists := userLatestReviews[user]; !exists || review.SubmittedAt > existingReview.SubmittedAt {
			userLatestReviews[user] = review
		}
	}

	return userLatestReviews
}

// addApprovedReviews adds users who have approved via reviews
func (c *Client) addApprovedReviews(lgtmUsers map[string]string, userLatestReviews map[string]*git.Review) {
	for user, latestReview := range userLatestReviews {
		if latestReview.State == "APPROVED" {
			lgtmUsers[user] = ""
			c.Debugf("Found APPROVED from user: %s (latest review)", user)
		}
	}
}

// processCommentVotesWithComments processes LGTM commands from provided or fetched comments
func (c *Client) processCommentVotesWithComments(comments []git.Comment, lgtmUsers map[string]string, userLatestReviews map[string]*git.Review, debugMode bool, ignoreUser string) error {
	// If no comments provided, fetch them
	if comments == nil {
		var err error
		comments, err = c.GetComments()
		if err != nil {
			return fmt.Errorf("failed to get comments: %w", err)
		}
	}

	ignoreCommentIndex := c.findIgnoreCommentIndex(comments, ignoreUser)

	for i, comment := range comments {
		if i == ignoreCommentIndex {
			c.Debugf("Skipping ignored comment at index %d from user: %s", i, comment.User.Login)
			continue
		}

		c.processLGTMComment(comment, lgtmUsers, userLatestReviews, debugMode)
	}

	return nil
}

// findIgnoreCommentIndex finds the index of the comment to ignore
func (c *Client) findIgnoreCommentIndex(comments []git.Comment, ignoreUser string) int {
	if ignoreUser == "" {
		return -1
	}

	for i := len(comments) - 1; i >= 0; i-- {
		if strings.EqualFold(comments[i].User.Login, ignoreUser) {
			body := comment.Normalize(comments[i].Body)
			if lgtm.RemoveLGTMRegexp.MatchString(body) {
				c.Debugf("Ignoring /remove-lgtm comment from user: %s at index %d", ignoreUser, i)
				return i
			}
			break
		}
	}

	return -1
}

// processLGTMComment processes a single LGTM-related comment
func (c *Client) processLGTMComment(commentObj git.Comment, lgtmUsers map[string]string, userLatestReviews map[string]*git.Review, debugMode bool) {
	user := strings.ToLower(commentObj.User.Login)
	body := comment.Normalize(commentObj.Body)

	switch {
	case lgtm.RemoveLGTMRegexp.MatchString(body):
		c.handleRemoveLGTM(user, lgtmUsers, userLatestReviews)
	case lgtm.LGTMCancelRegexp.MatchString(body):
		c.handleLGTMCancel(user, lgtmUsers, userLatestReviews)
	case lgtm.LGTMRegexp.MatchString(body):
		c.handleLGTM(user, lgtmUsers, userLatestReviews, debugMode)
	}
}

// handleRemoveLGTM processes /remove-lgtm commands
func (c *Client) handleRemoveLGTM(user string, lgtmUsers map[string]string, userLatestReviews map[string]*git.Review) {
	if c.canRemoveLGTM(user, userLatestReviews) {
		delete(lgtmUsers, user)
		c.Debugf("Found `/remove-lgtm` from user: %s", user)
	} else {
		c.Debugf("Ignoring `/remove-lgtm` from user: %s (has latest APPROVED review)", user)
	}
}

// handleLGTMCancel processes /lgtm cancel commands
func (c *Client) handleLGTMCancel(user string, lgtmUsers map[string]string, userLatestReviews map[string]*git.Review) {
	if c.canRemoveLGTM(user, userLatestReviews) {
		delete(lgtmUsers, user)
		c.Debugf("Found `/lgtm cancel` from user: %s", user)
	} else {
		c.Debugf("Ignoring `/lgtm cancel` from user: %s (has latest APPROVED review)", user)
	}
}

// handleLGTM processes /lgtm commands
func (c *Client) handleLGTM(user string, lgtmUsers map[string]string, userLatestReviews map[string]*git.Review, debugMode bool) {
	if strings.EqualFold(user, c.prSender) && !debugMode {
		c.Debugf("Skipping LGTM from PR author %s (not allowed)", user)
		return
	}

	if strings.EqualFold(user, c.prSender) && debugMode {
		c.Debugf("Debug mode: allowing PR author %s to give LGTM to their own PR", user)
	}

	if c.canAddLGTM(user, userLatestReviews) {
		lgtmUsers[user] = ""
		c.Debugf("Found /lgtm from user: %s", user)
	} else {
		c.Debugf("Ignoring /lgtm from user: %s (already has latest APPROVED review)", user)
	}
}

// canRemoveLGTM checks if LGTM can be removed for a user
func (c *Client) canRemoveLGTM(user string, userLatestReviews map[string]*git.Review) bool {
	latestReview, exists := userLatestReviews[user]
	return !exists || latestReview.State != "APPROVED"
}

// canAddLGTM checks if LGTM can be added for a user
func (c *Client) canAddLGTM(user string, userLatestReviews map[string]*git.Review) bool {
	latestReview, exists := userLatestReviews[user]
	return !exists || latestReview.State != "APPROVED"
}

// logCollectedUsers logs the collected LGTM users
func (c *Client) logCollectedUsers(lgtmUsers map[string]string, ignoreUser string) {
	if ignoreUser != "" {
		c.Debugf("Collected LGTM users (ignoring %s): %v", ignoreUser, lgtmUsers)
	} else {
		c.Debugf("Collected LGTM users: %v", lgtmUsers)
	}
}

// validatePermissionsAndCount validates permissions for each user and counts valid votes
func (c *Client) validatePermissionsAndCount(lgtmUsers map[string]string, requiredPerms []string) (int, map[string]string, error) {
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

	// Check if the error is due to trying to approve one's own PR
	if err != nil && strings.Contains(err.Error(), "Can not approve your own pull request") {
		c.Warnf("Cannot approve own PR (robot account limitation), posting comment instead: %v", err)
		// Post the approval message as a comment instead
		return c.PostComment(fmt.Sprintf("✅ **Auto-approved** (LGTM threshold met)\n\n%s\n\n> Note: Cannot create formal approval review due to GitHub's self-approval restriction.", message))
	}

	return err
}

// DismissApprove dismisses the token user's approval review
func (c *Client) DismissApprove(message string) error {
	// First try to get the authenticated user (token user) information
	authenticatedUser, _, err := c.client.Users.Get(c.ctx, "")
	var tokenUser string

	if err != nil {
		// If we get a 403 error (insufficient permissions), try to find bot approvals to dismiss
		if strings.Contains(err.Error(), "403") {
			c.Debugf("Cannot get authenticated user due to permissions (403), looking for bot approvals to dismiss")
			return c.dismissBotApproval(message)
		}
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	tokenUser = authenticatedUser.GetLogin()
	c.Debugf("Attempting to dismiss approval review by token user: %s", tokenUser)

	// Find and dismiss the user's approval
	reviewID, err := c.findLatestApprovalByUser(tokenUser)
	if err != nil {
		return err
	}

	return c.dismissReview(reviewID, message)
}

// dismissBotApproval attempts to dismiss approval reviews from bot accounts when we can't identify the token user
func (c *Client) dismissBotApproval(message string) error {
	botApprovals, err := c.findBotApprovals()
	if err != nil {
		return err
	}

	if len(botApprovals) == 0 {
		return fmt.Errorf("no bot approval reviews found to dismiss")
	}

	// Try to dismiss the bot approvals (typically there should be only one bot doing the approval)
	var dismissedCount int
	var lastErr error

	for botUser, reviewID := range botApprovals {
		if err := c.dismissReview(reviewID, message); err != nil {
			c.Errorf("Failed to dismiss approval from bot %s (review ID: %d): %v", botUser, reviewID, err)
			lastErr = err
		} else {
			c.Infof("✅ Successfully dismissed approval from bot: %s (review ID: %d)", botUser, reviewID)
			dismissedCount++
		}
	}

	if dismissedCount == 0 {
		return fmt.Errorf("failed to dismiss any bot approvals, last error: %w", lastErr)
	}

	c.Infof("Successfully dismissed %d bot approval(s)", dismissedCount)
	return nil
}

// findLatestApprovalByUser finds the latest APPROVED review by a specific user
func (c *Client) findLatestApprovalByUser(username string) (int64, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(c.ctx, c.owner, c.repo, c.prNum, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get reviews: %w", err)
	}

	var latestApprovalID int64 = 0
	for _, review := range reviews {
		if strings.EqualFold(review.GetUser().GetLogin(), username) && review.GetState() == "APPROVED" {
			if review.GetID() > latestApprovalID {
				latestApprovalID = review.GetID()
			}
		}
	}

	if latestApprovalID == 0 {
		return 0, fmt.Errorf("no approval review found for user %s to dismiss", username)
	}

	return latestApprovalID, nil
}

// findBotApprovals finds all latest APPROVED reviews by bot accounts
func (c *Client) findBotApprovals() (map[string]int64, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(c.ctx, c.owner, c.repo, c.prNum, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	botApprovals := make(map[string]int64) // username -> latest review ID
	for _, review := range reviews {
		username := review.GetUser().GetLogin()
		if review.GetState() == "APPROVED" {
			if c.isBotAccount(username) {
				if existingID, exists := botApprovals[username]; !exists || review.GetID() > existingID {
					botApprovals[username] = review.GetID()
					c.Debugf("Found APPROVED review from bot: %s (review ID: %d)", username, review.GetID())
				}
			} else {
				c.Debugf("Skipping non-bot approval from user: %s (review ID: %d)", username, review.GetID())
			}
		}
	}

	return botApprovals, nil
}

// dismissReview dismisses a specific review by ID
func (c *Client) dismissReview(reviewID int64, message string) error {
	dismissRequest := &github.PullRequestReviewDismissalRequest{
		Message: github.Ptr(message),
	}

	_, _, err := c.client.PullRequests.DismissReview(c.ctx, c.owner, c.repo, c.prNum, reviewID, dismissRequest)
	return err
}

// isBotAccount checks if a username appears to be a bot account
func (c *Client) isBotAccount(username string) bool {
	return slices.Contains(c.robotAccounts, username)
}

// MergePR merges the pull request with the specified method
func (c *Client) MergePR(method string) error {
	options := &github.PullRequestOptions{
		MergeMethod: method,
	}

	_, _, err := c.client.PullRequests.Merge(c.ctx, c.owner, c.repo, c.prNum, "", options)
	return err
}

// ClosePR closes the pull request without merging
func (c *Client) ClosePR() error {
	closed := "closed"
	pullRequest := &github.PullRequest{
		State: &closed,
	}

	_, _, err := c.client.PullRequests.Edit(c.ctx, c.owner, c.repo, c.prNum, pullRequest)
	return err
}

// RebasePR updates the PR branch with the base branch
func (c *Client) RebasePR() error {
	_, _, err := c.client.PullRequests.UpdateBranch(c.ctx, c.owner, c.repo, c.prNum, nil)
	return err
}

// GetAvailableMergeMethods retrieves the available merge methods for the pull request
func (c *Client) GetAvailableMergeMethods() ([]string, error) {
	pr, err := c.GetPR()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	// Get repository settings to determine available merge methods
	repo, _, err := c.client.Repositories.Get(c.ctx, c.owner, c.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository settings: %w", err)
	}

	var availableMethods []string

	// Check which merge methods are allowed in the repository
	// GitHub API returns boolean values for each merge method
	if repo.GetAllowRebaseMerge() && pr.Head.SHA != pr.Base.SHA {
		availableMethods = append(availableMethods, "rebase")
	}
	if repo.GetAllowSquashMerge() {
		availableMethods = append(availableMethods, "squash")
	}
	if repo.GetAllowMergeCommit() {
		availableMethods = append(availableMethods, "merge")
	}

	// If no methods are available, return an error
	if len(availableMethods) == 0 {
		return nil, fmt.Errorf("no merge methods are available for this repository")
	}

	return availableMethods, nil
}

// CheckRunsStatus checks if all check runs are successful with pagination
func (c *Client) CheckRunsStatus() (bool, []git.CheckRun, error) {
	pr, err := c.GetPR()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get PR: %w", err)
	}

	// Fetch all check runs with pagination
	allCheckRuns, err := c.fetchAllCheckRuns(pr.Head.SHA)
	if err != nil {
		return false, nil, err
	}

	// Analyze check runs and find failed ones
	failedChecks := c.analyzeCheckRuns(allCheckRuns)

	return len(failedChecks) == 0, failedChecks, nil
}

// fetchAllCheckRuns retrieves all check runs for a given SHA with pagination
func (c *Client) fetchAllCheckRuns(sha string) ([]*github.CheckRun, error) {
	var allCheckRuns []*github.CheckRun

	opts := &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100}, // GitHub max per page
	}

	for {
		checkRuns, resp, err := c.client.Checks.ListCheckRunsForRef(c.ctx, c.owner, c.repo, sha, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get check runs: %w", err)
		}

		allCheckRuns = append(allCheckRuns, checkRuns.CheckRuns...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allCheckRuns, nil
}

// analyzeCheckRuns analyzes check runs and returns failed ones
func (c *Client) analyzeCheckRuns(allCheckRuns []*github.CheckRun) []git.CheckRun {
	c.Debugf("Check run: selfCheckName: %q", c.selfCheckName)

	var failedChecks []git.CheckRun
	for _, check := range allCheckRuns {
		c.logCheckRunDetails(check)

		if c.isFailedCheck(check) {
			failedChecks = append(failedChecks, c.convertToGitCheckRun(check))
		}
	}

	return failedChecks
}

// logCheckRunDetails logs details of a check run
func (c *Client) logCheckRunDetails(check *github.CheckRun) {
	c.Debugf("Check run: %q, Status: %q, Conclusion: %q, URL: %q",
		check.GetName(), check.GetStatus(), check.GetConclusion(), check.GetHTMLURL())
}

// isFailedCheck determines if a check run should be considered failed
func (c *Client) isFailedCheck(check *github.CheckRun) bool {
	// Case 1: Completed check with failure/error conclusion
	if check.GetStatus() == "completed" {
		conclusion := check.GetConclusion()
		return conclusion != "success" && conclusion != "skipped"
	}

	// Case 2: Incomplete check (but exclude self-check)
	if check.GetStatus() != "completed" {
		return c.shouldIncludeIncompleteCheck(check)
	}

	return false
}

// shouldIncludeIncompleteCheck determines if an incomplete check should be considered failed
func (c *Client) shouldIncludeIncompleteCheck(check *github.CheckRun) bool {
	if c.selfCheckName == "" {
		return true
	}

	checkName := strings.TrimSpace(check.GetName())
	selfCheckSuffix := "/ " + c.selfCheckName

	return !strings.HasSuffix(checkName, selfCheckSuffix)
}

// convertToGitCheckRun converts a GitHub CheckRun to git.CheckRun
func (c *Client) convertToGitCheckRun(check *github.CheckRun) git.CheckRun {
	result := git.CheckRun{
		Name:   check.GetName(),
		Status: check.GetStatus(),
		URL:    check.GetHTMLURL(),
	}

	// Only set conclusion for completed checks
	if check.GetStatus() == "completed" {
		result.Conclusion = check.GetConclusion()
	}

	// Set app slug and check suite ID for GitHub Actions identification
	if check.App != nil {
		result.AppSlug = check.App.GetSlug()
	}
	if check.CheckSuite != nil {
		result.CheckSuiteID = check.CheckSuite.GetID()
	}

	return result
}

// AddLabels adds labels to the pull request
func (c *Client) AddLabels(labels []string) error {
	if len(labels) == 0 {
		return fmt.Errorf("no labels specified")
	}

	c.Debugf("Adding labels to PR #%d: %v", c.prNum, labels)

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

	c.Debugf("Removing labels from PR #%d: %v", c.prNum, labels)

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
	c.Debugf("Creating branch %s from %s", branchName, baseBranch)

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
	c.Debugf("Getting commits for PR #%d", c.prNum)

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
	c.Debugf("Creating PR: %s -> %s", head, base)

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
		Body:   pr.GetBody(),
		URL:    pr.GetHTMLURL(),
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
	c.Debugf("Cherry-picking commit %s to branch %s", commitSHA, targetBranch)

	// 1. Get and validate the commit to cherry-pick
	commit, parentSHA, err := c.prepareCommitForCherryPick(commitSHA)
	if err != nil {
		return err
	}

	// 2. Get target branch reference and trees
	targetRef, trees, err := c.prepareCherryPickTrees(targetBranch, parentSHA, commit)
	if err != nil {
		return err
	}

	// 3. Apply changes and create new tree
	newTree, err := c.createCherryPickTree(targetRef, trees)
	if err != nil {
		return err
	}

	// 4. Create and apply the cherry-pick commit
	return c.createAndApplyCherryPickCommit(commit, newTree, targetRef)
}

// cherryPickTrees holds the three trees needed for cherry-pick operation
type cherryPickTrees struct {
	target *github.Tree
	parent *github.Tree
	commit *github.Tree
}

// prepareCommitForCherryPick gets and validates the commit, returning the commit and parent SHA
func (c *Client) prepareCommitForCherryPick(commitSHA string) (*github.Commit, string, error) {
	commit, _, err := c.client.Git.GetCommit(c.ctx, c.owner, c.repo, commitSHA)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get commit %s: %w", commitSHA, err)
	}

	parentSHA, err := c.determineParentSHA(commit, commitSHA)
	if err != nil {
		return nil, "", err
	}

	return commit, parentSHA, nil
}

// determineParentSHA determines the appropriate parent SHA for cherry-pick
func (c *Client) determineParentSHA(commit *github.Commit, commitSHA string) (string, error) {
	if len(commit.Parents) == 0 {
		return "", fmt.Errorf("cannot cherry-pick initial commit %s (no parents)", commitSHA)
	}

	// Use the first parent (works for both regular and merge commits)
	parentSHA := commit.Parents[0].GetSHA()

	if len(commit.Parents) > 1 {
		c.Infof("Cherry-picking merge commit %s (using first parent %s)", commitSHA, parentSHA)
	}

	return parentSHA, nil
}

// prepareCherryPickTrees gets the target branch reference and all required trees
func (c *Client) prepareCherryPickTrees(targetBranch, parentSHA string, commit *github.Commit) (*github.Reference, *cherryPickTrees, error) {
	// Get target branch reference
	targetRef, _, err := c.client.Git.GetRef(c.ctx, c.owner, c.repo, "heads/"+targetBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target branch %s: %w", targetBranch, err)
	}

	// Get all required trees
	trees, err := c.fetchCherryPickTrees(targetRef.Object.GetSHA(), parentSHA, commit.Tree.GetSHA())
	if err != nil {
		return nil, nil, err
	}

	return targetRef, trees, nil
}

// fetchCherryPickTrees fetches the three trees needed for cherry-pick
func (c *Client) fetchCherryPickTrees(targetSHA, parentSHA, commitTreeSHA string) (*cherryPickTrees, error) {
	// Get target tree
	targetTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, targetSHA, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get target tree: %w", err)
	}

	// Get parent tree
	parentTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, parentSHA, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent tree: %w", err)
	}

	// Get commit tree
	commitTree, _, err := c.client.Git.GetTree(c.ctx, c.owner, c.repo, commitTreeSHA, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit tree: %w", err)
	}

	return &cherryPickTrees{
		target: targetTree,
		parent: parentTree,
		commit: commitTree,
	}, nil
}

// createCherryPickTree applies changes and creates the new tree
func (c *Client) createCherryPickTree(targetRef *github.Reference, trees *cherryPickTrees) (*github.Tree, error) {
	// Apply changes from commit to target tree
	newTreeEntries, err := c.applyCommitChanges(trees.target, trees.parent, trees.commit)
	if err != nil {
		return nil, fmt.Errorf("failed to apply commit changes: %w", err)
	}

	// Create the new tree
	newTree, _, err := c.client.Git.CreateTree(c.ctx, c.owner, c.repo, targetRef.Object.GetSHA(), newTreeEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to create new tree: %w", err)
	}

	return newTree, nil
}

// createAndApplyCherryPickCommit creates the cherry-pick commit and updates the target branch
func (c *Client) createAndApplyCherryPickCommit(commit *github.Commit, newTree *github.Tree, targetRef *github.Reference) error {
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

	c.Debugf("Cherry-pick successful, created commit: %s", createdCommit.GetSHA())
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

// BranchExists checks if a branch exists in the repository
func (c *Client) BranchExists(branchName string) (bool, error) {
	_, _, err := c.client.Git.GetRef(c.ctx, c.owner, c.repo, "heads/"+branchName)
	if err != nil {
		// Check if it's a 404 error (branch not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, nil
		}
		// For other errors, return the error
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}
	return true, nil
}

// GetWorkflowRunIDsFromCheckSuite retrieves workflow run IDs from a check suite
// Returns a slice of workflow run IDs associated with the check suite
func (c *Client) GetWorkflowRunIDsFromCheckSuite(checkSuiteID int64) ([]int64, error) {
	c.Debugf("Getting workflow runs for check suite ID: %d", checkSuiteID)

	// Get the check suite to retrieve the head SHA
	checkSuite, _, err := c.client.Checks.GetCheckSuite(c.ctx, c.owner, c.repo, checkSuiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get check suite %d: %w", checkSuiteID, err)
	}

	headSHA := checkSuite.GetHeadSHA()
	c.Debugf("Check suite head SHA: %s", headSHA)

	// List workflow runs for the head SHA
	opts := &github.ListWorkflowRunsOptions{
		HeadSHA: headSHA,
	}

	runs, _, err := c.client.Actions.ListRepositoryWorkflowRuns(c.ctx, c.owner, c.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs for SHA %s: %w", headSHA, err)
	}

	if len(runs.WorkflowRuns) == 0 {
		c.Warnf("No workflow runs found for check suite %d (SHA: %s)", checkSuiteID, headSHA)
		return nil, nil
	}

	// Collect workflow run IDs
	var runIDs []int64
	for _, run := range runs.WorkflowRuns {
		runIDs = append(runIDs, run.GetID())
		c.Debugf("Found workflow run: ID=%d, Name=%s, Status=%s, Conclusion=%s",
			run.GetID(), run.GetName(), run.GetStatus(), run.GetConclusion())
	}

	return runIDs, nil
}

// RerunWorkflowRunFailedJobs reruns failed jobs in a workflow run
func (c *Client) RerunWorkflowRunFailedJobs(runID int64) error {
	c.Debugf("Rerunning failed jobs for workflow run ID: %d", runID)

	_, err := c.client.Actions.RerunFailedJobsByID(c.ctx, c.owner, c.repo, runID)
	if err != nil {
		return fmt.Errorf("failed to rerun failed jobs for workflow run %d: %w", runID, err)
	}

	c.Infof("Successfully triggered rerun of failed jobs for workflow run ID: %d", runID)
	return nil
}
