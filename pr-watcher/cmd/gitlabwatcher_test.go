package cmd

import (
	"testing"
	"time"
)

func TestShouldIncludeMR(t *testing.T) {
	tests := []struct {
		name          string
		mr            GitLabMergeRequest
		minDays       int
		includeDrafts bool
		expected      bool
	}{
		{
			name: "MR older than threshold and not draft",
			mr: GitLabMergeRequest{
				DaysOpen: 10,
				Draft:    false,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      true,
		},
		{
			name: "MR newer than threshold",
			mr: GitLabMergeRequest{
				DaysOpen: 5,
				Draft:    false,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      false,
		},
		{
			name: "Draft MR excluded when includeDrafts is false",
			mr: GitLabMergeRequest{
				DaysOpen: 10,
				Draft:    true,
			},
			minDays:       7,
			includeDrafts: false,
			expected:      false,
		},
		{
			name: "Draft MR included when includeDrafts is true",
			mr: GitLabMergeRequest{
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
			result := shouldIncludeMR(tt.mr, tt.minDays, tt.includeDrafts)
			if result != tt.expected {
				t.Errorf("shouldIncludeMR() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitLabWatcherResult(t *testing.T) {
	result := GitLabWatcherResult{
		Group:       "test-group",
		ScanDate:    time.Now(),
		MinDaysOpen: 7,
		TotalProjects: 5,
		TotalOldMRs: 3,
		MergeRequests: []GitLabMergeRequest{
			{
				Project: "test-group/project1",
				IID:     123,
				Title:   "Test MR",
				Author:  "testuser",
				DaysOpen: 10,
			},
		},
	}

	if result.Group != "test-group" {
		t.Errorf("Expected group to be 'test-group', got %s", result.Group)
	}

	if result.TotalProjects != 5 {
		t.Errorf("Expected total projects to be 5, got %d", result.TotalProjects)
	}

	if len(result.MergeRequests) != 1 {
		t.Errorf("Expected 1 merge request, got %d", len(result.MergeRequests))
	}
}
