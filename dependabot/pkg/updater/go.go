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

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/version"
	"github.com/sirupsen/logrus"
)

// GoUpdater handles updating Go packages
type GoUpdater struct {
	*BaseUpdater
	// config contains Go-specific updater configuration
	config *config.GoUpdaterConfig
}

// NewGoUpdater creates a new Go language updater
func NewGoUpdater(projectPath string, goConfig *config.GoUpdaterConfig) *GoUpdater {
	commandOutputFile := ""
	if goConfig != nil {
		commandOutputFile = goConfig.CommandOutputFile
	}

	return &GoUpdater{
		BaseUpdater: NewBaseUpdater(projectPath, commandOutputFile),
		config:      goConfig,
	}
}

// packageUpdate represents a package that needs to be updated
type packageUpdate struct {
	PackageName  string
	FixedVersion string
	UseLatest    bool
	PackageDir   string
}

// UpdatePackages updates vulnerable Go packages to their fixed versions
// Supports mono repo scenarios by grouping vulnerabilities by PackageDir
// Uses batch update strategy to avoid version conflicts, especially for golang.org/x/* packages
func (g *GoUpdater) UpdatePackages(vulnerabilities []types.Vulnerability) (types.VulnFixResults, error) {
	if len(vulnerabilities) == 0 {
		return nil, nil
	}

	logrus.Info("Cleaning Go module cache before running updates...")
	if err := g.cleanGoModuleCache(); err != nil {
		logrus.Warnf("Failed to clean Go module cache: %v", err)
	} else {
		logrus.Debug("Go module cache cleaned successfully")
	}

	// Aggregate vulnerabilities by directory and package
	updatesByDir := g.aggregateVulnerabilities(vulnerabilities)

	result := types.VulnFixResults{}
	failedErrors := make([]error, 0)
	tidyDirs := make(map[string]bool)

	// Process each directory separately (for mono repo support)
	for goModDir, updates := range updatesByDir {
		logrus.Debugf("Processing directory: %s with %d package updates", goModDir, len(updates))

		// Group packages: golang.org/x/* vs others
		golangXPackages := []packageUpdate{}
		normalPackages := []packageUpdate{}

		for _, update := range updates {
			if g.isGolangStdExtension(update.PackageName) {
				update.UseLatest = true
				golangXPackages = append(golangXPackages, update)
			} else {
				normalPackages = append(normalPackages, update)
			}
		}

		// Execute batch updates
		updateSuccess := true

		// First, update normal packages with fixed versions
		if len(normalPackages) > 0 {
			logrus.Debugf("Updating %d normal packages with fixed versions", len(normalPackages))
			if err := g.executeBatchGoGet(goModDir, normalPackages); err != nil {
				logrus.Warnf("Failed to update normal packages: %v", err)
				updateSuccess = false
				failedErrors = append(failedErrors, err)
			}
		}

		// Then, update golang.org/x/* packages with @latest
		if len(golangXPackages) > 0 {
			logrus.Debugf("Updating %d golang.org/x/* packages with @latest", len(golangXPackages))
			if err := g.executeBatchGoGet(goModDir, golangXPackages); err != nil {
				logrus.Warnf("Failed to update golang.org/x/* packages: %v", err)
				updateSuccess = false
				failedErrors = append(failedErrors, err)
			}
		}

		// Mark directory for go mod tidy if any update was attempted
		if updateSuccess {
			tidyDirs[goModDir] = true
		}

		// Create fix results for all vulnerabilities in this directory
		for _, update := range updates {
			fixResult := types.VulnFixResult{
				Vulnerability: types.Vulnerability{
					PackageDir:       update.PackageDir,
					PackageName:      update.PackageName,
					FixedVersion:     update.FixedVersion,
					VulnerabilityIDs: []string{}, // Will be populated from original vulnerabilities
				},
				Success: updateSuccess,
			}
			if !updateSuccess {
				fixResult.Error = "batch update failed"
			}
			result = append(result, fixResult)
		}
	}

	// Run go mod tidy for all affected directories
	for goModDir := range tidyDirs {
		if err := g.runGoModTidy(goModDir); err != nil {
			logrus.Warnf("go mod tidy failed for directory %s: %v", goModDir, err)
			// Don't fail the entire operation for go mod tidy errors
		}

		// Check if vendor directory exists and run go mod vendor
		if g.hasVendorDirectory(goModDir) {
			if err := g.runGoModVendor(goModDir); err != nil {
				logrus.Warnf("go mod vendor failed for directory %s: %v", goModDir, err)
				// Don't fail the entire operation for go mod vendor errors
			}
		}
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
		return result, fmt.Errorf("failed to update %d out of %d packages", result.FixFailedVulnCount(), result.TotalVulnCount())
	}

	return result, nil
}

// GetLanguageType returns the language type this updater handles
func (g *GoUpdater) GetLanguageType() types.LanguageType {
	return types.LanguageGo
}

// aggregateVulnerabilities groups vulnerabilities by directory and package name,
// selecting the highest fixed version for each package
func (g *GoUpdater) aggregateVulnerabilities(vulnerabilities []types.Vulnerability) map[string][]packageUpdate {
	// Group by directory first, then by package
	dirPackageMap := make(map[string]map[string][]string) // goModDir -> packageName -> []fixedVersions

	for _, vuln := range vulnerabilities {
		goModDir := filepath.Join(g.projectPath, strings.TrimSuffix(vuln.PackageDir, "go.mod"))

		if dirPackageMap[goModDir] == nil {
			dirPackageMap[goModDir] = make(map[string][]string)
		}

		dirPackageMap[goModDir][vuln.PackageName] = append(
			dirPackageMap[goModDir][vuln.PackageName],
			vuln.FixedVersion,
		)
	}

	// Convert to packageUpdate list, selecting highest version for each package
	result := make(map[string][]packageUpdate)

	for goModDir, packageVersions := range dirPackageMap {
		updates := []packageUpdate{}

		for packageName, versions := range packageVersions {
			highestVersion := version.GetHighestVersion(versions...)

			updates = append(updates, packageUpdate{
				PackageName:  packageName,
				FixedVersion: highestVersion,
				PackageDir:   strings.TrimPrefix(goModDir, g.projectPath+"/"),
			})

			logrus.Debugf("Package %s: aggregated %d vulnerabilities, highest fixed version: %s",
				packageName, len(versions), highestVersion)
		}

		result[goModDir] = updates
	}

	return result
}

// isGolangStdExtension checks if a package is part of golang.org/x/* or google.golang.org/*
// These packages are commonly used as transitive dependencies and should use @latest to avoid conflicts
func (g *GoUpdater) isGolangStdExtension(packageName string) bool {
	prefixes := []string{
		"golang.org/x/",
		"google.golang.org/",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(packageName, prefix) {
			return true
		}
	}

	return false
}

// executeBatchGoGet executes a batch go get command for multiple packages
// If batch update fails, it falls back to updating packages individually
func (g *GoUpdater) executeBatchGoGet(goModDir string, updates []packageUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// Build batch command
	batchCmd := g.buildBatchCommand(updates)

	// Try batch update first
	logrus.Debugf("Attempting batch update: %s", batchCmd)

	packageArgs := g.buildPackageArgs(updates)
	args := append([]string{"get"}, packageArgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = goModDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err == nil && !strings.Contains(string(output), "no matching versions") {
		// Batch update succeeded
		logrus.Debugf("Batch update succeeded: %d packages", len(updates))

		if err := g.LogSuccessfulCommand(batchCmd); err != nil {
			logrus.Warnf("Failed to log successful command: %v", err)
		}

		return nil
	}

	// Batch update failed, log the failure and fall back to individual updates
	logrus.Warnf("Batch update failed (error: %v, output: %s), falling back to individual updates", err, string(output))

	if err := g.LogBlankLine(); err != nil {
		logrus.Warnf("Failed to log blank line: %v", err)
	}

	if err := g.LogComment("Batch update failed, retrying packages individually:"); err != nil {
		logrus.Warnf("Failed to log comment: %v", err)
	}

	if err := g.LogComment(fmt.Sprintf("Original batch: %s", batchCmd)); err != nil {
		logrus.Warnf("Failed to log comment: %v", err)
	}

	if err := g.LogBlankLine(); err != nil {
		logrus.Warnf("Failed to log blank line: %v", err)
	}

	// Try updating packages individually
	successCount := 0
	failedPackages := []string{}

	for _, update := range updates {
		singleCmd := g.buildSingleCommand(update)

		if err := g.executeSingleGoGet(goModDir, update); err != nil {
			logrus.Warnf("Failed to update %s: %v", update.PackageName, err)
			failedPackages = append(failedPackages, update.PackageName)

			// Log failed command
			if logErr := g.LogFailedCommand(singleCmd, err); logErr != nil {
				logrus.Warnf("Failed to log failed command: %v", logErr)
			}
		} else {
			logrus.Debugf("Successfully updated %s", update.PackageName)
			successCount++

			// Log successful command
			if logErr := g.LogSuccessfulCommand(singleCmd); logErr != nil {
				logrus.Warnf("Failed to log successful command: %v", logErr)
			}
		}
	}

	if err := g.LogBlankLine(); err != nil {
		logrus.Warnf("Failed to log blank line: %v", err)
	}

	if err := g.LogComment(fmt.Sprintf("Individual updates: %d/%d succeeded", successCount, len(updates))); err != nil {
		logrus.Warnf("Failed to log comment: %v", err)
	}

	logrus.Infof("Individual updates completed: %d/%d succeeded", successCount, len(updates))

	// Return error if some packages failed
	if len(failedPackages) > 0 {
		return fmt.Errorf("failed to update %d packages: %v", len(failedPackages), failedPackages)
	}

	return nil
}

// buildBatchCommand builds the complete batch go get command string
func (g *GoUpdater) buildBatchCommand(updates []packageUpdate) string {
	packageArgs := g.buildPackageArgs(updates)
	return "go get " + strings.Join(packageArgs, " ")
}

// buildSingleCommand builds a go get command for a single package
func (g *GoUpdater) buildSingleCommand(update packageUpdate) string {
	packageWithVersion := g.formatPackageVersion(update)
	return "go get " + packageWithVersion
}

// buildPackageArgs builds the list of package@version arguments
func (g *GoUpdater) buildPackageArgs(updates []packageUpdate) []string {
	args := make([]string, 0, len(updates))
	for _, update := range updates {
		args = append(args, g.formatPackageVersion(update))
	}
	return args
}

// formatPackageVersion formats a package with its version for go get command
func (g *GoUpdater) formatPackageVersion(update packageUpdate) string {
	if update.UseLatest {
		return fmt.Sprintf("%s@latest", update.PackageName)
	}
	return fmt.Sprintf("%s@v%s", update.PackageName, strings.TrimPrefix(update.FixedVersion, "v"))
}

// executeSingleGoGet executes go get for a single package
func (g *GoUpdater) executeSingleGoGet(goModDir string, update packageUpdate) error {
	packageWithVersion := g.formatPackageVersion(update)

	cmd := exec.Command("go", "get", packageWithVersion)
	cmd.Dir = goModDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("go get failed: %w, output: %s", err, string(output))
	}

	// Check for warnings
	outputStr := string(output)
	if strings.Contains(outputStr, "no matching versions") {
		return fmt.Errorf("no matching version found for %s", packageWithVersion)
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

	// Log successful command to output file if configured
	if err := g.LogSuccessfulCommand("go mod tidy"); err != nil {
		logrus.Warnf("Failed to log successful command: %v", err)
	}

	return nil
}

// hasVendorDirectory checks if a vendor directory exists in the given directory
func (g *GoUpdater) hasVendorDirectory(goModDir string) bool {
	vendorPath := filepath.Join(goModDir, "vendor")
	info, err := os.Stat(vendorPath)
	return err == nil && info.IsDir()
}

// runGoModVendor runs 'go mod vendor' command to update vendor directory
func (g *GoUpdater) runGoModVendor(goModDir string) error {
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = goModDir

	// Set environment variables for go command
	cmd.Env = os.Environ()

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("go mod vendor failed: %w, output: %s", err, string(output))
	}

	// Log successful command to output file if configured
	if err := g.LogSuccessfulCommand("go mod vendor"); err != nil {
		logrus.Warnf("Failed to log successful command: %v", err)
	}

	return nil
}

// cleanGoModuleCache clears the Go module cache to control disk usage during bulk updates
func (g *GoUpdater) cleanGoModuleCache() error {
	cmd := exec.Command("go", "clean", "-modcache")
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go clean -modcache failed: %w, output: %s", err, string(output))
	}

	return nil
}
