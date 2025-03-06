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
// Package fake provides fake mocking of fscopy interfaces
package fake

import (
	"io/fs"
	"time"
)

// FakeFileInfo is a struct that implements fs.FileInfo for testing purposes
type FakeFileInfo struct {
	// Path is the file path, returned when GetPath is called
	Path string

	// FileName is the base name of the file
	FileName string

	// Size is the length in bytes for regular files
	FileSize int64

	// Mode represents the file mode and permission bits
	FileMode fs.FileMode

	// ModTime is the file's last modification time
	FileModTime time.Time

	// IsDir indicates whether the file is a directory
	FileIsDir bool

	// Sys holds OS-specific file information
	FileSys any
}

// GetPath returns the file path
func (f *FakeFileInfo) GetPath() string { return f.Path }

func (f *FakeFileInfo) Name() string { return f.FileName }

func (f *FakeFileInfo) Size() int64 { return f.FileSize }

func (f *FakeFileInfo) Mode() fs.FileMode { return f.FileMode }

func (f *FakeFileInfo) ModTime() time.Time { return f.FileModTime }

func (f *FakeFileInfo) IsDir() bool { return f.FileIsDir }

func (f *FakeFileInfo) Sys() any { return f.FileSys }
