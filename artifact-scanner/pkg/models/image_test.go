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
	"testing"

	. "github.com/onsi/gomega"
)

func TestImage(t *testing.T) {
	tests := []struct {
		name     string
		image    Image
		expected string
	}{
		{
			name: "basic image URL",
			image: Image{
				Registry:   "docker.io",
				Repository: "library/nginx",
				Tag:        "latest",
			},
			expected: "docker.io/library/nginx:latest",
		},
		{
			name: "image with custom registry",
			image: Image{
				Registry:   "myregistry.com",
				Repository: "app/backend",
				Tag:        "v1.0.0",
			},
			expected: "myregistry.com/app/backend:v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(tt.image.URL()).To(Equal(tt.expected))
		})
	}
}

func TestComponentName(t *testing.T) {
	tests := []struct {
		name     string
		image    Image
		expected string
	}{
		{
			name: "simple repository",
			image: Image{
				Repository: "nginx",
			},
			expected: "nginx",
		},
		{
			name: "repository with path",
			image: Image{
				Repository: "library/nginx",
			},
			expected: "nginx",
		},
		{
			name: "bundle repository",
			image: Image{
				Repository: "app/backend-bundle",
			},
			expected: "backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(tt.image.ComponentName()).To(Equal(tt.expected))
		})
	}
}

func TestImageFromURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expected    Image
		expectError bool
	}{
		{
			name: "valid image with tag",
			url:  "docker.io/library/nginx:1.21",
			expected: Image{
				Registry:   "docker.io",
				Repository: "library/nginx",
				Tag:        "1.21",
				IsBundle:   false,
			},
			expectError: false,
		},
		{
			name: "valid image without tag",
			url:  "myregistry.com/app/backend",
			expected: Image{
				Registry:   "myregistry.com",
				Repository: "app/backend",
				Tag:        "latest",
				IsBundle:   false,
			},
			expectError: false,
		},
		{
			name: "bundle image",
			url:  "docker.io/library/app-bundle:latest",
			expected: Image{
				Registry:   "docker.io",
				Repository: "library/app-bundle",
				Tag:        "latest",
				IsBundle:   true,
			},
			expectError: false,
		},
		{
			name:        "invalid image URL",
			url:         "invalid@url",
			expected:    Image{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			result, err := ImageFromURL(tt.url)

			if tt.expectError {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(result.Registry).To(Equal(tt.expected.Registry))
				g.Expect(result.Repository).To(Equal(tt.expected.Repository))
				g.Expect(result.Tag).To(Equal(tt.expected.Tag))
				g.Expect(result.IsBundle).To(Equal(tt.expected.IsBundle))
			}
		})
	}
}
