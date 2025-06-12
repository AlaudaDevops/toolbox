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
dependabot --repo https://github.com/user/repo.git

# Specify branch
dependabot --repo https://github.com/user/repo.git --branch develop

# Enable automatic PR creation
dependabot --repo https://github.com/user/repo.git --create-pr=true

# View help information
dependabot --help
```

### Command Line Parameters

- `--dir`: Local project path (default: current directory)
- `--repo`: Remote repository URL (mutually exclusive with dir)
- `--branch`: Branch to clone, also used as PR target branch (default: main)
- `--create-pr`: Enable automatic PR creation (default: false)
- `--config`: External configuration file path (optional)

### Configuration System

DependaBot supports a three-tier configuration system, with priority from lowest to highest:

1. **Repository Configuration** (`.github/dependabot.yml` or `.github/dependabot.yaml`)
2. **External Configuration File** (specified by `--config` parameter)
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

#### External Configuration File

Create a configuration file anywhere and specify it using the `--config` parameter:

```yaml
pr:
  autoCreate: false
  labels:
    - "dependencies"
  assignees:
    - somebody

# Configure scanner and its parameters
scanner:
  type: "trivy"
  timeout: "8m"
  params:
    - "--ignore-unfixed"
    - "--scanners"
    - "vuln,secret"
```
