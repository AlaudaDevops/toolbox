package messages

import (
	"testing"
)

// TestFormatUserMentions tests the FormatUserMentions utility function
func TestFormatUserMentions(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "users without @ prefix",
			input:    []string{"user1", "user2"},
			expected: []string{"@user1", "@user2"},
		},
		{
			name:     "users with @ prefix",
			input:    []string{"@user1", "@user2"},
			expected: []string{"@user1", "@user2"},
		},
		{
			name:     "mixed users",
			input:    []string{"user1", "@user2", "user3"},
			expected: []string{"@user1", "@user2", "@user3"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single user",
			input:    []string{"testuser"},
			expected: []string{"@testuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUserMentions(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("FormatUserMentions() returned %d items, expected %d", len(result), len(tt.expected))
			}
			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("FormatUserMentions() result[%d] = %v, expected %v", i, result[i], expected)
				}
			}
		})
	}
}
