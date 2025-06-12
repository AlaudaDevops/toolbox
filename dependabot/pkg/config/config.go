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

// Package config provides configuration management for DependaBot
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

// DependaBotConfig represents the complete configuration for DependaBot (legacy format)
type DependaBotConfig struct {
	// Branch is the branch to clone and create PR against
	Branch string `yaml:"branch" json:"branch"`
	// BranchPrefix is the prefix for created branches
	BranchPrefix string `yaml:"branch_prefix" json:"branch_prefix"`
	// PRConfig contains PR-specific configuration
	PRConfig PRConfig `yaml:"pr" json:"pr"`
	// Scanner contains scanner configuration (supports multiple scanner types)
	Scanner ScannerConfig `yaml:"scanner" json:"scanner"`
}

// PRConfig contains pull request configuration
type PRConfig struct {
	AutoCreate *bool `yaml:"autoCreate" json:"autoCreate"`
	// Labels are labels to add to the created PR
	Labels []string `yaml:"labels" json:"labels"`
	// Assignees are users to assign to the created PR
	Assignees []string `yaml:"assignees" json:"assignees"`
}

func (p *PRConfig) NeedCreatePR() bool {
	return p.AutoCreate != nil && *p.AutoCreate
}

// ScannerConfig contains generic scanner configuration
type ScannerConfig struct {
	// Type specifies the scanner type (e.g., "trivy", "govulncheck")
	Type string `yaml:"type" json:"type"`
	// Timeout for scanner execution (e.g., "5m")
	Timeout string `yaml:"timeout" json:"timeout"`
	// Params contains scanner-specific parameters
	Params []string `yaml:"params" json:"params"`
}

// TrivyConfig contains Trivy scanning configuration (deprecated, use ScannerConfig)
type TrivyConfig struct {
	// Scanners specifies which trivy scanners to use
	Scanners []string `yaml:"scanners" json:"scanners"`
	// IgnoreUnfixed ignores vulnerabilities without fixes
	IgnoreUnfixed bool `yaml:"ignore_unfixed" json:"ignore_unfixed"`
	// Timeout for trivy scan (e.g., "5m")
	Timeout string `yaml:"timeout" json:"timeout"`
}

// PipelineScannerConfig represents scanner config for pipeline
type PipelineScannerConfig = ScannerConfig

// ConfigReader handles reading and merging configuration files
type ConfigReader struct{}

// NewConfigReader creates a new configuration reader
func NewConfigReader() *ConfigReader {
	return &ConfigReader{}
}

// ReadRepoConfig reads configuration from repository .github/dependabot.yml
// Supports both legacy format and GitHub Dependabot format
func (c *ConfigReader) ReadRepoConfig(projectPath string) (*DependaBotConfig, error) {
	// Try both .yml and .yaml extensions
	configPaths := []string{
		filepath.Join(projectPath, ".github", "dependabot.yml"),
		filepath.Join(projectPath, ".github", "dependabot.yaml"),
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			logrus.Debugf("Found repository configuration: %s", configPath)
			return c.readAndParseConfigFile(configPath)
		}
	}

	// No config file found, return empty config
	logrus.Debug("No repository configuration found, using defaults")
	return &DependaBotConfig{}, nil
}

// ReadExternalConfig reads configuration from external file specified by CLI
func (c *ConfigReader) ReadExternalConfig(configPath string) (*DependaBotConfig, error) {
	if configPath == "" {
		return &DependaBotConfig{}, nil
	}

	logrus.Debugf("Reading external configuration: %s", configPath)
	return c.readAndParseConfigFile(configPath)
}

// readAndParseConfigFile reads and parses a YAML configuration file
// Automatically detects format (legacy or GitHub Dependabot format)
func (c *ConfigReader) readAndParseConfigFile(configPath string) (*DependaBotConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Try to parse as GitHub Dependabot format first
	var githubConfig GitHubDependabotConfig
	if err := yaml.Unmarshal(data, &githubConfig); err == nil && githubConfig.Version == 2 && len(githubConfig.Updates) > 0 {
		logrus.Debugf("Detected GitHub Dependabot format in %s", configPath)
		return c.convertFromGitHubFormat(&githubConfig), nil
	}

	// Fall back to legacy format
	var legacyConfig DependaBotConfig
	if err := yaml.Unmarshal(data, &legacyConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s as either GitHub Dependabot format or legacy format: %w", configPath, err)
	}

	logrus.Debugf("Detected legacy format in %s", configPath)
	return &legacyConfig, nil
}

// convertFromGitHubFormat converts GitHub Dependabot format to our internal format
func (c *ConfigReader) convertFromGitHubFormat(githubConfig *GitHubDependabotConfig) *DependaBotConfig {
	config := &DependaBotConfig{}

	// Find the first Go module update configuration
	var goUpdate *DependabotUpdateConfig
	for _, update := range githubConfig.Updates {
		if update.PackageEcosystem == "gomod" {
			goUpdate = &update
			break
		}
	}

	if goUpdate == nil {
		return config
	}

	// Map labels and assignees
	config.PRConfig.Labels = goUpdate.Labels
	config.PRConfig.Assignees = goUpdate.Assignees

	return config
}

// MergeConfigs merges multiple configurations with priority order
// Later configurations override earlier ones
func (c *ConfigReader) MergeConfigs(configs ...*DependaBotConfig) *DependaBotConfig {
	merged := &DependaBotConfig{}

	for _, config := range configs {
		if config == nil {
			continue
		}

		// Merge simple fields (later config wins)
		if config.Branch != "" {
			merged.Branch = config.Branch
		}
		if config.BranchPrefix != "" {
			merged.BranchPrefix = config.BranchPrefix
		}
		// Merge PR config
		if config.PRConfig.AutoCreate != nil {
			merged.PRConfig.AutoCreate = config.PRConfig.AutoCreate
		}
		if len(config.PRConfig.Labels) > 0 {
			merged.PRConfig.Labels = config.PRConfig.Labels
		}
		if len(config.PRConfig.Assignees) > 0 {
			merged.PRConfig.Assignees = config.PRConfig.Assignees
		}
		if config.Scanner.Type != "" {
			merged.Scanner.Type = config.Scanner.Type
		}
		if config.Scanner.Timeout != "" {
			merged.Scanner.Timeout = config.Scanner.Timeout
		}
		if len(config.Scanner.Params) > 0 {
			merged.Scanner.Params = config.Scanner.Params
		}
	}

	return merged
}

// ApplyDefaults applies default values to configuration
func (c *ConfigReader) ApplyDefaults(config *DependaBotConfig) *DependaBotConfig {
	if config.Branch == "" {
		config.Branch = "main"
	}
	if config.BranchPrefix == "" {
		config.BranchPrefix = "dependabot/security-updates"
	}

	// Apply scanner defaults (prefer new format over legacy)
	// If neither scanner config nor trivy config is set, apply defaults to both for backward compatibility
	if config.Scanner.Type == "" {
		// Apply defaults to new scanner format
		config.Scanner.Type = "trivy"
		config.Scanner.Timeout = "5m"
		config.Scanner.Params = []string{"--scanners", "vuln"}
	}

	return config
}
