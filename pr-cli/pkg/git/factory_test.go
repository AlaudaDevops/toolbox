package git_test

import (
	"errors"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

// TestCreateClient demonstrates testing with mock ClientFactory
func TestCreateClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock factory and client
	mockFactory := mock_git.NewMockClientFactory(ctrl)
	mockClient := mock_git.NewMockGitClient(ctrl)

	// Test config
	config := &git.Config{
		Platform:      "test-platform",
		Token:         "test-token",
		Owner:         "test-owner",
		Repo:          "test-repo",
		PRNum:         123,
		PRSender:      "author",
		CommentSender: "commenter",
	}

	// Create a test logger
	logger := logrus.New()

	// Set expectation on mock factory
	mockFactory.EXPECT().
		CreateClient(gomock.Any(), config).
		Return(mockClient, nil).
		Times(1)

	// Register the mock factory
	git.RegisterFactory("test-platform", mockFactory)

	// Test CreateClient function
	client, err := git.CreateClient(logger, config)
	if err != nil {
		t.Errorf("CreateClient() error = %v, wantErr false", err)
	}
	if client != mockClient {
		t.Errorf("CreateClient() returned unexpected client")
	}
}

// TestCreateClient_UnsupportedPlatform tests error handling for unsupported platforms
func TestCreateClient_UnsupportedPlatform(t *testing.T) {
	config := &git.Config{
		Platform: "definitely-unsupported-platform-12345",
	}

	logger := logrus.New()

	_, err := git.CreateClient(logger, config)
	if err == nil {
		t.Error("CreateClient() error = nil, wantErr true")
	}
	expectedError := "unsupported platform: definitely-unsupported-platform-12345"
	if err.Error() != expectedError {
		t.Errorf("CreateClient() error = %v, want %v", err.Error(), expectedError)
	}
}

// TestCreateClient_FactoryError tests error handling when factory returns error
func TestCreateClient_FactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := mock_git.NewMockClientFactory(ctrl)

	config := &git.Config{
		Platform: "error-platform",
		Token:    "invalid-token",
	}

	expectedError := errors.New("invalid token")
	mockFactory.EXPECT().
		CreateClient(gomock.Any(), config).
		Return(nil, expectedError).
		Times(1)

	// Register the mock factory that will return error
	git.RegisterFactory("error-platform", mockFactory)

	logger := logrus.New()

	_, err := git.CreateClient(logger, config)
	if err != expectedError {
		t.Errorf("CreateClient() error = %v, want %v", err, expectedError)
	}
}

// TestRegisterFactory demonstrates testing factory registration
func TestRegisterFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := mock_git.NewMockClientFactory(ctrl)
	mockClient := mock_git.NewMockGitClient(ctrl)

	// Register the mock factory
	git.RegisterFactory("registration-test-platform", mockFactory)

	config := &git.Config{
		Platform: "registration-test-platform",
		Token:    "test-token",
	}

	// Set expectation - if the factory was registered properly,
	// CreateClient should call our mock
	mockFactory.EXPECT().
		CreateClient(gomock.Any(), config).
		Return(mockClient, nil).
		Times(1)

	// Test that the registered factory is used
	logger := logrus.New()

	client, err := git.CreateClient(logger, config)
	if err != nil {
		t.Errorf("CreateClient() with registered factory error = %v, wantErr false", err)
	}
	if client != mockClient {
		t.Errorf("CreateClient() did not use registered factory")
	}
}
