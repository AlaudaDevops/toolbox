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

// Package updater provides unified semantic version comparison functionality
package version

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// CompareVersions compares two semantic version strings and returns:
// -1 if version1 < version2
//
//	0 if version1 == version2
//	1 if version1 > version2
//
// This function handles version normalization (removing/adding "v" prefix as needed)
// and works for all programming languages that follow semantic versioning.
func CompareVersions(version1, version2 string) int {
	// Handle empty versions
	if version1 == "" && version2 == "" {
		return 0
	}
	if version1 == "" {
		return -1
	}
	if version2 == "" {
		return 1
	}

	// Normalize versions for semantic version parsing
	v1 := normalizeVersionForSemver(version1)
	v2 := normalizeVersionForSemver(version2)

	// Parse versions using semver library
	semVer1, err1 := semver.NewVersion(v1)
	semVer2, err2 := semver.NewVersion(v2)

	// If both versions parse successfully, use semver comparison
	if err1 == nil && err2 == nil {
		return semVer1.Compare(semVer2)
	}

	// Fallback to lexicographic comparison if semver parsing fails
	if v1 < v2 {
		return -1
	}
	if v1 > v2 {
		return 1
	}
	return 0
}

// normalizeVersionForSemver normalizes version strings for semantic version parsing
// Ensures version has proper format (adds "v" prefix if missing for some edge cases)
func normalizeVersionForSemver(version string) string {
	if version == "" {
		return version
	}

	// Remove "v" prefix for semver parsing (semver library handles this automatically)
	// but we need to handle some edge cases
	normalized := strings.TrimPrefix(version, "v")

	// Handle versions that might not be strictly semver (e.g., "1.0" -> "1.0.0")
	parts := strings.Split(normalized, ".")

	// If we have less than 3 parts, this might not be strict semver
	// The semver library will handle this, but let's ensure we have a valid format
	if len(parts) == 1 {
		// Version like "1" -> "1.0.0"
		normalized = normalized + ".0.0"
	} else if len(parts) == 2 {
		// Version like "1.2" -> "1.2.0"
		normalized = normalized + ".0"
	}

	return normalized
}

// GetHighestVersion returns the highest version from a slice of version strings
// Returns empty string if the slice is empty
func GetHighestVersion(versions ...string) string {
	if len(versions) == 0 {
		return ""
	}

	if len(versions) == 1 {
		return versions[0]
	}

	highest := versions[0]
	for i := 1; i < len(versions); i++ {
		if CompareVersions(versions[i], highest) > 0 {
			highest = versions[i]
		}
	}

	return highest
}
