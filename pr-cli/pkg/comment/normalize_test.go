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

package comment

import "testing"

func TestNormalize(t *testing.T) {
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
			name:     "escaped newline in middle should be preserved",
			input:    "/assign user1\\nuser2",
			expected: "/assign user1\\nuser2",
		},
		{
			name:     "multiple escaped sequences at end",
			input:    "/ready\\r\\n",
			expected: "/ready",
		},
		{
			name:     "multi-line comment with commands",
			input:    "/lgtm\n/ready",
			expected: "/lgtm\n/ready",
		},
		{
			name:     "multi-line comment with extra whitespace",
			input:    "  /lgtm  \n  /ready  ",
			expected: "/lgtm  \n  /ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsMultiLineCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "single command",
			input:    "/lgtm",
			expected: false,
		},
		{
			name:     "two commands",
			input:    "/lgtm\n/ready",
			expected: true,
		},
		{
			name:     "multiple commands with empty lines",
			input:    "/assign user1\n\n/lgtm\n/ready",
			expected: true,
		},
		{
			name:     "commands with spaces",
			input:    "  /lgtm  \n  /ready  ",
			expected: true,
		},
		{
			name:     "non-command text",
			input:    "This is not a command\n/lgtm",
			expected: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: false,
		},
		{
			name:     "only empty lines",
			input:    "\n\n\n",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMultiLineCommand(tt.input)
			if result != tt.expected {
				t.Errorf("IsMultiLineCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitCommandLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single command",
			input:    "/lgtm",
			expected: []string{"/lgtm"},
		},
		{
			name:     "two commands",
			input:    "/lgtm\n/ready",
			expected: []string{"/lgtm", "/ready"},
		},
		{
			name:     "multiple commands with empty lines",
			input:    "/assign user1\n\n/lgtm\n/ready",
			expected: []string{"/assign user1", "/lgtm", "/ready"},
		},
		{
			name:     "commands with spaces",
			input:    "  /lgtm  \n  /ready squash  ",
			expected: []string{"/lgtm", "/ready squash"},
		},
		{
			name:     "mixed content - only commands extracted",
			input:    "This is not a command\n/lgtm\nSome text\n/ready",
			expected: []string{"/lgtm", "/ready"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "no commands",
			input:    "This is not a command\nJust text",
			expected: []string{},
		},
		{
			name:     "commands with CRLF line endings",
			input:    "/label test\r\n/lgtm\r\n/check",
			expected: []string{"/label test", "/lgtm", "/check"},
		},
		{
			name:     "commands with mixed line endings",
			input:    "/label test\r\n/lgtm\n/check\r/ready",
			expected: []string{"/label test", "/lgtm", "/check", "/ready"},
		},
		{
			name:     "multi-line with /lgtm cancel should be transformed",
			input:    "/rebase\n/lgtm cancel\n/ready",
			expected: []string{"/rebase", "/remove-lgtm", "/ready"},
		},
		{
			name:     "single /lgtm cancel should be transformed",
			input:    "/lgtm cancel",
			expected: []string{"/remove-lgtm"},
		},
		{
			name:     "regular /lgtm should not be transformed",
			input:    "/rebase\n/lgtm\n/ready",
			expected: []string{"/rebase", "/lgtm", "/ready"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitCommandLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitCommandLines(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, cmd := range result {
				if cmd != tt.expected[i] {
					t.Errorf("SplitCommandLines(%q)[%d] = %q, want %q", tt.input, i, cmd, tt.expected[i])
				}
			}
		})
	}
}

func TestPreprocessSpecialCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "/lgtm cancel should be transformed to /remove-lgtm",
			input:    "/lgtm cancel",
			expected: "/remove-lgtm",
		},
		{
			name:     "regular /lgtm should not be changed",
			input:    "/lgtm",
			expected: "/lgtm",
		},
		{
			name:     "/lgtm with other args should not be changed",
			input:    "/lgtm with-deps",
			expected: "/lgtm with-deps",
		},
		{
			name:     "other commands should not be changed",
			input:    "/ready",
			expected: "/ready",
		},
		{
			name:     "/remove-lgtm should not be changed",
			input:    "/remove-lgtm",
			expected: "/remove-lgtm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreprocessSpecialCommands(tt.input)
			if result != tt.expected {
				t.Errorf("PreprocessSpecialCommands(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitRawCommandLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single command",
			input:    "/lgtm",
			expected: []string{"/lgtm"},
		},
		{
			name:     "two commands",
			input:    "/lgtm\n/ready",
			expected: []string{"/lgtm", "/ready"},
		},
		{
			name:     "multiple commands with empty lines",
			input:    "/assign user1\n\n/lgtm\n/ready",
			expected: []string{"/assign user1", "/lgtm", "/ready"},
		},
		{
			name:     "commands with spaces",
			input:    "  /lgtm  \n  /ready squash  ",
			expected: []string{"/lgtm", "/ready squash"},
		},
		{
			name:     "mixed content - only commands extracted",
			input:    "This is not a command\n/lgtm\nSome text\n/ready",
			expected: []string{"/lgtm", "/ready"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "no commands",
			input:    "This is not a command\nJust text",
			expected: []string{},
		},
		{
			name:     "commands with CRLF line endings",
			input:    "/label test\r\n/lgtm\r\n/check",
			expected: []string{"/label test", "/lgtm", "/check"},
		},
		{
			name:     "commands with mixed line endings",
			input:    "/label test\r\n/lgtm\n/check\r/ready",
			expected: []string{"/label test", "/lgtm", "/check", "/ready"},
		},
		{
			name:     "multi-line with /lgtm cancel should NOT be transformed (raw)",
			input:    "/rebase\n/lgtm cancel\n/ready",
			expected: []string{"/rebase", "/lgtm cancel", "/ready"},
		},
		{
			name:     "single /lgtm cancel should NOT be transformed (raw)",
			input:    "/lgtm cancel",
			expected: []string{"/lgtm cancel"},
		},
		{
			name:     "regular /lgtm should not be transformed",
			input:    "/rebase\n/lgtm\n/ready",
			expected: []string{"/rebase", "/lgtm", "/ready"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitRawCommandLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitRawCommandLines(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, cmd := range result {
				if cmd != tt.expected[i] {
					t.Errorf("SplitRawCommandLines(%q)[%d] = %q, want %q", tt.input, i, cmd, tt.expected[i])
				}
			}
		})
	}
}
