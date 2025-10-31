# PR Watcher CLI

A command-line tool for repository and organization maintenance, designed to help identify and track old pull requests across GitHub organizations and GitLab groups.

## Features

- **GitHub Support**:
  - Organization-wide PR scanning
  - Comprehensive GitHub PR reporting
- **GitLab Support**:
  - Group-wide MR scanning
  - Comprehensive GitLab MR reporting
- **Configurable age threshold**: Filter PRs/MRs based on how long they've been open
- **Comprehensive reporting**: Returns detailed JSON reports with all relevant information
- **File output support**: Save results to a file for further processing
- **Draft filtering**: Option to include or exclude draft pull/merge requests
- **Multiple state support**: Filter by state (open/opened, closed, merged, all)

## Prerequisites

- **For GitHub**: [GitHub CLI (gh)](https://cli.github.com/) installed and authenticated
- **For GitLab**: [GitLab CLI (glab)](https://gitlab.com/gitlab-org/cli) installed and authenticated
- Go 1.21 or later (for building from source)

## Installation

### From Source

```bash
git clone <repository-url>
cd pr-watcher
go build -o bin/pr-watcher main.go
```

### Using Go Install

```bash
go install github.com/alaudadevops/toolbox/pr-watcher@latest
```

## Usage

### GitHub - Basic Usage

List all PRs in an organization that have been open for more than 7 days:

```bash
pr-watcher watch-prs --org myorg --days 7
```

### GitLab - Basic Usage

List all MRs in a group that have been open for more than 7 days:

```bash
pr-watcher watch-mrs --group mygroup --days 7
```

### Advanced Usage

#### GitHub
```bash
# Include draft PRs and save to file
pr-watcher watch-prs --org myorg --days 14 --include-drafts --output old-prs.json

# Check PRs open for more than 30 days
pr-watcher watch-prs --org myorg --days 30 --output monthly-report.json

# Include all PR states (open, closed, merged)
pr-watcher watch-prs --org myorg --days 7 --state all
```

#### GitLab
```bash
# Include draft MRs and save to file
pr-watcher watch-mrs --group mygroup --days 14 --include-drafts --output old-mrs.json

# Check MRs open for more than 30 days
pr-watcher watch-mrs --group mygroup --days 30 --output monthly-report.json

# Include all MR states and specify GitLab host
pr-watcher watch-mrs --group mygroup --days 7 --state all --host gitlab.example.com
```

### Command Options

#### GitHub (`watch-prs`)
- `--org, -o`: GitHub organization name (required)
- `--days, -d`: Minimum number of days a PR should be open (default: 7)
- `--output, -f`: Output file path (optional, prints to stdout if not specified)
- `--include-drafts`: Include draft pull requests in results (default: false)
- `--state, -s`: PR state to filter by: open, closed, merged, all (default: open)

#### GitLab (`watch-mrs`)
- `--group, -g`: GitLab group name (required)
- `--days, -d`: Minimum number of days an MR should be open (default: 7)
- `--output, -f`: Output file path (optional, prints to stdout if not specified)
- `--include-drafts`: Include draft merge requests in results (default: false)
- `--state, -s`: MR state to filter by: opened, closed, merged, all (default: opened)
- `--host`: GitLab host (optional, defaults to gitlab.com)

## Output Format

The tool generates a comprehensive JSON report with the following structure:

```json
{
  "organization": "myorg",
  "scan_date": "2025-06-10T10:30:00Z",
  "min_days_open": 7,
  "total_repos": 25,
  "total_old_prs": 12,
  "pull_requests": [
    {
      "repository": "myorg/repo1",
      "number": 123,
      "title": "Add new feature",
      "author": "username",
      "created_at": "2025-05-20T15:30:00Z",
      "updated_at": "2025-05-25T09:15:00Z",
      "url": "https://github.com/myorg/repo1/pull/123",
      "state": "open",
      "draft": false,
      "days_open": 21,
      "labels": ["enhancement", "needs-review"],
      "assignees": ["reviewer1"],
      "reviewers": ["reviewer2"],
      "base_branch": "main",
      "head_branch": "feature/new-feature"
    }
  ]
}
```

## Use Cases

### Weekly PR Review

Create a weekly automation to identify stale PRs:

```bash
pr-watcher watch-prs --org myorg --days 7 --output weekly-prs.json
```

### Monthly Cleanup

Generate monthly reports for PRs that have been open for extended periods:

```bash
pr-watcher watch-prs --org myorg --days 30 --output monthly-cleanup.json
```

### Integration with External Tools

The JSON output can be easily integrated with:
- Slack notifications
- Email reports
- Dashboard tools
- Custom automation scripts

## Error Handling

The tool provides clear error messages for common issues:
- Missing GitHub CLI authentication
- Invalid organization names
- Network connectivity issues
- Permission errors

## WeChat Work Integration

Send PR/MR reports directly to WeChat Work (企业微信) webhooks. See [WECOM_INTEGRATION.md](WECOM_INTEGRATION.md) for detailed documentation.

### Quick Start

```bash
# Test the templates
./scripts/test-wecom.sh

# Send GitHub PR report to WeChat Work
./scripts/send-to-wecom.sh github example-output.json "$WECOM_WEBHOOK_URL"

# Send GitLab MR report to WeChat Work
./scripts/send-to-wecom.sh gitlab example-gitlab-output.json "$WECOM_WEBHOOK_URL"
```

### Integration with PR Watcher

```bash
# Watch PRs and send report to WeChat Work
pr-watcher watch-prs --org myorg --days 7 --output /tmp/prs.json
./scripts/send-to-wecom.sh github /tmp/prs.json "$WECOM_WEBHOOK_URL"

# Watch MRs and send report to WeChat Work
pr-watcher watch-mrs --group mygroup --days 7 --output /tmp/mrs.json
./scripts/send-to-wecom.sh gitlab /tmp/mrs.json "$WECOM_WEBHOOK_URL"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Architecture

The CLI is built using the Cobra framework and follows these principles:
- No global variables
- Function-based initialization
- Clean separation of concerns
- Comprehensive error handling

Each command is self-contained and can be easily extended or modified without affecting other functionality.
