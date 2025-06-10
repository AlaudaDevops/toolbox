package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// GitLabMergeRequest represents a GitLab merge request with relevant information
type GitLabMergeRequest struct {
	Project      string    `json:"project"`
	IID          int       `json:"iid"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	WebURL       string    `json:"web_url"`
	State        string    `json:"state"`
	Draft        bool      `json:"draft"`
	DaysOpen     int       `json:"days_open"`
	Labels       []string  `json:"labels"`
	Assignees    []string  `json:"assignees"`
	Reviewers    []string  `json:"reviewers"`
	TargetBranch string    `json:"target_branch"`
	SourceBranch string    `json:"source_branch"`
	Pipeline     string    `json:"pipeline_status"`
}

// GitLabWatcherResult represents the complete result of the GitLab MR watcher command
type GitLabWatcherResult struct {
	Group        string                `json:"group"`
	ScanDate     time.Time             `json:"scan_date"`
	MinDaysOpen  int                   `json:"min_days_open"`
	TotalProjects int                  `json:"total_projects"`
	TotalOldMRs  int                   `json:"total_old_mrs"`
	MergeRequests []GitLabMergeRequest `json:"merge_requests"`
}

// gitlabWatcherCmd represents the gitlab watch-mrs command
var gitlabWatcherCmd = &cobra.Command{
	Use:   "watch-mrs",
	Short: "Watch for old merge requests in a GitLab group",
	Long: `This command scans all projects in a GitLab group and identifies
merge requests that have been open for longer than the specified number of days.
It returns a comprehensive JSON report with all relevant MR information.

Example:
  pr-watcher watch-mrs --group mygroup --days 7 --output results.json`,
	RunE: runGitLabWatcher,
}

func init() {
	rootCmd.AddCommand(gitlabWatcherCmd)

	// Add flags for the GitLab MR watcher command
	gitlabWatcherCmd.Flags().StringP("group", "g", "", "GitLab group name (required)")
	gitlabWatcherCmd.Flags().IntP("days", "d", 7, "Minimum number of days an MR should be open to be included")
	gitlabWatcherCmd.Flags().StringP("output", "f", "", "Output file path (optional, prints to stdout if not specified)")
	gitlabWatcherCmd.Flags().BoolP("include-drafts", "", false, "Include draft merge requests in the results")
	gitlabWatcherCmd.Flags().StringP("state", "s", "opened", "MR state to filter by (opened, closed, merged, all)")
	gitlabWatcherCmd.Flags().StringP("host", "", "", "GitLab host (optional, defaults to gitlab.com)")

	// Mark required flags
	gitlabWatcherCmd.MarkFlagRequired("group")
}

// runGitLabWatcher executes the main logic for the GitLab MR watcher command
func runGitLabWatcher(cmd *cobra.Command, args []string) error {
	// Check if GitLab CLI is available
	if err := checkGitLabCLI(); err != nil {
		return err
	}

	// Get flag values
	group, _ := cmd.Flags().GetString("group")
	days, _ := cmd.Flags().GetInt("days")
	outputFile, _ := cmd.Flags().GetString("output")
	includeDrafts, _ := cmd.Flags().GetBool("include-drafts")
	state, _ := cmd.Flags().GetString("state")
	host, _ := cmd.Flags().GetString("host")

	// Set GitLab host if specified
	if host != "" {
		os.Setenv("GITLAB_HOST", host)
	}

	fmt.Printf("Scanning GitLab group '%s' for MRs older than %d days...\n", group, days)

	// Initialize the result structure
	result := GitLabWatcherResult{
		Group:         group,
		ScanDate:      time.Now(),
		MinDaysOpen:   days,
		MergeRequests: []GitLabMergeRequest{},
	}

	// Get all projects for the group
	projects, err := getGroupProjects(group)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	result.TotalProjects = len(projects)
	fmt.Printf("Found %d projects\n", len(projects))

	// Process each project
	for i, project := range projects {
		fmt.Printf("Processing project %d/%d: %s\n", i+1, len(projects), project)

		mrs, err := getProjectMRs(project, state)
		if err != nil {
			fmt.Printf("Warning: failed to get MRs for %s: %v\n", project, err)
			continue
		}

		// Filter MRs based on criteria
		for _, mr := range mrs {
			if shouldIncludeMR(mr, days, includeDrafts) {
				result.MergeRequests = append(result.MergeRequests, mr)
			}
		}
	}

	result.TotalOldMRs = len(result.MergeRequests)
	fmt.Printf("Found %d old MRs across all projects\n", result.TotalOldMRs)

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

// checkGitLabCLI verifies that GitLab CLI is available and authenticated
func checkGitLabCLI() error {
	// Check if glab is installed
	_, err := exec.LookPath("glab")
	if err != nil {
		return fmt.Errorf("GitLab CLI (glab) is not installed. Please install it from: https://gitlab.com/gitlab-org/cli")
	}

	// Check if glab is authenticated
	cmd := exec.Command("glab", "auth", "status")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("GitLab CLI is not authenticated. Please run: glab auth login")
	}

	return nil
}

// getGroupProjects fetches all projects for the given group
func getGroupProjects(group string) ([]string, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("groups/%s/projects", group), "--paginate")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute glab command: %w", err)
	}

	var projects []struct {
		PathWithNamespace string `json:"path_with_namespace"`
	}

	err = json.Unmarshal(output, &projects)
	if err != nil {
		return nil, fmt.Errorf("failed to parse projects JSON: %w", err)
	}

	var projectNames []string
	for _, project := range projects {
		projectNames = append(projectNames, project.PathWithNamespace)
	}

	return projectNames, nil
}

// getProjectMRs fetches all merge requests for the given project
func getProjectMRs(project, state string) ([]GitLabMergeRequest, error) {
	stateParam := "opened"
	if state != "opened" {
		stateParam = state
	}

	// URL encode the project path
	projectPath := project

	cmd := exec.Command("glab", "api", fmt.Sprintf("projects/%s/merge_requests", projectPath),
		"--paginate",
		"--field", fmt.Sprintf("state=%s", stateParam),
		"--field", "per_page=100")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute glab mr list: %w", err)
	}

	var glMRs []struct {
		IID          int       `json:"iid"`
		Title        string    `json:"title"`
		Author       struct {
			Username string `json:"username"`
		} `json:"author"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		WebURL       string    `json:"web_url"`
		State        string    `json:"state"`
		Draft        bool      `json:"draft"`
		Labels       []string  `json:"labels"`
		Assignees    []struct {
			Username string `json:"username"`
		} `json:"assignees"`
		Reviewers    []struct {
			Username string `json:"username"`
		} `json:"reviewers"`
		TargetBranch string `json:"target_branch"`
		SourceBranch string `json:"source_branch"`
		Pipeline     struct {
			Status string `json:"status"`
		} `json:"pipeline"`
	}

	err = json.Unmarshal(output, &glMRs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MR JSON: %w", err)
	}

	var mrs []GitLabMergeRequest
	for _, glMR := range glMRs {
		// Calculate days open
		daysOpen := int(time.Since(glMR.CreatedAt).Hours() / 24)

		// Extract assignees
		var assignees []string
		for _, assignee := range glMR.Assignees {
			assignees = append(assignees, assignee.Username)
		}

		// Extract reviewers
		var reviewers []string
		for _, reviewer := range glMR.Reviewers {
			reviewers = append(reviewers, reviewer.Username)
		}

		// Get pipeline status
		pipelineStatus := "none"
		if glMR.Pipeline.Status != "" {
			pipelineStatus = glMR.Pipeline.Status
		}

		mr := GitLabMergeRequest{
			Project:      project,
			IID:          glMR.IID,
			Title:        glMR.Title,
			Author:       glMR.Author.Username,
			CreatedAt:    glMR.CreatedAt,
			UpdatedAt:    glMR.UpdatedAt,
			WebURL:       glMR.WebURL,
			State:        glMR.State,
			Draft:        glMR.Draft,
			DaysOpen:     daysOpen,
			Labels:       glMR.Labels,
			Assignees:    assignees,
			Reviewers:    reviewers,
			TargetBranch: glMR.TargetBranch,
			SourceBranch: glMR.SourceBranch,
			Pipeline:     pipelineStatus,
		}

		mrs = append(mrs, mr)
	}

	return mrs, nil
}

// shouldIncludeMR determines if an MR should be included in the results
func shouldIncludeMR(mr GitLabMergeRequest, minDays int, includeDrafts bool) bool {
	// Check if MR is old enough
	if mr.DaysOpen < minDays {
		return false
	}

	// Check if we should include drafts
	if mr.Draft && !includeDrafts {
		return false
	}

	return true
}
