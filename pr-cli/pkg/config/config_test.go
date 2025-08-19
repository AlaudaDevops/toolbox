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

package config_test

import (
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("NewDefaultConfig", func() {
		It("should return a new config with default values", func() {
			cfg := config.NewDefaultConfig()

			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Platform).To(Equal("github"))
			Expect(cfg.LGTMThreshold).To(Equal(1))
			Expect(cfg.LGTMReviewEvent).To(Equal("APPROVE"))
			Expect(cfg.MergeMethod).To(Equal("squash"))
			Expect(cfg.SelfCheckName).To(Equal("pr-cli"))
			Expect(cfg.LogLevel).To(Equal("info"))
		})

		It("should set correct LGTM permissions", func() {
			cfg := config.NewDefaultConfig()

			expectedPermissions := []string{"admin", "write"}
			Expect(cfg.LGTMPermissions).To(HaveLen(2))
			Expect(cfg.LGTMPermissions).To(Equal(expectedPermissions))
		})
	})

	Describe("Validate", func() {
		DescribeTable("should validate configuration correctly",
			func(testCase string, cfg *config.Config, expectedError error) {
				err := cfg.Validate()

				if expectedError == nil {
					Expect(err).To(BeNil())
				} else {
					Expect(err).To(Equal(expectedError))
				}
			},
			Entry("valid configuration", "valid", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, nil),
			Entry("missing platform", "missing platform", &config.Config{
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrMissingPlatform),
			Entry("missing token", "missing token", &config.Config{
				Platform:       "github",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrMissingToken),
			Entry("missing owner", "missing owner", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrMissingOwner),
			Entry("missing repo", "missing repo", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrMissingRepo),
			Entry("invalid PR number (zero)", "invalid PR number (zero)", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          0,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrInvalidPRNum),
			Entry("invalid PR number (negative)", "invalid PR number (negative)", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          -1,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
			}, config.ErrInvalidPRNum),
			Entry("missing comment sender", "missing comment sender", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				TriggerComment: "test-comment",
			}, config.ErrMissingCommentSender),
			Entry("missing trigger comment", "missing trigger comment", &config.Config{
				Platform:      "github",
				Token:         "test-token",
				Owner:         "test-owner",
				Repo:          "test-repo",
				PRNum:         123,
				CommentSender: "test-sender",
			}, config.ErrMissingTriggerComment),
			Entry("multiple missing fields", "multiple missing fields", &config.Config{
				Platform: "github",
				Token:    "test-token",
				// Missing owner, repo, PRNum, CommentSender, TriggerComment
			}, config.ErrMissingOwner), // First error encountered
		)

		DescribeTable("should handle edge cases correctly",
			func(testCase string, cfg *config.Config, shouldPass bool) {
				err := cfg.Validate()

				if shouldPass {
					Expect(err).To(BeNil())
				} else {
					Expect(err).NotTo(BeNil())
				}
			},
			Entry("empty optional fields", "empty optional", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
				BaseURL:        "",
				SelfCheckName:  "",
				LogLevel:       "",
			}, true),
			Entry("zero LGTM threshold", "zero threshold", &config.Config{
				Platform:       "github",
				Token:          "test-token",
				Owner:          "test-owner",
				Repo:           "test-repo",
				PRNum:          123,
				CommentSender:  "test-sender",
				TriggerComment: "test-comment",
				LGTMThreshold:  0,
			}, true),
		)
	})

	Describe("Config struct", func() {
		It("should have all fields properly accessible", func() {
			cfg := &config.Config{
				Platform:        "github",
				Token:           "test-token",
				BaseURL:         "https://api.github.com",
				Owner:           "test-owner",
				Repo:            "test-repo",
				PRNum:           123,
				CommentSender:   "test-sender",
				TriggerComment:  "test-comment",
				Debug:           true,
				LGTMThreshold:   2,
				LGTMPermissions: []string{"admin"},
				LGTMReviewEvent: "COMMENT",
				MergeMethod:     "merge",
				SelfCheckName:   "custom-check",
				LogLevel:        "debug",
			}

			Expect(cfg.Platform).To(Equal("github"))
			Expect(cfg.Token).To(Equal("test-token"))
			Expect(cfg.BaseURL).To(Equal("https://api.github.com"))
			Expect(cfg.Owner).To(Equal("test-owner"))
			Expect(cfg.Repo).To(Equal("test-repo"))
			Expect(cfg.PRNum).To(Equal(123))
			Expect(cfg.CommentSender).To(Equal("test-sender"))
			Expect(cfg.TriggerComment).To(Equal("test-comment"))
			Expect(cfg.Debug).To(BeTrue())
			Expect(cfg.LGTMThreshold).To(Equal(2))
			Expect(cfg.LGTMPermissions).To(Equal([]string{"admin"}))
			Expect(cfg.LGTMReviewEvent).To(Equal("COMMENT"))
			Expect(cfg.MergeMethod).To(Equal("merge"))
			Expect(cfg.SelfCheckName).To(Equal("custom-check"))
			Expect(cfg.LogLevel).To(Equal("debug"))
		})

		It("should handle empty slices and strings correctly", func() {
			cfg := &config.Config{
				Platform:        "github",
				Token:           "test-token",
				Owner:           "test-owner",
				Repo:            "test-repo",
				PRNum:           123,
				CommentSender:   "test-sender",
				TriggerComment:  "test-comment",
				LGTMPermissions: []string{},
			}

			Expect(cfg.LGTMPermissions).To(BeEmpty())
			Expect(cfg.BaseURL).To(BeEmpty())
			Expect(cfg.SelfCheckName).To(BeEmpty())
			Expect(cfg.LogLevel).To(BeEmpty())
		})
	})
})
