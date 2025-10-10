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
	"path/filepath"
	"strings"

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

type issueFixture struct {
	Issue struct {
		Number int    `yaml:"number"`
		Title  string `yaml:"title"`
		State  string `yaml:"state"`
		Author string `yaml:"author"`
		Body   string `yaml:"body"`
		URL    string `yaml:"url"`
	} `yaml:"issue"`
}

var _ = ginkgo.Describe("HandleCheckboxIssue", func() {
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

	loadFixture := func(path string) issueFixture {
		var value issueFixture
		pkgtesting.MustLoadYaml(filepath.Join("testdata", filepath.FromSlash(path)), &value)
		return value
	}

	toIssue := func(f issueFixture) *git.Issue {
		return &git.Issue{
			Number: f.Issue.Number,
			Title:  f.Issue.Title,
			State:  f.Issue.State,
			Author: f.Issue.Author,
			Body:   f.Issue.Body,
			URL:    f.Issue.URL,
		}
	}

	buildIssueLabels := func(issue *git.Issue) (string, string) {
		if issue == nil {
			return "the issue", " in the issue"
		}
		ref := fmt.Sprintf("issue #%d", issue.Number)
		if strings.TrimSpace(issue.Title) != "" {
			ref = fmt.Sprintf("%s \"%s\"", ref, issue.Title)
		}
		success := ref
		failure := fmt.Sprintf(" in %s", ref)
		if strings.TrimSpace(issue.URL) != "" {
			success = fmt.Sprintf("[%s](%s)", ref, issue.URL)
			failure = fmt.Sprintf(" in [%s](%s)", ref, issue.URL)
		}
		return success, failure
	}

	loadGolden := func(path string) string {
		if path == "" {
			return ""
		}
		var result map[string]string
		pkgtesting.MustLoadYaml(filepath.Join("testdata", filepath.FromSlash(path)), &result)
		return result["updated_body"]
	}

	ginkgo.DescribeTable("checkbox issue command execution",
		func(args []string, fixturePath, goldenPath string, setup func(issue *git.Issue, golden string), assert func(err error, golden string, issue *git.Issue)) {
			fixture := loadFixture(fixturePath)
			issue := toIssue(fixture)
			golden := loadGolden(goldenPath)

			if setup != nil {
				setup(issue, golden)
			}

			err := testHandler.HandleCheckboxIssue(args)
			assert(err, golden, issue)
		},
		ginkgo.Entry("updates checkbox for explicit issue number",
			[]string{"110"},
			"checkbox_issue/direct_success/fixture.yaml",
			"checkbox_issue/direct_success/golden.yaml",
			func(issue *git.Issue, golden string) {
				mockClient.EXPECT().
					GetIssue(issue.Number).
					Return(issue, nil).
					Times(1)
				if golden != "" {
					mockClient.EXPECT().
						UpdateIssueBody(issue.Number, golden).
						Return(nil).
						Times(1)
				}
				successLabel, _ := buildIssueLabels(issue)
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, cfg.CommentSender, successLabel)).
					Return(nil).
					Times(1)
			},
			func(err error, golden string, _ *git.Issue) {
				Expect(err).To(BeNil())
				Expect(golden).NotTo(BeEmpty())
			},
		),
		ginkgo.Entry("updates checkbox via default search when no argument provided",
			[]string{},
			"checkbox_issue/default_success/fixture.yaml",
			"checkbox_issue/default_success/golden.yaml",
			func(issue *git.Issue, golden string) {
				mockClient.EXPECT().
					FindIssue(git.IssueSearchOptions{
						Title:  "Dependency Dashboard",
						Author: "alaudaa-renovate[bot]",
						State:  "open",
						Sort:   "created",
						Order:  "asc",
					}).
					Return(issue, nil).
					Times(1)
				if golden != "" {
					mockClient.EXPECT().
						UpdateIssueBody(issue.Number, golden).
						Return(nil).
						Times(1)
				}
				successLabel, _ := buildIssueLabels(issue)
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, cfg.CommentSender, successLabel)).
					Return(nil).
					Times(1)
			},
			func(err error, golden string, _ *git.Issue) {
				Expect(err).To(BeNil())
				Expect(golden).NotTo(BeEmpty())
			},
		),
		ginkgo.Entry("updates checkbox with overridden search parameters",
			[]string{"--title", "Custom Dashboard", "--author", "renovate"},
			"checkbox_issue/default_success/fixture.yaml",
			"checkbox_issue/default_success/golden.yaml",
			func(issue *git.Issue, golden string) {
				mockClient.EXPECT().
					FindIssue(git.IssueSearchOptions{
						Title:  "Custom Dashboard",
						Author: "renovate",
						State:  "open",
						Sort:   "created",
						Order:  "asc",
					}).
					Return(issue, nil).
					Times(1)
				if golden != "" {
					mockClient.EXPECT().
						UpdateIssueBody(issue.Number, golden).
						Return(nil).
						Times(1)
				}
				successLabel, _ := buildIssueLabels(issue)
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, cfg.CommentSender, successLabel)).
					Return(nil).
					Times(1)
			},
			func(err error, golden string, _ *git.Issue) {
				Expect(err).To(BeNil())
				Expect(golden).NotTo(BeEmpty())
			},
		),
		ginkgo.Entry("returns commented error when issue already checked",
			[]string{"310"},
			"checkbox_issue/already_checked/fixture.yaml",
			"",
			func(issue *git.Issue, _ string) {
				mockClient.EXPECT().
					GetIssue(issue.Number).
					Return(issue, nil).
					Times(1)
				_, failureLabel := buildIssueLabels(issue)
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxAlreadyCheckedTemplate, failureLabel)).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
		ginkgo.Entry("returns commented error when issue body missing",
			[]string{"402"},
			"checkbox_issue/missing_body/fixture.yaml",
			"",
			func(issue *git.Issue, _ string) {
				mockClient.EXPECT().
					GetIssue(issue.Number).
					Return(issue, nil).
					Times(1)
				_, failureLabel := buildIssueLabels(issue)
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxIssueBodyMissingTemplate, failureLabel)).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
		ginkgo.Entry("returns commented error when default search fails",
			[]string{},
			"checkbox_issue/not_found/fixture.yaml",
			"",
			func(_ *git.Issue, _ string) {
				mockClient.EXPECT().
					FindIssue(git.IssueSearchOptions{
						Title:  "Dependency Dashboard",
						Author: "alaudaa-renovate[bot]",
						State:  "open",
						Sort:   "created",
						Order:  "asc",
					}).
					Return(nil, fmt.Errorf("not found")).
					Times(1)
				mockClient.EXPECT().
					PostComment(messages.CheckboxIssueNotFoundTemplate).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
		ginkgo.Entry("returns commented error when argument is invalid number",
			[]string{"abc"},
			"checkbox_issue/not_found/fixture.yaml",
			"",
			func(_ *git.Issue, _ string) {
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxIssueInvalidNumberTemplate, "abc")).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
		ginkgo.Entry("returns commented error when option missing value",
			[]string{"--title"},
			"checkbox_issue/not_found/fixture.yaml",
			"",
			func(_ *git.Issue, _ string) {
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxIssueInvalidOptionTemplate, "--title")).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
		ginkgo.Entry("returns commented error when option unsupported",
			[]string{"--foo=bar"},
			"checkbox_issue/not_found/fixture.yaml",
			"",
			func(_ *git.Issue, _ string) {
				mockClient.EXPECT().
					PostComment(fmt.Sprintf(messages.CheckboxIssueInvalidOptionTemplate, "--foo=bar")).
					Return(nil).
					Times(1)
			},
			func(err error, _ string, _ *git.Issue) {
				Expect(err).ToNot(BeNil())
				_, isCommented := err.(*handler.CommentedError)
				Expect(isCommented).To(BeTrue())
			},
		),
	)
})
