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

// Package updater provides functionality to update vulnerable packages for multiple languages
package updater

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/sirupsen/logrus"
)

// Updater coordinates the updating of vulnerable packages across different languages
// It manages language-specific updaters and orchestrates the update process for all detected vulnerabilities.
type Updater struct {
	// projectPath is the path to the project
	projectPath string
	// languageUpdaters holds language-specific updaters
	languageUpdaters map[types.LanguageType]LanguageUpdater
}

// New creates a new Updater instance with automatic language detection
// projectPath: the path to the project to be updated
// Returns a pointer to an Updater instance
func New(projectPath string, updaterConfig config.UpdaterConfig) *Updater {
	updaters := make(map[types.LanguageType]LanguageUpdater)

	// Pre-register Go updater
	updaters[types.LanguageGo] = NewGoUpdater(projectPath, updaterConfig.Go)

	return &Updater{
		projectPath:      projectPath,
		languageUpdaters: updaters,
	}
}

// UpdatePackages updates vulnerable packages using appropriate language-specific updaters
// Returns detailed update results for PR creation
func (u *Updater) UpdatePackages(vulnerabilities []types.Vulnerability) (types.VulnFixResults, error) {
	result := types.VulnFixResults{}

	if len(vulnerabilities) == 0 {
		logrus.Info("No vulnerabilities to update")
		return result, nil
	}

	// Group vulnerabilities by language
	languageVulns := groupVulnerabilitiesByLanguage(vulnerabilities)

	var allErrors []string

	// Update packages for each detected language
	for language, vulns := range languageVulns {
		logrus.Debugf("Updating %s packages", language)

		updater, exists := u.languageUpdaters[language]
		if !exists {
			errorMsg := fmt.Sprintf("no updater available for language: %s", string(language))
			allErrors = append(allErrors, errorMsg)
			logrus.Errorf("Error: %s", errorMsg)

			// Add failed updates for this language
			for _, vuln := range vulns {
				failedUpdate := types.VulnFixResult{
					Vulnerability: vuln,
					Success:       false,
					Error:         errorMsg,
				}
				result = append(result, failedUpdate)
			}
			continue
		}

		// Language-specific validation is now handled internally by each updater

		// Update packages for this language
		fixResults, err := updater.UpdatePackages(vulns)
		if err != nil {
			allErrors = append(allErrors, err.Error())
			logrus.Errorf("Error: %s", err.Error())
		}
		result = append(result, fixResults...)
	}

	// Generate summary
	successCount := len(result.FixedVulns())
	failedCount := len(result.FixFailedVulns())

	// Print overall summary
	logrus.Debugf("=== Overall Update Summary ===")
	logrus.Debugf("  Total successfully updated: %d packages", successCount)
	logrus.Debugf("  Total failed to update: %d packages", failedCount)

	if len(allErrors) > 0 {
		logrus.Debugf("Overall errors encountered:")
		for _, errorMsg := range allErrors {
			logrus.Debugf("  - %s", errorMsg)
		}
		return result, fmt.Errorf("failed to update %d out of %d packages across all languages", failedCount, result.TotalVulnCount())
	}

	return result, nil
}

// sortVulnerabilities sorts vulnerabilities by multiple fields for stable ordering
// vulns: list of vulnerabilities to process
// Returns sorted vulnerabilities with stable ordering based on field declaration order
func sortVulnerabilities(vulns []types.Vulnerability) []types.Vulnerability {
	// Create a copy to avoid modifying the original slice
	sortedVulns := make([]types.Vulnerability, len(vulns))
	copy(sortedVulns, vulns)

	// Sort by multiple fields in declaration order for stable ordering
	sort.Slice(sortedVulns, func(i, j int) bool {
		keyI := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
			sortedVulns[i].PackageDir,
			sortedVulns[i].PackageName,
			sortedVulns[i].CurrentVersion,
			sortedVulns[i].FixedVersion,
			sortedVulns[i].Severity,
			sortedVulns[i].Language,
			strings.Join(sortedVulns[i].VulnerabilityIDs, ","))

		keyJ := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
			sortedVulns[j].PackageDir,
			sortedVulns[j].PackageName,
			sortedVulns[j].CurrentVersion,
			sortedVulns[j].FixedVersion,
			sortedVulns[j].Severity,
			sortedVulns[j].Language,
			strings.Join(sortedVulns[j].VulnerabilityIDs, ","))

		return keyI < keyJ
	})

	for _, vuln := range sortedVulns {
		logrus.Debug(vuln.String())
	}

	return sortedVulns
}

// groupVulnerabilitiesByLanguage groups vulnerabilities by their target language
// Returns a map with stable ordering of languages and vulnerabilities
func groupVulnerabilitiesByLanguage(vulnerabilities []types.Vulnerability) map[types.LanguageType][]types.Vulnerability {
	// First, group by language
	languageVulns := make(map[types.LanguageType][]types.Vulnerability)

	// Group vulnerabilities by the language field from trivy results
	for _, vuln := range vulnerabilities {
		var language types.LanguageType

		// Map trivy language strings to our LanguageType
		switch vuln.Language {
		case "go":
			language = types.LanguageGo
		case "python":
			language = types.LanguagePython
		case "node":
			language = types.LanguageNode
		default:
			language = types.LanguageUnknown
		}

		// Initialize language map if not exists
		if languageVulns[language] == nil {
			languageVulns[language] = make([]types.Vulnerability, 0)
		}

		languageVulns[language] = append(languageVulns[language], vuln)
	}

	// Sort vulnerabilities within each language for stable ordering
	for language, vulns := range languageVulns {
		languageVulns[language] = sortVulnerabilities(vulns)
	}

	return languageVulns
}

// RegisterLanguageUpdater registers a language-specific updater
// updater: the language-specific updater to register
func (u *Updater) RegisterLanguageUpdater(updater LanguageUpdater) {
	u.languageUpdaters[updater.GetLanguageType()] = updater
}

// GetSupportedLanguages returns the list of supported languages
// Returns a slice of supported LanguageType
func (u *Updater) GetSupportedLanguages() []types.LanguageType {
	languages := make([]types.LanguageType, 0, len(u.languageUpdaters))
	for lang := range u.languageUpdaters {
		languages = append(languages, lang)
	}
	return languages
}

// itoa is a helper function to convert int to string (since strconv.Itoa is not imported)
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
