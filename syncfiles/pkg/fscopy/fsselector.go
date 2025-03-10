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
	"io/fs"
	"path/filepath"

	goignore "github.com/monochromegane/go-gitignore"
)

// FileSystemSelector implements a file selector using the local file system
type FileSystemSelector struct {
}

var _ FileSelector = &FileSystemSelector{}

// ListFiles implements the FileSelector interface
func (s *FileSystemSelector) ListFiles(ctx context.Context, path string, filters ...FileFilter) ([]FileInfo, error) {

	matchedFiles := make([]FileInfo, 0, 10)
	// root never matches (accepts everything)
	// waits specific .syncignore files to check
	root := &IgnoreNode{path: path, matcher: goignore.DummyIgnoreMatcher(false), matcherConstructorFunc: goignore.NewGitIgnore}
	filters = append(filters, root)
	walkErr := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// not possible to read file, skipping
			return err
		}
		info, err := d.Info()
		if err != nil {
			// not possible to read file, skipping
			return err
		}
		fileInfo := fileInfoImp{path: path, FileInfo: info}

		allowed := false
		for _, filter := range filters {
			if walker, ok := filter.(FileTreeOperator); ok {
				if err := walker.WalkDirFunc(ctx, path, d, err); err != nil {
					// logger
					return err
				}
			}
			// directories are not filtered by filters
			if allowed || d.IsDir() {
				continue
			}
			fileAllowed, err := filter.IsFileAllowed(ctx, fileInfo)
			if err != nil {
				// logger
				return err
			}
			if fileAllowed {
				allowed = true
			}
		}
		if allowed {
			matchedFiles = append(matchedFiles, fileInfo)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return matchedFiles, nil
}
