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

// Package updater provides Go-specific package update functionality
package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/sirupsen/logrus"
)

// GoUpdater handles updating Go packages
type GoUpdater struct {
	// projectPath is the path to the project containing go.mod
	projectPath string
}

// NewGoUpdater creates a new Go language updater
func NewGoUpdater(projectPath string) *GoUpdater {
	return &GoUpdater{
		projectPath: projectPath,
	}
}

// UpdatePackages updates vulnerable Go packages to their fixed versions
// Supports mono repo scenarios by grouping vulnerabilities by PackageDir
func (g *GoUpdater) UpdatePackages(vulnerabilities []types.Vulnerability) (types.VulnFixResults, error) {
	if len(vulnerabilities) == 0 {
		return nil, nil
	}

	result := types.VulnFixResults{}
	failedErrors := make([]error, 0, len(vulnerabilities))
	// Process each directory separately
	for _, vuln := range vulnerabilities {
		err := g.updatePackage(vuln)
		if err != nil {
			failedErrors = append(failedErrors, err)
		}
		result = append(result, types.VulnFixResult{
			Vulnerability: vuln,
			Success:       err == nil,
			Error:         err.Error(),
		})
	}

	// Print overall summary
	logrus.Debugf("=== Overall Golang Update Summary ===")
	logrus.Debugf("  Successfully updated: %d packages", result.FixedVulnCount())
	logrus.Debugf("  Failed to update: %d packages", result.FixFailedVulnCount())

	if len(failedErrors) > 0 {
		logrus.Debugf("Errors encountered:")
		for _, err := range failedErrors {
			logrus.Debugf("  - %s", err.Error())
		}
		return result, fmt.Errorf("failed to update %d out of %d packages", result.FixedVulnCount(), result.TotalVulnCount())
	}

	return result, nil
}

// GetLanguageType returns the language type this updater handles
func (g *GoUpdater) GetLanguageType() types.LanguageType {
	return types.LanguageGo
}

// updatePackage updates a single Go package using go get
// Note: This method assumes the current working directory is already set to the correct package directory
func (g *GoUpdater) updatePackage(vuln types.Vulnerability) error {
	// Construct the go get command
	packageWithVersion := fmt.Sprintf("%s@v%s", vuln.PackageName, strings.TrimPrefix(vuln.FixedVersion, "v"))

	logrus.Debugf("Updating package: %s(%s)", vuln.PackageName, vuln.PackageDir)

	goModDir := filepath.Join(g.projectPath, strings.TrimSuffix(vuln.PackageDir, "go.mod"))

	cmd := exec.Command("go", "get", packageWithVersion)
	cmd.Dir = goModDir

	// Set environment variables for go command
	cmd.Env = os.Environ()

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("go get failed: %w, output: %s", err, string(output))
	}

	// Check if the output contains any warnings or errors
	outputStr := string(output)
	if strings.Contains(outputStr, "no matching versions") {
		return fmt.Errorf("no matching version found for %s", packageWithVersion)
	}

	err = g.runGoModTidy(goModDir)
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	return nil
}

// runGoModTidy runs 'go mod tidy' command to update go.sum
func (g *GoUpdater) runGoModTidy(goModDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = goModDir

	// Set environment variables for go command
	cmd.Env = os.Environ()

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w, output: %s", err, string(output))
	}

	return nil
}
