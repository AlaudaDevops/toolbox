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

package jira

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/models"
)

//go:embed table.tmpl
var tmplFS embed.FS

// templateFuncs returns a map of template functions
// Currently includes a function to convert strings to lowercase
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"lower": strings.ToLower,
	}
}

// RenderVulnerabilityTable renders a vulnerability table from scan results
// scanResult: The scan results containing vulnerability information
// Returns the rendered table as a string and any error that occurred
func RenderVulnerabilityTable(scanResult *models.ScanResult) (string, error) {
	tmplContent, err := tmplFS.ReadFile("table.tmpl")
	if err != nil {
		return "", fmt.Errorf("error reading template file: %w", err)
	}

	// sortVulnerabilities sorts vulnerabilities by package name and vulnerability ID
	sortVulnerabilities := func(vulns []models.Vulnerability) []models.Vulnerability {
		sortedVulns := make([]models.Vulnerability, len(vulns))
		copy(sortedVulns, vulns)

		sort.Slice(sortedVulns, func(i, j int) bool {
			if sortedVulns[i].PkgName != sortedVulns[j].PkgName {
				return sortedVulns[i].PkgName < sortedVulns[j].PkgName
			}
			return sortedVulns[i].VulnerabilityID < sortedVulns[j].VulnerabilityID
		})

		return sortedVulns
	}

	tmpl, err := template.New("vulnerability-table").
		Funcs(templateFuncs()).
		Funcs(template.FuncMap{"sortVulnerabilities": sortVulnerabilities}).
		Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, scanResult); err != nil {
		return "", fmt.Errorf("error rendering template: %w", err)
	}

	return buf.String(), nil
}
