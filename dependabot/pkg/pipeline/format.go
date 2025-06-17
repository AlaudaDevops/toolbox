package pipeline

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
)

// generateCommitMessage generates a commit message for the updates
func generateCommitMessage(result types.VulnFixResults) string {
	fixedVulns := result.FixedVulns()
	if len(fixedVulns) == 1 {
		update := fixedVulns[0]
		return fmt.Sprintf("fix: update %s from %s to %s\n\nFixes security vulnerabilities: %v",
			update.PackageName,
			update.CurrentVersion,
			update.FixedVersion,
			update.VulnerabilityIDs)
	}

	return fmt.Sprintf("fix: update %d vulnerable dependencies\n\nSecurity updates for multiple packages",
		len(fixedVulns))
}
