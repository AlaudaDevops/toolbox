# Plugin Release Bot

A CLI tool for executing plugin release related tasks.

## Features

- Create a Jira issue to notify the plugin owner to check the community release status and determine if a new plugin version needs to be published

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/alaudadevops/toolbox.git
cd toolbox/plugin-releaser

# Build the application
go build -o plugin-releaser cmd/main.go

# Move the binary to a location in your PATH (optional)
sudo mv plugin-releaser /usr/local/bin/
```

## Usage

### Create a Jira for plugin release check

#### Configuration File

Configuration file (`config.yaml`) example:

```yaml
jira:
  baseURL: <your jira base url>
  username: <your jira username>
  password: <your jira password>

  project: DEVOPS
  issueType: Job
  summary: "Check the community release status of {{.plugin}} and prepare for a new plugin version"
  priority: L1 - High
  customFields:
    # customfield_10001: # Sprint
    #   value: "{{ getActiveSprintId 44}}"
    #   type: number
    # customfield_13501:  # Cost of Delay
    #   value: 890
    # customfield_10006: # Story Points
    #   value: 0
  labels: # labels to apply to the issue
    - ReadyForPlanning

plugins:
  gitlab:
    owner: <jira username of the plugin owner>
    description: |-
      h2. Note
      This is the description of the plugin release check jira.
```

#### Command

```bash
plugin-releaser create-release-check-jira --config config.yaml --verbose
```




