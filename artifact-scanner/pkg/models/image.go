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
	"fmt"
	"strings"

	"github.com/distribution/reference"
)

// Owner represents the ownership information of an image
// Team: The team that owns the image
// JiraUser: The Jira user associated with the image
type Owner struct {
	Team     string `json:"team" yaml:"team"`
	JiraUser string `json:"jira_user" yaml:"jira_user"`
}

// Image represents a container image with its metadata
// Repository: The repository name of the image
// Tag: The tag of the image
// Owner: The ownership information of the image
// Registry: The registry where the image is stored (not serialized)
// IsBundle: Whether the image is a bundle (not serialized)
type Image struct {
	Repository string `json:"repository" yaml:"repository"`
	Tag        string `json:"tag" yaml:"tag"`
	Owner      Owner  `json:"owner" yaml:"owner"`

	Registry string `json:"-" yaml:"-"`
	IsBundle bool   `json:"-" yaml:"-"`
}

// URL returns the full URL of the image including registry, repository and tag
func (i *Image) URL() string {
	return fmt.Sprintf("%s/%s:%s", i.Registry, i.Repository, i.Tag)
}

// ComponentName extracts the component name from the repository
// Removes the registry prefix and "-bundle" suffix if present
func (i *Image) ComponentName() string {
	repository := i.Repository
	lastIndex := strings.LastIndex(repository, "/")
	if lastIndex != -1 {
		repository = repository[lastIndex+1:]
	}

	return strings.TrimSuffix(repository, "-bundle")
}

// ImageFromURL creates an Image struct from a URL string
// url: The full URL of the image
// Returns the parsed Image and any error that occurred
func ImageFromURL(url string) (Image, error) {
	ref, err := reference.ParseNamed(url)
	if err != nil {
		return Image{}, err
	}

	registry := reference.Domain(ref)

	repository := reference.Path(ref)

	tag := "latest"
	if tagged, ok := ref.(reference.Tagged); ok {
		tag = tagged.Tag()
	}

	image := Image{
		Repository: repository,
		Tag:        tag,
		Registry:   registry,
		IsBundle:   strings.HasSuffix(repository, "bundle"),
	}

	return image, nil
}
