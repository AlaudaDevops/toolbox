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

# Enable automatic PR creation
dependabot --repo.url https://github.com/user/repo.git --pr.autoCreate

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
-  `--repo.branch`     branch to clone and create PR against (default "main")
-  `--repo.url`        repository URL to clone and analyze (alternative to dir)

### Configuration System

DependaBot supports a three-tier configuration system, with priority from lowest to highest:

1. **Repository Configuration** (`.github/dependabot.yml` or `.github/dependabot.yaml`)
2. **Local Configuration File**: The first-matched configuration file from the following locations (in order of priority):
   1. specified by `--config` parameter
   2. `.dependabot.yaml` in the current directory
   3. `.dependabot.yml` in the home directory
3. **Command Line Parameters** (highest priority)

#### Repository Configuration File

The repository configuration file supports the GitHub dependabot configuration format, allowing seamless transition to this project for vulnerability management.

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
```

### Git Provider Support

Dependabot currently supports GitHub and Gitlab providers. You can specify the provider using the `--git.provider` parameter or configure it in the local configuration file.


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