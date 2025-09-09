/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package messages

import (
	"fmt"
	"slices"
	"strings"
)

// MessageTemplate defines the structure for message templates
type MessageTemplate struct {
	Template string
	Fields   []string
}

// TemplateData holds the data for template substitution
type TemplateData map[string]interface{}

// HelpMessage generates the help message template
func HelpMessage(lgtmThreshold int, lgtmPermissions []string, mergeMethod string) string {
	return fmt.Sprintf(`## 🤖 PR CLI Commands

Available commands for managing this Pull Request:

| Command | Usage | Description | Example |
|---------|-------|-------------|---------|
| **assign** | `+"`/assign user1 user2 ...`"+` | Assign reviewers to the PR | `+"`/assign @alice @bob`"+` |
| **unassign** | `+"`/unassign user1 user2 ...`"+` | Remove assigned reviewers | `+"`/unassign @alice @bob`"+` |
| **lgtm** | `+"`/lgtm`"+` | Approve the PR (requires permissions) | `+"`/lgtm`"+` |
| **remove-lgtm** | `+"`/remove-lgtm` or `/lgtm cancel`"+` | Dismiss your approval (requires permissions) | `+"`/remove-lgtm`"+` |
| **check** | `+"`/check [/cmd1 args... /cmd2 args...]`"+` | Check current LGTM status and check runs status, or execute multiple commands | `+"`/check` or `/check /assign user1 /merge squash`"+` |
| **batch** | `+"`/batch /cmd1 args... /cmd2 args...`"+` | Execute multiple commands in batch mode | `+"`/batch /assign user1 /merge squash`"+` |
| **merge** | `+"`/merge [method]`"+` | Merge the PR after checking permissions, checks, and LGTM status | `+"`/merge squash`"+` |
| **ready** | `+"`/ready [method]`"+` | Alias for merge command | `+"`/ready`"+` |
| **rebase** | `+"`/rebase`"+` | Rebase the PR branch against base | `+"`/rebase`"+` |
| **cherrypick** | `+"`/cherrypick <branch>`"+` | Create a cherrypick PR to a different branch | `+"`/cherrypick release-3.9`"+` |
| **label** | `+"`/label label1 label2 ...`"+` | Add labels to the PR | `+"`/label bug enhancement`"+` |
| **unlabel** | `+"`/unlabel label1 label2 ...`"+` | Remove labels from the PR | `+"`/unlabel bug`"+` |
| **retest** | `+"`/retest`"+` | Trigger retest of failed checks | `+"`/retest`"+` |
| **help** | `+"`/help`"+` | Display this help message | `+"`/help`"+` |

### 👍 LGTM Commands
| Command | Description |
|---------|-------------|
| `+"`/lgtm`"+` | Approve the PR and check if approval threshold is met |
| `+"`/remove-lgtm`"+` | Dismiss your own approval review |
| `+"`/lgtm cancel`"+` | Alternative syntax for dismissing approval |
| `+"`/check`"+` | Check current LGTM status without voting, or execute multiple commands |

### ️ Label Management
| Command | Description |
|---------|-------------|
| `+"`/label <labels>`"+` | Add one or more labels to the PR |
| `+"`/unlabel <labels>`"+` | Remove one or more labels from the PR |

### 📋 Merge Methods
| Method | Description |
|--------|-------------|
| `+"`auto`"+` | Automatically select the best available method (default) |
| `+"`merge`"+` | Create a merge commit |
| `+"`squash`"+` | Squash and merge all commits |
| `+"`rebase`"+` | Rebase and merge |

> 💡 **Auto Mode Priority**: When using `+"`auto`"+`, the system will automatically select the best available merge method in the following order: `+"`rebase`"+` > `+"`squash`"+` > `+"`merge`"+`.

### 🍒 Cherrypick Commands
| Command | Description | PR State |
|---------|-------------|----------|
| `+"`/cherrypick <branch>`"+` | Create a cherrypick PR to target branch | **Merged:** Creates immediately |
| `+"`/cherry-pick <branch>`"+` | Alternative syntax for cherrypick | **Open:** Schedules for after merge |
|  | | **Closed:** Creates from last commit |

### 🔍 Check Command
The `+"`/check`"+` command can show PR status or execute multiple commands:

| Usage | Description | Example |
|-------|-------------|---------|
| `+"`/check`"+` | Show PR status only | `+"`/check`"+` |
| `+"`/check /cmd1 /cmd2 ...`"+` | Execute multiple commands (same as `+"`/batch`"+`) | `+"`/check /assign user1 /merge squash`"+` |

> 💡 **Note:** For detailed information about multi-command execution, see the **Batch Command Execution** section below.

### 🔄 Batch Command Execution
The `+"`/batch`"+` command is a dedicated tool for executing multiple commands in batch mode:

| Usage | Description | Example |
|-------|-------------|---------|
| `+"`/batch /cmd1 /cmd2 ...`"+` | Execute multiple commands sequentially | `+"`/batch /assign user1 /merge squash`"+` |

**Supported Commands:**
- Regular commands: `+"`assign, unassign, merge, ready, rebase, label, unlabel, cherrypick, retest`"+`

**Restrictions:**
- Recursive batch calls are not allowed (`+"`/batch`"+` cannot be used within batch)
- LGTM commands (`+"`/lgtm, /remove-lgtm`"+`) are NOT supported in batch execution

**Example Usage:**
- `+"`/batch /assign user1 /merge rebase`"+` - Assign reviewer and merge with rebase
- `+"`/batch /assign user1 /label bug /retest`"+` - Assign reviewer, add label, trigger retest
- `+"`/batch /label ready /merge squash`"+` - Add label and merge with squash

**Notes:**
- Commands execute sequentially with results summarized
- All sub-commands use the same permissions as direct execution
- Execution continues even if some commands fail

### ⚙️ Configuration
- **LGTM Threshold:** %d approval(s) required
- **Required Permissions:** %s
- **Default Merge Method:** %s

> 💡 **Tip:** Use @username format for user mentions in assign/unassign commands`,
		lgtmThreshold, strings.Join(lgtmPermissions, ", "), mergeMethod)
}

// LGTM Messages
const (
	LGTMApprovalTemplate = `✅ **LGTM from @%s** (Permission: ` + "`%s`" + `)

Approving this PR...`

	LGTMPermissionDeniedTemplate = `❌ **LGTM Permission Denied**

@%s, you don't have sufficient permissions to approve this PR.

**Your permission:** ` + "`%s`" + `
**Required permissions:** %s

Only users with the required permissions can use the /lgtm command.`

	LGTMSelfApprovalTemplate = `ℹ️ **Self-approval not allowed**

@%s, as the PR author, you cannot approve your own PR.

However, I can show you the current LGTM status for this PR:`
)

// Error Messages
const (
	CommandErrorTemplate = `❌ **Command Failed**

Command: ` + "`%s`" + `
Error: %s

Please check the command usage or contact support if the issue persists.`
)

// Merge Messages
const (
	MergeInsufficientPermissionsTemplate = `❌ **Insufficient Permissions**

@%s, you don't have the required permissions to merge this PR.

**Your permission:** %s
**Required permissions:** %s
**PR creator:** @%s

You need either:
- Required repository permissions (%s), OR
- Be the creator of this PR`

	MergeChecksNotPassingTemplate = `⚠️ **Cannot merge PR: Some checks are not passing**

%s

Please wait for all checks to pass before merging.`

	MergeNotEnoughLGTMTemplate = `❌ **Cannot merge: Not enough LGTM approvals**

This PR has **%d/%d** valid LGTM approvals. **%d more approval(s) needed**.

Please ensure the PR has sufficient approvals before merging.`

	MergeFailedTemplate = `❌ **Merge failed**

Failed to merge PR #%d: %v

Please check the PR status and try again.`

	MergeSuccessTemplate = `🎉 **PR Successfully Merged!**

**Merge details:**
- **Method:** %s
- **Merged by:** @%s
- **LGTM votes:** %d/%d

**Approvers:**
%s

Thank you to all reviewers! 🙏`
)

// Assignment Messages
const (
	AssignmentGreetingTemplate = `%s

@%s has requested your review on this pull request. Please take a look when you have a moment. Thanks! 🙏`

	UnassignmentTemplate = `♻️ Removed %s from the review list. Thanks for your time!`
)

// Remove LGTM Messages
const (
	RemoveLGTMPermissionDeniedTemplate = `❌ **Remove LGTM Permission Denied**

@%s, you don't have sufficient permissions to dismiss approvals on this PR.

**Your permission:** ` + "`%s`" + `
**Required permissions:** %s

Only users with the required permissions can use the /remove-lgtm command.`

	RemoveLGTMDismissTemplate = `❌ **LGTM Removed** by @%s

Dismissing previous approval...`

	RemoveLGTMNoApprovalTemplate = `ℹ️ **No Approval to Remove**

@%s, you don't have any active approval review to dismiss on this PR.

**Your permission:** ` + "`%s`" + `

Use ` + "`/lgtm`" + ` first to approve the PR before you can remove your approval.`

	RemoveLGTMSuccessTemplate = `✅ **Approval Dismissed**

@%s has successfully dismissed their approval review.

**Permission:** ` + "`%s`" + `

Use ` + "`/lgtm`" + ` again to re-approve this PR if needed.`

	RemoveLGTMStatusTemplate = `✅ **Approval Dismissed Successfully**

@%s has dismissed their approval review.

**Updated LGTM Status:**
- Current valid approvals: **%d/%d**
- Approvals needed: **%d**

`
)

// Rebase Messages
const (
	RebaseFailedTemplate  = `❌ **Rebase failed**: %v`
	RebaseSuccessTemplate = `✅ **PR rebased successfully** on the base branch.`
)

// Cherry-pick Messages
const (
	CherryPickInvalidCommandTemplate = `❌ **Invalid cherrypick command**

Usage: ` + "`/cherrypick <target-branch>`" + `

Examples:
- ` + "`/cherrypick release-3.9`" + `
- ` + "`/cherrypick release-1.15`" + `

Please specify the target branch for the cherrypick.`

	CherryPickInsufficientPermissionsTemplate = `❌ **Insufficient Permissions**

@%s, you don't have the required permissions to create a cherrypick PR.

**Your permission:** %s
**Required permissions:** %s
**PR creator:** @%s

You need either:
- Required repository permissions (%s), OR
- Be the creator of this PR`

	CherryPickClosedPRTemplate = `❌ **Cannot cherrypick PR**

PR #%d has an unknown state. Cherrypick can be performed on:
- **Merged PRs** (cherrypick is created immediately)
- **Open PRs** (cherrypick is scheduled for when the PR merges)
- **Closed PRs** (cherrypick the last commit)

Current PR state: %s`

	CherryPickErrorTemplate = `❌ **Cherry Pick Failed**

Failed to cherry-pick changes from PR #%d to branch ` + "`%s`" + `:
* Requested by: @%s
* Error: ` + "`%s`" + `

*Possible causes:*
* **🔀 Merge conflicts** - Changes conflict with target branch
* **🍴 Fork PR** - Commits may not be available in target repository  
* **🔒 Branch protection rules** - Target branch has restrictions
* **📁 Binary file conflicts** - Binary files cannot be auto-merged
* **🔗 Missing dependencies** - Required commits not present in target branch
* **❌ Invalid branch name** - Target branch doesn't exist

*Manual resolution options:*
* Create the cherry-pick PR manually using git commands
* Resolve conflicts locally and create PR
* For fork PRs, ensure commits are accessible in target repository

Please resolve any issues and try again.`

	CherryPickSuccessTemplate = `✅ **Cherry Pick Successful**

Successfully cherry-picked changes from PR #%d to branch ` + "`%s`" + `.

*Details:*
* Source PR: #%d
* Cherry-pick PR: #%d
* Target Branch: ` + "`%s`" + `
* Cherry-picked by: @%s
* Latest commit SHA: ` + "`%s`" + ``

	CherryPickScheduledTemplate = `✅ We will cherry-pick this PR to the branch ` + "`%s`" + ` upon merge.`
)

// LGTM Status Messages
const (
	LGTMStatusReadyTemplate = `✅ **LGTM Status - Ready to Merge**

This PR has received **%d/%d** valid LGTM approvals and meets the approval threshold.

**LGTM Summary:**
%s

The PR is now ready for merge! 🎉`

	LGTMStatusPendingTemplate = `⏳ **LGTM Status**

This PR currently has **%d/%d** valid LGTM approvals. **%d more approval(s) needed** to meet the threshold.

**Current LGTM Votes:**
%s

**Required permissions:** %s`

	LGTMStatusTipTemplate = `

>  **Tip:** Use ` + "`/lgtm`" + ` to approve this PR if you have the required permissions.`

	CheckRunsFailedHeaderTemplate = `

⚠️ **Check Runs Status - Some checks are not passing**

| Check Name | Status |
|------------|--------|`

	CheckRunsFailedFooterTemplate = `

> **Note:** All checks must pass before this PR can be merged.`

	CheckRunsPassedTemplate = `

✅ **Check Runs Status - All checks are passing**

This PR is ready for merge from a technical perspective!`
)

// Utility functions for creating user mentions and tables
func FormatUserMentions(users []string) []string {
	mentions := make([]string, len(users))
	for i, user := range users {
		if strings.HasPrefix(user, "@") {
			mentions[i] = user
		} else {
			mentions[i] = "@" + user
		}
	}
	return mentions
}

func BuildUsersTable(lgtmUsers map[string]string, robotUsers map[string]bool) string {
	usersTable := "\n| User | Permission | Valid |\n|------|------------|-------|\n"
	for user, permission := range lgtmUsers {
		if robotUsers[user] {
			continue
		}
		usersTable += fmt.Sprintf("| @%s | `%s` | ✅ |\n", user, permission)
	}
	return usersTable
}

func BuildCheckStatusTable(failedChecks []CheckStatus) string {
	statusTable := "\n| Check Name | Status |\n|------------|--------|\n"
	for _, check := range failedChecks {
		status := check.Conclusion
		if status == "" {
			status = check.Status
		}
		checkName := check.Name
		if check.URL != "" {
			checkName = fmt.Sprintf("[%s](%s)", check.Name, check.URL)
		}
		statusTable += fmt.Sprintf("| %s | `%s` |\n", checkName, status)
	}
	return statusTable
}

func BuildLGTMUsersTable(lgtmUsers map[string]string, lgtmPermissions []string, robotUsers map[string]bool) string {
	usersTable := "\n| User | Permission | Valid |\n|------|------------|-------|\n"
	for user, permission := range lgtmUsers {
		if robotUsers[user] {
			continue
		}
		hasPermission := slices.Contains(lgtmPermissions, permission)
		validMark := "✅"
		if !hasPermission {
			validMark = "❌"
		}
		usersTable += fmt.Sprintf("| @%s | `%s` | %s |\n", user, permission, validMark)
	}
	return usersTable
}

func BuildCheckRunsStatusSection(allPassed bool, failedChecks []CheckStatus) string {
	if allPassed {
		return CheckRunsPassedTemplate
	}

	statusTable := CheckRunsFailedHeaderTemplate
	for _, check := range failedChecks {
		status := check.Conclusion
		if status == "" {
			status = check.Status
		}
		checkName := check.Name
		if check.URL != "" {
			checkName = fmt.Sprintf("[%s](%s)", check.Name, check.URL)
		}
		statusTable += fmt.Sprintf("\n| %s | `%s` |", checkName, status)
	}
	statusTable += CheckRunsFailedFooterTemplate
	return statusTable
}

// CheckStatus represents the status of a check run
type CheckStatus struct {
	Name       string
	Status     string
	Conclusion string
	URL        string
}

// LGTMStatusOptions contains options for generating LGTM status message
type LGTMStatusOptions struct {
	ValidVotes       int
	LGTMUsers        map[string]string
	LGTMThreshold    int
	LGTMPermissions  []string
	RobotAccounts    []string
	IncludeThreshold bool
	ChecksPassed     bool
	FailedChecks     []CheckStatus
}

// BuildLGTMStatusMessage generates a formatted LGTM status message with check runs status
func BuildLGTMStatusMessage(opts LGTMStatusOptions) string {
	// Build robot users map for filtering
	robotUsers := make(map[string]bool)
	for _, robot := range opts.RobotAccounts {
		for user := range opts.LGTMUsers {
			if user == robot {
				robotUsers[user] = true
				break
			}
		}
	}

	// Build users table for status message
	usersTable := BuildLGTMUsersTable(opts.LGTMUsers, opts.LGTMPermissions, robotUsers)

	// Build base LGTM status message
	var message string
	if opts.ValidVotes >= opts.LGTMThreshold {
		message = fmt.Sprintf(LGTMStatusReadyTemplate, opts.ValidVotes, opts.LGTMThreshold, usersTable)
	} else {
		// Not enough LGTM votes
		message = fmt.Sprintf(LGTMStatusPendingTemplate, opts.ValidVotes, opts.LGTMThreshold,
			opts.LGTMThreshold-opts.ValidVotes, usersTable, strings.Join(opts.LGTMPermissions, ", "))

		if opts.IncludeThreshold {
			message += LGTMStatusTipTemplate
		}
	}

	// Add check runs status information
	message += BuildCheckRunsStatusSection(opts.ChecksPassed, opts.FailedChecks)

	return message
}
