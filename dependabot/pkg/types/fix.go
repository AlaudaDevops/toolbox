package types

import "fmt"

// VulnFixResult represents a single package update
type VulnFixResult struct {
	Vulnerability `json:",inline" yaml:",inline"`

	// Success indicates whether this package update was successful
	Success bool
	// Error contains the error message if update failed
	Error string
}

// VulnFixResults represents the comprehensive result of an update operation
type VulnFixResults []VulnFixResult

func (v VulnFixResults) FixedVulns() []Vulnerability {
	vulns := make([]Vulnerability, 0, len(v))
	for _, result := range v {
		if result.Success {
			vulns = append(vulns, result.Vulnerability)
		}
	}
	return vulns
}

func (v VulnFixResults) FixedVulnCount() int {
	count := 0
	for _, result := range v {
		if result.Success {
			count++
		}
	}
	return count
}

func (v VulnFixResults) FixFailedVulns() []Vulnerability {
	vulns := make([]Vulnerability, 0, len(v))
	for _, result := range v {
		if !result.Success {
			vulns = append(vulns, result.Vulnerability)
		}
	}
	return vulns
}

func (v VulnFixResults) FixFailedVulnCount() int {
	count := 0
	for _, result := range v {
		if !result.Success {
			count++
		}
	}
	return count
}

// TotalVulnCount returns the total number of vulnerabilities
func (v VulnFixResults) TotalVulnCount() int {
	return len(v)
}

// Summary returns a summary of the fix operation
func (v VulnFixResults) FixSummary() string {
	totalCount := v.TotalVulnCount()
	if totalCount == 0 {
		return "No vulnerabilities found to update"
	}

	return fmt.Sprintf("Updated %d packages successfully, %d failed", v.FixedVulnCount(), v.FixFailedVulnCount())
}
