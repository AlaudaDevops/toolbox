package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// PullRequest represents a GitHub pull request with relevant information
type PullRequest struct {
	Repository   string    `json:"repository"`
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	URL          string    `json:"url"`
	State        string    `json:"state"`
	Draft        bool      `json:"draft"`
	DaysOpen     int       `json:"days_open"`
	Labels       []string  `json:"labels"`
	Assignees    []string  `json:"assignees"`
	Reviewers    []string  `json:"reviewers"`
	BaseBranch   string    `json:"base_branch"`
	HeadBranch   string    `json:"head_branch"`
}

// PRWatcherResult represents the complete result of the PR watcher command
type PRWatcherResult struct {
	Organization  string        `json:"organization"`
	ScanDate      time.Time     `json:"scan_date"`
	MinDaysOpen   int           `json:"min_days_open"`
	TotalRepos    int           `json:"total_repos"`
	TotalOldPRs   int           `json:"total_old_prs"`
	PullRequests  []PullRequest `json:"pull_requests"`
}

// prWatcherCmd represents the prwatcher command
var prWatcherCmd = &cobra.Command{
	Use:   "watch-prs",
	Short: "Watch for old pull requests in an organization",
	Long: `This command scans all repositories in a GitHub organization and identifies
pull requests that have been open for longer than the specified number of days.
It returns a comprehensive JSON report with all relevant PR information.

Example:
  pr-watcher watch-prs --org myorg --days 7 --output results.json`,
	RunE: runPRWatcher,
}

func init() {
	rootCmd.AddCommand(prWatcherCmd)

	// Add flags for the PR watcher command
	prWatcherCmd.Flags().StringP("org", "o", "", "GitHub organization name (required)")
	prWatcherCmd.Flags().IntP("days", "d", 7, "Minimum number of days a PR should be open to be included")
	prWatcherCmd.Flags().StringP("output", "f", "", "Output file path (optional, prints to stdout if not specified)")
	prWatcherCmd.Flags().BoolP("include-drafts", "", false, "Include draft pull requests in the results")
	prWatcherCmd.Flags().StringP("state", "s", "open", "PR state to filter by (open, closed, merged, all)")

	// Mark required flags
	prWatcherCmd.MarkFlagRequired("org")
}

// runPRWatcher executes the main logic for the PR watcher command
func runPRWatcher(cmd *cobra.Command, args []string) error {
	// Get flag values
	org, _ := cmd.Flags().GetString("org")
	days, _ := cmd.Flags().GetInt("days")
	outputFile, _ := cmd.Flags().GetString("output")
	includeDrafts, _ := cmd.Flags().GetBool("include-drafts")
	state, _ := cmd.Flags().GetString("state")

	fmt.Printf("Scanning organization '%s' for PRs older than %d days...\n", org, days)

	// Initialize the result structure
	result := PRWatcherResult{
		Organization: org,
		ScanDate:     time.Now(),
		MinDaysOpen:  days,
		PullRequests: []PullRequest{},
	}

	// Get all repositories for the organization
	repos, err := getOrgRepositories(org)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	result.TotalRepos = len(repos)
	fmt.Printf("Found %d repositories\n", len(repos))

	// Process each repository
	for i, repo := range repos {
		fmt.Printf("Processing repository %d/%d: %s\n", i+1, len(repos), repo)

		prs, err := getRepositoryPRs(repo, state)
		if err != nil {
			fmt.Printf("Warning: failed to get PRs for %s: %v\n", repo, err)
			continue
		}

		// Filter PRs based on criteria
		for _, pr := range prs {
			if shouldIncludePR(pr, days, includeDrafts) {
				result.PullRequests = append(result.PullRequests, pr)
			}
		}
	}

	result.TotalOldPRs = len(result.PullRequests)
	fmt.Printf("Found %d old PRs across all repositories\n", result.TotalOldPRs)

	// Convert to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Output the result
	if outputFile != "" {
		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Results saved to %s\n", outputFile)
	} else {
		fmt.Println(string(jsonData))
	}

	return nil
}

// getOrgRepositories fetches all repositories for the given organization
func getOrgRepositories(org string) ([]string, error) {
	cmd := exec.Command("gh", "repo", "list", org, "--limit", "1000", "--json", "name")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gh command: %w", err)
	}

	var repos []struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal(output, &repos)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository JSON: %w", err)
	}

	var repoNames []string
	for _, repo := range repos {
		repoNames = append(repoNames, fmt.Sprintf("%s/%s", org, repo.Name))
	}

	return repoNames, nil
}

// getRepositoryPRs fetches all pull requests for the given repository
func getRepositoryPRs(repo, state string) ([]PullRequest, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--repo", repo,
		"--state", state,
		"--limit", "100",
		"--json", "number,title,author,createdAt,updatedAt,url,state,isDraft,labels,assignees,reviewRequests,baseRefName,headRefName")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gh pr list: %w", err)
	}

	var ghPRs []struct {
		Number         int       `json:"number"`
		Title          string    `json:"title"`
		Author         struct {
			Login string `json:"login"`
		} `json:"author"`
		CreatedAt      time.Time `json:"createdAt"`
		UpdatedAt      time.Time `json:"updatedAt"`
		URL            string    `json:"url"`
		State          string    `json:"state"`
		IsDraft        bool      `json:"isDraft"`
		Labels         []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignees      []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		ReviewRequests []struct {
			Login string `json:"login"`
		} `json:"reviewRequests"`
		BaseRefName    string `json:"baseRefName"`
		HeadRefName    string `json:"headRefName"`
	}

	err = json.Unmarshal(output, &ghPRs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PR JSON: %w", err)
	}

	var prs []PullRequest
	for _, ghPR := range ghPRs {
		// Calculate days open
		daysOpen := int(time.Since(ghPR.CreatedAt).Hours() / 24)

		// Extract labels
		var labels []string
		for _, label := range ghPR.Labels {
			labels = append(labels, label.Name)
		}

		// Extract assignees
		var assignees []string
		for _, assignee := range ghPR.Assignees {
			assignees = append(assignees, assignee.Login)
		}

		// Extract reviewers
		var reviewers []string
		for _, reviewer := range ghPR.ReviewRequests {
			reviewers = append(reviewers, reviewer.Login)
		}

		pr := PullRequest{
			Repository: repo,
			Number:     ghPR.Number,
			Title:      ghPR.Title,
			Author:     ghPR.Author.Login,
			CreatedAt:  ghPR.CreatedAt,
			UpdatedAt:  ghPR.UpdatedAt,
			URL:        ghPR.URL,
			State:      ghPR.State,
			Draft:      ghPR.IsDraft,
			DaysOpen:   daysOpen,
			Labels:     labels,
			Assignees:  assignees,
			Reviewers:  reviewers,
			BaseBranch: ghPR.BaseRefName,
			HeadBranch: ghPR.HeadRefName,
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

// shouldIncludePR determines if a PR should be included in the results
func shouldIncludePR(pr PullRequest, minDays int, includeDrafts bool) bool {
	// Check if PR is old enough
	if pr.DaysOpen < minDays {
		return false
	}

	// Check if we should include drafts
	if pr.Draft && !includeDrafts {
		return false
	}

	return true
}
