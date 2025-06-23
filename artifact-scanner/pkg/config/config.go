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

package config

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
// Jira: Jira configuration settings
// Ops: OPS API configuration settings
type Config struct {
	Jira     Jira     `json:"jira" yaml:"jira"`
	Ops      Ops      `json:"ops" yaml:"ops"`
	Users    []User   `json:"users" yaml:"users"`
	Registry Registry `json:"registry" yaml:"registry"`
}

// Jira represents Jira configuration settings
// BaseURL: The base URL of the Jira instance
// Username: The username for Jira authentication
// Password: The password for Jira authentication
type Jira struct {
	BaseURL  string `json:"baseURL" yaml:"baseURL"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// Ops represents OPS API configuration settings
// BaseURL: The base URL of the OPS API
type Ops struct {
	BaseURL string `json:"baseURL" yaml:"baseURL"`
}

// Registry represents the registry that pulls the images
type Registry struct {
	Address string `json:"address" yaml:"address"`
}

type User struct {
	Email string   `json:"email" yaml:"email"`
	Jira  JiraUser `json:"jira" yaml:"jira"`
}

// JiraUser represents a Jira user
type JiraUser struct {
	// Jira username
	User string `json:"user" yaml:"user"`
	// Jira team key
	Team string `json:"team" yaml:"team"`
}

// Load loads the configuration from a YAML file
// path: The path to the configuration file
// Returns the loaded configuration and any error that occurred
func Load(path string) (*Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return config, nil
}

type contextKey struct{}

var (
	ContextKeyConfig contextKey = contextKey{}
)

// InjectContext injects the Config into the context
// ctx: The context to inject the Config into
// Returns the context with the Config injected
func (c *Config) InjectContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextKeyConfig, c)
}

// FromContext returns the Config from the context
// ctx: The context to get the Config from
// Returns the Config from the context
func FromContext(ctx context.Context) *Config {
	return ctx.Value(ContextKeyConfig).(*Config)
}

// GetJiraUser returns the Jira user for a given email
// email: The email of the user
// Returns the Jira user for the given email
func (c *Config) GetJiraUser(email string) *JiraUser {
	for _, u := range c.Users {
		if u.Email == email {
			return &u.Jira
		}
	}
	return nil
}
