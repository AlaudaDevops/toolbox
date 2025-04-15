/*
Copyright 2024 The AlaudaDevops Authors.

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

package models

// Severity levels for vulnerabilities
const (
	SeverityCritical = "CRITICAL"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityLow      = "LOW"
)

// Priority levels for Jira issues
const (
	PriorityCritical = "L0 - Critical"
	PriorityHigh     = "L1 - High"
	PriorityMedium   = "L2 - Medium"
	PriorityLow      = "L3 - Low"
)

// SeverityToPriority maps vulnerability severity levels to Jira priority levels
var SeverityToPriority = map[string]string{
	SeverityCritical: PriorityCritical,
	SeverityHigh:     PriorityHigh,
	SeverityMedium:   PriorityMedium,
	SeverityLow:      PriorityLow,
}

// SeverityList represents a list of severity levels
type SeverityList []string

// Highest returns the highest severity level from the list
// Returns an empty string if the list is empty
func (s SeverityList) Highest() string {
	if len(s) == 0 {
		return ""
	}

	severityOrder := []string{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
	}

	severityFound := make(map[string]bool)
	for _, value := range s {
		severityFound[value] = true
	}

	for _, severity := range severityOrder {
		if severityFound[severity] {
			return severity
		}
	}
	return ""
}

// HighestPriority returns the Jira priority corresponding to the highest severity
func (s SeverityList) HighestPriority() string {
	return SeverityToPriority[s.Highest()]
}

// ScanResult represents the results of a vulnerability scan
// OS: Operating system vulnerabilities
// Lang: Language-specific vulnerabilities
// Secret: Secret scanning results
// OSImage: The base OS image used
type ScanResult struct {
	OS      []Vulnerability `json:"os"`
	Lang    []Vulnerability `json:"lang"`
	Secret  []interface{}   `json:"secret"`
	OSImage string          `json:"os_image"`
}

// Vulnerability represents a single vulnerability finding
// Target: The target of the vulnerability
// VulnerabilityID: The unique identifier of the vulnerability
// Severity: The severity level of the vulnerability
// PkgName: The name of the affected package
// InstalledVersion: The currently installed version
// FixedVersion: The version that fixes the vulnerability
// Title: The title of the vulnerability
// Description: The description of the vulnerability
type Vulnerability struct {
	Target           string `json:"Target"`
	VulnerabilityID  string `json:"VulnerabilityID"`
	Severity         string `json:"Severity"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
}

// Priority returns the Jira priority corresponding to the highest severity in the scan result
func (s *ScanResult) Priority() string {
	return SeverityToPriority[s.Severity()]
}

// TotalVulnerabilities returns the total number of vulnerabilities found
func (s *ScanResult) TotalVulnerabilities() int {
	return len(s.OS) + len(s.Lang)
}

// Severity returns the highest severity level found in the scan result
func (s *ScanResult) Severity() string {
	list := SeverityList{}
	for _, v := range s.OS {
		list = append(list, v.Severity)
	}
	for _, v := range s.Lang {
		list = append(list, v.Severity)
	}
	return list.Highest()
}

// ScanResults represents a map of images to their scan results
type ScanResults map[Image]*ScanResult

// Priority returns the highest Jira priority across all scan results
func (r ScanResults) Priority() string {
	list := SeverityList{}
	for _, result := range r {
		list = append(list, result.Severity())
	}

	return list.HighestPriority()
}
