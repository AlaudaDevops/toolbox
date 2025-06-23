/*
Copyright 2024 The AlaudaDevops Authors.

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
	"context"
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/models"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/bundle"
	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/config"
	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/jira"
	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/ops"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"knative.dev/pkg/logging"
)

type Options struct {
	valuesPath string
	directory  string
	branch     string
	configPath string
	bundle     string
}

func ScanCmd(ctx context.Context, name string) *cobra.Command {
	opts := &Options{}

	scanCmd := &cobra.Command{
		Use:          "scan [command options] values",
		SilenceUsage: true,
		Example:      fmt.Sprintf("%s scan values.yaml", name),
		Short:        "scan image in values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(ctx)
		},
	}
	opts.AddFlags(scanCmd.Flags())

	return scanCmd
}

func (s *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&s.valuesPath, "values", "values.yaml", `values file path`)
	flags.StringVar(&s.directory, "dir", "", `directory to be scanned, the directory should contains plugins that follow "alauda/artifacts" structure`)
	flags.StringVar(&s.branch, "branch", "main", `branch name`)
	flags.StringVar(&s.configPath, "config", "config.yaml", `config file path`)
	flags.StringVar(&s.bundle, "bundle", "", `bundle to be scanned, multiple bundles can be separated by comma`)
}

func (s *Options) run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	cfg, err := config.Load(s.configPath)
	if err != nil {
		return err
	}

	ctx = cfg.InjectContext(ctx)

	var imageSource models.ImageSource
	if s.directory != "" {
		bundleNames := []string{}
		if s.bundle != "" {
			bundleNames = strings.Split(s.bundle, ",")
		}

		imageSource = models.NewDirSource(s.directory, bundleNames)
	} else {
		imageSource = models.NewValuesSource(s.valuesPath, s.bundle)
	}

	images, err := imageSource.GetImages(ctx)
	if err != nil {
		return err
	}

	opsClient := ops.NewClient(cfg.Ops.BaseURL)
	scanner := bundle.NewScanner(opsClient)

	jiraClient, err := jira.NewClient(cfg.Jira.BaseURL, cfg.Jira.Username, cfg.Jira.Password)
	if err != nil {
		return fmt.Errorf("failed to create jira client: %w", err)
	}

	for _, image := range images {

		if image.Type != models.ImageTypeBundle {
			// todo: only handle bundles currently
			continue
		}

		logger.Infof("==== start to scan bundle:%s ====", image.URL())

		owner, err := opsClient.GetOwner(image.URL())
		if err != nil {
			logger.Errorw("failed to get owner", "err", err)
			return err
		}

		if owner == nil {
			owner = &image.Owner
		}

		results, err := scanner.Scan(ctx, image)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			logger.Infof("no vulnerabilities found")
			continue
		}

		version, err := jiraClient.GetComponentVersion(ctx, owner.Team, image.ComponentName())
		if err != nil {
			logger.Errorf("failed to get jira component version: %s", err.Error())
			continue
		}

		parentOptions := []jira.IssueOption{
			jira.WithAssignee(owner.JiraUser),
			jira.WithProject(owner.Team),
			jira.WithSummary(fmt.Sprintf("漏洞 - %s", image.Repository)),
			jira.WithPriority(results.Priority()),
			jira.WithLabels(s.branch, image.Repository, image.Tag),
			jira.WithType(jira.IssueTypeJob),
		}

		logger.Infof("creating jira issue for bundle")
		query := s.searchJQL(owner.Team, []string{s.branch, image.Repository})
		parentIssue, err := jiraClient.FindOrCreateIssue(ctx, query, parentOptions...)
		if err != nil {
			logger.Errorf("failed to create jira issue: %s", err.Error())
			return err
		}
		logger.Infof("bundle issue created: %s", parentIssue.Key)

		for relatedImage, result := range results {
			description, err := jira.RenderVulnerabilityTable(result)
			if err != nil {
				logger.Errorf("failed to render vulnerability table: %s", err.Error())
				return err
			}
			logger.Infof("creating jira issue for image: %s", relatedImage.URL())

			childOptions := []jira.IssueOption{
				jira.WithProject(owner.Team),
				jira.WithAssignee(owner.JiraUser),
				jira.WithSummary(fmt.Sprintf("漏洞 - %s", relatedImage.Repository)),
				jira.WithDescription(description),
				jira.WithPriority(result.Priority()),
				jira.WithLabels(s.branch, relatedImage.Repository, relatedImage.Tag),
				jira.WithType(jira.IssueTypeVulnerability),
				jira.WithAffectsVersion(version),
				jira.WithCustomField(map[string]interface{}{
					jira.CustomFieldCVEScore: 0,
					jira.CustomFieldCVEID:    "无",
				}),
			}

			query := s.searchJQL(owner.Team, []string{s.branch, relatedImage.Repository, relatedImage.Tag})
			childIssue, err := jiraClient.CreateOrUpdateIssue(ctx, query, childOptions...)
			if err != nil {
				logger.Infof("failed to create jira issue: %s", err.Error())
				return err
			}

			logger.Infof("image issue created: %s", childIssue.Key)

			if err := jiraClient.LinkIssue(ctx, parentIssue, childIssue); err != nil {
				logger.Infof("failed to link jira issue: %s", err.Error())
				return err
			}

			logger.Infof("issue linked, parent: %s, child: %s", parentIssue.Key, childIssue.Key)
		}
	}

	return nil
}

func (s *Options) searchJQL(project string, labels []string) *jql.JQL {
	query := &jql.JQL{}
	query = query.FilterBy("project", project)

	for _, label := range labels {
		query = query.And(func() {
			query.FilterBy("labels", label)
		})
	}

	return query
}
