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
- **Webhook Server**: Run as a service to receive GitHub/GitLab webhooks directly
- **PR Event Handling**: Trigger GitHub Actions workflows on PR open/update events

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

## Documentation

📖 **Choose the right documentation for your needs:**

| Documentation | Purpose | Audience |
|---------------|---------|----------|
| **This README** | Project overview, installation, and quick start | New users getting started |
| **[📋 CLI Reference](docs/usage.md)** | Complete command-line usage guide | Users needing detailed CLI documentation |
| **[🔧 Pipeline Integration](pipeline/README.md)** | Tekton Pipeline setup and configuration | DevOps teams setting up automation |
| **[🌐 Webhook Service](docs/webhook-usage.md)** | Webhook server deployment and configuration | Teams running PR CLI as a service |
| **[⚡ Webhook Quick Start](docs/webhook-quick-start.md)** | Quick setup guide for webhook service | Users setting up webhook service quickly |

## Quick Start

### Basic Usage

```bash
# Process a single command
pr-cli --platform github \
       --repo-owner owner \
       --repo-name repo \
       --pr-num 123 \
       --comment-sender user \
       --token $GITHUB_TOKEN \
       --trigger-comment "/assign user1 user2"

# Process multiple commands in one comment
pr-cli --platform github \
       --repo-owner owner \
       --repo-name repo \
       --pr-num 123 \
       --comment-sender user \
       --token $GITHUB_TOKEN \
       --trigger-comment "/lgtm
/ready squash"
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

Quick reference for all available PR comment commands:

| Command | Description | Parameters | Example |
|---------|-------------|------------|---------|
| `/assign` | Assign reviewers to PR | `user1 user2 ...` | `/assign alice bob` |
| `/unassign` | Remove reviewers from PR | `user1 user2 ...` | `/unassign alice` |
| `/lgtm` | Add LGTM approval | `[message]` | `/lgtm` or `/lgtm Great work!` |
| `/lgtm cancel` | Remove LGTM approval (alias) | - | `/lgtm cancel` |
| `/remove-lgtm` | Remove LGTM approval | `[message]` | `/remove-lgtm` |
| `/merge` | Merge the PR | `[method]` | `/merge` or `/merge squash` |
| `/ready` | Merge the PR (alias for `/merge`) | `[method]` | `/ready` or `/ready squash` |
| `/close` | Close the PR without merging | - | `/close` |
| `/rebase` | Rebase the PR | - | `/rebase` |
| `/check` | Check PR status or execute multiple commands | `[/cmd1 args... /cmd2 args...]` | `/check` or `/check /assign user1 /merge rebase` |
| `/batch` | Execute multiple commands in batch mode | `/cmd1 args... [/cmd2 args...]` | `/batch /assign user1 /merge squash` |
| `/label` | Add labels to PR | `label1 label2 ...` | `/label bug enhancement` |
| `/unlabel` | Remove labels from PR | `label1 label2 ...` | `/unlabel bug` |
| `/checkbox` | Check unchecked boxes in the PR description | - | `/checkbox` |
| `/checkbox-issue` | Check unchecked boxes in a specific or default dependency dashboard issue | `[issue] [--title ...] [--author ...]` | `/checkbox-issue 42` |
| `/cherry-pick` | Cherry-pick to branches | `branch1 branch2 ...` | `/cherry-pick release/v1.0` |
| `/cherrypick` | Cherry-pick to branches (alias) | `branch1 branch2 ...` | `/cherrypick release/v1.0` |
| `/retest` | Trigger retest of failed checks | - | `/retest` |
| `/help` | Show available commands | - | `/help` |

### Multi-line Commands

You can execute multiple commands in a single comment by placing each command on a separate line. This is particularly useful for common workflows:

```
/assign alice bob
/lgtm
/ready
```

**Multi-line Command Features:**
- **Line-based execution**: Each line starting with `/` is treated as a separate command
- **Command validation**: Each command is validated individually with proper permission checks
- **Error handling**: If any command fails, you'll see detailed results for all executed commands
- **Backward compatibility**: Existing single-line and `/batch` commands continue to work as before

**Example Multi-line Comments:**
```
# Review workflow
/assign reviewer1 reviewer2
/lgtm Looks good to me!
/ready squash

# Label and merge
/label enhancement
/merge rebase

# Quick approval
/lgtm
/ready
```

> 📘 **Need More Details?** For complete command syntax, configuration options, and troubleshooting, see the **[CLI Reference](docs/usage.md)**. For Pipeline integration, see **[Pipeline Integration](pipeline/README.md)**.

### Checkbox Issue Command

Use `/checkbox-issue` to toggle unchecked boxes inside a standalone issue. You can pass an explicit issue number or rely on the default search for the open `Dependency Dashboard` created by `alaudaa-renovate`. Optional `--title` and `--author` flags customize the search criteria, for example `/checkbox-issue --title "Security Dashboard" --author team-bot`.

### Built-in Commands

Built-in commands use the `/__` prefix (double underscore) and are designed for internal system usage rather than direct user interaction. These commands bypass normal validation checks and are typically triggered by pipeline automation or system processes.

| Command | Description | Use Case | Example |
|---------|-------------|----------|---------|
| `/__post-merge-cherry-pick` | Execute post-merge cherry-pick operations | Pipeline finally tasks after PR merge | `/__post-merge-cherry-pick` |

#### Built-in Command Rules:
- **Prefix**: All built-in commands must start with `/__` (slash + double underscore)
- **Purpose**: Internal system automation, not user-triggered actions
- **Validation**: Skip comment sender validation (no permission checks)
- **Status Checks**: Automatically bypass PR status checks when appropriate

#### Usage Example:
```bash
# Pipeline/automation usage
pr-cli --platform github \
       --repo-owner owner \
       --repo-name repo \
       --pr-num 123 \
       --comment-sender system \
       --trigger-comment "/__post-merge-cherry-pick"
```

### Merge Methods
- `merge` - Create merge commit
- `squash` - Squash merge (default)
- `rebase` - Rebase merge

> 📘 **Pipeline Integration**: For Tekton Pipeline integration and detailed command usage, see [pipeline/README.md](pipeline/README.md)

## Webhook Service Mode

PR CLI can run as a standalone webhook server that receives events directly from GitHub/GitLab:

```bash
# Start webhook server
pr-cli serve \
  --webhook-secret="your-secret" \
  --allowed-repos="myorg/*" \
  --token="$GITHUB_TOKEN"
```

### PR Event Handling

The webhook service can also trigger GitHub Actions workflows when PRs are opened or updated:

```bash
# Enable PR event handling to trigger workflows
pr-cli serve \
  --pr-event-enabled \
  --workflow-file=.github/workflows/pr-check.yml \
  --webhook-secret="your-secret" \
  --allowed-repos="myorg/*"
```

When a PR is opened or updated, the service triggers the specified workflow with inputs like `pr_number`, `pr_action`, `head_ref`, `head_sha`, `base_ref`, and `sender`.

> 📘 **Full Documentation**: See [Webhook Service Guide](docs/webhook-usage.md) for complete configuration options and deployment instructions.

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
- `--merge-method`: Merge method (merge/squash/rebase, default: squash)
- `--debug`: Enable debug mode (skip validation, allow PR creator self-approval)
- `--verbose`: Enable verbose logging (debug level logs)
- `--lgtm-permissions`: Comma-separated list of permissions required for LGTM (default: admin,write)
- `--lgtm-review-event`: Review event type for LGTM (default: APPROVE)
- `--self-check-name`: Name for self-check (default: pr-cli)
- `--robot-accounts`: Comma-separated list of bot accounts for managing bot approval reviews
- `--base-url`: API base URL (optional, defaults per platform)
- `--use-git-cli-for-cherrypick`: Use Git CLI for cherry-pick operations (default: true)
- `--results-dir`: Directory to write results files (default: /tekton/results)
- `--version`, `-v`: Show version information
- `--output`, `-o`: Output format for version (text|json)

### LGTM Configuration

```bash
# Set LGTM threshold
pr-cli --lgtm-threshold 2 --trigger-comment "/lgtm" ...

# Custom merge method
pr-cli --merge-method squash --trigger-comment "/merge" ...
```

## Tekton Results

When used in Tekton pipelines, pr-cli writes result files to the configured results directory (specified by `--results-dir`). The following table shows all available results:

| Result Name | Value | Condition | Description |
|-------------|-------|-----------|-------------|
| `merge-successful` | `"true"` | When merge operation completes successfully | Indicates that a PR has been successfully merged |
| `has-cherry-pick-comments` | `"true"`/`"false"` | During `/merge` or `/ready` operations | Indicates if there are cherry-pick comments in the PR |

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
