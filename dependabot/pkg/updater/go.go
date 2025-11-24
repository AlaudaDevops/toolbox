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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	PackageName     string
	FixedVersion    string
	UseLatest       bool
	PackageDir      string
	ResolvedVersion string
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
	latestVersionCache := make(map[string]string)

	// Process each directory separately (for mono repo support)
	for goModDir, updates := range updatesByDir {
		logrus.Debugf("Processing directory: %s with %d package updates", goModDir, len(updates))

		sortedUpdates := sortPackageUpdates(updates)
		for i := range sortedUpdates {
			if g.isGolangStdExtension(sortedUpdates[i].PackageName) {
				sortedUpdates[i].UseLatest = true
			}
		}

		resolvedUpdates := g.resolvePackageVersions(goModDir, sortedUpdates, latestVersionCache)

		// Group packages: golang.org/x/* vs others
		golangXPackages := []packageUpdate{}
		normalPackages := []packageUpdate{}

		for _, update := range resolvedUpdates {
			if update.UseLatest {
				golangXPackages = append(golangXPackages, update)
			} else {
				normalPackages = append(normalPackages, update)
			}
		}

		// Execute batch updates
		updateSuccess := true
		failureReason := ""

		// First, update normal packages with fixed versions
		if len(normalPackages) > 0 {
			logrus.Debugf("Updating %d normal packages with fixed versions", len(normalPackages))
			if err := g.executeBatchGoGet(goModDir, normalPackages); err != nil {
				logrus.Warnf("Failed to update normal packages: %v", err)
				updateSuccess = false
				failedErrors = append(failedErrors, err)
				failureReason = "batch update failed"
			}
		}

		// Then, update golang.org/x/* packages with @latest
		if len(golangXPackages) > 0 {
			logrus.Debugf("Updating %d golang.org/x/* packages with @latest", len(golangXPackages))
			if err := g.executeBatchGoGet(goModDir, golangXPackages); err != nil {
				logrus.Warnf("Failed to update golang.org/x/* packages: %v", err)
				updateSuccess = false
				failedErrors = append(failedErrors, err)
				if failureReason == "" {
					failureReason = "batch update failed"
				}
			}
		}

		// Mark directory for go mod tidy if any update was attempted
		if updateSuccess {
			tidyDirs[goModDir] = true
		}

		g.appendResults(&result, updates, updateSuccess, failureReason)
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

func (g *GoUpdater) appendResults(result *types.VulnFixResults, updates []packageUpdate, success bool, failureReason string) {
	if updates == nil {
		return
	}

	for _, update := range updates {
		fixResult := types.VulnFixResult{
			Vulnerability: types.Vulnerability{
				PackageDir:       update.PackageDir,
				PackageName:      update.PackageName,
				FixedVersion:     update.FixedVersion,
				VulnerabilityIDs: []string{},
			},
			Success: success,
		}
		if !success {
			if failureReason != "" {
				fixResult.Error = failureReason
			} else {
				fixResult.Error = "batch update failed"
			}
		}

		*result = append(*result, fixResult)
	}
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

	updates = sortPackageUpdates(updates)

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
	sortedUpdates := sortPackageUpdates(updates)
	args := make([]string, 0, len(sortedUpdates))
	for _, update := range sortedUpdates {
		args = append(args, g.formatPackageVersion(update))
	}
	return args
}

// sortPackageUpdates returns a new slice sorted by package name and version
func sortPackageUpdates(updates []packageUpdate) []packageUpdate {
	if len(updates) <= 1 {
		return append([]packageUpdate(nil), updates...)
	}

	sorted := make([]packageUpdate, len(updates))
	copy(sorted, updates)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].PackageName == sorted[j].PackageName {
			return sorted[i].FixedVersion < sorted[j].FixedVersion
		}
		return sorted[i].PackageName < sorted[j].PackageName
	})

	return sorted
}

func normalizeVersion(version string) string {
	trimmed := strings.TrimPrefix(version, "@")
	trimmed = strings.TrimPrefix(trimmed, "v")
	return "v" + trimmed
}

func (g *GoUpdater) resolvePackageVersions(goModDir string, updates []packageUpdate, cache map[string]string) []packageUpdate {
	resolved := make([]packageUpdate, len(updates))
	for i, update := range updates {
		resolved[i] = update
		if !update.UseLatest {
			continue
		}

		version, ok := cache[update.PackageName]
		if !ok {
			moduleVersion, err := g.fetchLatestModuleVersion(goModDir, update.PackageName)
			if err != nil {
				logrus.Warnf("Failed to resolve latest version for %s using module context: %v", update.PackageName, err)
				proxyVersion, proxyErr := g.fetchLatestVersionFromProxy(update.PackageName)
				if proxyErr != nil {
					logrus.Warnf("Failed to resolve latest version for %s via proxy: %v, falling back to fixed version %s", update.PackageName, proxyErr, update.FixedVersion)
					cache[update.PackageName] = ""
					resolved[i].ResolvedVersion = update.FixedVersion
					continue
				}
				cache[update.PackageName] = proxyVersion
				version = proxyVersion
			} else {
				cache[update.PackageName] = moduleVersion
				version = moduleVersion
			}
		}

		if version == "" {
			// Cache hit for previous failure, use fixed version deterministically
			resolved[i].ResolvedVersion = update.FixedVersion
			continue
		}

		resolved[i].ResolvedVersion = version
	}

	return resolved
}

func (g *GoUpdater) fetchLatestModuleVersion(goModDir, packageName string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "-mod=mod", fmt.Sprintf("%s@latest", packageName))
	cmd.Dir = goModDir
	env := append([]string{}, os.Environ()...)
	env = append(env, "GO111MODULE=on", "GOWORK=off")
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go list failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	var moduleInfo struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &moduleInfo); err != nil {
		return "", fmt.Errorf("parse go list output: %w", err)
	}
	if moduleInfo.Version == "" {
		return "", fmt.Errorf("empty version returned for %s", packageName)
	}

	return moduleInfo.Version, nil
}

func (g *GoUpdater) fetchLatestVersionFromProxy(packageName string) (string, error) {
	endpoint := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escapeModulePath(packageName))
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("proxy returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var moduleInfo struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&moduleInfo); err != nil {
		return "", fmt.Errorf("parse proxy response: %w", err)
	}
	if moduleInfo.Version == "" {
		return "", fmt.Errorf("proxy response missing version")
	}

	return moduleInfo.Version, nil
}

func escapeModulePath(modulePath string) string {
	segments := strings.Split(modulePath, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// formatPackageVersion formats a package with its version for go get command
func (g *GoUpdater) formatPackageVersion(update packageUpdate) string {
	if update.ResolvedVersion != "" {
		return fmt.Sprintf("%s@%s", update.PackageName, normalizeVersion(update.ResolvedVersion))
	}
	if update.UseLatest {
		return fmt.Sprintf("%s@latest", update.PackageName)
	}
	return fmt.Sprintf("%s@%s", update.PackageName, normalizeVersion(update.FixedVersion))
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
