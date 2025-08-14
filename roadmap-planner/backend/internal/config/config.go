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
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Debug  bool   `mapstructure:"debug"`
	Logger Logger `mapstructure:"logger"`
	Jira   Jira   `mapstructure:"jira"`
	Server Server `mapstructure:"server"`
	Cache  Cache  `mapstructure:"cache"`
}

// Logger represents logger configuration settings
type Logger struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
	Encoding    string `mapstructure:"encoding"`
}

// Jira represents Jira configuration settings
type Jira struct {
	BaseURL  string `mapstructure:"base_url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Project  string `mapstructure:"project"`
}

// Server represents server configuration settings
type Server struct {
	Port int  `mapstructure:"port"`
	CORS CORS `mapstructure:"cors"`
}

// CORS represents CORS configuration settings
type CORS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// Cache represents cache configuration settings
type Cache struct {
	TTL             string `mapstructure:"ttl"`
	RefreshInterval string `mapstructure:"refresh_interval"`
}

// Load loads the configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.roadmap-planner")

	// Set default values
	viper.SetDefault("debug", false)
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.development", false)
	viper.SetDefault("logger.encoding", "json")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.cors.allowed_origins", []string{"http://localhost:3000"})
	viper.SetDefault("jira.project", "DEVOPS")
	viper.SetDefault("cache.ttl", "5m")
	viper.SetDefault("cache.refresh_interval", "1m")

	// Environment variable mapping
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Bind environment variables
	viper.BindEnv("jira.base_url", "JIRA_BASE_URL")
	viper.BindEnv("jira.username", "JIRA_USERNAME")
	viper.BindEnv("jira.password", "JIRA_PASSWORD")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("debug", "DEBUG")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if config.Jira.BaseURL == "" {
		return nil, fmt.Errorf("jira.base_url is required")
	}
	if config.Jira.Username == "" {
		return nil, fmt.Errorf("jira.username is required")
	}
	if config.Jira.Password == "" {
		return nil, fmt.Errorf("jira.password is required")
	}

	return &config, nil
}
