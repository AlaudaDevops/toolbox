# SyncFiles

A powerful file synchronization tool for complex file operations

## Overview

SyncFiles is designed to handle complex file synchronization scenarios, particularly useful for documentation and content management. It allows you to:

1. Copy files from multiple source directories to a target directory
2. Create symbolic links between files and directories
3. Apply gitignore-style filtering to control which files are synchronized
4. Use templated paths with placeholders for flexible directory structures

## Features

- **Multi-source synchronization**: Copy files from multiple source directories
- **Symbolic linking**: Create symbolic links with automatic relative path calculation
- **Gitignore-style filtering**: Filter files using `.syncignore` files
- **Templated paths**: Use placeholders like `<name>` for dynamic path generation
- **Configurable via YAML**: Simple configuration using YAML files
- **Preserve file attributes**: Maintain file permissions and attributes during copying

## Installation

```bash
# Clone the repository
git clone https://github.com/AlaudaDevops/toolbox.git

# Build the syncfiles tool
cd toolbox/syncfiles
go build -o syncfiles

# Optional: Move to a directory in your PATH
sudo mv syncfiles /usr/local/bin/
```

## Usage

The basic command structure is:

```bash
syncfiles copy --config <config-file.yaml>
```

### Configuration File

SyncFiles uses a YAML configuration file to define sources and targets. Here's an example:

```yaml
sources:
- name: project-a  # Custom name for the source
  dir:
    path: ../path/to/project-a  # Path to source directory

- name: project-b
  dir:
    path: ../path/to/project-b

target:
  copyTo: imported-docs  # Destination directory for copied files
  linkTo: docs  # Base directory for symbolic links
  links:
  - from: public/<name>  # Source path for linking (with placeholder)
    target: public/<name>  # Target path for the link (with placeholder)
  - from: shared/crds
    target: shared/crds/<name>
  - from: en/apis/kubernetes_apis
    target: en/apis/kubernetes_apis/<name>
  - from: en
    target: en/<name>
```

In this configuration:
- `<name>` is a placeholder that will be replaced with the source name
- Files are copied from source directories to `imported-docs`
- Symbolic links are created in the `docs` directory pointing to the copied files

### Command Line Options

```
Usage:
  syncfiles [command]

Available Commands:
  copy        Copy files from multiple sources to a target based on a configuration file
  help        Help about any command

Flags:
  -l, --log-level string   Set the logging level (debug, info, warn, error, panic, fatal). Defaults to info
```

## Example Workflow

1. Create a configuration file `sync-config.yaml`:

```yaml
sources:
- name: docs-v1
  dir:
    path: ./source-docs/v1

target:
  copyTo: imported
  linkTo: website/content
  links:
  - from: en
    target: en/<name>
```

2. Run the synchronization:

```bash
syncfiles copy --config sync-config.yaml
```

This will:
- Copy all files from `./source-docs/v1` to `./imported/docs-v1`
- Create a symbolic link from `./website/content/en/docs-v1` to `../../../imported/docs-v1/en`

## Architecture

SyncFiles is built around three core operations:

```
1. Select Files → 2. Copy Files → 3. Create Symbolic Links
```

The tool uses a modular architecture with well-defined interfaces for file selection, copying, and linking, making it extensible for different use cases.

