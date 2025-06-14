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

package pr

import (
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/updater"
)

// PRInfo represents basic information about a pull request
type PRInfo struct {
	// Number is the pull request number
	Number int `json:"number"`
	// Title is the pull request title
	Title string `json:"title"`
	// State is the pull request state (open, closed, merged)
	State string `json:"state"`
}

type PRCreateOption struct {
	Labels        []string              `json:"labels" yaml:"labels"`
	Assignees     []string              `json:"assignees" yaml:"assignees"`
	UpdateSummary updater.UpdateSummary `json:"update_summary" yaml:"update_summary"`
}

// PRCreator defines the interface for creating pull requests
type PRCreator interface {
	// CreatePR creates a pull request based on the update result
	CreatePR(repo *config.RepoConfig, sourceBranch string, option PRCreateOption) error

	// GetPlatformType returns the type of platform (github, gitlab, etc.)
	GetPlatformType() string
}

// NewPRCreator creates a new PRCreator based on the git provider configuration
func NewPRCreator(provider config.GitProviderConfig, workingDir string) (PRCreator, error) {
	switch provider.Provider {
	case "github":
		return NewGitHubPRCreator(provider.BaseURL, provider.Token, workingDir), nil
	case "gitlab":
		return NewGitLabPRCreator(provider.BaseURL, provider.Token, workingDir)
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", provider.Provider)
	}
}

// generatePRTitle generates a title for the pull request
func generatePRTitle(result *updater.UpdateSummary) string {
	successCount := len(result.SuccessfulUpdates)

	if successCount == 1 {
		update := result.SuccessfulUpdates[0]
		return fmt.Sprintf("chore(deps): bump %s from %s to %s", update.PackageName, update.CurrentVersion, update.FixedVersion)
	}

	// Group by language for multiple updates
	languageGroups := make(map[updater.LanguageType]int)
	for _, update := range result.SuccessfulUpdates {
		languageGroups[updater.LanguageType(update.Language)]++
	}

	if len(languageGroups) == 1 {
		// All updates are for the same language
		for language := range languageGroups {
			return fmt.Sprintf("chore(deps): bump %d %s dependencies", successCount, language)
		}
	}

	// Multiple languages
	return fmt.Sprintf("chore(deps): bump %d dependencies across multiple languages", successCount)
}

// GeneratePRBody generates the body/description for the pull request
func GeneratePRBody(result *updater.UpdateSummary) string {
	var body strings.Builder

	body.WriteString("## 🔒 Security Updates\n\n")
	body.WriteString("This pull request updates dependencies to fix security vulnerabilities identified by Trivy scanning.\n\n")

	// Group updates by language
	languageGroups := make(map[updater.LanguageType][]updater.PackageUpdate)
	for _, update := range result.SuccessfulUpdates {
		languageGroups[updater.LanguageType(update.Language)] = append(languageGroups[updater.LanguageType(update.Language)], update)
	}

	// Generate updates by language
	for language, updates := range languageGroups {
		body.WriteString(fmt.Sprintf("### %s Dependencies\n\n", strings.Title(string(language))))

		for _, update := range updates {
			body.WriteString(fmt.Sprintf("- [%s]**%s**(%s): %s → %s\n",
				update.Severity, update.PackageName, update.PackageDir, update.CurrentVersion, update.FixedVersion))

			if len(update.VulnerabilityIDs) > 0 {
				body.WriteString(fmt.Sprintf("  - 🔍 Fixes: %s\n", strings.Join(update.VulnerabilityIDs, ", ")))
			}
		}
		body.WriteString("\n")
	}

	// Add summary information
	body.WriteString("## 📊 Update Summary\n\n")
	body.WriteString(fmt.Sprintf("- **Total packages updated**: %d\n", len(result.SuccessfulUpdates)))
	body.WriteString(fmt.Sprintf("- **Project path**: %s\n", result.ProjectPath))
	body.WriteString(fmt.Sprintf("- **Updated at**: %s\n", result.Timestamp))

	if len(result.FailedUpdates) > 0 {
		body.WriteString(fmt.Sprintf("- **Failed updates**: %d (see logs for details)\n", len(result.FailedUpdates)))
	}

	body.WriteString("\n## 🤖 Automated by DependaBot\n\n")
	body.WriteString("This PR was automatically created by DependaBot based on Trivy security scan results.\n")
	body.WriteString("Please review the changes and merge if everything looks good.\n")

	return body.String()
}
