package pipeline

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
)

// generateCommitMessage generates a commit message for the updates
func generateCommitMessage(result *updater.UpdateSummary) string {
	if len(result.SuccessfulUpdates) == 1 {
		update := result.SuccessfulUpdates[0]
		return fmt.Sprintf("fix: update %s from %s to %s\n\nFixes security vulnerabilities: %v",
			update.PackageName,
			update.CurrentVersion,
			update.FixedVersion,
			update.VulnerabilityIDs)
	}

	return fmt.Sprintf("fix: update %d vulnerable dependencies\n\nSecurity updates for multiple packages",
		len(result.SuccessfulUpdates))
}
