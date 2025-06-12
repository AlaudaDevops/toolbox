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

package config

// GitHubDependabotConfig represents GitHub Dependabot compatible configuration
type GitHubDependabotConfig struct {
	Version int                      `yaml:"version" json:"version"`
	Updates []DependabotUpdateConfig `yaml:"updates" json:"updates"`
}

// DependabotUpdateConfig represents a single package ecosystem update configuration
type DependabotUpdateConfig struct {
	PackageEcosystem string   `yaml:"package-ecosystem" json:"package-ecosystem"`
	Labels           []string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Assignees        []string `yaml:"assignees,omitempty" json:"assignees,omitempty"`
}
