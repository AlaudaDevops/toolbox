# Artifact Scanner

This is a command-line tool for scanning and managing DevOps images, mainly used to scan images for vulnerabilities and synchronize results to Jira.

## Features

- Scan container images and image bundles for vulnerabilities
- Automatically create and manage Jira tickets
- Automatically associate vulnerabilities of related images

## Usage

### Basic Usage

```bash
artifact-scanner scan [options]
```

### Command Line Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `--values` | Path to values file | values.yaml |
| `--branch` | Branch name | main |
| `--config` | Path to configuration file | config.yaml |
| `--bundle` | Bundle to scan, empty string means all | empty |

### Configuration File

Configuration file (`config.yaml`) example:

```yaml
jira:
  baseURL: https://your-jira-instance.atlassian.net # Jira access URL
  username: your-jira-username # Jira username
  password: your-jira-password # Jira password

ops:
  baseURL: https://ops-api-instance # API address of the scanning system provided by operations team
```

## Workflow

1. Read configuration file to get Jira and OPS API access addresses
2. Read values file to get information about images to be scanned
3. Parse bundle images, scan each image for vulnerabilities
4. Calculate priority of Jira issues based on scan results
4. Create or update tickets in Jira:
   - Create parent ticket for the bundle
   - Create child tickets for each related image
   - Link parent tickets and child tickets