# PR CLI Pipeline Integration

This document describes how to integrate PR CLI with Tekton Pipelines for automated Pull Request management through comment commands.

📖 **Other Documentation:**
- **[Main README](../README.md)** - Project overview and quick start
- **[CLI Reference](../docs/usage.md)** - Complete command-line usage guide

## Overview

The Pipeline automatically triggers when users post specific comment commands on PRs, executing corresponding PR management operations using the PR CLI tool.

**Trigger Pattern**: `^/(help|rebase|lgtm|remove-lgtm|cherry-pick|assign|merge|ready|unassign|label|unlabel|check|retest|batch)([ \t].*)?$`

> 📘 **Pipeline Integration**: This document provides detailed command categorization and pipeline-specific configuration. For basic command usage, see [../README.md](../README.md#supported-commands)

## Command Categories

### Direct Trigger Commands

These commands can be directly triggered by PR comments and will automatically start the pipeline:

| Command | Purpose | Parameters | Example |
|---------|---------|------------|---------|
| `/assign` | Assign reviewers to PR | `user1 user2 ...` | `/assign alice bob` |
| `/unassign` | Remove reviewers from PR | `user1 user2 ...` | `/unassign alice` |
| `/lgtm` | Add LGTM approval | `[message]` | `/lgtm` or `/lgtm Great work!` |
| `/lgtm cancel` | Remove LGTM approval (alias) | - | `/lgtm cancel` |
| `/remove-lgtm` | Remove LGTM approval | `[message]` | `/remove-lgtm` |
| `/merge` | Merge the PR | `[method]` | `/merge` or `/merge squash` |
| `/ready` | Merge the PR (alias for `/merge`) | `[method]` | `/ready` or `/ready squash` |
| `/rebase` | Rebase the PR | - | `/rebase` |
| `/check` | Check PR status or execute multiple commands | `[/cmd1 args... /cmd2 args...]` | `/check` or `/check /assign user1 /merge rebase` |
| `/batch` | Execute multiple commands in batch mode | `/cmd1 args... [/cmd2 args...]` | `/batch /assign user1 /merge squash` |
| `/label` | Add labels to PR | `label1 label2 ...` | `/label bug enhancement` |
| `/unlabel` | Remove labels from PR | `label1 label2 ...` | `/unlabel bug` |
| `/cherry-pick` | Cherry-pick to branches | `branch1 branch2 ...` | `/cherry-pick release/v1.0` |
| `/cherrypick` | Cherry-pick to branches (alias) | `branch1 branch2 ...` | `/cherrypick release/v1.0` |
| `/retest` | Trigger retest of failed checks | - | `/retest` |
| `/help` | Show available commands | - | `/help` |

### Commands Requiring Indirect Trigger

> 💡 **Note:** Some commands supported by pr-cli may not be directly triggerable via PR comments due to pipeline configuration. These commands can be executed indirectly through `/check` or `/batch` until their trigger patterns are added to the pipeline configuration. This allows for gradual rollout and testing of new features.

## Multi-Command Execution

Both `/check` and `/batch` commands support executing multiple commands in a single operation:

### Check Command
**Primary Purpose:** Check PR status, with optional multi-command execution
```bash
/check                                    # Show PR status only
/check /assign user1 /merge rebase       # Execute commands (same as /batch)
```

### Batch Command
**Primary Purpose:** Execute multiple commands in batch mode
```bash
/batch /assign user1 /merge squash       # Assign reviewer then merge with squash
/batch /assign user1 /label bug /retest  # Assign reviewer, add label, trigger retest
```

**Restrictions (Both Commands):**
- Built-in commands (starting with `__`) are blocked for security
- LGTM commands (`/lgtm`, `/remove-lgtm`) are NOT supported in multi-command execution
- Recursive batch calls (`/batch` within `/batch`) are not allowed

**Key Features:**
- Commands run sequentially with results summarized in one comment
- All commands use the same permissions as direct execution
- Execution continues even if some commands fail

## Pipeline Configuration Parameters

The Pipeline supports the following configurable parameters:

### Required Parameters

| Parameter | Description |
|-----------|-------------|
| `trigger_comment` | The comment command to execute |
| `repo_owner` | GitHub/GitLab repository owner |
| `repo_name` | Repository name |
| `pull_request_number` | PR number |
| `comment_sender` | Username of the comment author |
| `git_auth_secret` | Secret containing the API token |

### Optional Parameters

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| `git_auth_secret_key` | `git-provider-token` | The key in git_auth_secret that contains the token |
| `git_comment_auth_secret` | `github-credentials` | Secret containing the token for posting comments |
| `git_comment_auth_secret_key` | `token` | The key in git_comment_auth_secret that contains the comment token |
| `image` | `registry.alauda.cn:60070/devops/toolbox/pr-cli:latest` | Container image for pr-cli tool |
| `lgtm_permissions` | `admin,write,read` | Permission levels required for LGTM, allow read permission for internal repositories |
| `lgtm_threshold` | `1` | LGTM approval count threshold |
| `lgtm_review_event` | `APPROVE` | LGTM review event type |
| `merge_method` | `squash` | Default merge method |
| `self_check_name` | `pr-manage` | Self-check name |
| `platform` | `github` | The platform to use (github, gitlab, gitee) |
| `debug` | `false` | Enable debug mode (skip validation, allow PR creator self-approval) |
| `verbose` | `true` | Enable verbose logging (debug level logs) |
| `robot_accounts` | `alaudabot,dependabot,renovate,alaudaa-renovate,edge-katanomi-app2,edge-katanomi-app2[bot]` | List of bot accounts for managing bot approval reviews |

## Permission Description

- **admin**: Repository administrator permissions
- **write**: Write permissions
- **read**: Read-only permissions

LGTM functionality requires users to have `admin`, `write` or `read` permissions by default. (configurable via `lgtm_permissions` parameter)

## Usage Notes

1. **Comment Format**: Comments must start with `/` followed by supported commands
2. **Permission Check**: Some operations require appropriate repository permissions
3. **Status Check**: Merge operations will check if all required status checks pass
4. **Bot Users**: Bot users like `alaudabot`, `dependabot`, `renovate` etc. are handled specially

## Example Workflows

### Typical PR Review Process:

1. **Check Status**: `/check` - View current PR status
2. **Assign Reviewers**: `/assign @reviewer1 @reviewer2` - Assign reviewers
3. **Code Review**: Reviewers examine code and provide feedback
4. **Approval**: `/lgtm` - Reviewers approve the PR
5. **Merge**: `/merge` - Merge the PR (if all conditions are met)

### Maintenance Operations:

1. **Add Labels**: `/label bug high-priority` - Categorize PR
2. **Rebase**: `/rebase` - Update PR baseline
3. **Cherry Pick**: `/cherry-pick hotfix/v1.0` - Apply to other branches

Through these comment commands, teams can efficiently manage the entire lifecycle of Pull Requests.

## PipelineRun Example

To enable `pr-manage`, add the following `PipelineRun` configuration to your `.tekton` directory:

```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: pr-manage
  annotations:
    pipelinesascode.tekton.dev/pipeline: "https://raw.githubusercontent.com/AlaudaDevops/toolbox/main/pr-cli/pipeline/pr-manage.yaml"
    pipelinesascode.tekton.dev/on-comment: "^/(help|rebase|lgtm|remove-lgtm|cherry-pick|assign|merge|ready|unassign|label|unlabel|check|retest|batch)([ \\t].*)?$"
    pipelinesascode.tekton.dev/max-keep-runs: "5"
spec:
  pipelineRef:
    name: pr-manage
  params:
    - name: trigger_comment
      value: |
        {{ trigger_comment }}
    - name: repo_owner
      value: "{{ repo_owner }}"
    - name: repo_name
      value: "{{ repo_name }}"
    - name: pull_request_number
      value: "{{ pull_request_number }}"
    - name: comment_sender
      value: "{{ sender }}"
    - name: git_auth_secret
      value: "{{ git_auth_secret }}"
    #
    # Optional parameters (value is the default):
    #
    # The key in git_auth_secret that contains the token (default: git-provider-token)
    # - name: git_auth_secret_key
    #   value: "git-provider-token"
    #
    # Optional: Separate secret for posting comments (default: github-credentials)
    # - name: git_comment_auth_secret
    #   value: "github-credentials"
    # - name: git_comment_auth_secret_key
    #   value: "token"
    #
    # Container image for pr-cli tool (default: registry.alauda.cn:60070/devops/toolbox/pr-cli:latest)
    # - name: image
    #   value: "registry.alauda.cn:60070/devops/toolbox/pr-cli:latest"
    #
    # The /lgtm threshold needed of approvers for a PR to be approved (default: 1)
    # - name: lgtm_threshold
    #   value: "1"
    #
    # The permissions the user need to trigger a lgtm (default: admin,write)
    # - name: lgtm_permissions
    #   value: "admin,write"
    #
    # The review event when lgtm is triggered, can be APPROVE,
    # REQUEST_CHANGES, or COMMENT if setting to empty string it will be set as
    # PENDING (default: APPROVE)
    # - name: lgtm_review_event
    #   value: "APPROVE"
    #
    # The merge method to use. Can be one of: merge, squash, rebase (default: squash)
    # - name: merge_method
    #   value: "squash"
    #
    # The name used for self-check status (default: pr-manage)
    # - name: self_check_name
    #   value: "pr-manage"
    #
    # Enable debug mode (skip validation, allow PR creator self-approval) (default: false)
    # - name: debug
    #   value: "false"
    #
    # Enable verbose logging (debug level logs) (default: false)
    # - name: verbose
    #   value: "false"
    #
    # The platform to use, can be one of: github, gitlab, gitee (default: github)
    # - name: platform
    #  value: "github"
    #
    # The robot accounts for managing bot approval reviews.
    # - name: robot_accounts
    #   value: "alaudabot,dependabot,renovate"
```

### Required Parameters

- `trigger_comment`: The comment command to execute
- `repo_owner`: GitHub/GitLab repository owner
- `repo_name`: Repository name
- `pull_request_number`: PR number
- `comment_sender`: Username of the comment author
- `git_auth_secret`: Secret containing the API token

### Optional Parameters

All other parameters have default values and can be customized as needed for your specific requirements.
