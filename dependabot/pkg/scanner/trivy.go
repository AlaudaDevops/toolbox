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

// Package scanner provides Trivy-specific scanner implementation
package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/version"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
	"github.com/sirupsen/logrus"
)

// TrivyScanner implements the Scanner interface for Trivy security scanner
type TrivyScanner struct {
	// scanPath is the directory to scan
	scanPath string
	// tempDir is the temporary directory for scan results
	tempDir string
	// config contains scanner configuration
	config config.ScannerConfig
}

// NewTrivyScanner creates a new Trivy scanner instance
func NewTrivyScanner(scanPath string, config config.ScannerConfig) *TrivyScanner {
	return &TrivyScanner{
		scanPath: scanPath,
		config:   config,
	}
}

// GetName returns the name of the scanner
func (t *TrivyScanner) GetName() string {
	return "trivy"
}

// Scan performs the security scan and returns vulnerabilities
func (t *TrivyScanner) Scan() ([]types.Vulnerability, error) {
	logrus.Infof("Running %s scan", t.GetName())

	// Check if trivy is installed
	if err := CheckTrivyInstalled(); err != nil {
		return nil, fmt.Errorf("trivy CLI check failed: %w", err)
	}

	// Create temporary directory for scan results
	tempDir, err := os.MkdirTemp("", "dependabot-trivy-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	t.tempDir = tempDir

	// Generate temporary file path for results
	resultsFile := filepath.Join(tempDir, "trivy-results.json")

	logrus.Debugf("Running Trivy scan on directory: %s", t.scanPath)
	logrus.Debugf("Scan results will be saved to: %s", resultsFile)

	// Build command arguments from configuration
	args, err := t.buildCommandArgs(resultsFile)
	if err != nil {
		t.Cleanup()
		return nil, fmt.Errorf("failed to build command arguments: %w", err)
	}

	// Log the command being executed
	logrus.Debugf("Executing: trivy %s", strings.Join(args, " "))

	// Execute trivy command
	cmd := exec.Command("trivy", args...)
	cmd.Dir = t.scanPath

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp directory on error
		t.Cleanup()
		return nil, fmt.Errorf("trivy scan failed: %w, output: %s", err, string(output))
	}

	// Print trivy output for user information
	if len(output) > 0 {
		logrus.Debugf("Trivy scan output:%s", string(output))
	}

	// Verify that the results file was created
	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		t.Cleanup()
		return nil, fmt.Errorf("trivy results file was not created: %s", resultsFile)
	}

	logrus.Info("✅ Trivy scan completed successfully")
	logrus.Debugf("Results saved to: %s", resultsFile)

	// Parse the results
	vulnerabilities, err := ParseTrivyResults(resultsFile)
	if err != nil {
		t.Cleanup()
		return nil, fmt.Errorf("failed to parse trivy results: %w", err)
	}

	return vulnerabilities, nil
}

// Cleanup removes the temporary directory and all its contents
func (t *TrivyScanner) Cleanup() error {
	if t.tempDir == "" {
		return nil
	}

	logrus.Debugf("Cleaning up temporary directory: %s", t.tempDir)

	err := os.RemoveAll(t.tempDir)
	if err != nil {
		return fmt.Errorf("failed to cleanup temp directory %s: %w", t.tempDir, err)
	}

	t.tempDir = ""
	return nil
}

// buildCommandArgs builds the trivy command arguments from configuration
func (t *TrivyScanner) buildCommandArgs(resultsFile string) ([]string, error) {
	args := []string{
		"repo",
		"--format", "json",
		"--output", resultsFile,
	}

	// Apply timeout if specified
	if t.config.Timeout != "" {
		args = append(args, "--timeout", t.config.Timeout)
	}

	// Process parameters from configuration
	if len(t.config.Params) > 0 {
		args = append(args, t.config.Params...)
	} else {
		// Default parameters if none specified
		args = append(args, "--scanners", "vuln")
	}

	// Add target directory
	args = append(args, t.scanPath)

	return args, nil
}

// CheckTrivyInstalled checks if trivy CLI is available
func CheckTrivyInstalled() error {
	cmd := exec.Command("trivy", "version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trivy CLI is not installed or not available in PATH: %wOutput: %s", err, string(output))
	}

	logrus.Debugf("Trivy CLI is available:%s", string(output))
	return nil
}

// ParseTrivyResults parses trivy scan results from a JSON file and returns Go package vulnerabilities
func ParseTrivyResults(filePath string) ([]types.Vulnerability, error) {
	if filePath == "" {
		return nil, fmt.Errorf("trivy result file path cannot be empty")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open trivy result file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read trivy result file: %w", err)
	}

	var trivyReport trivyTypes.Report
	if err := json.Unmarshal(data, &trivyReport); err != nil {
		return nil, fmt.Errorf("failed to parse trivy result JSON: %w", err)
	}

	return extractGoVulnerabilities(trivyReport), nil
}

// extractGoVulnerabilities extracts Go package vulnerabilities from trivy results
func extractGoVulnerabilities(trivyReport trivyTypes.Report) []types.Vulnerability {
	var vulnerabilities []types.Vulnerability

	for _, result := range trivyReport.Results {
		// Only process Go module targets
		if !isGoTarget(result) {
			continue
		}

		packageVulns := make(map[string]types.Vulnerability, len(result.Vulnerabilities))

		for _, vuln := range result.Vulnerabilities {
			// Skip vulnerabilities without fixed versions
			if vuln.FixedVersion == "" {
				continue
			}
			fixedVersion := getNearestFixedVersion(vuln.InstalledVersion, vuln.FixedVersion, string(result.Type))
			if fixedVersion == "" {
				continue
			}
			if pkgVuln, exists := packageVulns[vuln.PkgName]; exists {
				pkgVuln.VulnerabilityIDs = append(pkgVuln.VulnerabilityIDs, vuln.VulnerabilityID)
				pkgVuln.FixedVersion = version.GetHighestVersion(pkgVuln.FixedVersion, fixedVersion)
				pkgVuln.Severity = getHighestSeverity(pkgVuln.Severity, string(vuln.Severity))
				packageVulns[vuln.PkgName] = pkgVuln
			} else {
				packageVulns[vuln.PkgName] = types.Vulnerability{
					PackageDir:       result.Target,
					PackageName:      vuln.PkgName,
					CurrentVersion:   normalizeVersionForLanguage(vuln.InstalledVersion, string(result.Type)),
					FixedVersion:     fixedVersion,
					VulnerabilityIDs: []string{vuln.VulnerabilityID},
					Severity:         string(vuln.Severity),
					Language:         getLanguageFromResultType(string(result.Type)),
				}
			}
		}

		// Traverse the map in a deterministic order by sorting the keys
		var pkgNames []string
		for pkgName := range packageVulns {
			pkgNames = append(pkgNames, pkgName)
		}
		sort.Strings(pkgNames)
		for _, pkgName := range pkgNames {
			vulnerabilities = append(vulnerabilities, packageVulns[pkgName])
		}
	}

	return vulnerabilities
}

func getHighestSeverity(severities ...string) string {
	// getHighestSeverity returns the highest severity from a list of severity strings.
	// The order of severity from highest to lowest is: CRITICAL > HIGH > MEDIUM > LOW > UNKNOWN.
	// If no valid severity is found, returns an empty string.
	severityOrder := map[string]int{
		"CRITICAL": 5,
		"HIGH":     4,
		"MEDIUM":   3,
		"LOW":      2,
		"UNKNOWN":  1,
	}

	highest := ""
	highestRank := 0
	for _, sev := range severities {
		rank, ok := severityOrder[strings.ToUpper(sev)]
		if !ok {
			continue
		}
		if rank > highestRank {
			highest = strings.ToUpper(sev)
			highestRank = rank
		}
	}
	return highest
}

// isGoTarget checks if the target is a Go module
func isGoTarget(result trivyTypes.Result) bool {
	return result.Type == "gomod" ||
		result.Class == "lang-pkgs" ||
		strings.HasSuffix(result.Target, "go.mod") ||
		strings.HasSuffix(result.Target, "go.sum")
}

// normalizeVersionForLanguage normalizes version strings based on the target language
func normalizeVersionForLanguage(version, resultType string) string {
	if version == "" {
		return version
	}

	// For Go modules, add v prefix if needed
	if resultType == "gomod" {
		// If version already starts with 'v', return as is
		if strings.HasPrefix(version, "v") {
			return version
		}

		// If version looks like a semantic version (x.y.z), add 'v' prefix
		if isSemanticVersion(version) {
			return "v" + version
		}

		// For other cases (like commit hashes, branch names), return as is
		return version
	}

	// For Python packages, typically no v prefix
	if resultType == "pip" || resultType == "pipenv" || resultType == "poetry" {
		// Remove v prefix if present for Python packages
		if strings.HasPrefix(version, "v") {
			return strings.TrimPrefix(version, "v")
		}
		return version
	}

	// For Node.js packages, typically no v prefix
	if resultType == "npm" || resultType == "yarn" {
		// Remove v prefix if present for Node packages
		if strings.HasPrefix(version, "v") {
			return strings.TrimPrefix(version, "v")
		}
		return version
	}

	// Default behavior - return as is
	return version
}

// getLanguageFromResultType maps trivy result types to language names
func getLanguageFromResultType(resultType string) string {
	switch resultType {
	case "gomod":
		return "go"
	case "pip", "pipenv", "poetry":
		return "python"
	case "npm", "yarn":
		return "node"
	case "composer":
		return "php"
	case "rubygems":
		return "ruby"
	case "cargo":
		return "rust"
	case "nuget":
		return "dotnet"
	case "maven", "gradle":
		return "java"
	default:
		return "unknown"
	}
}

// isSemanticVersion checks if a version string looks like semantic version
func isSemanticVersion(version string) bool {
	// Simple check for semantic version pattern: x.y.z or x.y.z-suffix
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	// Check if first part is numeric
	for _, char := range parts[0] {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// getNearestFixedVersion returns the nearest fixed version that is greater than the installed version.
// It splits the fixedVersions string (comma-separated), filters out versions less than or equal to installedVersion,
// and returns the smallest version that is greater than installedVersion. Returns empty string if none found.
func getNearestFixedVersion(installedVersion, fixedVersions, resultType string) string {
	if fixedVersions == "" {
		return ""
	}

	// Normalize installed version for comparison
	installed := normalizeVersionForLanguage(installedVersion, resultType)

	// Split and normalize all fixed versions
	var candidates []string
	for _, v := range strings.Split(fixedVersions, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		vNorm := normalizeVersionForLanguage(v, resultType)
		if version.CompareVersions(vNorm, installed) > 0 {
			candidates = append(candidates, vNorm)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	nearest := candidates[0]
	for _, v := range candidates[1:] {
		if version.CompareVersions(v, nearest) < 0 {
			nearest = v
		}
	}
	return nearest
}
