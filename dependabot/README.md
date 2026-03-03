# DependaBot

This project is a dependency management bot that helps manage project dependencies and automatically updates packages when security vulnerabilities are detected.

## Workflow

1. Automatically scan projects for security vulnerabilities using configured scanners (default: trivy)
2. Filter out Go packages that need upgrading (supports multiple languages, currently focused on Go)
3. Navigate to the project's go.mod directory and upgrade vulnerable packages to fixed versions using go get
4. Create a branch and pull request based on the changes, using gh cli for GitHub pull requests

## Features

- 🔍 Support for multiple security scanners (Trivy, GovulnCheck, etc.)
  - [x] Trivy
  - [ ] GoVulnCheck
- 🎯 Support for multi-language dependency updates
  - [x] Go
  - [ ] Python
  - [ ] Node.js
- 🔌 Support for multiple git service providers
  - [x] GitHub
  - [ ] GitLab
- 📦 Automatic package upgrades to fixed versions using language-specific package managers
- 📊 Detailed update logs and error reports
- ⚙️ Support for configuration files and command-line parameters
- 🌿 Automatic Pull Request creation
- 📁 Submodule cloning suprt
- 🚀 Custom script execution hooks (Pre-scan, Post-scan, Pre-commit, Post-commit)
- 📝 Go get commands output generation

## Installation

### Install from Source

```bash
# Clone repository
git clone https://github.com/AlaudaDevOps/toolbox/dependabot.git
cd dependabot

# Build project
make install
```

### Go Install

```bash
go install github.com/AlaudaDevops/toolbox/dependabot@main

which dependabot
```

## Usage

### Basic Usage

```bash
# Local project mode (automatic scanning)
dependabot --dir /path/to/your/project

# Remote repository mode (clone + automatic scanning)
dependabot --repo.url https://github.com/user/repo.git

# Specify branch
dependabot --repo.url https://github.com/user/repo.git --repo.branch develop

# Enable automatic PR creation (this will also enable branch push)
dependabot --repo.url https://github.com/user/repo.git --pr.autoCreate

# Enable automatic branch push only (without PR creation)
dependabot --repo.url https://github.com/user/repo.git --pr.pushBranch

# Clone with submodules
dependabot --repo.url https://github.com/user/repo.git --repo.includeSubmodules

# View help information
dependabot --help
```

### Command Line Parameters

-  `--config`          config file
-  `--debug`           enable debug log output
-  `--dir`             path to project directory containing go.mod (default: current directory) (default ".")
-  `--git.baseUrl`     Base API URL of the Git provider (e.g., https://api.github.com for GitHub, https://gitlab.example.com for GitLab) (default "https://api.github.com")
-  `--git.provider`    Git provider type (e.g., github, gitlab) (default "github")
-  `--git.token`       Access token for the Git provider (used for authentication and PR creation)
-  `--pr.autoCreate`   enable automatic PR creation
-  `--pr.pushBranch`   enable automatic push branch (automatically enabled when --pr.autoCreate is true)
-  `--runtime.cleanupTempDirs` cleanup temporary directories after execution (default true)
-  `--repo.branch`     branch to clone and create PR against (default "main")
-  `--repo.url`        repository URL to clone and analyze (alternative to dir)
-  `--repo.includeSubmodules` include submodules when cloning repository (default: false)

### Configuration System

DependaBot supports a three-tier configuration system, with priority from lowest to highest:

1. **Repository Configuration** (searched in order):
   - `.dependabot.yml` in project root directory
   - `.dependabot.yaml` in project root directory
   - `.github/dependabot.yml`
   - `.github/dependabot.yaml`
2. **Local Configuration File**: The first-matched configuration file from the following locations (in order of priority):
   1. specified by `--config` parameter
   2. `.dependabot.yaml` in the current directory
   3. `.dependabot.yml` in the home directory
3. **Command Line Parameters** (highest priority)

#### Repository Configuration File

The repository configuration file supports both the GitHub dependabot configuration format and our custom format, allowing seamless transition to this project for vulnerability management.

Example GitHub dependabot configuration:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod" # See documentation for possible values
    directory: "/" # Location of package manifests
    schedule:
      interval: "yearly"
    open-pull-requests-limit: 0
    groups:
      gomod:
        update-types:
          - patch
          - minor
        applies-to: security-updates
        patterns:
          - "*"
    reviewers:
      - somebody
    assignees:
      - somebody
```

#### Local Configuration File

Create a configuration file anywhere and specify it using the `--config` parameter:

```yaml
pr:
  autoCreate: false
  pushBranch: false
  labels:
    - dependencies
  assignees:
    - somebody

git:
  provider: github
  token: xxx

# Configure scanner and its parameters
scanner:
  type: "trivy"
  timeout: "8m"
  params:
    - "--ignore-unfixed"
    - "--scanners"
    - "vuln,secret"

# Runtime behavior configuration
runtime:
  cleanupTempDirs: true  # set false to keep temporary directories for troubleshooting

# Custom script configuration for pipeline hooks
hooks:
  # Pre-verify-scan script: executed before verification security scanning
  # Use case: apply existing remediation commands and verify whether updates are still needed
  preVerifyScan:
    script: |
      #!/bin/bash
      echo "Running pre-verify-scan setup..."
    timeout: "15m"
    continueOnError: false  # Verification stage falls back to regular flow if this script fails

  # Post-verify-scan script: executed after verification security scanning
  # Use case: cleanup workspace after verification stage before regular flow
  postVerifyScan:
    script: |
      #!/bin/bash
      echo "Running post-verify-scan cleanup..."
    timeout: "10m"
    continueOnError: false  # Pipeline stops if cleanup fails

  # Pre-scan script: executed before security scanning
  # Use case: prepare environment, install dependencies, run tests
  preScan:
    script: |
      #!/bin/bash
      echo "Running pre-scan setup..."
    timeout: "10m"
    continueOnError: false  # Pipeline will stop if this script fails

  # Post-scan script: executed after security scanning
  # Use case: process scan results, generate reports, send notifications
  postScan:
    script: |
      #!/bin/bash
      echo "Processing scan results..."
      # Add custom logic to process vulnerability scan results
    timeout: "5m"
    continueOnError: true  # Pipeline will continue even if this script fails

  # Pre-commit script: executed before committing changes
  # Use case: validate changes, run additional checks, format code
  preCommit:
    script: |
      #!/bin/bash
      echo "Running pre-commit checks..."
    timeout: "10m"
    continueOnError: true  # Pipeline will continue even if this script fails

  # Post-commit script: executed after committing changes
  # Use case: run tests, trigger CI/CD, send notifications
  postCommit:
    script: |
      #!/bin/bash
      echo "Running post-commit tasks..."
      # Add custom logic like running tests, triggering CI/CD
    timeout: "15m"
    continueOnError: true  # Pipeline will continue even if this script fails

# Updater configuration
updater:
  go:
    # Indicate the file to store the go get commands
    commandOutputFile: ".tekton/patches/dependabot-go-get-commands.sh"
```

`runtime.cleanupTempDirs` is a global switch. When set to `false`, DependaBot keeps temporary directories created by repository cloning and scanner execution.

#### Verify Existing Remediation Before Regenerating Commands

If your repository already stores remediation commands (for example in `.tekton/patches/dependabot-go-get-commands.sh`), you can use `preVerifyScan`/`postVerifyScan` to avoid duplicate PRs:

```yaml
hooks:
  preVerifyScan:
    script: |
      #!/bin/bash
      set -ex
      make clean-patches-default apply-patches-default upgrade-go-dependencies-default

  postVerifyScan:
    script: |
      #!/bin/bash
      set -ex
      make clean-patches-default

  preScan:
    script: |
      #!/bin/bash
      set -ex
      SKIP_DEPENDABOT_COMMANDS=true make apply-patches upgrade-go-dependencies
```

With this setup:

1. DependaBot first verifies vulnerabilities after applying existing remediation commands.
2. If no vulnerabilities remain, it exits early and skips PR generation.
3. If vulnerabilities remain, it cleans up and continues with the regular update flow to regenerate commands.

### Git Provider Support

DependaBot currently supports GitHub and Gitlab providers. You can specify the provider using the `--git.provider` parameter or configure it in the local configuration file.

```yaml
# github provider example
git:
  provider: github
  token: xxx
```

```yaml
# gitlab provider example
git:
  provider: gitlab
  token: xxx
  baseUrl: https://gitlab.example.com
```

### Notice Configuration

Notice configuration is used to send notifications about the vulnerability updates.

Currently, only WeCom webhook is supported.

```yaml
notice:
  type: "wecom"  # or "wechat"
  params:
    webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
```

### Generated Go Get Commands

When DependaBot detects vulnerabilities in Go dependencies, it generates a script file containing the necessary `go get` commands to upgrade the vulnerable packages to their fixed versions. This file is created at the path specified in the `updater.go.commandOutputFile` configuration.

Example of generated commands:

```bash
go get github.com/cloudflare/circl@v1.6.1
go get github.com/go-jose/go-jose/v3@v3.0.4
go get github.com/go-jose/go-jose/v4@v4.0.5
go get github.com/golang-jwt/jwt/v4@v4.5.2
go get github.com/open-policy-agent/opa@v1.4.0
go mod tidy
```

These commands can be executed manually or integrated into CI/CD pipelines to automatically apply the security updates.

### Pipeline Execution Flow

DependaBot pipeline executes in the following order:

1. **Git Clone** - Clone the repository
2. **Pre-verify-scan Hook** (Optional) - Apply existing remediation state before verification scan
3. **Verification Security Scanning** (Optional) - Check whether existing remediation already resolves vulnerabilities
4. **Post-verify-scan Hook** (Optional) - Cleanup workspace after verification stage
5. **Early Exit** (Conditional) - Stop pipeline when verification scan finds no vulnerabilities
6. **Pre-scan Hook** - Prepare environment before regular security scanning
7. **Security Scanning** - Scan for vulnerabilities using configured scanner
8. **Post-scan Hook** - Process scan results, generate reports
9. **Package Updates** - Update vulnerable packages to fixed versions
10. **Pre-commit Hook** - Validate changes before committing
11. **Commit Changes** - Create branch, commit and push changes
12. **Post-commit Hook** - Run tests, trigger CI/CD after commit
13. **PR Creation** - Create pull request (if enabled)
14. **Notification** - Send notification about updates (if configured)

Each hook is optional and can be configured with custom scripts, timeout settings, and error handling behavior.
