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

package fscopy

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

// FileSystemSelector implements a file selector using the local file system
type FileSystemSelector struct {
}

var _ FileSelector = &FileSystemSelector{}

// ListFiles implements the FileSelector interface
func (s *FileSystemSelector) ListFiles(ctx context.Context, path string, filters ...FileFilter) ([]FileInfo, error) {

	// filepath.Walk(path, func(path string, d fs.FileInfo, err error) error {
	// 	return nil
	// })
	// ignoreFiles := make(map[string]struct{})

	filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		dirPath := filepath.Dir(path)
		if d.Name() == ".syncignore" {
			// load ignore file
			return nil
		}

		fmt.Println("--====> ", path, "===dirpath>", dirPath)
		return nil
	})
	return nil, nil
}
