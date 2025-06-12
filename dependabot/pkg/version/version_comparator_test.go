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

package version

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVersionComparator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Version Comparator Suite")
}

var _ = Describe("Unified Version Comparator", func() {
	Context("when comparing semantic versions", func() {
		It("should compare basic semantic versions correctly", func() {
			// Test basic semantic version comparison
			Expect(CompareVersions("1.0.0", "1.0.1")).To(Equal(-1))
			Expect(CompareVersions("1.0.1", "1.0.0")).To(Equal(1))
			Expect(CompareVersions("1.0.0", "1.0.0")).To(Equal(0))

			// Test major version differences
			Expect(CompareVersions("1.0.0", "2.0.0")).To(Equal(-1))
			Expect(CompareVersions("2.0.0", "1.0.0")).To(Equal(1))

			// Test minor version differences
			Expect(CompareVersions("1.1.0", "1.2.0")).To(Equal(-1))
			Expect(CompareVersions("1.2.0", "1.1.0")).To(Equal(1))
		})

		It("should handle 'v' prefix correctly", func() {
			Expect(CompareVersions("v1.0.0", "v1.0.1")).To(Equal(-1))
			Expect(CompareVersions("v1.0.1", "v1.0.0")).To(Equal(1))
			Expect(CompareVersions("v1.0.0", "v1.0.0")).To(Equal(0))

			// Mixed prefix and no prefix
			Expect(CompareVersions("v1.0.0", "1.0.1")).To(Equal(-1))
			Expect(CompareVersions("1.0.1", "v1.0.0")).To(Equal(1))
		})

		It("should handle pre-release versions", func() {
			// Pre-release versions should be less than release versions
			Expect(CompareVersions("1.0.0-alpha", "1.0.0")).To(Equal(-1))
			Expect(CompareVersions("1.0.0", "1.0.0-alpha")).To(Equal(1))

			// Compare different pre-release versions
			Expect(CompareVersions("1.0.0-alpha", "1.0.0-beta")).To(Equal(-1))
			Expect(CompareVersions("1.0.0-beta", "1.0.0-alpha")).To(Equal(1))
		})

		It("should handle empty versions", func() {
			Expect(CompareVersions("", "")).To(Equal(0))
			Expect(CompareVersions("", "1.0.0")).To(Equal(-1))
			Expect(CompareVersions("1.0.0", "")).To(Equal(1))
		})

		It("should handle different version lengths", func() {
			Expect(CompareVersions("1.0", "1.0.0")).To(Equal(0))
			Expect(CompareVersions("1.0.0", "1.0")).To(Equal(0))
			Expect(CompareVersions("1.0", "1.0.1")).To(Equal(-1))
			Expect(CompareVersions("1.0.1", "1.0")).To(Equal(1))
		})

		It("should work for Go versions", func() {
			Expect(CompareVersions("v1.20.0", "v1.21.0")).To(Equal(-1))
			Expect(CompareVersions("v1.21.0", "v1.20.0")).To(Equal(1))
		})

		It("should work for Python versions", func() {
			Expect(CompareVersions("3.8.0", "3.9.0")).To(Equal(-1))
			Expect(CompareVersions("3.9.0", "3.8.0")).To(Equal(1))
		})

		It("should work for Node.js versions", func() {
			Expect(CompareVersions("16.0.0", "18.0.0")).To(Equal(-1))
			Expect(CompareVersions("18.0.0", "16.0.0")).To(Equal(1))
		})
	})

	Context("when finding the highest version", func() {
		It("should return empty string for empty slice", func() {
			Expect(GetHighestVersion()).To(Equal(""))
		})

		It("should return the single version for single element slice", func() {
			Expect(GetHighestVersion("1.0.0")).To(Equal("1.0.0"))
		})

		It("should find the highest version from multiple versions", func() {
			versions := []string{"1.0.0", "1.2.0", "1.1.0", "2.0.0", "1.1.5", "2.0.0-beta.1"}
			Expect(GetHighestVersion(versions...)).To(Equal("2.0.0"))
		})

		It("should handle versions with 'v' prefix", func() {
			versions := []string{"v1.0.0", "v1.2.0", "v1.1.0"}
			Expect(GetHighestVersion(versions...)).To(Equal("v1.2.0"))
		})

		It("should handle mixed prefix and no prefix", func() {
			versions := []string{"v1.0.0", "1.2.0", "v1.1.0"}
			Expect(GetHighestVersion(versions...)).To(Equal("1.2.0"))
		})
	})
})
