package github

import (
	"sort"
	"strings"

	pkgtesting "github.com/AlaudaDevops/pkg/testing"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

type reviewFixture struct {
	Description string             `json:"description" yaml:"description"`
	PRSender    string             `json:"pr_sender" yaml:"pr_sender"`
	Reviews     []reviewFixtureRow `json:"reviews" yaml:"reviews"`
}

type reviewFixtureRow struct {
	User        string `json:"user" yaml:"user"`
	State       string `json:"state" yaml:"state"`
	SubmittedAt string `json:"submitted_at" yaml:"submitted_at"`
}

type expectedApprovalsFixture struct {
	Description   string   `json:"description" yaml:"description"`
	ApprovedUsers []string `json:"approved_users" yaml:"approved_users"`
}

var _ = ginkgo.Describe("GitHub review approvals", func() {
	ginkgo.Describe("effective approval aggregation", func() {
		type testCase struct {
			description  string
			reviewsFile  string
			expectedFile string
		}

		ginkgo.DescribeTable("collects LGTM users from reviews",
			func(tc testCase) {
				reviewData := reviewFixture{}
				pkgtesting.MustLoadYaml(tc.reviewsFile, &reviewData)

				expected := expectedApprovalsFixture{}
				pkgtesting.MustLoadYaml(tc.expectedFile, &expected)

				client := &Client{
					Logger:   logrus.New(),
					prSender: reviewData.PRSender,
				}

				reviews := make([]git.Review, len(reviewData.Reviews))
				for i := range reviewData.Reviews {
					row := reviewData.Reviews[i]
					reviews[i] = git.Review{
						User: git.User{
							Login: row.User,
						},
						State:       row.State,
						SubmittedAt: row.SubmittedAt,
					}
				}

				latestReviews := client.findLatestActionableReviews(reviews)
				approvedReviews := client.findEffectiveApprovals(reviews)

				lgtmUsers := make(map[string]string)
				client.addApprovedReviews(lgtmUsers, latestReviews, approvedReviews)

				collectedUsers := make([]string, 0, len(lgtmUsers))
				for user := range lgtmUsers {
					collectedUsers = append(collectedUsers, user)
				}
				sort.Strings(collectedUsers)

				expectedUsers := make([]string, len(expected.ApprovedUsers))
				for i, user := range expected.ApprovedUsers {
					expectedUsers[i] = strings.ToLower(user)
				}
				sort.Strings(expectedUsers)

				Expect(collectedUsers).To(Equal(expectedUsers), "test case: %s", tc.description)
			},
			ginkgo.Entry("retains approval after comment", testCase{
				description:  "should retain approval when the latest review is only a comment",
				reviewsFile:  "testdata/review_effective_approvals/retain_after_comment/reviews.yaml",
				expectedFile: "testdata/review_effective_approvals/retain_after_comment/expected.yaml",
			}),
			ginkgo.Entry("removes approval after changes requested", testCase{
				description:  "should remove approval when changes are requested after approval",
				reviewsFile:  "testdata/review_effective_approvals/revoked_by_changes/reviews.yaml",
				expectedFile: "testdata/review_effective_approvals/revoked_by_changes/expected.yaml",
			}),
			ginkgo.Entry("restores approval when re-approved after changes", testCase{
				description:  "should restore approval when a new approval follows requested changes",
				reviewsFile:  "testdata/review_effective_approvals/reapprove_after_changes/reviews.yaml",
				expectedFile: "testdata/review_effective_approvals/reapprove_after_changes/expected.yaml",
			}),
			ginkgo.Entry("removes approval after dismissal", testCase{
				description:  "should remove approval when the latest review is dismissed",
				reviewsFile:  "testdata/review_effective_approvals/revoked_by_dismissed/reviews.yaml",
				expectedFile: "testdata/review_effective_approvals/revoked_by_dismissed/expected.yaml",
			}),
		)
	})
})
