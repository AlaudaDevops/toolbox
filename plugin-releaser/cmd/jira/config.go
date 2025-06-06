// Package jira provides configuration structures and utilities for Jira issue management
// This package defines the configuration format for creating and managing Jira issues
// in the plugin release automation workflow
package jira

import (
	"os"

	"github.com/AlaudaDevops/toolbox/plugin-releaser/pkg/types"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

// JiraConfig represents the complete Jira configuration including connection settings and issue metadata
// It embeds the base JiraConfig from types package and extends it with issue-specific metadata
type JiraConfig struct {
	types.JiraConfig `yaml:",inline"`
	IssueMeta        `yaml:",inline"`
}

// IssueMeta contains metadata for creating Jira issues
// This structure defines all the standard fields that can be set when creating a Jira issue
type IssueMeta struct {
	// Project key in Jira (e.g., "DEVOPS")
	Project string `yaml:"project"`
	// Type of issue to create (e.g., "Task", "Bug", "Story")
	IssueType string `yaml:"issueType"`
	// Brief description of the issue
	Summary string `yaml:"summary"`
	// Detailed description of the issue
	Description string `yaml:"description"`
	// Priority level (e.g., "High", "Medium", "Low")
	Priority string `yaml:"priority"`
	// List of labels to apply to the issue
	Labels []string `yaml:"labels"`

	// Custom field values specific to the Jira instance
	CustomFields map[string]CustomFieldValue `yaml:"customFields"`
}

// GetCustomFields converts the custom field values to the format expected by the Jira API
// Returns a map where keys are custom field IDs and values are properly typed values
func (i *IssueMeta) GetCustomFields() map[string]any {
	customFields := make(map[string]any)
	for k, v := range i.CustomFields {
		customFields[k] = v.GetValue()
	}
	return customFields
}

// CustomFieldValueType represents the data type of a custom field value
// Jira custom fields can have different types and need to be handled accordingly
type CustomFieldValueType string

const (
	// CustomFieldValueTypeNumber represents numeric custom fields (integers, floats)
	CustomFieldValueTypeNumber CustomFieldValueType = "number"
	// CustomFieldValueTypeString represents text-based custom fields
	CustomFieldValueTypeString CustomFieldValueType = "string"
)

// CustomFieldValue represents a custom field with its value and type information
// This allows for proper type conversion when setting custom field values in Jira
type CustomFieldValue struct {
	// The actual value to set for the custom field
	Value any `yaml:"value"`
	// The expected type for proper conversion
	Type CustomFieldValueType `yaml:"type"`
}

// GetValue returns the custom field value converted to the appropriate type
// This method handles type conversion based on the specified Type field
func (c *CustomFieldValue) GetValue() any {
	if c.Type == CustomFieldValueTypeNumber {
		return cast.ToInt(c.Value)
	} else if c.Type == CustomFieldValueTypeString {
		return cast.ToString(c.Value)
	}
	return c.Value
}

// PluginIssueMeta extends IssueMeta with plugin-specific metadata
// This structure represents issue metadata for a specific plugin, including ownership information
type PluginIssueMeta struct {
	IssueMeta `yaml:",inline"`
	// Username of the person responsible for this plugin
	Owner string `yaml:"owner"`
}

// Merge combines this PluginIssueMeta with another IssueMeta, filling in missing values
// The current values take precedence over the provided defaults
// other: the default IssueMeta to merge with
// Returns the current PluginIssueMeta with merged values
func (i *PluginIssueMeta) Merge(other *IssueMeta) *PluginIssueMeta {
	if other == nil {
		return i
	}

	if i.Project == "" {
		i.Project = other.Project
	}
	if i.IssueType == "" {
		i.IssueType = other.IssueType
	}
	if i.Summary == "" {
		i.Summary = other.Summary
	}
	if i.Description == "" {
		i.Description = other.Description
	}
	if i.Priority == "" {
		i.Priority = other.Priority
	}
	if len(i.Labels) == 0 {
		i.Labels = other.Labels
	}
	if len(i.CustomFields) == 0 {
		i.CustomFields = other.CustomFields
	}
	return i
}

// Config is the root configuration structure for the plugin-releaser tool
// It contains global Jira settings and plugin-specific configurations
type Config struct {
	// Global Jira configuration and default issue settings
	Jira JiraConfig `yaml:"jira"`
	// Plugin-specific configurations indexed by plugin name
	Plugins map[string]PluginIssueMeta `yaml:"plugins"`
}

// LoadConfig loads configuration from a YAML file
// path: the file path to the configuration file
// Returns the parsed Config struct and any error that occurred during loading
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
