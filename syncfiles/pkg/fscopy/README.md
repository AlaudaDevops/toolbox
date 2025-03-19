# File System Copy (fscopy)

The `fscopy` package provides robust file system operations for the syncfiles tool, focusing on file selection, copying, and symbolic linking with support for gitignore-style filtering.

## Overview

This package implements the core functionality for the syncfiles tool, enabling efficient file synchronization across directories with advanced filtering capabilities. It handles three main operations:

```mermaid
graph LR
    A[selectFiles] -->|list of files| B[copyFiles]
    B -->|link targets| C[linkFiles]
```

## Components

### File Selection

The file selection system uses a flexible filtering mechanism that supports:

- Directory traversal using `filepath.WalkDir`
- Gitignore-style pattern matching
- Symlink detection and handling
- Custom file filtering based on path patterns

### File Copying

The copying system:

- Preserves file permissions and attributes
- Maintains directory structure
- Handles relative path transformations
- Efficiently copies file contents

### Symbolic Linking

The linking system:

- Creates symbolic links between directories
- Supports templated link paths with placeholders
- Calculates relative paths automatically
- Handles existing links gracefully

## Interfaces

The package defines several key interfaces:

- `FileSelector`: For selecting files based on filters
- `FileCopier`: For copying files between directories
- `FileFilter`: For filtering files based on custom criteria
- `FileTreeOperator`: For performing operations during directory traversal

## Usage Examples

### Selecting Files

```go
selector := &FileSystemSelector{}
files, err := selector.ListFiles(ctx, "/path/to/source")
```

### Copying Files

```go
copier := &FileSystemCopier{}
err := copier.Copy(ctx, "/path/to/source", "/path/to/destination", files...)
```

### Creating Symbolic Links

```go
copier := &FileSystemCopier{}
err := copier.Link(ctx, "/path/to/source", "/path/to/destination", linkRequests...)
```

## Advanced Features

- **Gitignore Support**: Files can be filtered using `.syncignore` files with gitignore syntax
- **Symlink Detection**: Properly handles symbolic links during traversal
- **Templated Paths**: Supports path templates with placeholders like `<n>` for dynamic path generation
