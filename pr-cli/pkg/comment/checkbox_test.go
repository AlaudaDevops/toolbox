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

package comment_test

import (
	pkgtesting "github.com/AlaudaDevops/pkg/testing"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type toggleFixture struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

var _ = ginkgo.Describe("Checkbox helpers", func() {
	ginkgo.Describe("ToggleAllUncheckedCheckboxes", func() {
		type testCase struct {
			description string
			sourceFile  string
			goldenFile  string
		}

		ginkgo.DescribeTable("should toggle unchecked checkboxes",
			func(tc testCase) {
				sourcePath := tc.sourceFile
				goldenPath := tc.goldenFile

				source := toggleFixture{}
				pkgtesting.MustLoadYaml(sourcePath, &source)

				golden := toggleFixture{}
				pkgtesting.MustLoadYaml(goldenPath, &golden)

				updated, count := comment.ToggleAllUncheckedCheckboxes(source.Input)
				Expect(updated).To(Equal(golden.Output))

				if comment.HasUncheckedCheckbox(source.Input) {
					Expect(count).To(BeNumerically(">", 0))
				} else {
					Expect(count).To(Equal(0))
				}
			},
			ginkgo.Entry("basic checkbox toggle", testCase{
				description: "should toggle all unchecked checkboxes",
				sourceFile:  "testdata/toggle_basic/source.yaml",
				goldenFile:  "testdata/toggle_basic/golden.yaml",
			}),
			ginkgo.Entry("no unchecked checkbox", testCase{
				description: "should detect no unchecked checkbox",
				sourceFile:  "testdata/toggle_none/source.yaml",
				goldenFile:  "testdata/toggle_none/golden.yaml",
			}),
		)
	})
})
