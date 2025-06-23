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

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"knative.dev/pkg/logging"
)

func TestParser(t *testing.T) {
	// Use real test data
	testDataPath := filepath.Join("testdata", "artifacts")

	// Initialize context with logger
	logger, _ := logging.NewLogger("test", "debug")
	ctx := logging.WithLogger(context.Background(), logger)

	t.Run("ParseAll", func(t *testing.T) {
		g := NewWithT(t)

		parser := NewParser(testDataPath, nil)
		artifacts, err := parser.Parse(ctx)
		g.Expect(err).To(BeNil())

		expectedPlugins := 3
		g.Expect(artifacts).To(HaveLen(expectedPlugins))

		expectedPluginNames := []string{"connectors-operator", "gitlab-ce-operator", "harbor-robot-gen"}
		for _, pluginName := range expectedPluginNames {
			g.Expect(artifacts).To(HaveKey(pluginName))
		}

		plugin := artifacts["connectors-operator"]
		g.Expect(plugin.Name).To(Equal("connectors-operator"))
		g.Expect(string(plugin.Metadata.PackageType)).To(Equal("OperatorBundle"))
		g.Expect(plugin.Metadata.Owners).ToNot(BeEmpty())
		g.Expect(plugin.Metadata.Owners[0].Email).To(Equal("user1@example.com"))
		g.Expect(plugin.Metadata.Channels).ToNot(BeEmpty())
		g.Expect(plugin.Metadata.Channels[0].Channel).To(Equal("alpha"))
		g.Expect(plugin.Metadata.Channels[0].Repository).To(Equal("devops/connectors-operator-bundle"))
		g.Expect(plugin.Versions).To(HaveKeyWithValue("alpha", "v1.1.0-beta.126.gf70d7e4"))
		g.Expect(plugin.Artifact.Channels).ToNot(BeEmpty())
		g.Expect(plugin.Artifact.Channels[0].Channel).To(Equal("alpha"))
		g.Expect(plugin.Artifact.Channels[0].Artifacts).ToNot(BeEmpty())
		g.Expect(plugin.Artifact.Channels[0].Artifacts[0].Repository).To(Equal("devops/connectors-operator-bundle"))
		g.Expect(string(plugin.Artifact.Channels[0].Artifacts[0].Type)).To(Equal("Bundle"))
		g.Expect(artifacts.GetBundleOrChart("connectors-operator")).To(Equal([]Artifact{
			{
				Repository: "devops/connectors-operator-bundle",
				Tag:        "v1.1.0-beta.126.gf70d7e4",
				Type:       "Bundle",
			},
		}))

		g.Expect(artifacts.GetBundleOrChart("gitlab-ce-operator")).To(Equal([]Artifact{
			{
				Repository: "devops/gitlab-ce-operator-bundle",
				Tag:        "v17.12.0-beta.21.g5e337e0",
				Type:       "Bundle",
			},
		}))

		harborPlugin := artifacts["harbor-robot-gen"]
		g.Expect(string(harborPlugin.Metadata.PackageType)).To(Equal("ModulePlugin"))
		g.Expect(harborPlugin.Metadata.Owners[0].Email).To(Equal("user1@example.com"))
		g.Expect(harborPlugin.Metadata.Channels[0].Channel).To(Equal("default"))
		g.Expect(harborPlugin.Metadata.Channels[0].Repository).To(Equal("devops/chart-harbor-robot-gen"))
		g.Expect(harborPlugin.Metadata.Channels[0].Stage).To(Equal("alpha"))
		g.Expect(harborPlugin.Versions).To(HaveKeyWithValue("default", "v0.13.0-gb3a73ed"))
		g.Expect(harborPlugin.Artifact.Channels).ToNot(BeEmpty())
		g.Expect(harborPlugin.Artifact.Channels[0].Channel).To(Equal("default"))
		g.Expect(harborPlugin.Artifact.Channels[0].Artifacts).ToNot(BeEmpty())
		g.Expect(harborPlugin.Artifact.Channels[0].Artifacts[0].Repository).To(Equal("devops/chart-harbor-robot-gen"))
		g.Expect(string(harborPlugin.Artifact.Channels[0].Artifacts[0].Type)).To(Equal("Chart"))
		g.Expect(artifacts.GetBundleOrChart("harbor-robot-gen")).To(Equal([]Artifact{
			{
				Repository: "devops/chart-harbor-robot-gen",
				Tag:        "v0.13.0-gb3a73ed",
				Type:       "Chart",
			},
		}))
	})

	t.Run("ParseMultipleSpecificPlugins", func(t *testing.T) {
		g := NewWithT(t)

		parser := NewParser(testDataPath, []string{"gitlab-ce-operator", "harbor-robot-gen"})
		artifacts, err := parser.Parse(ctx)
		g.Expect(err).To(BeNil())

		g.Expect(artifacts).To(HaveLen(2))
		g.Expect(artifacts).To(HaveKey("gitlab-ce-operator"))
		g.Expect(artifacts).To(HaveKey("harbor-robot-gen"))

		// Verify different package types
		g.Expect(string(artifacts["gitlab-ce-operator"].Metadata.PackageType)).To(Equal("OperatorBundle"))
		g.Expect(string(artifacts["harbor-robot-gen"].Metadata.PackageType)).To(Equal("ModulePlugin"))
	})

	t.Run("ParseNonExistentPlugin", func(t *testing.T) {
		g := NewWithT(t)

		parser := NewParser(testDataPath, []string{"non-existent-plugin"})
		_, err := parser.Parse(ctx)
		g.Expect(err).ToNot(BeNil())
	})

	t.Run("ParseInvalidPlugin", func(t *testing.T) {
		g := NewWithT(t)

		// connectors-operator-test only has versions.yaml, missing metadata.yaml and artifacts.yaml
		parser := NewParser(testDataPath, []string{"connectors-operator-test"})
		artifacts, err := parser.Parse(ctx)
		g.Expect(err).To(BeNil())

		// Should return empty artifacts since the plugin is skipped
		g.Expect(artifacts).To(BeEmpty())
	})

	t.Run("ParseInvalidBasePath", func(t *testing.T) {
		g := NewWithT(t)

		parser := NewParser("/non/existent/path", []string{})
		_, err := parser.Parse(ctx)
		g.Expect(err).ToNot(BeNil())
	})
}
