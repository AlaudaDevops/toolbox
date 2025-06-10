package cmd

import (
	"testing"
	"time"
)

func TestShouldIncludePR(t *testing.T) {
	tests := []struct {
		name          string
		pr            PullRequest
		minDays       int
		includeDrafts bool
		expected      bool
	}{
		{
			name: "PR older than threshold and not draft",
			pr: PullRequest{
				DaysOpen: 10,
				Draft:    false,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      true,
		},
		{
			name: "PR newer than threshold",
			pr: PullRequest{
				DaysOpen: 5,
				Draft:    false,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      false,
		},
		{
			name: "Draft PR excluded when includeDrafts is false",
			pr: PullRequest{
				DaysOpen: 10,
				Draft:    true,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      false,
		},
		{
			name: "Draft PR included when includeDrafts is true",
			pr: PullRequest{
				DaysOpen: 10,
				Draft:    true,
			},
			minDays:       7,
			includeDrafts: true,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludePR(tt.pr, tt.minDays, tt.includeDrafts)
			if result != tt.expected {
				t.Errorf("shouldIncludePR() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPRWatcherResult(t *testing.T) {
	result := PRWatcherResult{
		Organization: "test-org",
		ScanDate:     time.Now(),
		MinDaysOpen:  7,
		TotalRepos:   5,
		TotalOldPRs:  3,
		PullRequests: []PullRequest{
			{
				Repository: "test-org/repo1",
				Number:     123,
				Title:      "Test PR",
				Author:     "testuser",
				DaysOpen:   10,
			},
		},
	}

	if result.Organization != "test-org" {
		t.Errorf("Expected organization to be 'test-org', got %s", result.Organization)
	}

	if result.TotalRepos != 5 {
		t.Errorf("Expected total repos to be 5, got %d", result.TotalRepos)
	}

	if len(result.PullRequests) != 1 {
		t.Errorf("Expected 1 pull request, got %d", len(result.PullRequests))
	}
}
