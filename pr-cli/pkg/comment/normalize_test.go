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
