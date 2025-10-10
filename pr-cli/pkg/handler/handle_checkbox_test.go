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

package handler_test

import (
	"fmt"

	pkgtesting "github.com/AlaudaDevops/pkg/testing"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

type handlerFixture struct {
	PR struct {
		Body string `yaml:"body"`
		URL  string `yaml:"url"`
	} `yaml:"pr"`
	Comments []struct {
		ID   int64  `yaml:"id"`
		URL  string `yaml:"url"`
		Body string `yaml:"body"`
	} `yaml:"comments"`
}

var _ = ginkgo.Describe("HandleCheckbox", func() {
	var (
		ctrl        *gomock.Controller
		mockClient  *mock_git.MockGitClient
		testHandler *handler.PRHandler
		cfg         *config.Config
	)

	createHandler := func() {
		cfg = &config.Config{
			CommentSender: "reviewer",
		}
		var err error
		testHandler, err = handler.NewPRHandlerWithClient(logrus.New(), cfg, mockClient, "author")
		Expect(err).To(BeNil())
	}

	ginkgo.BeforeEach(func() {
		ctrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = mock_git.NewMockGitClient(ctrl)
		createHandler()
	})

	ginkgo.AfterEach(func() {
		ctrl.Finish()
	})

	type testCase struct {
		description    string
		fixture        string
		golden         string
		expectPRUpdate bool
		assertion      func(err error, expectedBody string, fixture handlerFixture)
		expectedMsg    func(string, string) string
	}

	loadFixture := func(path string) handlerFixture {
		var value handlerFixture
		pkgtesting.MustLoadYaml(path, &value)
		return value
	}

	buildPRLabels := func(pr *git.PullRequest) (string, string) {
		if pr == nil || pr.URL == "" {
			return "the pull request description", " in the pull request description"
		}
		return fmt.Sprintf("[the pull request description](%s)", pr.URL),
			fmt.Sprintf(" in [the pull request description](%s)", pr.URL)
	}

	ginkgo.DescribeTable("checkbox command execution",
		func(tc testCase) {
			fixture := loadFixture(tc.fixture)
			prInfo := &git.PullRequest{
				Body: fixture.PR.Body,
				URL:  fixture.PR.URL,
			}
			mockClient.EXPECT().
				GetPR().
				Return(prInfo, nil).
				Times(1)
			prSuccessLabel, prFailureLabel := buildPRLabels(prInfo)

			var goldenBody string
			if tc.golden != "" {
				golden := map[string]string{}
				pkgtesting.MustLoadYaml(tc.golden, &golden)
				goldenBody = golden["updated_body"]
			}

			if tc.expectPRUpdate {
				mockClient.EXPECT().
					UpdatePRBody(goldenBody).
					Return(nil).
					Times(1)
			}
			expectedMessage := tc.expectedMsg(prSuccessLabel, prFailureLabel)
			mockClient.EXPECT().
				PostComment(expectedMessage).
				Return(nil).
				Times(1)

			err := testHandler.HandleCheckbox(nil)
			tc.assertion(err, goldenBody, fixture)
		},
		ginkgo.Entry("updates pull request body checkboxes", testCase{
			description:    "should toggle all unchecked boxes in the PR description",
			fixture:        "testdata/handler_pr_body/fixture.yaml",
			golden:         "testdata/handler_pr_body/golden.yaml",
			expectPRUpdate: true,
			assertion: func(err error, expectedBody string, _ handlerFixture) {
				Expect(err).To(BeNil())
				Expect(expectedBody).NotTo(BeEmpty())
			},
			expectedMsg: func(prSuccessLabel, _ string) string {
				return fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, cfg.CommentSender, prSuccessLabel)
			},
		}),
		ginkgo.Entry("returns commented error when already checked", testCase{
			description: "should notify when all checkboxes already checked",
			fixture:     "testdata/handler_already_checked/fixture.yaml",
			assertion: func(err error, _ string, _ handlerFixture) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
			expectedMsg: func(_ string, prFailureLabel string) string {
				return fmt.Sprintf(messages.CheckboxAlreadyCheckedTemplate, prFailureLabel)
			},
		}),
		ginkgo.Entry("returns commented error when description missing", testCase{
			description: "should notify when description missing",
			fixture:     "testdata/handler_missing_description/fixture.yaml",
			assertion: func(err error, _ string, _ handlerFixture) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
			expectedMsg: func(_, _ string) string {
				return messages.CheckboxDescriptionNotFoundTemplate
			},
		}),
	)
})
