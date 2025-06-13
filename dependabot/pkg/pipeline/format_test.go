package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
	. "github.com/onsi/gomega"
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

	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read test input
			inputData, err := os.ReadFile(filepath.Join("testdata", tt.inputFile))
			g.Expect(err).NotTo(HaveOccurred(), "Failed to read input file %s", tt.inputFile)

			var updateSummary updater.UpdateSummary
			err = json.Unmarshal(inputData, &updateSummary)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal input data from %s", tt.inputFile)

			// Read expected output
			expected, err := os.ReadFile(filepath.Join("testdata", tt.goldenFile))
			g.Expect(err).NotTo(HaveOccurred(), "Failed to read expected output file %s", tt.goldenFile)

			// Generate commit message
			got := generateCommitMessage(&updateSummary)

			// Remove trailing newlines for comparison
			expectedStr := strings.TrimRight(string(expected), "\n")
			gotStr := strings.TrimRight(got, "\n")
			g.Expect(gotStr).To(Equal(expectedStr))
		})
	}
}
