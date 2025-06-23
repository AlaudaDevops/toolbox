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

package artifactsdir

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"knative.dev/pkg/logging"
)

const (
	ArtifactsFile = "artifacts.yaml"
	MetadataFile  = "metadata.yaml"
	VersionsFile  = "versions.yaml"
)

// Parser handles parsing of artifacts directory structure
type Parser struct {
	basePath    string
	pluginNames []string // Empty slice means parse all plugins
}

// NewParser creates a new Parser instance
func NewParser(basePath string, pluginNames []string) *Parser {
	return &Parser{
		basePath:    basePath,
		pluginNames: pluginNames,
	}
}

// Parse parses plugins from the directory based on the configured plugin names
// If pluginNames is empty, it parses all plugins in the directory
func (p *Parser) Parse(ctx context.Context) (Plugins, error) {
	logger := logging.FromContext(ctx).With("basePath", p.basePath)

	artifacts := make(Plugins)
	var plugins []string
	if len(p.pluginNames) > 0 {
		plugins = p.pluginNames
	}

	if len(plugins) == 0 {
		entries, err := os.ReadDir(p.basePath)
		if err != nil {
			logger.Errorw("failed to read directory", "error", err)
			return nil, fmt.Errorf("failed to read directory %s: %w", p.basePath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				plugins = append(plugins, entry.Name())
			}
		}
	}

	for _, pluginName := range plugins {
		logger := logger.With("plugin", pluginName)

		plugin, err := p.parsePlugin(ctx, pluginName)
		if err != nil {
			logger.Errorw("failed to parse plugin", "error", err)

			if len(p.pluginNames) > 0 {
				return nil, fmt.Errorf("failed to parse plugin %s: %w", pluginName, err)
			}

			continue // Continue with other plugins when parsing all
		}

		if plugin == nil {
			logger.Warnw("skipping plugin", "plugin", pluginName)
			continue
		}

		artifacts[pluginName] = *plugin
	}

	return artifacts, nil
}

// parsePlugin parses a single plugin directory
func (p *Parser) parsePlugin(ctx context.Context, pluginName string) (*Plugin, error) {
	logger := logging.FromContext(ctx).With("plugin", pluginName)

	pluginDir := filepath.Join(p.basePath, pluginName)

	if _, err := os.Stat(pluginDir); err != nil {
		logger.Warnw("visit plugin directory failed", "path", pluginDir, "err", err)
		return nil, fmt.Errorf("visit plugin directory failed: %s, err: %w", pluginDir, err)
	}

	requiredFiles := []string{ArtifactsFile, MetadataFile, VersionsFile}
	missingFiles := []string{}

	for _, file := range requiredFiles {
		filePath := filepath.Join(pluginDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, file)
		}
	}

	if len(missingFiles) > 0 {
		logger.Warnw("missing required files",
			"missingFiles", missingFiles,
			"requiredFiles", requiredFiles)
		return nil, nil
	}

	versions, err := p.parseVersionsFile(ctx, filepath.Join(pluginDir, VersionsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse versions file: %w", err)
	}

	metadata, err := p.parseMetadataFile(ctx, filepath.Join(pluginDir, MetadataFile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %w", err)
	}

	artifacts, err := p.parseArtifactsFile(ctx, filepath.Join(pluginDir, ArtifactsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse artifacts file: %w", err)
	}

	plugin := &Plugin{
		Name:     pluginName,
		Versions: versions,
		Metadata: metadata,
		Artifact: artifacts,
	}

	logger.Debugw("successfully parsed plugin")
	return plugin, nil
}

// parseVersionsFile parses the versions.yaml file
func (p *Parser) parseVersionsFile(ctx context.Context, filePath string) (PluginVersion, error) {
	logger := logging.FromContext(ctx).With("file", filePath)
	logger.Debugw("parsing versions file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var versions PluginVersion
	if err := yaml.Unmarshal(data, &versions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	logger.Debugw("successfully parsed versions file", "channels", len(versions))
	return versions, nil
}

// parseMetadataFile parses the metadata.yaml file
func (p *Parser) parseMetadataFile(ctx context.Context, filePath string) (PluginMetadata, error) {
	logger := logging.FromContext(ctx).With("file", filePath)
	logger.Debugw("parsing metadata file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to read file: %w", err)
	}

	var metadata PluginMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	logger.Debugw("successfully parsed metadata file", "packageType", metadata.PackageType)
	return metadata, nil
}

// parseArtifactsFile parses the artifacts.yaml file
func (p *Parser) parseArtifactsFile(ctx context.Context, filePath string) (PluginArtifact, error) {
	logger := logging.FromContext(ctx).With("file", filePath)
	logger.Debugw("parsing artifacts file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return PluginArtifact{}, fmt.Errorf("failed to read file: %w", err)
	}

	var artifacts PluginArtifact
	if err := yaml.Unmarshal(data, &artifacts); err != nil {
		return PluginArtifact{}, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	logger.Debugw("successfully parsed artifacts file", "channels", len(artifacts.Channels))
	return artifacts, nil
}
