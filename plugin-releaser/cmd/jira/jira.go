// Package jira provides Jira command functionality
// This file defines the jira subcommand for creating Jira issues
package jira

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/plugin-releaser/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// logger is the package-level logger instance
var logger = logrus.New()

func init() {
	// Configure logger for CLI-friendly output
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    false,
		TimestampFormat:  "15:04:05",
		DisableTimestamp: false,
	})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(os.Stdout)
}

// NewJiraCmd creates and returns the jira subcommand
// Returns a cobra.Command configured for creating Jira issues
func NewJiraCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-release-check-jira",
		Short: "Create a Jira issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Setup verbose logging if requested
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				logger.SetLevel(logrus.DebugLevel)
				logger.Debug("Verbose logging enabled")
			}

			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = "config.yaml"
			}

			logger.WithField("config", configPath).Info("Loading configuration file")
			cfg, err := LoadConfig(configPath)
			if err != nil {
				logger.WithError(err).WithField("config", configPath).Error("Failed to load configuration")
				return fmt.Errorf("failed to load config: %w", err)
			}
			logger.Debug("Configuration loaded successfully")

			logger.WithFields(logrus.Fields{
				"baseURL":  cfg.Jira.BaseURL,
				"project":  cfg.Jira.Project,
				"username": cfg.Jira.Username,
			}).Info("Initializing Jira client")

			client, err := jira.NewClientWithConfig(&cfg.Jira.JiraConfig)
			if err != nil {
				logger.WithError(err).Error("Failed to create Jira client")
				return fmt.Errorf("failed to create jira client: %w", err)
			}
			logger.Debug("Jira client created successfully")

			logger.WithField("pluginCount", len(cfg.Plugins)).Info("Processing plugins")

			for pluginName, issueMeta := range cfg.Plugins {
				logger.WithField("plugin", pluginName).Info("Processing plugin")
				issueMeta := issueMeta.Merge(&cfg.Jira.IssueMeta)
				if err := createPluginIssue(ctx, client, issueMeta, pluginName); err != nil {
					logger.WithError(err).WithField("plugin", pluginName).Error("Failed to process plugin")
					return err
				}
				logger.WithField("plugin", pluginName).Info("Plugin processed successfully")
			}

			logger.Info("All plugins processed successfully")
			return nil
		},
	}

	cmd.Flags().String("config", "", "Config file path (default: ./config.yaml)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")

	return cmd
}

func templateFuncMap(jiraClient *jira.Client) template.FuncMap {
	return template.FuncMap{
		"getActiveSprintId": func(boardID int) (int, error) {
			logger.WithField("boardId", boardID).Debug("Getting active sprint ID")
			sprint, err := jiraClient.GetActiveSprint(context.Background(), boardID)
			if err != nil {
				logger.WithError(err).WithField("boardId", boardID).Error("Failed to get active sprint")
				return 0, fmt.Errorf("failed to get active sprint: %w", err)
			}
			return sprint.ID, nil
		},
	}
}

func createPluginIssue(ctx context.Context, jiraClient *jira.Client, issueMeta *PluginIssueMeta, pluginName string) error {
	logger := logger.WithField("plugin", pluginName)

	logger.Debug("Creating template data for issue")
	templateData := map[string]interface{}{
		"plugin": pluginName,
		"issue":  issueMeta,
	}

	logger.Debug("Rendering template for issue metadata")
	if err := RenderStructTemplate(issueMeta, templateData, templateFuncMap(jiraClient)); err != nil {
		logger.WithError(err).Error("Failed to render template")
		return fmt.Errorf("failed to render template: %w", err)
	}
	logger.Debug("Template rendered successfully")

	logger.Debug("Building issue options")
	var opts []jira.IssueOption
	opts = append(opts, jira.WithProject(issueMeta.Project))
	opts = append(opts, jira.WithType(issueMeta.IssueType))
	opts = append(opts, jira.WithSummary(issueMeta.Summary))
	opts = append(opts, jira.WithDescription(issueMeta.Description))
	if issueMeta.Priority != "" {
		opts = append(opts, jira.WithPriority(issueMeta.Priority))
	}

	if issueMeta.Owner != "" {
		opts = append(opts, jira.WithAssignee(issueMeta.Owner))
		logger.WithField("assignee", issueMeta.Owner).Debug("Issue assignee set")
	}

	botLabels := GenerateIssueTags(pluginName)
	labels := append(issueMeta.Labels, botLabels...)
	opts = append(opts, jira.WithLabels(labels...))
	logger.WithField("labels", labels).Debug("Issue labels configured")

	if len(issueMeta.CustomFields) > 0 {
		opts = append(opts, jira.WithCustomField(issueMeta.GetCustomFields()))
		logger.WithField("customField", issueMeta.GetCustomFields()).Debug("Custom fields configured")
	}

	// Build JQL query to find existing issues
	jqlQuery := jql.NewJQL(issueMeta.Project)
	for _, label := range botLabels {
		jqlQuery = jqlQuery.And(func() {
			jqlQuery.FilterBy("labels", label)
		})
	}

	logger.WithFields(logrus.Fields{
		"project": issueMeta.Project,
		"summary": issueMeta.Summary,
		"labels":  botLabels,
	}).Info("Creating or finding Jira issue")

	issue, err := jiraClient.FindOrCreateIssue(ctx, jqlQuery, opts...)
	if err != nil {
		logger.WithError(err).Error("Failed to create or find Jira issue")
		return fmt.Errorf("failed to create or find jira issue: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"issueKey": issue.Key,
		"plugin":   pluginName,
	}).Info("Jira issue created or found successfully")

	return nil
}

// GenerateIssueTags returns a tag with the format "bot-<pluginName>-YYYYMM"
// pluginName: the name of the plugin
// Returns a slice of tags including the release bot identifier and plugin-specific tag
func GenerateIssueTags(pluginName string) []string {
	now := time.Now()
	tags := []string{
		"created-by-release-bot",
		fmt.Sprintf("%s-%d%02d", pluginName, now.Year(), int(now.Month())),
	}
	logger.WithFields(logrus.Fields{
		"plugin": pluginName,
		"tags":   tags,
	}).Debug("Generated issue tags")
	return tags
}

// RenderStructTemplate renders template strings in all fields of a struct using YAML serialization
// s: pointer to the struct to render
// templateData: data to use for template rendering
// Returns error if serialization, template parsing or execution fails
func RenderStructTemplate(s interface{}, templateData map[string]interface{}, templateFuncMap template.FuncMap) (err error) {
	// Recover from panic that might occur during marshaling
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to marshal struct to YAML: %v", r)
		}
	}()

	// Serialize struct to YAML
	yamlData, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal struct to YAML: %w", err)
	}

	// Parse and execute template on YAML string
	var output strings.Builder
	tmpl, err := template.New("struct").Funcs(templateFuncMap).Parse(string(yamlData))
	if err != nil {
		return fmt.Errorf("failed to parse YAML template: %w", err)
	}

	err = tmpl.Execute(&output, templateData)
	if err != nil {
		return fmt.Errorf("failed to execute YAML template: %w", err)
	}

	// Deserialize rendered YAML back to struct
	// Handle panic that might occur during unmarshaling
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to unmarshal rendered YAML: %v", r)
		}
	}()

	err = yaml.Unmarshal([]byte(output.String()), s)
	if err != nil {
		return fmt.Errorf("failed to unmarshal rendered YAML: %w", err)
	}

	return nil
}
