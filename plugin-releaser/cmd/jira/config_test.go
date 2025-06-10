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

var _ = Describe("IssueMeta", func() {
	Describe("GetCustomFields", func() {
		var issueMeta *jira.IssueMeta

		Context("when custom fields are properly configured", func() {
			BeforeEach(func() {
				issueMeta = &jira.IssueMeta{
					CustomFields: map[string]jira.CustomFieldValue{
						"customfield_10001": {
							Value: "42",
							Type:  jira.CustomFieldValueTypeNumber,
						},
						"customfield_10002": {
							Value: 123,
							Type:  jira.CustomFieldValueTypeNumber,
						},
						"customfield_10003": {
							Value: "test-string",
							Type:  jira.CustomFieldValueTypeString,
						},
						"customfield_10004": {
							Value: 456,
							Type:  jira.CustomFieldValueTypeString,
						},
					},
				}
			})

			It("should convert all custom fields to proper types", func() {
				result := issueMeta.GetCustomFields()

				Expect(result).To(HaveLen(4))
				Expect(result["customfield_10001"]).To(Equal(42))
				Expect(result["customfield_10002"]).To(Equal(123))
				Expect(result["customfield_10003"]).To(Equal("test-string"))
				Expect(result["customfield_10004"]).To(Equal("456"))
			})
		})

		Context("when custom fields map is empty", func() {
			BeforeEach(func() {
				issueMeta = &jira.IssueMeta{
					CustomFields: map[string]jira.CustomFieldValue{},
				}
			})

			It("should return an empty map", func() {
				result := issueMeta.GetCustomFields()

				Expect(result).To(BeEmpty())
			})
		})

		Context("when custom fields map is nil", func() {
			BeforeEach(func() {
				issueMeta = &jira.IssueMeta{
					CustomFields: nil,
				}
			})

			It("should return an empty map", func() {
				result := issueMeta.GetCustomFields()

				Expect(result).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("CustomFieldValue", func() {
	Describe("GetValue", func() {
		var customField *jira.CustomFieldValue

		Context("when type is number", func() {
			Context("with string number value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: "123",
						Type:  jira.CustomFieldValueTypeNumber,
					}
				})

				It("should convert to integer", func() {
					result := customField.GetValue()
					Expect(result).To(Equal(123))
				})
			})

			Context("with integer value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: 456,
						Type:  jira.CustomFieldValueTypeNumber,
					}
				})

				It("should return as integer", func() {
					result := customField.GetValue()
					Expect(result).To(Equal(456))
				})
			})

			Context("with float value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: 12.34,
						Type:  jira.CustomFieldValueTypeNumber,
					}
				})

				It("should convert to integer", func() {
					result := customField.GetValue()
					Expect(result).To(Equal(12))
				})
			})

			Context("with invalid number string", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: "not-a-number",
						Type:  jira.CustomFieldValueTypeNumber,
					}
				})

				It("should return zero", func() {
					result := customField.GetValue()
					Expect(result).To(Equal(0))
				})
			})
		})

		Context("when type is string", func() {
			Context("with string value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: "test-value",
						Type:  jira.CustomFieldValueTypeString,
					}
				})

				It("should return as string", func() {
					result := customField.GetValue()
					Expect(result).To(Equal("test-value"))
				})
			})

			Context("with integer value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: 789,
						Type:  jira.CustomFieldValueTypeString,
					}
				})

				It("should convert to string", func() {
					result := customField.GetValue()
					Expect(result).To(Equal("789"))
				})
			})

			Context("with boolean value", func() {
				BeforeEach(func() {
					customField = &jira.CustomFieldValue{
						Value: true,
						Type:  jira.CustomFieldValueTypeString,
					}
				})

				It("should convert to string", func() {
					result := customField.GetValue()
					Expect(result).To(Equal("true"))
				})
			})
		})

		Context("when type is unknown", func() {
			BeforeEach(func() {
				customField = &jira.CustomFieldValue{
					Value: "raw-value",
					Type:  "unknown-type",
				}
			})

			It("should return the original value", func() {
				result := customField.GetValue()
				Expect(result).To(Equal("raw-value"))
			})
		})

		Context("when type is empty", func() {
			BeforeEach(func() {
				customField = &jira.CustomFieldValue{
					Value: "some-value",
					Type:  "",
				}
			})

			It("should return the original value", func() {
				result := customField.GetValue()
				Expect(result).To(Equal("some-value"))
			})
		})
	})
})

var _ = Describe("PluginIssueMeta", func() {
	Describe("Merge", func() {
		var (
			pluginMeta  *jira.PluginIssueMeta
			defaultMeta *jira.IssueMeta
		)

		BeforeEach(func() {
			defaultMeta = &jira.IssueMeta{
				Project:     "DEFAULT_PROJECT",
				IssueType:   "Task",
				Summary:     "Default Summary",
				Description: "Default Description",
				Priority:    "Medium",
				Labels:      []string{"default", "label"},
				CustomFields: map[string]jira.CustomFieldValue{
					"default_field": {
						Value: "default_value",
						Type:  jira.CustomFieldValueTypeString,
					},
				},
			}
		})

		Context("when plugin meta has all empty values", func() {
			BeforeEach(func() {
				pluginMeta = &jira.PluginIssueMeta{
					IssueMeta: jira.IssueMeta{},
					Owner:     "test-owner",
				}
			})

			It("should merge all values from default meta", func() {
				result := pluginMeta.Merge(defaultMeta)

				Expect(result.Project).To(Equal("DEFAULT_PROJECT"))
				Expect(result.IssueType).To(Equal("Task"))
				Expect(result.Summary).To(Equal("Default Summary"))
				Expect(result.Description).To(Equal("Default Description"))
				Expect(result.Priority).To(Equal("Medium"))
				Expect(result.Labels).To(Equal([]string{"default", "label"}))
				Expect(result.CustomFields).To(HaveKey("default_field"))
				Expect(result.Owner).To(Equal("test-owner"))
			})

			It("should return the same instance", func() {
				result := pluginMeta.Merge(defaultMeta)
				Expect(result).To(BeIdenticalTo(pluginMeta))
			})
		})

		Context("when plugin meta has some values set", func() {
			BeforeEach(func() {
				pluginMeta = &jira.PluginIssueMeta{
					IssueMeta: jira.IssueMeta{
						Project: "PLUGIN_PROJECT",
						Summary: "Plugin Summary",
						Labels:  []string{"plugin", "specific"},
					},
					Owner: "plugin-owner",
				}
			})

			It("should keep existing values and merge missing ones", func() {
				result := pluginMeta.Merge(defaultMeta)

				Expect(result.Project).To(Equal("PLUGIN_PROJECT"))
				Expect(result.IssueType).To(Equal("Task"))
				Expect(result.Summary).To(Equal("Plugin Summary"))
				Expect(result.Description).To(Equal("Default Description"))
				Expect(result.Priority).To(Equal("Medium"))
				Expect(result.Labels).To(Equal([]string{"plugin", "specific"}))
				Expect(result.CustomFields).To(HaveKey("default_field"))
				Expect(result.Owner).To(Equal("plugin-owner"))
			})
		})

		Context("when plugin meta has all values set", func() {
			BeforeEach(func() {
				pluginMeta = &jira.PluginIssueMeta{
					IssueMeta: jira.IssueMeta{
						Project:     "PLUGIN_PROJECT",
						IssueType:   "Bug",
						Summary:     "Plugin Summary",
						Description: "Plugin Description",
						Priority:    "High",
						Labels:      []string{"plugin", "labels"},
						CustomFields: map[string]jira.CustomFieldValue{
							"plugin_field": {
								Value: "plugin_value",
								Type:  jira.CustomFieldValueTypeString,
							},
						},
					},
					Owner: "plugin-owner",
				}
			})

			It("should keep all plugin values and not merge from default", func() {
				result := pluginMeta.Merge(defaultMeta)

				Expect(result.Project).To(Equal("PLUGIN_PROJECT"))
				Expect(result.IssueType).To(Equal("Bug"))
				Expect(result.Summary).To(Equal("Plugin Summary"))
				Expect(result.Description).To(Equal("Plugin Description"))
				Expect(result.Priority).To(Equal("High"))
				Expect(result.Labels).To(Equal([]string{"plugin", "labels"}))
				Expect(result.CustomFields).To(HaveKey("plugin_field"))
				Expect(result.CustomFields).ToNot(HaveKey("default_field"))
				Expect(result.Owner).To(Equal("plugin-owner"))
			})
		})

		Context("when default meta is nil", func() {
			BeforeEach(func() {
				pluginMeta = &jira.PluginIssueMeta{
					IssueMeta: jira.IssueMeta{
						Project: "PLUGIN_PROJECT",
					},
					Owner: "plugin-owner",
				}
				defaultMeta = nil
			})

			It("should not panic and keep existing values", func() {
				Expect(func() {
					result := pluginMeta.Merge(defaultMeta)
					Expect(result.Project).To(Equal("PLUGIN_PROJECT"))
					Expect(result.Owner).To(Equal("plugin-owner"))
				}).ToNot(Panic())
			})
		})

		Context("when merging with empty slices and maps", func() {
			BeforeEach(func() {
				pluginMeta = &jira.PluginIssueMeta{
					IssueMeta: jira.IssueMeta{
						Labels:       []string{},                         // Empty slice
						CustomFields: map[string]jira.CustomFieldValue{}, // Empty map
					},
				}
			})

			It("should merge empty slices and maps with default values", func() {
				result := pluginMeta.Merge(defaultMeta)

				Expect(result.Labels).To(Equal([]string{"default", "label"}))
				Expect(result.CustomFields).To(HaveKey("default_field"))
			})
		})
	})
})
