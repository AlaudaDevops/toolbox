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

package git

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Global registry of platform factories
var factories = make(map[string]ClientFactory)

// RegisterFactory registers a platform factory globally
func RegisterFactory(platform string, factory ClientFactory) {
	factories[strings.ToLower(platform)] = factory
}

// CreateClient creates a client for the specified platform
func CreateClient(logger *logrus.Logger, config *Config) (GitClient, error) {
	platform := strings.ToLower(config.Platform)
	factory, exists := factories[platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", config.Platform)
	}

	return factory.CreateClient(logger, config)
}
