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

// Package types provides common types for the dependabot package
package types

import "fmt"

// LanguageType represents different programming languages
type LanguageType string

const (
	// LanguageGo represents Go programming language
	LanguageGo LanguageType = "go"
	// LanguagePython represents Python programming language (for future implementation)
	LanguagePython LanguageType = "python"
	// LanguageNode represents Node.js/JavaScript (for future implementation)
	LanguageNode LanguageType = "node"
	// LanguageUnknown represents unknown programming language
	LanguageUnknown LanguageType = "unknown"
)

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

// String returns a formatted string representation of the vulnerability
func (v Vulnerability) String() string {
	return fmt.Sprintf("Vulnerability{Package: %s@%s (fixed: %s), Severity: %s, Language: %s, Dir: %s, IDs: %s}",
		v.PackageName, v.CurrentVersion, v.FixedVersion, v.Severity, v.Language, v.PackageDir, v.VulnerabilityIDs)
}
