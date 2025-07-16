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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// DependaBotConfig represents the complete configuration for DependaBot (legacy format)
type DependaBotConfig struct {
	// Repo contains repository information
	Repo RepoConfig `yaml:"repo" json:"repo" mapstructure:"repo"`
	// PR contains PR-specific configuration
	PR PRConfig `yaml:"pr" json:"pr" mapstructure:"pr"`
	// Scanner contains scanner configuration (supports multiple scanner types)
	Scanner ScannerConfig `yaml:"scanner" json:"scanner" mapstructure:"scanner"`
	// Git contains git provider configuration
	Git GitProviderConfig `yaml:"git" json:"git" mapstructure:"git"`
	// Notice contains notice configuration
	Notice NoticeConfig `yaml:"notice" json:"notice" mapstructure:"notice"`
	// Scripts contains custom script configuration for pipeline hooks
	Scripts ScriptsConfig `yaml:"scripts" json:"scripts" mapstructure:"scripts"`
	// Updater contains updater-specific configuration
	Updater UpdaterConfig `yaml:"updater" json:"updater" mapstructure:"updater"`
}

type NoticeConfig struct {
	// Type is the type of notice (e.g., "slack", "dingtalk")
	Type string `yaml:"type" json:"type" mapstructure:"type"`
	// Params contains notice-specific parameters
	Params map[string]interface{} `yaml:"params" json:"params" mapstructure:"params"`
}

type RepoConfig struct {
	// URL is the repository URL (e.g., "https://github.com/example/repo")
	URL string `yaml:"url" json:"url" mapstructure:"url"`
	// Branch is the repository branch (e.g., "main")
	Branch string `yaml:"branch" json:"branch" mapstructure:"branch"`
	// IncludeSubmodules indicates whether to clone submodules
	IncludeSubmodules *bool `yaml:"includeSubmodules" json:"includeSubmodules" mapstructure:"includeSubmodules"`
}

type GitProviderConfig struct {
	// Provider is the type of git provider (e.g., "github", "gitlab")
	Provider string `yaml:"provider" json:"provider" mapstructure:"provider"`
	// BaseURL is the base URL of the git provider (e.g., "https://github.com")
	BaseURL string `yaml:"baseURL" json:"baseURL" mapstructure:"baseURL"`
	// Token is the authentication token for the git provider
	Token string `yaml:"token" json:"token" mapstructure:"token"`
}

// PRConfig contains pull request configuration
type PRConfig struct {
	AutoCreate *bool `yaml:"autoCreate" json:"autoCreate" mapstructure:"autoCreate"`
	// PushBranch controls whether to push the branch to remote repository
	PushBranch *bool `yaml:"pushBranch" json:"pushBranch" mapstructure:"pushBranch"`
	// Labels are labels to add to the created PR
	Labels []string `yaml:"labels" json:"labels" mapstructure:"labels"`
	// Assignees are users to assign to the created PR
	Assignees []string `yaml:"assignees" json:"assignees" mapstructure:"assignees"`
}

// UpdaterConfig contains updater-specific configuration
type UpdaterConfig struct {
	// Go contains Go-specific updater configuration
	Go *GoUpdaterConfig `yaml:"go" json:"go" mapstructure:"go"`
}

// GoUpdaterConfig contains Go-specific updater configuration
type GoUpdaterConfig struct {
	// CommandOutputFile is the file path to output successful go get commands
	// If empty, no output will be written
	CommandOutputFile string `yaml:"commandOutputFile" json:"commandOutputFile" mapstructure:"commandOutputFile"`
}

func (p *PRConfig) NeedCreatePR() bool {
	return p.AutoCreate != nil && *p.AutoCreate
}

// NeedPushBranch determines whether to push branch based on configuration
// If autocreate is true, push branch automatically becomes true
func (p *PRConfig) NeedPushBranch() bool {
	// If autocreate is true, automatically push branch
	if p.NeedCreatePR() {
		return true
	}
	// Otherwise, check the explicit pushBranch setting
	return p.PushBranch != nil && *p.PushBranch
}

// ScannerConfig contains generic scanner configuration
type ScannerConfig struct {
	// Type specifies the scanner type (e.g., "trivy", "govulncheck")
	Type string `yaml:"type" json:"type" mapstructure:"type"`
	// Timeout for scanner execution (e.g., "5m")
	Timeout string `yaml:"timeout" json:"timeout" mapstructure:"timeout"`
	// Params contains scanner-specific parameters
	Params []string `yaml:"params" json:"params" mapstructure:"params"`
}

// PipelineScannerConfig represents scanner config for pipeline
type PipelineScannerConfig = ScannerConfig

// ScriptsConfig contains custom script configuration for pipeline hooks
type ScriptsConfig struct {
	// PreScan contains script to execute before security scanning
	PreScan *ScriptConfig `yaml:"preScan" json:"preScan" mapstructure:"preScan"`
	// PreCommit contains script to execute before committing changes
	PreCommit *ScriptConfig `yaml:"preCommit" json:"preCommit" mapstructure:"preCommit"`
}

// ScriptConfig contains configuration for a single script
type ScriptConfig struct {
	// Script contains the script content to execute
	Script string `yaml:"script" json:"script" mapstructure:"script"`
	// Timeout for script execution (e.g., "5m", "30s")
	Timeout string `yaml:"timeout" json:"timeout" mapstructure:"timeout"`
	// ContinueOnError determines whether to continue pipeline execution if script fails
	ContinueOnError bool `yaml:"continueOnError" json:"continueOnError" mapstructure:"continueOnError"`
}

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
	var goUpdate *GitHubDependabotUpdateConfig
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
	config.PR.Labels = goUpdate.Labels
	config.PR.Assignees = goUpdate.Assignees

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

		// Merge repo fields
		if config.Repo.URL != "" {
			merged.Repo.URL = config.Repo.URL
		}
		if config.Repo.Branch != "" {
			merged.Repo.Branch = config.Repo.Branch
		}
		if config.Repo.IncludeSubmodules != nil {
			merged.Repo.IncludeSubmodules = config.Repo.IncludeSubmodules
		}
		// Merge PR config
		if config.PR.AutoCreate != nil {
			merged.PR.AutoCreate = config.PR.AutoCreate
		}
		if config.PR.PushBranch != nil {
			merged.PR.PushBranch = config.PR.PushBranch
		}
		if len(config.PR.Labels) > 0 {
			merged.PR.Labels = config.PR.Labels
		}
		if len(config.PR.Assignees) > 0 {
			merged.PR.Assignees = config.PR.Assignees
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
		// Merge Git provider configuration
		if config.Git.Provider != "" {
			merged.Git.Provider = config.Git.Provider
		}
		if config.Git.BaseURL != "" {
			merged.Git.BaseURL = config.Git.BaseURL
		}
		if config.Git.Token != "" {
			merged.Git.Token = config.Git.Token
		}
		// Merge Notice configuration
		if config.Notice.Type != "" {
			merged.Notice.Type = config.Notice.Type
		}
		if len(config.Notice.Params) > 0 {
			merged.Notice.Params = config.Notice.Params
		}
		// Merge Scripts configuration
		if config.Scripts.PreScan != nil {
			merged.Scripts.PreScan = config.Scripts.PreScan
		}
		if config.Scripts.PreCommit != nil {
			merged.Scripts.PreCommit = config.Scripts.PreCommit
		}
		if config.Updater.Go != nil {
			merged.Updater.Go = config.Updater.Go
		}
	}

	return merged
}

// ApplyDefaults applies default values to configuration
func (c *ConfigReader) ApplyDefaults(config *DependaBotConfig) *DependaBotConfig {
	// Apply scanner defaults (prefer new format over legacy)
	// If neither scanner config nor trivy config is set, apply defaults to both for backward compatibility
	if config.Scanner.Type == "" {
		// Apply defaults to new scanner format
		config.Scanner.Type = "trivy"
		config.Scanner.Timeout = "5m"
		config.Scanner.Params = []string{"--scanners", "vuln"}
	}

	// Apply default for IncludeSubmodules if not set
	if config.Repo.IncludeSubmodules == nil {
		defaultIncludeSubmodules := false
		config.Repo.IncludeSubmodules = &defaultIncludeSubmodules
	}

	return config
}

// GetIncludeSubmodules returns the IncludeSubmodules value with default fallback
func (r *RepoConfig) GetIncludeSubmodules() bool {
	return r.IncludeSubmodules != nil && *r.IncludeSubmodules
}

// String implements fmt.Stringer interface for better debugging experience
func (c *DependaBotConfig) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		logrus.Errorf("Failed to marshal config to JSON: %v", err)
		return fmt.Sprintf("%+v", c)
	}
	return string(data)
}
