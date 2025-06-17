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

// Package updater provides language-agnostic vulnerability package updating functionality.
//
// Architecture Design Principles:
//
//  1. Simplified Interface Design:
//     The LanguageUpdater interface is intentionally minimal with only two methods:
//     - UpdatePackages: Core functionality to update vulnerable packages
//     - GetLanguageType: Returns the language this updater handles
//
//  2. Language-Specific Internal Processing:
//     Each language implementation receives trivy vulnerability data and handles
//     all language-specific logic internally, including:
//     - Project structure validation (e.g., finding go.mod, requirements.txt)
//     - Package version normalization (e.g., Go's "v" prefix handling)
//     - Package manager command execution (e.g., "go get", "pip install")
//     - Post-update tasks (e.g., "go mod tidy")
//
//  3. Trivy Result Integration:
//     The design leverages trivy's standardized JSON output format. All languages
//     use the same trivy vulnerability structure, making the interface consistent
//     and reducing the need for language-specific data transformations at the
//     interface level.
//
// 4. Flexibility and Extensibility:
//   - Easy to add new languages by implementing the simple LanguageUpdater interface
//   - Language detection is handled separately and can be enhanced independently
//   - Each language updater can evolve its internal logic without affecting others
//   - Testing is simplified with mock implementations
//
// 5. Separation of Concerns:
//   - Interface focuses on the core "update packages" responsibility
//   - Language detection is handled by a separate LanguageDetector interface
//   - Project validation and package management details are encapsulated within
//     each language implementation
//
// This design provides better maintainability, testability, and extensibility
// compared to a more complex interface with many methods.
package updater

import "github.com/AlaudaDevops/toolbox/dependabot/pkg/types"

// LanguageUpdater defines the simplified interface for language-specific package updaters
// Each language implementation receives trivy vulnerabilities and handles the update process internally
type LanguageUpdater interface {
	// UpdatePackages is the core method that updates vulnerable packages for the specific language
	// It receives vulnerabilities from trivy and handles all language-specific logic internally
	UpdatePackages(vulnerabilities []types.Vulnerability) (types.VulnFixResults, error)

	// GetLanguageType returns the language type this updater handles
	GetLanguageType() types.LanguageType
}
