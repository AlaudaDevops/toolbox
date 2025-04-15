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

func TestNewValuesSource(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name       string
		valuesPath string
		bundle     string
	}{
		{
			name:       "with values path only",
			valuesPath: "values.yaml",
			bundle:     "",
		},
		{
			name:       "with values path and bundle",
			valuesPath: "values.yaml",
			bundle:     "test-bundle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := NewValuesSource(tc.valuesPath, tc.bundle)
			g.Expect(source).NotTo(BeNil())

			vs := source.(*ValuesSource)
			g.Expect(vs.valuesPath).To(Equal(tc.valuesPath))
			g.Expect(vs.bundle).To(Equal(tc.bundle))
		})
	}
}

func TestValuesSource_GetImages(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name      string
		path      string
		bundle    string
		expectErr bool
		expectLen int
	}{
		{
			name:      "valid yaml without bundle filter",
			path:      "testdata/valid.yaml",
			bundle:    "",
			expectErr: false,
			expectLen: 2,
		},
		{
			name:      "valid yaml with bundle filter",
			path:      "testdata/valid.yaml",
			bundle:    "test-bundle",
			expectErr: false,
			expectLen: 1,
		},
		{
			name:      "empty yaml",
			path:      "testdata/empty.yaml",
			bundle:    "",
			expectErr: false,
			expectLen: 0,
		},
		{
			name:      "file not found",
			path:      "non-existent-file.yaml",
			bundle:    "",
			expectErr: true,
			expectLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := NewValuesSource(tc.path, tc.bundle)
			images, err := source.GetImages()

			if tc.expectErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(images).To(HaveLen(tc.expectLen))

				// 仅在有结果时检查
				if tc.expectLen > 0 {
					// 第一个测试用例的额外检查
					if tc.name == "valid yaml without bundle filter" {
						g.Expect(images[0].Repository).To(Equal("app1"))
						g.Expect(images[0].Registry).To(Equal("registry.example.com"))
						g.Expect(images[1].Repository).To(Equal("test-bundle"))
						g.Expect(images[1].IsBundle).To(BeTrue())
					}

					// 第二个测试用例的额外检查
					if tc.name == "valid yaml with bundle filter" {
						g.Expect(images[0].Repository).To(Equal("test-bundle"))
						g.Expect(images[0].IsBundle).To(BeTrue())
					}
				}
			}
		})
	}
}
