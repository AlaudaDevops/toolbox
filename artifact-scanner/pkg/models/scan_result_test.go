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

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSeverityList_Highest(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    SeverityList
		expected string
	}{
		{
			name:     "empty list",
			input:    SeverityList{},
			expected: "",
		},
		{
			name:     "single critical severity",
			input:    SeverityList{SeverityCritical},
			expected: SeverityCritical,
		},
		{
			name:     "single high severity",
			input:    SeverityList{SeverityHigh},
			expected: SeverityHigh,
		},
		{
			name:     "single medium severity",
			input:    SeverityList{SeverityMedium},
			expected: SeverityMedium,
		},
		{
			name:     "single low severity",
			input:    SeverityList{SeverityLow},
			expected: SeverityLow,
		},
		{
			name:     "mixed severities returns highest (critical)",
			input:    SeverityList{SeverityLow, SeverityMedium, SeverityCritical, SeverityHigh},
			expected: SeverityCritical,
		},
		{
			name:     "mixed severities returns highest (high)",
			input:    SeverityList{SeverityLow, SeverityMedium, SeverityHigh},
			expected: SeverityHigh,
		},
		{
			name:     "mixed severities returns highest (medium)",
			input:    SeverityList{SeverityLow, SeverityMedium},
			expected: SeverityMedium,
		},
		{
			name:     "unknown severity only",
			input:    SeverityList{"UNKNOWN"},
			expected: "",
		},
		{
			name:     "mixed with unknown severity",
			input:    SeverityList{"UNKNOWN", SeverityLow, SeverityMedium},
			expected: SeverityMedium,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.Highest()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestSeverityList_HighestPriority(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    SeverityList
		expected string
	}{
		{
			name:     "empty list",
			input:    SeverityList{},
			expected: "",
		},
		{
			name:     "critical severity",
			input:    SeverityList{SeverityCritical},
			expected: PriorityCritical,
		},
		{
			name:     "high severity",
			input:    SeverityList{SeverityHigh},
			expected: PriorityHigh,
		},
		{
			name:     "medium severity",
			input:    SeverityList{SeverityMedium},
			expected: PriorityMedium,
		},
		{
			name:     "low severity",
			input:    SeverityList{SeverityLow},
			expected: PriorityLow,
		},
		{
			name:     "mixed severities returns highest priority",
			input:    SeverityList{SeverityLow, SeverityMedium, SeverityCritical},
			expected: PriorityCritical,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.HighestPriority()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestScanResult_Priority(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    *ScanResult
		expected string
	}{
		{
			name: "empty scan result",
			input: &ScanResult{
				OS:   []Vulnerability{},
				Lang: []Vulnerability{},
			},
			expected: "",
		},
		{
			name: "only OS vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityHigh},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{},
			},
			expected: PriorityHigh,
		},
		{
			name: "only Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{},
				Lang: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
			},
			expected: PriorityMedium,
		},
		{
			name: "mixed OS and Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{
					{Severity: SeverityCritical},
					{Severity: SeverityHigh},
				},
			},
			expected: PriorityCritical,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.Priority()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestScanResult_TotalVulnerabilities(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    *ScanResult
		expected int
	}{
		{
			name: "empty scan result",
			input: &ScanResult{
				OS:   []Vulnerability{},
				Lang: []Vulnerability{},
			},
			expected: 0,
		},
		{
			name: "only OS vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityHigh},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{},
			},
			expected: 2,
		},
		{
			name: "only Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{},
				Lang: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
			},
			expected: 2,
		},
		{
			name: "mixed OS and Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{
					{Severity: SeverityCritical},
					{Severity: SeverityHigh},
				},
			},
			expected: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.TotalVulnerabilities()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestScanResult_Severity(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    *ScanResult
		expected string
	}{
		{
			name: "empty scan result",
			input: &ScanResult{
				OS:   []Vulnerability{},
				Lang: []Vulnerability{},
			},
			expected: "",
		},
		{
			name: "only OS vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityHigh},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{},
			},
			expected: SeverityHigh,
		},
		{
			name: "only Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{},
				Lang: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
			},
			expected: SeverityMedium,
		},
		{
			name: "mixed OS and Lang vulnerabilities",
			input: &ScanResult{
				OS: []Vulnerability{
					{Severity: SeverityMedium},
					{Severity: SeverityLow},
				},
				Lang: []Vulnerability{
					{Severity: SeverityCritical},
					{Severity: SeverityHigh},
				},
			},
			expected: SeverityCritical,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.Severity()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestScanResults_Priority(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    ScanResults
		expected string
	}{
		{
			name:     "empty scan results",
			input:    ScanResults{},
			expected: "",
		},
		{
			name: "single scan result",
			input: ScanResults{
				Image{
					Repository: "image1",
				}: &ScanResult{
					OS: []Vulnerability{
						{Severity: SeverityHigh},
					},
					Lang: []Vulnerability{},
				},
			},
			expected: PriorityHigh,
		},
		{
			name: "multiple scan results with different severities",
			input: ScanResults{
				Image{
					Repository: "image1",
				}: &ScanResult{
					OS: []Vulnerability{
						{Severity: SeverityMedium},
					},
					Lang: []Vulnerability{},
				},
				Image{
					Repository: "image2",
				}: &ScanResult{
					OS: []Vulnerability{},
					Lang: []Vulnerability{
						{Severity: SeverityCritical},
					},
				},
				Image{
					Repository: "image3",
				}: &ScanResult{
					OS: []Vulnerability{
						{Severity: SeverityLow},
					},
					Lang: []Vulnerability{
						{Severity: SeverityHigh},
					},
				},
			},
			expected: PriorityCritical,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.Priority()
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}
