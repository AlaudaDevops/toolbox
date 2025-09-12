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

package cmd

import (
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
)

func TestNormalizeComment(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple command",
			input:    "/ready",
			expected: "/ready",
		},
		{
			name:     "command with escaped newline at end",
			input:    "/ready\\n",
			expected: "/ready",
		},
		{
			name:     "command with escaped carriage return at end",
			input:    "/ready\\r",
			expected: "/ready",
		},
		{
			name:     "command with actual newline",
			input:    "/ready\n",
			expected: "/ready",
		},
		{
			name:     "command with spaces and escaped newline",
			input:    "  /ready\\n  ",
			expected: "/ready",
		},
		{
			name:     "command with arguments and escaped newline",
			input:    "/merge squash\\n",
			expected: "/merge squash",
		},
		{
			name:     "escaped newline in middle should be preserved",
			input:    "/assign user1\\nuser2",
			expected: "/assign user1\\nuser2",
		},
		{
			name:     "command with tab character (not escaped sequence)",
			input:    "/assign\tuser1",
			expected: "/assign\tuser1",
		},
		{
			name:     "command with escaped tab in middle should be preserved",
			input:    "/assign user1\\tuser2",
			expected: "/assign user1\\tuser2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
		{
			name:     "escaped newline in middle of whitespace",
			input:    "  \\n  ",
			expected: "",
		},
		{
			name:     "multiple escaped sequences at end",
			input:    "/ready\\r\\n",
			expected: "/ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comment.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("comment.Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseCommandWithNormalization(t *testing.T) {
	option := NewPROption()

	tests := []struct {
		name         string
		comment      string
		expectedCmd  string
		expectedArgs []string
		expectError  bool
	}{
		{
			name:         "simple ready command",
			comment:      "/ready",
			expectedCmd:  "ready",
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			name:         "ready command with escaped newline",
			comment:      "/ready\\n",
			expectedCmd:  "ready",
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			name:         "merge command with method",
			comment:      "/merge squash",
			expectedCmd:  "merge",
			expectedArgs: []string{"squash"},
			expectError:  false,
		},
		{
			name:         "assign command with users",
			comment:      "/assign user1 user2",
			expectedCmd:  "assign",
			expectedArgs: []string{"user1", "user2"},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First normalize the comment
			normalizedComment := comment.Normalize(tt.comment)

			cmd, args, err := option.parseCommand(normalizedComment)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseCommand(%q) expected error, got none", tt.comment)
				}
				return
			}

			if err != nil {
				t.Errorf("parseCommand(%q) unexpected error: %v", tt.comment, err)
				return
			}

			if cmd != tt.expectedCmd {
				t.Errorf("parseCommand(%q) cmd = %q, want %q", tt.comment, cmd, tt.expectedCmd)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("parseCommand(%q) args length = %d, want %d", tt.comment, len(args), len(tt.expectedArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("parseCommand(%q) args[%d] = %q, want %q", tt.comment, i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}
