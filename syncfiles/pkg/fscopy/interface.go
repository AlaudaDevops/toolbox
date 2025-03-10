/*    Copyright 2025 AlaudaDevops authors

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
// Package fscopy provides file manipulation utilities
package fscopy

import (
	"context"
	"io/fs"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
)

// FileSelector selects files from a given path respecting .syncignore files rules
// in the given path or in its subfolders returning a list of allowed files
type FileSelector interface {
	ListFiles(ctx context.Context, path string, filters ...FileFilter) ([]ifs.FileInfo, error)
}

// FileFilter checks if a files is allowed to be copied
type FileFilter interface {
	IsFileAllowed(ctx context.Context, info ifs.FileInfo) (bool, error)
}

// FileTreeOperator operator to walk a file tree and do its own processing
type FileTreeOperator interface {
	WalkDirFunc(ctx context.Context, path string, d fs.DirEntry, err error) error
}

// FileCopier copies files from a given path to a destination
// base is the original base path
// dst is the destination path
// files is the list of files to copy
type FileCopier interface {
	Copy(ctx context.Context, base, dst string, files ...ifs.FileInfo) error
}

// fileInfoImp private implementation of the FileInfo interface
type fileInfoImp struct {
	fs.FileInfo
	path string
}

func (f fileInfoImp) GetPath() string { return f.path }

type LinkRequest struct {
	Source      string
	Destination string
}
