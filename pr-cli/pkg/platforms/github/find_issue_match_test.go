package github

import (
	"path/filepath"

	pkgtesting "github.com/AlaudaDevops/pkg/testing"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/google/go-github/v74/github"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type issueFixture struct {
	Number int      `yaml:"number"`
	Title  string   `yaml:"title"`
	Author string   `yaml:"author"`
	State  string   `yaml:"state"`
	Labels []string `yaml:"labels"`
	Body   string   `yaml:"body"`
}

type issueCollection struct {
	Description string         `yaml:"description"`
	Issues      []issueFixture `yaml:"issues"`
}

type optionsFixture struct {
	Description string                 `yaml:"description"`
	Options     git.IssueSearchOptions `yaml:"options"`
}

type goldenFixture struct {
	Description string `yaml:"description"`
	Expected    struct {
		Found  bool `yaml:"found"`
		Number int  `yaml:"number"`
	} `yaml:"expected"`
}

var _ = ginkgo.Describe("issueMatchesFilters", func() {
	type testCase struct {
		description string
		baseDir     string
	}

	loadIssues := func(path string) []*github.Issue {
		data := issueCollection{}
		pkgtesting.MustLoadYaml(path, &data)

		issues := make([]*github.Issue, 0, len(data.Issues))
		for _, item := range data.Issues {
			labels := make([]*github.Label, 0, len(item.Labels))
			for _, name := range item.Labels {
				labels = append(labels, &github.Label{Name: github.String(name)})
			}

			issue := &github.Issue{
				Number: github.Int(item.Number),
				Title:  github.String(item.Title),
				State:  github.String(item.State),
				Body:   github.String(item.Body),
				User:   &github.User{Login: github.String(item.Author)},
				Labels: labels,
			}
			issues = append(issues, issue)
		}

		return issues
	}

	loadOptions := func(path string) git.IssueSearchOptions {
		data := optionsFixture{}
		pkgtesting.MustLoadYaml(path, &data)
		return data.Options
	}

	loadGolden := func(path string) goldenFixture {
		result := goldenFixture{}
		pkgtesting.MustLoadYaml(path, &result)
		return result
	}

	findMatch := func(issues []*github.Issue, opts git.IssueSearchOptions) *github.Issue {
		for _, issue := range issues {
			if issueMatchesFilters(issue, opts) {
				return issue
			}
		}
		return nil
	}

	ginkgo.DescribeTable("matches the expected issue when filters apply",
		func(tc testCase) {
			base := filepath.Join("testdata", "find_issue_match", tc.baseDir)
			issues := loadIssues(filepath.Join(base, "issues.yaml"))
			options := loadOptions(filepath.Join(base, "options.yaml"))
			golden := loadGolden(filepath.Join(base, "golden.yaml"))

			match := findMatch(issues, options)
			if golden.Expected.Found {
				Expect(match).NotTo(BeNil(), "expected a matching issue")
				Expect(match.GetNumber()).To(Equal(golden.Expected.Number))
			} else {
				Expect(match).To(BeNil())
			}
		},
		ginkgo.Entry("default dependency dashboard match", testCase{
			description: "should find issue by title and author",
			baseDir:     "default_match",
		}),
		ginkgo.Entry("missing author fallback", testCase{
			description: "should ignore author when not provided",
			baseDir:     "no_author",
		}),
		ginkgo.Entry("label mismatch exclusion", testCase{
			description: "should skip issues missing required labels",
			baseDir:     "label_filtered",
		}),
		ginkgo.Entry("author bot alias support", testCase{
			description: "should match author even with bot suffix variations",
			baseDir:     "author_variant",
		}),
	)
})
