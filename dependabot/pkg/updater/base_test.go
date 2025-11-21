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

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaseUpdater_LogSuccessfulCommand(t *testing.T) {
	tests := []struct {
		name              string
		projectPath       string
		commandOutputFile string
		command           string
		wantErr           bool
		setupFunc         func() // Optional setup function
		cleanupFunc       func() // Optional cleanup function
	}{
		{
			name:              "empty output file should return nil",
			projectPath:       "", // Will be set in test
			commandOutputFile: "",
			command:           "go mod tidy",
			wantErr:           false,
		},
		{
			name:              "successful command logging",
			projectPath:       "", // Will be set in test
			commandOutputFile: "commands.log",
			command:           "go mod tidy",
			wantErr:           false,
		},
		{
			name:              "multiple commands logging",
			projectPath:       "", // Will be set in test
			commandOutputFile: "commands.log",
			command:           "go mod download",
			wantErr:           false,
		},
		{
			name:              "create nested directory structure",
			projectPath:       "", // Will be set in test
			commandOutputFile: "logs/debug/commands.log",
			command:           "go build",
			wantErr:           false,
		},
		{
			name:              "command with special characters",
			projectPath:       "", // Will be set in test
			commandOutputFile: "commands.log",
			command:           "go test -v ./... && echo 'test completed'",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a separate temporary directory for each test case
			tempDir := t.TempDir()
			tt.projectPath = tempDir

			updater := NewBaseUpdater(tt.projectPath, tt.commandOutputFile)

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			err := updater.LogSuccessfulCommand(tt.command)

			if tt.cleanupFunc != nil {
				tt.cleanupFunc()
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("BaseUpdater.LogSuccessfulCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If no error expected and output file is specified, verify the file was created and contains the command
			if !tt.wantErr && tt.commandOutputFile != "" {
				outputPath := filepath.Join(tt.projectPath, tt.commandOutputFile)
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("Expected output file to be created at %s", outputPath)
				}

				// Read the file content to verify the command was written
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Errorf("Failed to read output file: %v", err)
				}

				// Check that the file contains the expected header
				if !strings.HasPrefix(string(content), CommandOutputHeader) {
					t.Errorf("File content does not start with expected header. Got: %q", string(content))
				}

				// Check that the command was appended after the header
				expectedCommand := tt.command + "\n"
				if !strings.Contains(string(content), expectedCommand) {
					t.Errorf("File content does not contain expected command. Got: %q, want to contain: %q", string(content), expectedCommand)
				}
			}
		})
	}
}

func TestBaseUpdater_LogSuccessfulCommand_AppendMode(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := "commands.log"
	updater := NewBaseUpdater(tempDir, outputFile)

	// Log first command
	err := updater.LogSuccessfulCommand("first command")
	if err != nil {
		t.Fatalf("Failed to log first command: %v", err)
	}

	// Log second command
	err = updater.LogSuccessfulCommand("second command")
	if err != nil {
		t.Fatalf("Failed to log second command: %v", err)
	}

	// Verify both commands are in the file
	outputPath := filepath.Join(tempDir, outputFile)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check that the file contains the expected header
	if !strings.HasPrefix(string(content), CommandOutputHeader) {
		t.Errorf("File content does not start with expected header. Got: %q", string(content))
	}

	// Check that both commands are appended after the header
	expectedContent := CommandOutputHeader + "first command\nsecond command\n"
	if string(content) != expectedContent {
		t.Errorf("File content = %q, want %q", string(content), expectedContent)
	}
}

func TestBaseUpdater_LogComment(t *testing.T) {
	tests := []struct {
		name            string
		comment         string
		wantContent     string
	}{
		{
			name:        "comment with hash prefix",
			comment:     "# This is a comment",
			wantContent: "# This is a comment\n",
		},
		{
			name:        "comment without hash prefix",
			comment:     "This is a comment",
			wantContent: "# This is a comment\n",
		},
		{
			name:        "empty comment",
			comment:     "",
			wantContent: "# \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outputFile := "commands.log"
			updater := NewBaseUpdater(tempDir, outputFile)

			err := updater.LogComment(tt.comment)
			if err != nil {
				t.Errorf("LogComment() error = %v", err)
				return
			}

			// Read file and verify
			outputPath := filepath.Join(tempDir, outputFile)
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			expectedContent := CommandOutputHeader + tt.wantContent
			if string(content) != expectedContent {
				t.Errorf("File content = %q, want %q", string(content), expectedContent)
			}
		})
	}
}

func TestBaseUpdater_LogFailedCommand(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := "commands.log"
	updater := NewBaseUpdater(tempDir, outputFile)

	testErr := fmt.Errorf("version conflict")
	err := updater.LogFailedCommand("go get pkg@v1.0.0", testErr)
	if err != nil {
		t.Fatalf("LogFailedCommand() error = %v", err)
	}

	// Read file and verify
	outputPath := filepath.Join(tempDir, outputFile)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedContent := CommandOutputHeader +
		"# FAILED: go get pkg@v1.0.0\n" +
		"# Error: version conflict\n"

	if string(content) != expectedContent {
		t.Errorf("File content = %q, want %q", string(content), expectedContent)
	}
}

func TestBaseUpdater_LogBlankLine(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := "commands.log"
	updater := NewBaseUpdater(tempDir, outputFile)

	err := updater.LogBlankLine()
	if err != nil {
		t.Fatalf("LogBlankLine() error = %v", err)
	}

	// Read file and verify
	outputPath := filepath.Join(tempDir, outputFile)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedContent := CommandOutputHeader + "\n"
	if string(content) != expectedContent {
		t.Errorf("File content = %q, want %q", string(content), expectedContent)
	}
}

func TestBaseUpdater_MixedLogging(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := "commands.log"
	updater := NewBaseUpdater(tempDir, outputFile)

	// Simulate a realistic logging scenario
	_ = updater.LogComment("Starting updates")
	_ = updater.LogSuccessfulCommand("go get pkg1@v1.0.0")
	_ = updater.LogBlankLine()
	_ = updater.LogComment("Batch update failed, retrying individually")
	_ = updater.LogSuccessfulCommand("go get pkg2@v2.0.0")
	_ = updater.LogFailedCommand("go get pkg3@v3.0.0", fmt.Errorf("version conflict"))

	// Read file and verify structure
	outputPath := filepath.Join(tempDir, outputFile)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedContent := CommandOutputHeader +
		"# Starting updates\n" +
		"go get pkg1@v1.0.0\n" +
		"\n" +
		"# Batch update failed, retrying individually\n" +
		"go get pkg2@v2.0.0\n" +
		"# FAILED: go get pkg3@v3.0.0\n" +
		"# Error: version conflict\n"

	if string(content) != expectedContent {
		t.Errorf("File content = %q, want %q", string(content), expectedContent)
	}
}
