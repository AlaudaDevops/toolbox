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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
// Jira: Jira configuration settings
// Ops: OPS API configuration settings
type Config struct {
	Jira Jira `json:"jira" yaml:"jira"`
	Ops  Ops  `json:"ops" yaml:"ops"`
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
