# CLI Reference Guide

This is the complete command-line reference for PR CLI. For a quick overview, see the [main README](../README.md). For Pipeline integration, see [Pipeline documentation](../pipeline/README.md).

> 📖 **Quick Navigation:**
> - [Global Flags](#global-flags) - Command-line parameters and environment variables
> - [Command Reference](#command-reference) - Detailed syntax for each command  
> - [Configuration Examples](#configuration-examples) - Platform-specific setups
> - [Troubleshooting](#troubleshooting) - Common issues and solutions

## Global Flags

| Flag | Environment Variable | Description | Required |
|------|---------------------|-------------|----------|
| `--platform` | `PR_PLATFORM` | Platform (github/gitlab) (default: github) | Yes |
| `--token` | `PR_TOKEN` | API token | Yes |
| `--repo-owner` | `PR_REPO_OWNER` | Repository owner | Yes |
| `--repo-name` | `PR_REPO_NAME` | Repository name | Yes |
| `--pr-num` | `PR_PR_NUM` | Pull request number | Yes |
| `--comment-sender` | `PR_COMMENT_SENDER` | Comment author | Yes |
| `--trigger-comment` | `PR_TRIGGER_COMMENT` | Comment to process | Yes |
| `--base-url` | `PR_BASE_URL` | API base URL (optional, defaults per platform) | No |
| `--verbose` | `PR_VERBOSE` | Enable verbose logging (debug level logs) | No |
| `--debug` | `PR_DEBUG` | Enable debug mode (skip validations, allow PR self-approval) | No |
| `--lgtm-threshold` | `PR_LGTM_THRESHOLD` | LGTM threshold (default: 1) | No |
| `--lgtm-permissions` | `PR_LGTM_PERMISSIONS` | Required permissions for LGTM (default: admin,write) | No |
| `--lgtm-review-event` | `PR_LGTM_REVIEW_EVENT` | Review event type for LGTM (default: APPROVE) | No |
| `--merge-method` | `PR_MERGE_METHOD` | Merge method (default: squash) | No |
| `--self-check-name` | `PR_SELF_CHECK_NAME` | Name of the tool's own check run to exclude (default: pr-cli) | No |

## Command Reference

This section provides detailed syntax and behavior for each command. For a quick command overview, see the [main README](../README.md#supported-commands).

> 💡 **Pro Tip**: Most commands can be combined using the `/check` command. See the [`/check` section](#check) for details.

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

#### `/close`

Close the pull request without merging.

**Syntax:**
```
/close
```

**Example:**
```bash
pr-cli --trigger-comment "/close"
```

**Behavior:**
- Closes the PR without merging it
- Checks if PR is already closed and provides appropriate feedback
- Posts a confirmation comment when successfully closed
- Cannot be undone through pr-cli (must be reopened manually through the platform)

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

Check the current status of the pull request or execute sub-commands.

**Syntax:**
```
/check [sub-commands...]
```

**Examples:**
```bash
# Check current status only
pr-cli --trigger-comment "/check"

# Execute single sub-command
pr-cli --trigger-comment "/check /assign user1"

# Execute multiple sub-commands
pr-cli --trigger-comment "/check /assign user1 user2 /label bug"
```

**Basic Output (no sub-commands):**
- Current LGTM count and threshold
- Status check results
- Merge eligibility
- Assigned reviewers

**Multi-command Support:**
The `/check` command can also execute multiple commands in sequence (same functionality as `/batch`).

> 💡 **Note:** For detailed information about multi-command execution, restrictions, and supported commands, see the `/batch` command section below.

#### `/batch`

Execute multiple commands in batch mode in a single operation.

**Syntax:**
```
/batch /cmd1 args... /cmd2 args... /cmd3 args...
```

**Examples:**
```bash
# Assign reviewers and merge with squash
pr-cli --trigger-comment "/batch /assign user1 user2 /merge squash"

# Add labels and assign reviewer
pr-cli --trigger-comment "/batch /label bug enhancement /assign reviewer1"

# Complex workflow: assign, label, and merge
pr-cli --trigger-comment "/batch /assign user1 /label ready /merge rebase"

# Close PR without merging (useful for rejecting changes)
pr-cli --trigger-comment "/batch /label wontfix /close"
```

**Supported Commands:**
The `/batch` command can execute most PR management commands in sequence:
- `/assign` - Assign reviewers
- `/unassign` - Remove reviewers
- `/merge` - Merge the PR
- `/ready` - Merge the PR (alias)
- `/close` - Close the PR without merging
- `/rebase` - Rebase the PR
- `/label` - Add labels
- `/unlabel` - Remove labels
- `/cherrypick` - Cherry-pick to branches
- `/retest` - Retry tests

**Restrictions:**
- Built-in commands (starting with `__`) are blocked for security
- LGTM commands (`/lgtm`, `/remove-lgtm`) are NOT supported in batch execution
- Recursive batch calls (`/batch` within `/batch`) are not allowed

**Key Features:**
- Commands execute sequentially with results summarized in one comment
- All commands use the same permissions as direct execution
- Execution continues even if some commands fail
- Failed commands are clearly marked in the summary

> 💡 **Note:** Some commands supported by pr-cli may require indirect execution through `/check` or `/batch` if they haven't been added to pipeline trigger configurations yet.

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
export PR_PLATFORM=github
export PR_TOKEN=$GITHUB_TOKEN
export PR_REPO_OWNER=myorg
export PR_REPO_NAME=myrepo
export PR_LGTM_THRESHOLD=2
export PR_LGTM_PERMISSIONS=admin,write
export PR_MERGE_METHOD=squash

pr-cli --pr-num 123 --comment-sender alice --trigger-comment "/lgtm"
```

### GitLab Configuration

```bash
export PR_PLATFORM=gitlab
export PR_TOKEN=$GITLAB_TOKEN
export PR_BASE_URL=https://gitlab.example.com/api/v4
export PR_REPO_OWNER=mygroup
export PR_REPO_NAME=myproject
export PR_DEBUG=true
export PR_SELF_CHECK_NAME=pr-cli

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

**Already Closed:**
```
❌ PR #123 is already closed

Cannot close a PR that is already in closed state.
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

### Debug and Verbose Modes

#### Verbose Mode
Enable verbose logging to see debug level logs:

```bash
pr-cli --verbose --trigger-comment "/check"
```

#### Debug Mode
Enable debug mode to skip comment sender validation and allow PR creators to approve their own PR:

```bash
pr-cli --debug --trigger-comment "/check"
```

#### Combined Usage
Use both flags together for full debug capabilities:

```bash
pr-cli --verbose --debug --trigger-comment "/check"
```

### Common Issues

1. **API Rate Limits**: Use authenticated requests and consider implementing backoff
2. **Network Timeouts**: Check connectivity to platform APIs
3. **Token Permissions**: Ensure token has sufficient scopes
4. **Branch Protection**: Verify branch protection rules allow the operation

### Support

For issues and support:
- Check the logs with `--verbose` for detailed debugging information
- Review the [troubleshooting guide](../README.md#troubleshooting)
- Open an issue in the repository
