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

package models

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/artifactsdir"
	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/config"
	"gopkg.in/yaml.v3"
)

// ImageSource defines the interface for retrieving images
type ImageSource interface {
	GetImages(ctx context.Context) ([]Image, error)
}

// ValuesSource represents a source of images from a values file
// valuesPath: Path to the values file
// bundle: Optional bundle name to filter images
type ValuesSource struct {
	valuesPath string
	bundle     string
}

// NewValuesSource creates a new ValuesSource instance
// valuesPath: Path to the values file
// bundle: Optional bundle name to filter images
func NewValuesSource(valuesPath, bundle string) ImageSource {
	return &ValuesSource{
		valuesPath: valuesPath,
		bundle:     bundle,
	}
}

// GetImages retrieves images from the values file
// Returns a list of images and any error that occurred
func (v *ValuesSource) GetImages(ctx context.Context) ([]Image, error) {
	bytes, err := os.ReadFile(v.valuesPath)
	if err != nil {
		return nil, err
	}

	values := &Values{}
	if err := yaml.Unmarshal(bytes, values); err != nil {
		return nil, err
	}

	images := make([]Image, 0)
	for _, item := range values.Global.Images {

		imageType := ImageTypeImage
		if strings.HasSuffix(item.Repository, "bundle") {
			imageType = ImageTypeBundle
		}
		if strings.Contains(item.Repository, "chart") {
			imageType = ImageTypeChart
		}

		image := Image{
			Repository: item.Repository,
			Tag:        item.Tag,
			Owner:      item.Owner,
			Registry:   values.Global.Registry.Address,
			Type:       imageType,
		}

		if v.bundle == "" || (v.bundle != "" && v.bundle == item.Repository) {
			images = append(images, image)
		}
	}

	return images, nil
}

// Values represents the structure of a values file
// Global: Global configuration settings
type Values struct {
	Global Global `json:"global" yaml:"global"`
}

// Global represents the global section of a values file
// Registry: Registry configuration
// Images: Map of image configurations
type Global struct {
	Registry Registry         `json:"registry" yaml:"registry,omitempty"`
	Images   map[string]Image `json:"images" yaml:"images,omitempty"`
}

// Registry represents registry configuration
// Address: The registry address
type Registry struct {
	Address string `json:"address" yaml:"address"`
}

// DirSource represents a source of images from a directory
// the directory should be a directory contains plugins, the plugins folder structure should be like definitions of alauda/artifacts
// dirPath: Path to the directory contains plugins
type DirSource struct {
	dirPath string
	plugins []string
}

func NewDirSource(dirPath string, plugins []string) ImageSource {
	return &DirSource{
		dirPath: dirPath,
		plugins: plugins,
	}
}

// GetImages retrieves images from the artifacts directory
// up to now, only return all bundle and chart images
func (d *DirSource) GetImages(ctx context.Context) ([]Image, error) {
	plugins, err := artifactsdir.NewParser(d.dirPath, d.plugins).Parse(ctx)
	if err != nil {
		return nil, err
	}

	pluginNames := make([]string, 0)
	if len(d.plugins) > 0 {
		pluginNames = d.plugins
	} else {
		for _, plugin := range plugins {
			pluginNames = append(pluginNames, plugin.Name)
		}
	}

	images := make([]Image, 0)

	for _, pluginName := range pluginNames {
		plugin, ok := plugins[pluginName]
		if !ok {
			return nil, fmt.Errorf("plugin %s not found", pluginName)
		}

		chartOrBundles, err := plugin.GetBundleOrChart()
		if err != nil {
			return nil, err
		}

		if len(plugin.Metadata.Owners) == 0 {
			return nil, fmt.Errorf("plugin %s has no owners", pluginName)
		}

		cfg := config.FromContext(ctx)
		jiraUser := cfg.GetJiraUser(plugin.Metadata.Owners[0].Email)
		if jiraUser == nil {
			return nil, fmt.Errorf("cannot find jira user by plugin's owner '%s', please add user mappings in the config.yaml. plugin: %s", plugin.Metadata.Owners[0].Email, pluginName)
		}

		for _, artifact := range chartOrBundles {
			images = append(images, Image{
				Repository: artifact.Repository,
				Tag:        artifact.Tag,
				Type:       ImageType(artifact.Type),
				Registry:   cfg.Registry.Address,
				Owner: Owner{
					Team:     jiraUser.Team,
					JiraUser: jiraUser.User,
				},
			})
		}
	}

	return images, nil
}
