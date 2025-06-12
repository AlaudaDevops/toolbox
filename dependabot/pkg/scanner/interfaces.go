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

// Package scanner provides interfaces and types for security scanning
package scanner

import (
	"fmt"

	"github.com/alauda-devops/toolbox/dependabot/pkg/config"
)

// Scanner is the interface that all security scanners must implement
type Scanner interface {
	// Scan performs the security scan and returns vulnerabilities
	Scan() ([]Vulnerability, error)
	// Cleanup cleans up any temporary resources created during scanning
	Cleanup() error
	// GetName returns the name of the scanner
	GetName() string
}

// ScanResult represents the result of a security scan
type ScanResult struct {
	// Vulnerabilities is the list of vulnerabilities found
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	// ScannerName is the name of the scanner that produced this result
	ScannerName string `json:"scanner_name"`
	// ScanPath is the path that was scanned
	ScanPath string `json:"scan_path"`
}

// Vulnerability represents a security vulnerability in a package
type Vulnerability struct {
	// PackageDir is the directory of the vulnerable package
	PackageDir string `json:"package_dir"`
	// PackageName is the name of the vulnerable package
	PackageName string `json:"package_name"`
	// CurrentVersion is the current version of the package
	CurrentVersion string `json:"current_version"`
	// FixedVersion is the version that fixes the vulnerability
	FixedVersion string `json:"fixed_version"`
	// VulnerabilityIDs are the identifiers of the vulnerabilities
	VulnerabilityIDs []string `json:"vulnerability_ids"`
	// Severity is the severity level of the vulnerability
	Severity string `json:"severity"`
	// Language indicates the programming language ecosystem this vulnerability belongs to
	Language string `json:"language,omitempty"`
}

// NewScanner creates a new scanner instance based on the provided configuration
func NewScanner(scanPath string, config config.ScannerConfig) (Scanner, error) {
	switch config.Type {
	case "trivy":
		return NewTrivyScanner(scanPath, config), nil
	case "govulncheck":
		// Future implementation
		return nil, fmt.Errorf("govulncheck scanner not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported scanner type: %s", config.Type)
	}
}
