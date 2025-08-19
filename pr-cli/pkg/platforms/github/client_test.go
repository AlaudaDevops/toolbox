package github

import (
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/sirupsen/logrus"
)

// TestFactory_CreateClient demonstrates testing the GitHub factory implementation
func TestFactory_CreateClient(t *testing.T) {
	factory := &Factory{}

	config := &git.Config{
		Platform:      "github",
		Token:         "test-token",
		BaseURL:       "", // Use default GitHub API
		Owner:         "test-owner",
		Repo:          "test-repo",
		PRNum:         123,
		PRSender:      "author",
		CommentSender: "commenter",
	}

	logger := logrus.New()

	client, err := factory.CreateClient(logger, config)
	if err != nil {
		t.Errorf("Factory.CreateClient() error = %v, wantErr false", err)
	}

	if client == nil {
		t.Error("Factory.CreateClient() returned nil client")
	}

	// Verify the client is of the correct type
	githubClient, ok := client.(*Client)
	if !ok {
		t.Error("Factory.CreateClient() did not return GitHub Client")
	}

	// Verify the client has the correct configuration
	if githubClient.owner != config.Owner {
		t.Errorf("Factory.CreateClient() owner = %v, want %v", githubClient.owner, config.Owner)
	}
	if githubClient.repo != config.Repo {
		t.Errorf("Factory.CreateClient() repo = %v, want %v", githubClient.repo, config.Repo)
	}
	if githubClient.prNum != config.PRNum {
		t.Errorf("Factory.CreateClient() prNum = %v, want %v", githubClient.prNum, config.PRNum)
	}
	if githubClient.prSender != config.PRSender {
		t.Errorf("Factory.CreateClient() prSender = %v, want %v", githubClient.prSender, config.PRSender)
	}
	if githubClient.commentSender != config.CommentSender {
		t.Errorf("Factory.CreateClient() commentSender = %v, want %v", githubClient.commentSender, config.CommentSender)
	}
}

// TestFactory_CreateClient_EnterpriseURL demonstrates testing with enterprise GitHub URL
func TestFactory_CreateClient_EnterpriseURL(t *testing.T) {
	factory := &Factory{}

	config := &git.Config{
		Platform:      "github",
		Token:         "test-token",
		BaseURL:       "https://github.enterprise.com/api/v3",
		Owner:         "test-owner",
		Repo:          "test-repo",
		PRNum:         123,
		PRSender:      "author",
		CommentSender: "commenter",
	}

	logger := logrus.New()

	client, err := factory.CreateClient(logger, config)
	if err != nil {
		t.Errorf("Factory.CreateClient() with enterprise URL error = %v, wantErr false", err)
	}

	if client == nil {
		t.Error("Factory.CreateClient() returned nil client with enterprise URL")
	}

	// The client should be created successfully with enterprise URL
	_, ok := client.(*Client)
	if !ok {
		t.Error("Factory.CreateClient() did not return GitHub Client with enterprise URL")
	}
}

// TestFactory_CreateClient_InvalidEnterpriseURL demonstrates error handling for invalid URLs
func TestFactory_CreateClient_InvalidEnterpriseURL(t *testing.T) {
	factory := &Factory{}

	config := &git.Config{
		Platform: "github",
		Token:    "test-token",
		BaseURL:  "://invalid-url", // Invalid URL format
		Owner:    "test-owner",
		Repo:     "test-repo",
		PRNum:    123,
	}

	logger := logrus.New()

	_, err := factory.CreateClient(logger, config)
	if err == nil {
		t.Error("Factory.CreateClient() with invalid enterprise URL error = nil, wantErr true")
	}
}
