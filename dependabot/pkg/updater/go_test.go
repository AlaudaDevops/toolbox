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

package updater

import (
	"testing"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
)

func TestGoUpdater_isGolangStdExtension(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		want        bool
	}{
		{
			name:        "golang.org/x/crypto package",
			packageName: "golang.org/x/crypto",
			want:        true,
		},
		{
			name:        "golang.org/x/net package",
			packageName: "golang.org/x/net",
			want:        true,
		},
		{
			name:        "golang.org/x/oauth2 package",
			packageName: "golang.org/x/oauth2",
			want:        true,
		},
		{
			name:        "golang.org/x/sys package",
			packageName: "golang.org/x/sys",
			want:        true,
		},
		{
			name:        "google.golang.org/grpc package",
			packageName: "google.golang.org/grpc",
			want:        true,
		},
		{
			name:        "google.golang.org/protobuf package",
			packageName: "google.golang.org/protobuf",
			want:        true,
		},
		{
			name:        "github.com/containerd/containerd package",
			packageName: "github.com/containerd/containerd",
			want:        false,
		},
		{
			name:        "github.com/go-jose/go-jose/v4 package",
			packageName: "github.com/go-jose/go-jose/v4",
			want:        false,
		},
		{
			name:        "github.com/golang-jwt/jwt/v4 package",
			packageName: "github.com/golang-jwt/jwt/v4",
			want:        false,
		},
		{
			name:        "golang.org package without x",
			packageName: "golang.org/somepackage",
			want:        false,
		},
		{
			name:        "empty package name",
			packageName: "",
			want:        false,
		},
	}

	updater := NewGoUpdater("/tmp/test", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.isGolangStdExtension(tt.packageName)
			if got != tt.want {
				t.Errorf("GoUpdater.isGolangStdExtension(%q) = %v, want %v", tt.packageName, got, tt.want)
			}
		})
	}
}

func TestGoUpdater_aggregateVulnerabilities(t *testing.T) {
	tests := []struct {
		name             string
		vulnerabilities  []types.Vulnerability
		wantDirCount     int
		wantPackageCount map[string]int // directory -> package count
		wantVersions     map[string]map[string]string // directory -> package -> highest version
	}{
		{
			name: "single vulnerability",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
			},
			wantDirCount: 1,
			wantPackageCount: map[string]int{
				"/tmp/test": 1,
			},
			wantVersions: map[string]map[string]string{
				"/tmp/test": {
					"golang.org/x/crypto": "v0.35.0",
				},
			},
		},
		{
			name: "multiple vulnerabilities same package different versions",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.45.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.40.0",
				},
			},
			wantDirCount: 1,
			wantPackageCount: map[string]int{
				"/tmp/test": 1,
			},
			wantVersions: map[string]map[string]string{
				"/tmp/test": {
					"golang.org/x/crypto": "v0.45.0", // highest version
				},
			},
		},
		{
			name: "multiple packages with multiple vulnerabilities",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.45.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/net",
					FixedVersion: "v0.36.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/net",
					FixedVersion: "v0.38.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/containerd/containerd",
					FixedVersion: "v1.7.29",
				},
			},
			wantDirCount: 1,
			wantPackageCount: map[string]int{
				"/tmp/test": 3,
			},
			wantVersions: map[string]map[string]string{
				"/tmp/test": {
					"golang.org/x/crypto":               "v0.45.0",
					"golang.org/x/net":                  "v0.38.0",
					"github.com/containerd/containerd": "v1.7.29",
				},
			},
		},
		{
			name: "mono repo with multiple directories",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
				{
					PackageDir:   "submodule/go.mod",
					PackageName:  "golang.org/x/net",
					FixedVersion: "v0.38.0",
				},
				{
					PackageDir:   "submodule/go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.40.0",
				},
			},
			wantDirCount: 2,
			wantPackageCount: map[string]int{
				"/tmp/test":           1,
				"/tmp/test/submodule": 2,
			},
			wantVersions: map[string]map[string]string{
				"/tmp/test": {
					"golang.org/x/crypto": "v0.35.0",
				},
				"/tmp/test/submodule": {
					"golang.org/x/net":    "v0.38.0",
					"golang.org/x/crypto": "v0.40.0",
				},
			},
		},
		{
			name: "version without v prefix",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "0.35.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.45.0",
				},
			},
			wantDirCount: 1,
			wantPackageCount: map[string]int{
				"/tmp/test": 1,
			},
			wantVersions: map[string]map[string]string{
				"/tmp/test": {
					"golang.org/x/crypto": "v0.45.0", // highest version with v prefix
				},
			},
		},
		{
			name:             "empty vulnerabilities",
			vulnerabilities:  []types.Vulnerability{},
			wantDirCount:     0,
			wantPackageCount: map[string]int{},
			wantVersions:     map[string]map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewGoUpdater("/tmp/test", nil)
			result := updater.aggregateVulnerabilities(tt.vulnerabilities)

			// Check directory count
			if len(result) != tt.wantDirCount {
				t.Errorf("aggregateVulnerabilities() directory count = %v, want %v", len(result), tt.wantDirCount)
			}

			// Check package count per directory
			for dir, wantCount := range tt.wantPackageCount {
				if gotCount := len(result[dir]); gotCount != wantCount {
					t.Errorf("aggregateVulnerabilities() package count for dir %q = %v, want %v", dir, gotCount, wantCount)
				}
			}

			// Check versions
			for dir, packages := range tt.wantVersions {
				for packageName, wantVersion := range packages {
					found := false
					for _, update := range result[dir] {
						if update.PackageName == packageName {
							found = true
							if update.FixedVersion != wantVersion {
								t.Errorf("aggregateVulnerabilities() version for package %q in dir %q = %v, want %v",
									packageName, dir, update.FixedVersion, wantVersion)
							}
							break
						}
					}
					if !found {
						t.Errorf("aggregateVulnerabilities() package %q not found in dir %q", packageName, dir)
					}
				}
			}
		})
	}
}

func TestGoUpdater_aggregateVulnerabilities_grouping(t *testing.T) {
	tests := []struct {
		name                  string
		vulnerabilities       []types.Vulnerability
		wantNormalPackages    int
		wantGolangXPackages   int
	}{
		{
			name: "mixed golang.org/x/* and normal packages",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/net",
					FixedVersion: "v0.38.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/oauth2",
					FixedVersion: "v0.27.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/containerd/containerd",
					FixedVersion: "v1.7.29",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/go-jose/go-jose/v4",
					FixedVersion: "v4.0.5",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/golang-jwt/jwt/v4",
					FixedVersion: "v4.5.2",
				},
			},
			wantNormalPackages:  3,
			wantGolangXPackages: 3,
		},
		{
			name: "only golang.org/x/* packages",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "golang.org/x/net",
					FixedVersion: "v0.38.0",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "google.golang.org/grpc",
					FixedVersion: "v1.50.0",
				},
			},
			wantNormalPackages:  0,
			wantGolangXPackages: 3,
		},
		{
			name: "only normal packages",
			vulnerabilities: []types.Vulnerability{
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/containerd/containerd",
					FixedVersion: "v1.7.29",
				},
				{
					PackageDir:   "go.mod",
					PackageName:  "github.com/go-jose/go-jose/v4",
					FixedVersion: "v4.0.5",
				},
			},
			wantNormalPackages:  2,
			wantGolangXPackages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewGoUpdater("/tmp/test", nil)
			result := updater.aggregateVulnerabilities(tt.vulnerabilities)

			// Get the packages for the default directory
			packages := result["/tmp/test"]

			// Count normal vs golang.org/x/* packages
			normalCount := 0
			golangXCount := 0

			for _, pkg := range packages {
				if updater.isGolangStdExtension(pkg.PackageName) {
					golangXCount++
				} else {
					normalCount++
				}
			}

			if normalCount != tt.wantNormalPackages {
				t.Errorf("normal packages count = %v, want %v", normalCount, tt.wantNormalPackages)
			}

			if golangXCount != tt.wantGolangXPackages {
				t.Errorf("golang.org/x/* packages count = %v, want %v", golangXCount, tt.wantGolangXPackages)
			}
		})
	}
}

func TestGoUpdater_UpdatePackages_emptyVulnerabilities(t *testing.T) {
	tempDir := t.TempDir()
	updater := NewGoUpdater(tempDir, nil)

	result, err := updater.UpdatePackages([]types.Vulnerability{})

	if err != nil {
		t.Errorf("UpdatePackages() with empty vulnerabilities should not return error, got %v", err)
	}

	if result != nil {
		t.Errorf("UpdatePackages() with empty vulnerabilities should return nil, got %v", result)
	}
}

func TestGoUpdater_commandOutputFile_integration(t *testing.T) {
	tempDir := t.TempDir()
	commandOutputFile := "commands.log"

	updater := NewGoUpdater(tempDir, &config.GoUpdaterConfig{
		CommandOutputFile: commandOutputFile,
	})

	// Test that the config is set correctly
	if updater.config == nil {
		t.Fatal("GoUpdater config should not be nil")
	}

	if updater.config.CommandOutputFile != commandOutputFile {
		t.Errorf("CommandOutputFile = %q, want %q", updater.config.CommandOutputFile, commandOutputFile)
	}

	// Test that BaseUpdater has the correct commandOutputFile
	if updater.BaseUpdater.commandOutputFile != commandOutputFile {
		t.Errorf("BaseUpdater.commandOutputFile = %q, want %q", updater.BaseUpdater.commandOutputFile, commandOutputFile)
	}
}

func TestGoUpdater_GetLanguageType(t *testing.T) {
	updater := NewGoUpdater("/tmp/test", nil)

	if got := updater.GetLanguageType(); got != types.LanguageGo {
		t.Errorf("GetLanguageType() = %v, want %v", got, types.LanguageGo)
	}
}

func TestNewGoUpdater_nilConfig(t *testing.T) {
	updater := NewGoUpdater("/tmp/test", nil)

	if updater == nil {
		t.Fatal("NewGoUpdater() should not return nil")
	}

	if updater.config != nil {
		t.Errorf("config should be nil when nil is passed")
	}

	if updater.BaseUpdater.commandOutputFile != "" {
		t.Errorf("commandOutputFile should be empty when config is nil, got %q", updater.BaseUpdater.commandOutputFile)
	}
}

func TestNewGoUpdater_withConfig(t *testing.T) {
	config := &config.GoUpdaterConfig{
		CommandOutputFile: "test-commands.log",
	}

	updater := NewGoUpdater("/tmp/test", config)

	if updater == nil {
		t.Fatal("NewGoUpdater() should not return nil")
	}

	if updater.config != config {
		t.Error("config should be set")
	}

	if updater.BaseUpdater.commandOutputFile != config.CommandOutputFile {
		t.Errorf("commandOutputFile = %q, want %q", updater.BaseUpdater.commandOutputFile, config.CommandOutputFile)
	}
}

func TestGoUpdater_formatPackageVersion(t *testing.T) {
	tests := []struct {
		name   string
		update packageUpdate
		want   string
	}{
		{
			name: "normal package with v prefix",
			update: packageUpdate{
				PackageName:  "github.com/example/pkg",
				FixedVersion: "v1.2.3",
				UseLatest:    false,
			},
			want: "github.com/example/pkg@v1.2.3",
		},
		{
			name: "normal package without v prefix",
			update: packageUpdate{
				PackageName:  "github.com/example/pkg",
				FixedVersion: "1.2.3",
				UseLatest:    false,
			},
			want: "github.com/example/pkg@v1.2.3",
		},
		{
			name: "package with @latest",
			update: packageUpdate{
				PackageName:  "golang.org/x/crypto",
				FixedVersion: "v0.35.0",
				UseLatest:    true,
			},
			want: "golang.org/x/crypto@latest",
		},
	}

	updater := NewGoUpdater("/tmp/test", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.formatPackageVersion(tt.update)
			if got != tt.want {
				t.Errorf("formatPackageVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoUpdater_buildBatchCommand(t *testing.T) {
	tests := []struct {
		name    string
		updates []packageUpdate
		want    string
	}{
		{
			name: "single package",
			updates: []packageUpdate{
				{
					PackageName:  "github.com/example/pkg",
					FixedVersion: "v1.2.3",
					UseLatest:    false,
				},
			},
			want: "go get github.com/example/pkg@v1.2.3",
		},
		{
			name: "multiple packages",
			updates: []packageUpdate{
				{
					PackageName:  "github.com/example/pkg1",
					FixedVersion: "v1.2.3",
					UseLatest:    false,
				},
				{
					PackageName:  "github.com/example/pkg2",
					FixedVersion: "v2.3.4",
					UseLatest:    false,
				},
			},
			want: "go get github.com/example/pkg1@v1.2.3 github.com/example/pkg2@v2.3.4",
		},
		{
			name: "mixed @latest and version",
			updates: []packageUpdate{
				{
					PackageName:  "github.com/example/pkg1",
					FixedVersion: "v1.2.3",
					UseLatest:    false,
				},
				{
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
					UseLatest:    true,
				},
			},
			want: "go get github.com/example/pkg1@v1.2.3 golang.org/x/crypto@latest",
		},
	}

	updater := NewGoUpdater("/tmp/test", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.buildBatchCommand(tt.updates)
			if got != tt.want {
				t.Errorf("buildBatchCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoUpdater_buildSingleCommand(t *testing.T) {
	tests := []struct {
		name   string
		update packageUpdate
		want   string
	}{
		{
			name: "normal package",
			update: packageUpdate{
				PackageName:  "github.com/example/pkg",
				FixedVersion: "v1.2.3",
				UseLatest:    false,
			},
			want: "go get github.com/example/pkg@v1.2.3",
		},
		{
			name: "package with @latest",
			update: packageUpdate{
				PackageName:  "golang.org/x/crypto",
				FixedVersion: "v0.35.0",
				UseLatest:    true,
			},
			want: "go get golang.org/x/crypto@latest",
		},
	}

	updater := NewGoUpdater("/tmp/test", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.buildSingleCommand(tt.update)
			if got != tt.want {
				t.Errorf("buildSingleCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoUpdater_buildPackageArgs(t *testing.T) {
	tests := []struct {
		name    string
		updates []packageUpdate
		want    []string
	}{
		{
			name: "single package",
			updates: []packageUpdate{
				{
					PackageName:  "github.com/example/pkg",
					FixedVersion: "v1.2.3",
					UseLatest:    false,
				},
			},
			want: []string{"github.com/example/pkg@v1.2.3"},
		},
		{
			name: "multiple packages",
			updates: []packageUpdate{
				{
					PackageName:  "github.com/example/pkg1",
					FixedVersion: "v1.2.3",
					UseLatest:    false,
				},
				{
					PackageName:  "golang.org/x/crypto",
					FixedVersion: "v0.35.0",
					UseLatest:    true,
				},
			},
			want: []string{
				"github.com/example/pkg1@v1.2.3",
				"golang.org/x/crypto@latest",
			},
		},
	}

	updater := NewGoUpdater("/tmp/test", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.buildPackageArgs(tt.updates)

			if len(got) != len(tt.want) {
				t.Errorf("buildPackageArgs() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildPackageArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
