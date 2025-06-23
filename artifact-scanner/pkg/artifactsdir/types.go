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

import "fmt"

// Plugins is a map of plugin name to Plugin
type Plugins map[string]Plugin

// Plugin represents a single plugin with all its configuration
type Plugin struct {
	Name     string         `json:"name" yaml:"name"`
	Versions PluginVersion  `json:"version" yaml:"version"`
	Metadata PluginMetadata `json:"metadata" yaml:"metadata"`
	Artifact PluginArtifact `json:"artifacts" yaml:"artifacts"`
}

// PluginVersion represents the versions.yaml content
// It's a map of channel name to version string
type PluginVersion map[string]string

type PackageType string

const (
	PackageTypeOperatorBundle PackageType = "OperatorBundle"
	PackageTypeModulePlugin   PackageType = "ModulePlugin"
)

// PluginMetadata represents the metadata.yaml content
type PluginMetadata struct {
	PackageType PackageType `json:"packageType" yaml:"packageType"`
	Owners      []Owner     `json:"owners" yaml:"owners"`
	Channels    []Channel   `json:"channels" yaml:"channels"`
}

// Owner represents an owner in metadata.yaml
type Owner struct {
	Email string `json:"email" yaml:"email"`
}

// Channel represents a channel configuration in metadata.yaml
type Channel struct {
	Channel        string `json:"channel" yaml:"channel"`
	DefaultChannel bool   `json:"defaultChannel" yaml:"defaultChannel"`
	Repository     string `json:"repository" yaml:"repository"`
	Stage          string `json:"stage" yaml:"stage"`
}

// PluginArtifact represents the artifacts.yaml content
type PluginArtifact struct {
	Channels []ArtifactChannel `json:"channels" yaml:"channels"`
}

// ArtifactChannel represents a channel with its artifacts in artifacts.yaml
type ArtifactChannel struct {
	Channel   string     `json:"channel" yaml:"channel"`
	Version   string     `json:"version" yaml:"version"`
	Artifacts []Artifact `json:"artifacts" yaml:"artifacts"`
}

type ArtifactType string

const (
	ArtifactTypeBundle ArtifactType = "Bundle"
	ArtifactTypeChart  ArtifactType = "Chart"
	ArtifactTypeImage  ArtifactType = "Image"
)

// Artifact represents a single artifact
type Artifact struct {
	Repository string       `json:"repository" yaml:"repository"`
	Tag        string       `json:"tag" yaml:"tag"`
	Digest     string       `json:"digest" yaml:"digest"`
	Type       ArtifactType `json:"type" yaml:"type"` // Bundle, Image, Chart
}

// GetBundleOrChart returns the images of a plugin, not include related images
// it will return the images of all channels of the plugin
func (plugins *Plugins) GetBundleOrChart(pluginName string) ([]Artifact, error) {
	plugin, ok := (*plugins)[pluginName]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	artifacts, err := plugin.GetBundleOrChart()
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

// GetBundleOrChart returns the images of a plugin, not include related images
// it will return the images of all channels of the plugin
func (plugin *Plugin) GetBundleOrChart() ([]Artifact, error) {
	artifacts := []Artifact{}
	for _, channel := range plugin.Metadata.Channels {
		name := channel.Channel
		version := plugin.Versions[name]
		if version == "" {
			continue
		}

		var artifactType = ArtifactTypeBundle
		if plugin.Metadata.PackageType == PackageTypeModulePlugin {
			artifactType = ArtifactTypeChart
		}

		contains := false
		for _, artifact := range artifacts {
			if artifact.Repository == channel.Repository && artifact.Tag == version && artifact.Type == artifactType {
				contains = true
				break
			}
		}

		if !contains {
			artifacts = append(artifacts, Artifact{
				Repository: channel.Repository,
				Tag:        version,
				Type:       artifactType,
			})
		}
	}
	return artifacts, nil
}
