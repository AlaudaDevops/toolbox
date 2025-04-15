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
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ImageSource defines the interface for retrieving images
type ImageSource interface {
	GetImages() ([]Image, error)
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
func (v *ValuesSource) GetImages() ([]Image, error) {
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
		image := Image{
			Repository: item.Repository,
			Tag:        item.Tag,
			Owner:      item.Owner,
			Registry:   values.Global.Registry.Address,
			IsBundle:   strings.HasSuffix(item.Repository, "bundle"),
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
