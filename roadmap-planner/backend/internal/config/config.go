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
	Debug   bool    `mapstructure:"debug"`
	Logger  Logger  `mapstructure:"logger"`
	Jira    Jira    `mapstructure:"jira"`
	Server  Server  `mapstructure:"server"`
	Cache   Cache   `mapstructure:"cache"`
	Metrics Metrics `mapstructure:"metrics"`
}

// Logger represents logger configuration settings
type Logger struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
	Encoding    string `mapstructure:"encoding"`
}

// Jira represents Jira configuration settings
type Jira struct {
	BaseURL  string   `mapstructure:"base_url"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	Project  string   `mapstructure:"project"`
	Quarters []string `mapstructure:"quarters"`
}

// Server represents server configuration settings
type Server struct {
	Port            int    `mapstructure:"port"`
	CORS            CORS   `mapstructure:"cors"`
	StaticFilesPath string `mapstructure:"static_files_path"`
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

// Metrics represents metrics system configuration
type Metrics struct {
	Enabled            bool               `mapstructure:"enabled"`
	CollectionInterval string             `mapstructure:"collection_interval"`
	HistoricalDays     int                `mapstructure:"historical_days"`
	Prometheus         PrometheusConfig   `mapstructure:"prometheus"`
	Filters            []FilterConfig     `mapstructure:"filters"`
	Calculators        []CalculatorConfig `mapstructure:"calculators"`
}

// PrometheusConfig represents Prometheus exporter configuration
type PrometheusConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
}

// CalculatorConfig represents configuration for a metric calculator
type CalculatorConfig struct {
	Name    string                 `mapstructure:"name"`
	Enabled bool                   `mapstructure:"enabled"`
	Options map[string]interface{} `mapstructure:"options"`
}

// FilterConfig represents configuration for a data filter
// used to filter data using options
type FilterConfig struct {
	Name    string                 `mapstructure:"name"`
	Enabled bool                   `mapstructure:"enabled"`
	Options map[string]interface{} `mapstructure:"options"`
}

func (c *Metrics) GetFilter(name string) *FilterConfig {
	for _, filter := range c.Filters {
		if filter.Name == name {
			return &filter
		}
	}
	return nil
}

func (c *Metrics) GetCalculator(name string) *CalculatorConfig {
	for _, calculator := range c.Calculators {
		if calculator.Name == name {
			return &calculator
		}
	}
	return nil
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
	viper.SetDefault("server.static_file_path", "../frontend/build")
	viper.SetDefault("server.cors.allowed_origins", []string{"http://localhost:3000"})
	viper.SetDefault("jira.project", "DEVOPS")
	viper.SetDefault("jira.quarters", []string{"2025Q1", "2025Q2", "2025Q3", "2025Q4", "2026Q1", "2026Q2", "2026Q4"})
	viper.SetDefault("cache.ttl", "5m")
	viper.SetDefault("cache.refresh_interval", "1m")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.collection_interval", "5m")
	viper.SetDefault("metrics.historical_days", 365)
	viper.SetDefault("metrics.prometheus.enabled", true)
	viper.SetDefault("metrics.prometheus.path", "/metrics")
	viper.SetDefault("metrics.prometheus.namespace", "roadmap")
	viper.SetDefault("metrics.filters", []FilterConfig{})

	// Environment variable mapping
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Bind environment variables
	_ = viper.BindEnv("jira.base_url", "JIRA_BASE_URL")
	_ = viper.BindEnv("jira.username", "JIRA_USERNAME")
	_ = viper.BindEnv("jira.password", "JIRA_PASSWORD")
	_ = viper.BindEnv("server.static_files_path", "STATIC_FILES_PATH")
	_ = viper.BindEnv("server.port", "SERVER_PORT")
	_ = viper.BindEnv("debug", "DEBUG")
	_ = viper.BindEnv("metrics.enabled", "METRICS_ENABLED")
	_ = viper.BindEnv("metrics.collection_interval", "METRICS_COLLECTION_INTERVAL")
	_ = viper.BindEnv("metrics.historical_days", "METRICS_HISTORICAL_DAYS")

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

	return &config, nil
}
