# PR CLI

A command-line tool for managing GitHub and GitLab Pull Requests through comment commands.

## Overview

PR CLI is a tool that processes comment commands on pull requests and executes corresponding actions using platform APIs. It supports operations like assigning reviewers, LGTM voting, merging, rebasing, and more.

## Features

- **Multi-platform support**: GitHub and GitLab
- **Comment-driven operations**: Execute actions through PR comments
- **Flexible LGTM system**: Configurable threshold and permissions
- **Multiple merge methods**: merge, squash, rebase
- **Reviewer management**: Assign and unassign reviewers
- **Status checks**: Verify CI/CD status before operations

## Installation

### From Source

```bash
git clone https://github.com/AlaudaDevops/toolbox.git
cd toolbox/pr-cli
make build
```

### Using Go

```bash
go install github.com/AlaudaDevops/toolbox/pr-cli@latest
```

## Quick Start

### Basic Usage

```bash
# Process a comment command
pr-cli --platform github \
       --repo-owner owner \
       --repo-name repo \
       --pr-num 123 \
       --comment-sender user \
       --token $GITHUB_TOKEN \
       --trigger-comment "/assign user1 user2"
```

### Environment Variables

You can use environment variables instead of command line flags:

```bash
export PR_PLATFORM=github
export PR_TOKEN=$GITHUB_TOKEN
export PR_REPO_OWNER=owner
export PR_REPO_NAME=repo
```

## Supported Commands

| Command | Description | Parameters | Example |
|---------|-------------|------------|---------|
| `/assign` | Assign reviewers to PR | `user1 user2 ...` | `/assign alice bob` |
| `/unassign` | Remove reviewers from PR | `user1 user2 ...` | `/unassign alice` |
| `/lgtm` | Add LGTM approval | `[message]` | `/lgtm` or `/lgtm Great work!` |
| `/lgtm cancel` | Remove LGTM approval (alias) | - | `/lgtm cancel` |
| `/remove-lgtm` | Remove LGTM approval | `[message]` | `/remove-lgtm` |
| `/merge` | Merge the PR | `[method]` | `/merge` or `/merge squash` |
| `/ready` | Merge the PR (alias for `/merge`) | `[method]` | `/ready` or `/ready squash` |
| `/rebase` | Rebase the PR | - | `/rebase` |
| `/check` | Check PR status | - | `/check` |
| `/label` | Add labels to PR | `label1 label2 ...` | `/label bug enhancement` |
| `/unlabel` | Remove labels from PR | `label1 label2 ...` | `/unlabel bug` |
| `/cherry-pick` | Cherry-pick to branches | `branch1 branch2 ...` | `/cherry-pick release/v1.0` |
| `/cherrypick` | Cherry-pick to branches (alias) | `branch1 branch2 ...` | `/cherrypick release/v1.0` |
| `/help` | Show available commands | - | `/help` |

### Merge Methods
- `merge` - Create merge commit
- `squash` - Squash merge  
- `rebase` - Rebase merge (default)

> 📘 **Pipeline Integration**: For Tekton Pipeline integration and detailed command usage, see [pipeline/README.md](pipeline/README.md)

## Configuration

### Command Line Flags

- `--platform`: Platform (github/gitlab)
- `--token`: API token
- `--repo-owner`: Repository owner
- `--repo-name`: Repository name
- `--pr-num`: Pull request number
- `--comment-sender`: Comment author
- `--trigger-comment`: Comment content to process
- `--lgtm-threshold`: Minimum LGTM approvals (default: 1)
- `--merge-method`: Merge method (merge/squash/rebase, default: merge)

### LGTM Configuration

```bash
# Set LGTM threshold
pr-cli --lgtm-threshold 2 --trigger-comment "/lgtm" ...

# Custom merge method
pr-cli --merge-method squash --trigger-comment "/merge" ...
```

## Development

### Prerequisites

- Go 1.24.6 or later
- Make

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Running Tests

```bash
go test ./...
```

## Project Structure

```
pr-cli/
├── cmd/                   # CLI command definitions
├── pkg/
│   ├── config/           # Configuration management
│   ├── git/              # Git platform abstraction
│   ├── handler/          # Business logic handlers
│   └── platforms/        # Platform implementations
├── internal/
│   └── version/          # Version information
├── testing/              # Test utilities and mocks
└── docs/                 # Documentation
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.