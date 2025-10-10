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

package cmd

import (
	"fmt"
	"path/filepath"

	pkgtesting "github.com/AlaudaDevops/pkg/testing"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type parseCommandFixture struct {
	Cases []parseCommandCase `yaml:"cases"`
}

type parseCommandCase struct {
	Description string   `yaml:"description"`
	Comment     string   `yaml:"comment"`
	Command     string   `yaml:"command"`
	Args        []string `yaml:"args"`
}

var checkboxIssueArgumentCases = loadParseCommandCases()
var checkboxIssueArgumentCaseIndex = indexParseCommandCases(checkboxIssueArgumentCases)

func loadParseCommandCases() []parseCommandCase {
	var fixture parseCommandFixture
	pkgtesting.MustLoadYaml(filepath.Join("testdata", "checkbox_issue_args", "cases.yaml"), &fixture)
	return fixture.Cases
}

func indexParseCommandCases(cases []parseCommandCase) map[string]parseCommandCase {
	indexed := make(map[string]parseCommandCase, len(cases))
	for _, tc := range cases {
		indexed[tc.Description] = tc
	}
	return indexed
}

func getParseCommandCase(description string) parseCommandCase {
	tc, ok := checkboxIssueArgumentCaseIndex[description]
	if !ok {
		panic(fmt.Sprintf("missing parse command case: %s", description))
	}
	return tc
}

var _ = ginkgo.Describe("parseCommand argument handling", func() {
	var option *PROption

	ginkgo.BeforeEach(func() {
		option = NewPROption()
	})

	ginkgo.DescribeTable("parses checkbox-issue arguments with quoting",
		func(tc parseCommandCase) {
			parsed, err := option.parseCommand(tc.Comment)
			Expect(err).NotTo(HaveOccurred())

			Expect(parsed.Command).To(Equal(tc.Command))
			Expect(parsed.Args).To(Equal(tc.Args))
		},
		ginkgo.Entry("handles quoted flags for title and author", getParseCommandCase("handles quoted flags for title and author")),
		ginkgo.Entry("handles equals style quoting for title and author", getParseCommandCase("handles equals style quoting for title and author")),
	)
})
