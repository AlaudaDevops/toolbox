/*
	Copyright 2025 AlaudaDevops authors

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
	"errors"
	"strings"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
)

// CopyConfig represents the configuration for copying files from multiple sources.
type CopyConfig struct {
	// List of source configurations
	Sources []CopySource `json:"sources"`
	// Target configuration for copying
	Target *CopyTarget `json:"target,omitempty"`
}

// CopySource defines the source configuration for copying files.
type CopySource struct {
	Name string `json:"name"` // Custom name for the source
	// Repository is currently not supported
	Dir *Directory `json:"dir"` // Directory information (mutually exclusive with Repository)
}

// Directory contains information about a local directory.
type Directory struct {
	Path string `json:"path"` // Path to the local directory
}

// CopyTarget defines the target configuration for copying files.
type CopyTarget struct {
	CopyTo string      `json:"copyTo"` // Destination directory for copied files
	LinkTo string      `json:"linkTo"` // Base directory for relative paths
	Links  []CopyLinks `json:"links"`  // List of link configurations
}

// CopyLinks defines the linking configuration for files or directories.
type CopyLinks struct {
	From   string `json:"from"`   // Source path for linking
	Target string `json:"target"` // Target path for the link
}

// Validate whole configuration
// TODO: Refactor separating all validation methods into each separated struct
func (c *CopyConfig) Validate(ctx context.Context) error {
	if c == nil {
		return errors.New("config should not be nil")
	}
	errs := make([]error, 0, 5)
	if len(c.Sources) == 0 {
		errs = append(errs, errors.New("config.sources should have at least one configuration"))
	}
	for _, source := range c.Sources {
		if source.Name == "" {
			errs = append(errs, errors.New("config.sources.name should not be empty"))
		}
		if source.Dir == nil {
			errs = append(errs, errors.New("config.sources.dir should not be nil"))
		}
	}
	if c.Target.CopyTo == "" || c.Target.LinkTo == "" || c.Target.CopyTo == c.Target.LinkTo {
		errs = append(errs, errors.New("config.target.copyTo and base should point to different folders"))
	}
	if len(c.Target.Links) == 0 {
		errs = append(errs, errors.New("config.target.links should have at least one configuration"))
	}
	for _, link := range c.Target.Links {
		if link.From == "" || link.Target == "" {
			errs = append(errs, errors.New("config.target.links.from and target should not be empty"))
		}
	}
	return errors.Join(errs...)
}

// Default configuration for CopyTarget
func (CopyTarget) Default() *CopyTarget {
	return &CopyTarget{
		CopyTo: "imported-docs",
		LinkTo: "docs",
		Links: []CopyLinks{
			{From: "public/<name>", Target: "public/<name>"},
			{From: "shared/crds", Target: "shared/crds/<name>"},
			{From: "en/apis/kubernetes_apis", Target: "en/apis/kubernetes_apis/<name>"},
			{From: "en", Target: "en/<name>"},
		},
	}
}

// Parse returns the list of link requests for the given source
func (t CopyTarget) Parse(source CopySource) (links []ifs.LinkRequest) {
	for _, link := range t.Links {
		links = append(links, link.Parse(source))
	}
	return
}

// Parse returns the link request for the given source
func (t CopyLinks) Parse(source CopySource) ifs.LinkRequest {
	// there is only one placeholder <name>
	// simplified implementation for now
	return ifs.LinkRequest{
		Source:      strings.ReplaceAll(t.From, "<name>", source.Name),
		Destination: strings.ReplaceAll(t.Target, "<name>", source.Name),
	}
}
