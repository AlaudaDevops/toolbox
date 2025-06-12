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

package scanner

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/google/go-cmp/cmp"
)

func loadJson(filePath string, obj any) (err error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(fileData, obj)
}

func TestExtractGoVulnerabilities(t *testing.T) {
	var trivyReport types.Report
	err := loadJson("./testdata/go_vuln.json", &trivyReport)
	if err != nil {
		t.Fatalf("failed to load trivy report: %v", err)
	}

	vulns := extractGoVulnerabilities(trivyReport)

	var expectedVulns []Vulnerability
	err = loadJson("./testdata/go_vuln_golden.json", &expectedVulns)
	if err != nil {
		t.Fatalf("failed to load golden vulns: %v", err)
	}

	if diff := cmp.Diff(expectedVulns, vulns); diff != "" {
		t.Errorf("extractGoVulnerabilities() mismatch (-want +got):%s", diff)
	}
}

func TestGetNearestFixedVersion(t *testing.T) {
	tests := []struct {
		name             string
		installedVersion string
		fixedVersions    string
		resultType       string
		want             string
	}{
		{
			name:             "single higher version",
			installedVersion: "v1.7.25",
			fixedVersions:    "1.7.27",
			resultType:       "gomod",
			want:             "v1.7.27",
		},
		{
			name:             "multiple, one higher",
			installedVersion: "v1.7.25",
			fixedVersions:    "1.7.24, 1.7.27, 1.6.38",
			resultType:       "gomod",
			want:             "v1.7.27",
		},
		{
			name:             "multiple, all lower",
			installedVersion: "v1.7.25",
			fixedVersions:    "1.7.24, 1.6.38",
			resultType:       "gomod",
			want:             "",
		},
		{
			name:             "multiple, all higher, pick nearest",
			installedVersion: "v1.7.25",
			fixedVersions:    "1.7.27, 1.8.0, 2.0.0",
			resultType:       "gomod",
			want:             "v1.7.27",
		},
		{
			name:             "mixed v prefix",
			installedVersion: "v1.7.25",
			fixedVersions:    "v1.7.27, 1.8.0, v2.0.0",
			resultType:       "gomod",
			want:             "v1.7.27",
		},
		{
			name:             "empty fixed",
			installedVersion: "v1.7.25",
			fixedVersions:    "",
			resultType:       "gomod",
			want:             "",
		},
		{
			name:             "installed no v, fixed with v",
			installedVersion: "1.7.25",
			fixedVersions:    "v1.7.27, v1.8.0",
			resultType:       "gomod",
			want:             "v1.7.27",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNearestFixedVersion(tt.installedVersion, tt.fixedVersions, tt.resultType)
			if got != tt.want {
				t.Errorf("getNearestFixedVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
