# Testing

This directory contains testing utilities, mocks, and test data for the PR CLI application.

## Generated Mocks

The following mock interfaces are automatically generated using GoMock:

- `GitClient` → `testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/git_client.go`
- `ClientFactory` → `testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/client_factory.go`

## Generating Mocks

```bash
# Generate all mocks
make generate

# Clean and regenerate mocks
make clean-mocks && make generate

# Full development workflow (includes mock generation)
make dev
```

## Using Mocks in Tests

### GitClient Mock Example

```go
package handler_test

import (
    "testing"
    "github.com/golang/mock/gomock"
    mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

func TestPRHandler_HandleHelp(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Create a mock GitClient
    mockClient := mock_git.NewMockGitClient(ctrl)

    // Set expectations
    mockClient.EXPECT().
        PostComment(gomock.Any()).
        Return(nil).
        Times(1)

    // Use the mock in your test...
}
```

### ClientFactory Mock Example

```go
package git_test

import (
    "testing"
    "github.com/golang/mock/gomock"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
    mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

func TestCreateClient(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockFactory := mock_git.NewMockClientFactory(ctrl)
    mockClient := mock_git.NewMockGitClient(ctrl)

    config := &git.Config{Platform: "test-platform"}

    mockFactory.EXPECT().
        CreateClient(gomock.Any(), config).
        Return(mockClient, nil).
        Times(1)

    git.RegisterFactory("test-platform", mockFactory)

    logger := logrus.New()
    client, err := git.CreateClient(logger, config)
    // Assert results...
}
```

## Available Mock Methods

### MockGitClient Methods
- `GetPR() (*PullRequest, error)`
- `CheckPRStatus(expectedState string) error`
- `PostComment(message string) error`
- `GetComments() ([]Comment, error)`
- `GetReviews() ([]Review, error)`
- `GetUserPermission(username string) (string, error)`
- `CheckUserPermissions(username string, requiredPerms []string) (bool, string, error)`
- `AssignReviewers(reviewers []string) error`
- `RemoveReviewers(reviewers []string) error`
- `GetLGTMVotes(requiredPerms []string) (int, map[string]string, error)`
- `ApprovePR(message string) error`
- `MergePR(method string) error`
- `RebasePR() error`
- `CheckRunsStatus() (bool, []CheckRun, error)`

### MockClientFactory Methods
- `CreateClient(logger *logrus.Logger, config *Config) (GitClient, error)`

## Test Organization

Tests are organized as follows:

### Unit Tests with Mocks
- `pkg/handler/pr_handler_test.go` - Tests PRHandler using MockGitClient
- `pkg/git/factory_test.go` - Tests factory functions using MockClientFactory

### Integration Tests
- `pkg/platforms/github/client_test.go` - Tests real GitHub factory implementation
- `pkg/platforms/gitlab/client_test.go` - Tests real GitLab factory implementation (when implemented)

### Generated Mocks
- `testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/` - Auto-generated mock files

## Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler -v
go test github.com/AlaudaDevops/toolbox/pr-cli/pkg/git -v
go test github.com/AlaudaDevops/toolbox/pr-cli/pkg/platforms/github -v

# Run with coverage
make test-coverage
```

## Mock Expectations

GoMock provides powerful expectation setting:

```go
// Exact parameter matching
mockClient.EXPECT().PostComment("exact message").Return(nil)

// Any parameter
mockClient.EXPECT().PostComment(gomock.Any()).Return(nil)

// Custom matcher
mockClient.EXPECT().PostComment(gomock.Not(gomock.Eq(""))).Return(nil)

// Call count
mockClient.EXPECT().PostComment(gomock.Any()).Times(1)
mockClient.EXPECT().PostComment(gomock.Any()).AnyTimes()

// Return sequence
mockClient.EXPECT().GetPR().Return(nil, errors.New("first call fails"))
mockClient.EXPECT().GetPR().Return(&git.PullRequest{Number: 123}, nil)
```

## Best Practices

1. Always call `ctrl.Finish()` in defer to verify all expectations were met
2. Use specific expectations where possible rather than `gomock.Any()`
3. Set appropriate call counts with `.Times(n)` or `.AnyTimes()`
4. Use separate test packages (`package foo_test`) to avoid import cycles
5. Create meaningful test data that represents real scenarios
6. Clean up global state between tests when necessary