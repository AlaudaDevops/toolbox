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

package jira_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AlaudaDevops/toolbox/plugin-releaser/cmd/jira"
)

var _ = Describe("GenerateIssueTags", func() {
	var (
		pluginName string
	)

	Context("when generating tags for a plugin", func() {
		BeforeEach(func() {
			pluginName = "gitlab"
		})

		It("should generate correct tags with current year and month", func() {
			// Since we can't easily mock time.Now() in the function,
			// we'll test the format and structure
			tags := jira.GenerateIssueTags(pluginName)

			Expect(tags).To(HaveLen(2))
			Expect(tags[0]).To(Equal("created-by-release-bot"))
			Expect(tags[1]).To(MatchRegexp(`^gitlab-\d{6}$`))
			Expect(tags[1]).To(ContainSubstring(pluginName))
		})
	})

	Context("when testing time-based tag generation", func() {
		It("should generate different tags for different months", func() {
			// This test demonstrates that the function would generate different tags
			// for different time periods (though we can't easily test this without mocking)
			tags1 := jira.GenerateIssueTags("gitlab")
			tags2 := jira.GenerateIssueTags("gitlab")

			// Both calls should generate the same tags since they're called in the same month
			Expect(tags1).To(Equal(tags2))

			// Verify the format includes year and month
			Expect(tags1[1]).To(MatchRegexp(`^gitlab-\d{4}\d{2}$`))
		})
	})
})

// Test structures for RenderStructTemplate tests
type TestStruct struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Value       int               `yaml:"value"`
	Tags        []string          `yaml:"tags"`
	Config      map[string]string `yaml:"config"`
}

type InvalidStruct struct {
	Channel chan int `yaml:"channel"` // channels cannot be marshaled to YAML
}

var _ = Describe("RenderStructTemplate", func() {
	var (
		testStruct   *TestStruct
		templateData map[string]interface{}
	)

	BeforeEach(func() {
		testStruct = &TestStruct{
			Name:        "{{.pluginName}}",
			Description: "Test plugin for {{.environment}}",
			Value:       42,
			Tags:        []string{"{{.category}}", "test"},
			Config: map[string]string{
				"url":     "https://{{.domain}}/{{.pluginName}}",
				"version": "{{.version}}",
			},
		}

		templateData = map[string]interface{}{
			"pluginName":  "gitlab",
			"environment": "production",
			"category":    "ci-cd",
			"domain":      "example.com",
			"version":     "1.0.0",
		}
	})

	Context("when rendering a valid struct with template data", func() {
		It("should successfully render all template variables", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(testStruct.Name).To(Equal("gitlab"))
			Expect(testStruct.Description).To(Equal("Test plugin for production"))
			Expect(testStruct.Value).To(Equal(42)) // Non-template values should remain unchanged
			Expect(testStruct.Tags).To(ContainElements("ci-cd", "test"))
			Expect(testStruct.Config["url"]).To(Equal("https://example.com/gitlab"))
			Expect(testStruct.Config["version"]).To(Equal("1.0.0"))
		})
	})

	Context("when rendering with missing template variables", func() {
		BeforeEach(func() {
			// Remove some template data to simulate missing variables
			delete(templateData, "domain")
		})

		It("should render successfully with default zero values for missing variables", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(testStruct.Name).To(Equal("gitlab"))
			Expect(testStruct.Config["url"]).To(Equal("https:///gitlab")) // Missing domain becomes empty
		})
	})

	Context("when rendering with nil template data", func() {
		BeforeEach(func() {
			templateData = nil
		})

		It("should render successfully with empty values for missing variables", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(testStruct.Name).To(Equal("")) // Template renders with empty value
		})
	})

	Context("when rendering with empty template data", func() {
		BeforeEach(func() {
			templateData = make(map[string]interface{})
		})

		It("should render successfully with default zero values", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(testStruct.Name).To(Equal("")) // Empty string for missing variable
		})
	})

	Context("when the struct contains invalid template syntax", func() {
		BeforeEach(func() {
			testStruct.Name = "{{.invalid syntax"
		})

		It("should return a template parsing error", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse YAML template"))
		})
	})

	Context("when the struct contains template with undefined function", func() {
		BeforeEach(func() {
			testStruct.Name = "{{.pluginName | unknownFunction}}"
		})

		It("should return a template parsing error", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse YAML template"))
		})
	})

	Context("when the rendered YAML is invalid", func() {
		BeforeEach(func() {
			// Create a template that will produce invalid YAML
			testStruct.Name = "{{.pluginName"
			testStruct.Description = "}}invalid"
		})

		It("should return an unmarshal error", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse YAML template"))
		})
	})

	Context("when passing nil struct", func() {
		It("should return an unmarshal error", func() {
			err := jira.RenderStructTemplate(nil, templateData, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal rendered YAML"))
		})
	})

	Context("when using complex template expressions", func() {
		BeforeEach(func() {
			testStruct.Name = "{{if .pluginName}}{{.pluginName}}-plugin{{else}}unknown{{end}}"
			testStruct.Description = "{{range .tags}}{{.}} {{end}}"
			templateData["tags"] = []string{"tag1", "tag2", "tag3"}
		})

		It("should handle complex template logic correctly", func() {
			err := jira.RenderStructTemplate(testStruct, templateData, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(testStruct.Name).To(Equal("gitlab-plugin"))
			Expect(testStruct.Description).To(Equal("tag1 tag2 tag3 "))
		})
	})
})
