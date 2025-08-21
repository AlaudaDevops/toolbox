/*
Copyright 2024 The AlaudaDevops Authors.

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

package jira

import (
	// "context"
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		username string
		password string
		project  string
		wantErr  bool
	}{
		{
			name:     "valid parameters",
			baseURL:  "https://test.atlassian.net",
			username: "test@example.com",
			password: "test-token",
			project:  "TEST",
			wantErr:  false,
		},
		{
			name:     "invalid base URL",
			baseURL:  "invalid-url",
			username: "test@example.com",
			password: "test-token",
			project:  "TEST",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL, tt.username, tt.password, tt.project)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("NewClient() returned nil client")
			}
		})
	}
}

func TestCreateMilestoneRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.CreateMilestoneRequest
		wantErr bool
	}{
		{
			name: "valid quarter format",
			req: models.CreateMilestoneRequest{
				Name:     "Test Milestone",
				Quarter:  "2025Q1",
				PillarID: "123",
			},
			wantErr: false,
		},
		{
			name: "invalid quarter format - no Q",
			req: models.CreateMilestoneRequest{
				Name:     "Test Milestone",
				Quarter:  "20251",
				PillarID: "123",
			},
			wantErr: true,
		},
		{
			name: "invalid quarter format - wrong quarter number",
			req: models.CreateMilestoneRequest{
				Name:     "Test Milestone",
				Quarter:  "2025Q5",
				PillarID: "123",
			},
			wantErr: true,
		},
		{
			name: "invalid quarter format - wrong year",
			req: models.CreateMilestoneRequest{
				Name:     "Test Milestone",
				Quarter:  "25Q1",
				PillarID: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// Mock test for client methods (would require actual Jira instance for integration tests)
func TestClient_Methods(t *testing.T) {
	// This is a placeholder for integration tests
	// In a real scenario, you would either:
	// 1. Use a test Jira instance
	// 2. Mock the HTTP responses
	// 3. Use dependency injection to mock the underlying Jira client

	t.Skip("Integration tests require actual Jira instance or mocking")

	// Example of what integration tests would look like:
	/*
		client, err := NewClient("https://test.atlassian.net", "test", "token", "TEST")
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()

		// Test connection
		err = client.TestConnection(ctx)
		if err != nil {
			t.Errorf("TestConnection() failed: %v", err)
		}

		// Test getting pillars
		pillars, err := client.GetPillars(ctx)
		if err != nil {
			t.Errorf("GetPillars() failed: %v", err)
		}

		if len(pillars) == 0 {
			t.Log("No pillars found (this might be expected for test environment)")
		}
	*/
}
