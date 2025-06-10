# Getting Started with PR Watcher CLI

This guide will help you get started with the PR Watcher CLI tool for both GitHub and GitLab.

## Quick Start

### 1. Build the Application
```bash
make build
```

### 2. Check Prerequisites
```bash
./scripts/check-prerequisites.sh
```

### 3. Run Your First Scan

#### GitHub
```bash
# Replace 'your-org' with your actual GitHub organization
./bin/pr-watcher watch-prs --org your-org --days 7
```

#### GitLab
```bash
# Replace 'your-group' with your actual GitLab group
./bin/pr-watcher watch-mrs --group your-group --days 7
```

## Common Use Cases

### Weekly PR/MR Review

#### GitHub - Scan for PRs older than 7 days:
```bash
./bin/pr-watcher watch-prs --org myorg --days 7 --output weekly-review.json
```

#### GitLab - Scan for MRs older than 7 days:
```bash
./bin/pr-watcher watch-mrs --group mygroup --days 7 --output weekly-review.json
```

### Monthly Cleanup

#### GitHub - Find PRs that have been stale for 30+ days:
```bash
./bin/pr-watcher watch-prs --org myorg --days 30 --output monthly-cleanup.json
```

#### GitLab - Find MRs that have been stale for 30+ days:
```bash
./bin/pr-watcher watch-mrs --group mygroup --days 30 --output monthly-cleanup.json
```

### Include Draft PRs/MRs

#### GitHub
```bash
./bin/pr-watcher watch-prs --org myorg --days 14 --include-drafts --output all-old-prs.json
```

#### GitLab
```bash
./bin/pr-watcher watch-mrs --group mygroup --days 14 --include-drafts --output all-old-mrs.json
```

### Check All PR/MR States

#### GitHub - Include closed and merged PRs:
```bash
./bin/pr-watcher watch-prs --org myorg --days 7 --state all --output all-prs.json
```

#### GitLab - Include closed and merged MRs:
```bash
./bin/pr-watcher watch-mrs --group mygroup --days 7 --state all --output all-mrs.json
```

### GitLab Self-Hosted Instance
```bash
./bin/pr-watcher watch-mrs --group mygroup --days 7 --host gitlab.example.com
```

## Output Format

The tool outputs comprehensive JSON with the following information for each PR/MR:

### GitHub PRs
- Repository name
- PR number and title
- Author information
- Creation and update timestamps
- Current state (open/closed/merged)
- Days since creation
- Labels, assignees, and reviewers
- Branch information

### GitLab MRs
- Project name
- MR IID and title
- Author information
- Creation and update timestamps
- Current state (opened/closed/merged)
- Days since creation
- Labels, assignees, and reviewers
- Branch information
- Pipeline status

## Integration Examples

### Slack Notification Script
```bash
#!/bin/bash
# weekly-notification.sh

# GitHub scan
./bin/pr-watcher watch-prs --org myorg --days 7 --output weekly-github-prs.json
GITHUB_COUNT=$(jq '.total_old_prs' weekly-github-prs.json)

# GitLab scan
./bin/pr-watcher watch-mrs --group mygroup --days 7 --output weekly-gitlab-mrs.json
GITLAB_COUNT=$(jq '.total_old_mrs' weekly-gitlab-mrs.json)

# Send to Slack
TOTAL=$((GITHUB_COUNT + GITLAB_COUNT))
if [ "$TOTAL" -gt 0 ]; then
    echo "Found $GITHUB_COUNT old GitHub PRs and $GITLAB_COUNT old GitLab MRs" | slack-cli send
fi
```

### Email Report
```bash
#!/bin/bash
# Generate and email weekly report

# GitHub
./bin/pr-watcher watch-prs --org myorg --days 7 --output github-report.json
echo "=== GitHub PRs ===" > report.txt
jq -r '.pull_requests[] | "PR #\(.number): \(.title) by \(.author) (\(.days_open) days old)"' github-report.json >> report.txt

# GitLab
./bin/pr-watcher watch-mrs --group mygroup --days 7 --output gitlab-report.json
echo -e "\n=== GitLab MRs ===" >> report.txt
jq -r '.merge_requests[] | "MR !\(.iid): \(.title) by \(.author) (\(.days_open) days old)"' gitlab-report.json >> report.txt

mail -s "Weekly PR/MR Report" team@company.com < report.txt
```

## Troubleshooting

### GitHub CLI Not Authenticated
```bash
gh auth login
```

### GitLab CLI Not Authenticated
```bash
glab auth login
```

### Permission Issues

#### GitHub
Ensure your GitHub token has the following scopes:
- `repo` (for private repositories)
- `read:org` (for organization access)

#### GitLab
Ensure your GitLab token has the following scopes:
- `read_api` (for API access)
- `read_repository` (for repository access)

### Large Organizations/Groups
For organizations or groups with many repositories/projects, the scan might take some time. Consider:
- Running during off-peak hours
- Using filtering by repository/project if possible
- Setting up as a scheduled job

### GitLab Self-Hosted
For self-hosted GitLab instances, use the `--host` flag:
```bash
./bin/pr-watcher watch-mrs --group mygroup --host gitlab.internal.company.com --days 7
```

## Development

### Running Tests
```bash
make test
```

### Building for Multiple Platforms
```bash
make build-all
```

### Code Formatting
```bash
make fmt
```

## Automation Ideas

1. **GitHub Actions**: Set up a workflow to run weekly scans
2. **Cron Jobs**: Schedule regular scans on your CI/CD server
3. **Dashboard Integration**: Parse JSON output into monitoring dashboards
4. **Custom Notifications**: Build custom notification logic based on PR age, labels, or authors

## Support

For issues, feature requests, or contributions, please refer to the main README.md file.
