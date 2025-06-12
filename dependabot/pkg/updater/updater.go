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
	"time"

	"github.com/sirupsen/logrus"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/scanner"
)

// Updater coordinates the updating of vulnerable packages across different languages
// It manages language-specific updaters and orchestrates the update process for all detected vulnerabilities.
type Updater struct {
	// projectPath is the path to the project
	projectPath string
	// languageUpdaters holds language-specific updaters
	languageUpdaters map[LanguageType]LanguageUpdater
}

// New creates a new Updater instance with automatic language detection
// projectPath: the path to the project to be updated
// Returns a pointer to an Updater instance
func New(projectPath string) *Updater {
	updaters := make(map[LanguageType]LanguageUpdater)

	// Pre-register Go updater
	updaters[LanguageGo] = NewGoUpdater(projectPath)

	return &Updater{
		projectPath:      projectPath,
		languageUpdaters: updaters,
	}
}

// UpdatePackages updates vulnerable packages using appropriate language-specific updaters
// Returns detailed update results for PR creation
func (u *Updater) UpdatePackages(vulnerabilities []scanner.Vulnerability) (*UpdateSummary, error) {
	startTime := time.Now()

	result := &UpdateSummary{
		ProjectPath:       u.projectPath,
		TotalPackages:     0,
		SuccessfulUpdates: []PackageUpdate{},
		FailedUpdates:     []PackageUpdate{},
		Timestamp:         startTime.Format(time.RFC3339),
	}

	if len(vulnerabilities) == 0 {
		logrus.Info("No vulnerabilities to update")
		result.Summary = "No vulnerabilities found to update"
		return result, nil
	}

	// Group vulnerabilities by language
	languageVulns := groupVulnerabilitiesByLanguage(vulnerabilities)

	if len(languageVulns) == 0 {
		logrus.Info("No vulnerabilities found for supported languages")
		result.Summary = "No vulnerabilities found for supported languages"
		return result, nil
	}

	var allErrors []string

	// Update packages for each detected language
	for language, vulns := range languageVulns {
		logrus.Debugf("Updating %s packages", language)

		updater, exists := u.languageUpdaters[language]
		if !exists {
			errorMsg := "no updater available for language: " + string(language)
			allErrors = append(allErrors, errorMsg)
			logrus.Errorf("Error: %s", errorMsg)

			// Add failed updates for this language
			for _, vuln := range vulns {
				failedUpdate := PackageUpdate{
					Vulnerability: vuln,
					Success:       false,
					Error:         errorMsg,
				}
				result.FailedUpdates = append(result.FailedUpdates, failedUpdate)
				result.TotalPackages++
			}
			continue
		}

		// Language-specific validation is now handled internally by each updater

		// Update packages for this language
		if err := updater.UpdatePackages(vulns); err != nil {
			errorMsg := "failed to update " + string(language) + " packages: " + err.Error()
			allErrors = append(allErrors, errorMsg)
			logrus.Errorf("Error: %s", errorMsg)

			// Add failed updates for this language
			for _, vuln := range vulns {
				failedUpdate := PackageUpdate{
					Vulnerability: vuln,
					Success:       false,
					Error:         err.Error(),
				}
				result.FailedUpdates = append(result.FailedUpdates, failedUpdate)
				result.TotalPackages++
			}
		} else {
			// Add successful updates
			for _, vuln := range vulns {
				successUpdate := PackageUpdate{
					Vulnerability: vuln,
					Success:       true,
					Error:         "",
				}
				result.SuccessfulUpdates = append(result.SuccessfulUpdates, successUpdate)
				result.TotalPackages++
			}
		}
	}

	// Generate summary
	successCount := len(result.SuccessfulUpdates)
	failedCount := len(result.FailedUpdates)

	// Print overall summary
	logrus.Debugf("=== Overall Update Summary ===")
	logrus.Debugf("  Total successfully updated: %d packages", successCount)
	logrus.Debugf("  Total failed to update: %d packages", failedCount)

	result.Summary = "Updated " + itoa(successCount) + " packages successfully, " + itoa(failedCount) + " failed"

	if len(allErrors) > 0 {
		logrus.Debugf("Overall errors encountered:")
		for _, errorMsg := range allErrors {
			logrus.Debugf("  - %s", errorMsg)
		}
		return result, fmt.Errorf("failed to update %d out of %d packages across all languages", failedCount, result.TotalPackages)
	}

	return result, nil
}

// groupVulnerabilitiesByLanguage groups vulnerabilities by their target language
// and merges multiple vulnerabilities for the same package, selecting the highest fixed version
func groupVulnerabilitiesByLanguage(vulnerabilities []scanner.Vulnerability) map[LanguageType][]scanner.Vulnerability {
	// First, group by language
	languageVulns := make(map[LanguageType][]scanner.Vulnerability)

	// Group vulnerabilities by the language field from trivy results
	for _, vuln := range vulnerabilities {
		var language LanguageType

		// Map trivy language strings to our LanguageType
		switch vuln.Language {
		case "go":
			language = LanguageGo
		case "python":
			language = LanguagePython
		case "node":
			language = LanguageNode
		default:
			language = LanguageUnknown
		}

		// Initialize language map if not exists
		if languageVulns[language] == nil {
			languageVulns[language] = make([]scanner.Vulnerability, 0)
		}

		languageVulns[language] = append(languageVulns[language], vuln)
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
func (u *Updater) GetSupportedLanguages() []LanguageType {
	languages := make([]LanguageType, 0, len(u.languageUpdaters))
	for lang := range u.languageUpdaters {
		languages = append(languages, lang)
	}
	return languages
}

// itoa is a helper function to convert int to string (since strconv.Itoa is not imported)
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
