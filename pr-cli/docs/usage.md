# Usage Guide

This guide provides detailed information on using the PR CLI tool for managing pull requests.

## Command Reference

### Global Flags

| Flag | Environment Variable | Description | Required |
|------|---------------------|-------------|----------|
| `--platform` | `PR_CLI_PLATFORM` | Platform (github/gitlab) (default: github) | Yes |
| `--token` | `PR_CLI_TOKEN` | API token | Yes |
| `--repo-owner` | `PR_CLI_REPO_OWNER` | Repository owner | Yes |
| `--repo-name` | `PR_CLI_REPO_NAME` | Repository name | Yes |
| `--pr-num` | `PR_CLI_PR_NUM` | Pull request number | Yes |
| `--comment-sender` | `PR_CLI_COMMENT_SENDER` | Comment author | Yes |
| `--trigger-comment` | `PR_CLI_TRIGGER_COMMENT` | Comment to process | Yes |
| `--base-url` | `PR_CLI_BASE_URL` | API base URL (optional, defaults per platform) | No |
| `--debug` | `PR_CLI_DEBUG` | Enable debug mode | No |
| `--lgtm-threshold` | `PR_CLI_LGTM_THRESHOLD` | LGTM threshold (default: 1) | No |
| `--lgtm-permissions` | `PR_CLI_LGTM_PERMISSIONS` | Required permissions for LGTM (default: admin,write) | No |
| `--lgtm-review-event` | `PR_CLI_LGTM_REVIEW_EVENT` | Review event type for LGTM (default: APPROVE) | No |
| `--merge-method` | `PR_CLI_MERGE_METHOD` | Merge method (default: rebase) | No |
| `--self-check-name` | `PR_CLI_SELF_CHECK_NAME` | Name of the tool's own check run to exclude (default: pr-cli) | No |

### Comment Commands

#### `/assign`

Assign reviewers to the pull request.

**Syntax:**
```
/assign @user1 @user2 user3
```

**Examples:**
```bash
pr-cli --trigger-comment "/assign alice bob"
pr-cli --trigger-comment "/assign @team/reviewers"
```

**Behavior:**
- Adds specified users as reviewers
- Removes the '@' prefix if present
- Ignores already assigned reviewers
- Posts a comment confirming the assignment

#### `/unassign`

Remove reviewers from the pull request.

**Syntax:**
```
/unassign @user1 @user2 user3
```

**Examples:**
```bash
pr-cli --trigger-comment "/unassign alice"
pr-cli --trigger-comment "/unassign @bob @charlie"
```

#### `/lgtm`

Add an LGTM (Looks Good To Me) approval.

**Syntax:**
```
/lgtm [message]
```

**Examples:**
```bash
pr-cli --trigger-comment "/lgtm"
pr-cli --trigger-comment "/lgtm Great work!"
```

**Behavior:**
- Checks if user has required permissions
- Adds an approval review
- Posts a comment with LGTM status
- Shows current LGTM count vs threshold

#### `/lgtm cancel` or `/remove-lgtm`

Remove an LGTM approval.

**Syntax:**
```
/lgtm cancel [message]
/remove-lgtm [message]
```

**Examples:**
```bash
pr-cli --trigger-comment "/lgtm cancel"
pr-cli --trigger-comment "/remove-lgtm Found an issue"
```

#### `/merge` or `/ready`

Merge the pull request. The `/ready` command is an alias for `/merge`.

**Syntax:**
```
/merge [method]
/ready [method]
```

**Examples:**
```bash
pr-cli --trigger-comment "/merge"
pr-cli --trigger-comment "/ready"
pr-cli --trigger-comment "/merge squash"
pr-cli --merge-method rebase --trigger-comment "/merge"
```

**Merge Methods:**
- `merge`: Create a merge commit
- `squash`: Squash and merge
- `rebase`: Rebase and merge

**Prerequisites:**
- Sufficient LGTM approvals
- All status checks passing
- PR is in mergeable state

#### `/rebase`

Rebase the pull request against the target branch.

**Syntax:**
```
/rebase
```

**Example:**
```bash
pr-cli --trigger-comment "/rebase"
```

#### `/check`

Check the current status of the pull request.

**Syntax:**
```
/check
```

**Example:**
```bash
pr-cli --trigger-comment "/check"
```

**Output includes:**
- Current LGTM count and threshold
- Status check results
- Merge eligibility
- Assigned reviewers

#### `/label`

Add labels to the pull request.

**Syntax:**
```
/label label1 label2 label3
```

**Example:**
```bash
pr-cli --trigger-comment "/label bug enhancement"
```

#### `/unlabel`

Remove labels from the pull request.

**Syntax:**
```
/unlabel label1 label2 label3
```

**Example:**
```bash
pr-cli --trigger-comment "/unlabel bug"
```

#### `/cherrypick`

Cherry-pick the pull request to other branches.

**Syntax:**
```
/cherrypick branch1 branch2
```

**Example:**
```bash
pr-cli --trigger-comment "/cherrypick release/v1.0 hotfix/urgent"
```

#### `/help`

Display available commands and their usage.

**Syntax:**
```
/help
```

**Example:**
```bash
pr-cli --trigger-comment "/help"
```

## CLI Commands

### `completion`

Generate autocompletion scripts for your shell.

**Syntax:**
```bash
pr-cli completion [bash|zsh|fish|powershell]
```

**Examples:**
```bash
# Bash
pr-cli completion bash > /etc/bash_completion.d/pr-cli

# Zsh
pr-cli completion zsh > "${fpath[1]}/_pr-cli"

# Fish
pr-cli completion fish > ~/.config/fish/completions/pr-cli.fish
```

### `version`

Display version information.

**Syntax:**
```bash
pr-cli version [flags]
```

**Flags:**
- `-o, --output`: Output format (text|json) (default: text)

**Examples:**
```bash
pr-cli version
pr-cli version --output json
```

### `help`

Get help about any command.

**Syntax:**
```bash
pr-cli help [command]
```

**Examples:**
```bash
pr-cli help
pr-cli help completion
```

## Configuration Examples

### GitHub Configuration

```bash
export PR_CLI_PLATFORM=github
export PR_CLI_TOKEN=$GITHUB_TOKEN
export PR_CLI_REPO_OWNER=myorg
export PR_CLI_REPO_NAME=myrepo
export PR_CLI_LGTM_THRESHOLD=2
export PR_CLI_LGTM_PERMISSIONS=admin,write
export PR_CLI_MERGE_METHOD=rebase

pr-cli --pr-num 123 --comment-sender alice --trigger-comment "/lgtm"
```

### GitLab Configuration

```bash
export PR_CLI_PLATFORM=gitlab
export PR_CLI_TOKEN=$GITLAB_TOKEN
export PR_CLI_BASE_URL=https://gitlab.example.com/api/v4
export PR_CLI_REPO_OWNER=mygroup
export PR_CLI_REPO_NAME=myproject
export PR_CLI_DEBUG=true
export PR_CLI_SELF_CHECK_NAME=pr-cli

pr-cli --pr-num 456 --comment-sender bob --trigger-comment "/merge"
```

## LGTM System

### Threshold Configuration

Set the minimum number of LGTM approvals required:

```bash
pr-cli --lgtm-threshold 3 --trigger-comment "/check"
```

### Permission Requirements

Control who can give LGTM approvals:

```bash
pr-cli --lgtm-permissions admin,write --trigger-comment "/lgtm"
```

**Available permissions:**
- `admin`: Repository administrators
- `write`: Users with write access
- `read`: Users with read access

### LGTM Workflow

1. User comments `/lgtm` on a PR
2. System checks user permissions
3. If authorized, adds approval review
4. Updates LGTM count
5. Posts status comment
6. Checks if threshold is met

## Error Handling

### Common Errors

**Insufficient Permissions:**
```
Error: User 'alice' does not have required permissions (admin,write) for LGTM. Current permission: read
```

**Missing LGTM Threshold:**
```
Error: Insufficient LGTM approvals. Required: 2, Current: 1
```

**Failed Status Checks:**
```
Error: Cannot merge PR. Status checks are failing:
- ci/travis-ci: failure
- ci/codecov: pending
```

**Already Merged:**
```
Error: Pull request #123 is already merged
```

### Recovery Strategies

1. **Permission Issues**: Contact repository admin to adjust permissions
2. **Status Checks**: Fix failing tests and wait for checks to pass
3. **Merge Conflicts**: Rebase the PR using `/rebase` command
4. **LGTM Threshold**: Get additional approvals or adjust threshold

## Integration Examples

### GitHub Actions

```yaml
name: PR CLI
on:
  issue_comment:
    types: [created]

jobs:
  pr-cli:
    if: github.event.issue.pull_request
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Process PR comment
      run: |
        pr-cli \
          --platform github \
          --token ${{ secrets.GITHUB_TOKEN }} \
          --repo-owner ${{ github.repository_owner }} \
          --repo-name ${{ github.event.repository.name }} \
          --pr-num ${{ github.event.issue.number }} \
          --comment-sender ${{ github.event.comment.user.login }} \
          --trigger-comment "${{ github.event.comment.body }}"
```

### GitLab CI

```yaml
pr-cli:
  script:
    - pr-cli
      --platform gitlab
      --token $GITLAB_TOKEN
      --repo-owner $CI_PROJECT_NAMESPACE
      --repo-name $CI_PROJECT_NAME
      --pr-num $CI_MERGE_REQUEST_IID
      --comment-sender $GITLAB_USER_LOGIN
      --trigger-comment "$COMMENT_BODY"
  only:
    - merge_requests
  when: manual
```

## Troubleshooting

### Debug Mode

Enable debug mode for verbose logging, skip comment sender validation, and allow PR creators to approve their own PR:

```bash
pr-cli --debug --trigger-comment "/check"
```

### Common Issues

1. **API Rate Limits**: Use authenticated requests and consider implementing backoff
2. **Network Timeouts**: Check connectivity to platform APIs
3. **Token Permissions**: Ensure token has sufficient scopes
4. **Branch Protection**: Verify branch protection rules allow the operation

### Support

For issues and support:
- Check the logs with `--debug`
- Review the [troubleshooting guide](../README.md#troubleshooting)
- Open an issue in the repository
