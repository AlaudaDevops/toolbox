package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		inputFile  string
		goldenFile string
	}{
		{
			name:       "single update",
			inputFile:  "single_update.json",
			goldenFile: "single_update.golden",
		},
		{
			name:       "multiple updates",
			inputFile:  "multiple_updates.json",
			goldenFile: "multiple_updates.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read test input
			inputData, err := os.ReadFile(filepath.Join("testdata", tt.inputFile))
			require.NoError(t, err)

			var updateSummary updater.UpdateSummary
			err = json.Unmarshal(inputData, &updateSummary)
			require.NoError(t, err)

			// Read expected output
			expected, err := os.ReadFile(filepath.Join("testdata", tt.goldenFile))
			require.NoError(t, err)

			// Generate commit message
			got := generateCommitMessage(&updateSummary)

			// Remove trailing newlines for comparison
			expectedStr := strings.TrimRight(string(expected), "\n")
			gotStr := strings.TrimRight(got, "\n")
			assert.Equal(t, expectedStr, gotStr)
		})
	}
}
