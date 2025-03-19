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
	"os"
	"path/filepath"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	goignore "github.com/monochromegane/go-gitignore"
)

// FileSystemSelector implements a file selector using the local file system
type FileSystemSelector struct {
}

var _ FileSelector = &FileSystemSelector{}

// ListFiles implements the FileSelector interface
func (s *FileSystemSelector) ListFiles(ctx context.Context, path string, filters ...FileFilter) ([]ifs.FileInfo, error) {
	log := logger.GetLogger(ctx)
	log.Debug("listing files in path ", path)
	matchedFiles := make([]ifs.FileInfo, 0, 10)
	// root never matches (accepts everything)
	// waits specific .syncignore files to check
	if len(filters) == 0 {
		root := &IgnoreNode{path: path, matcher: goignore.DummyIgnoreMatcher(false), matcherConstructorFunc: goignore.NewGitIgnore}
		filters = append(filters, root)
	}
	walkErr := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// not possible to read file, skipping
			return err
		}
		info, err := d.Info()
		if err != nil {
			// not possible to read file, skipping
			log.Warn("error reading file ", path, " error: ", err)
			return nil
		}

		filesToEvaluate := make([]fileInfoImp, 0, 10)
		fileInfo := fileInfoImp{path: path, FileInfo: info}
		filesToEvaluate = append(filesToEvaluate, fileInfo)
		log.Debug("file info: ", fileInfo, " dirInfo: ", d)

		// Walkdir does not walk symlinks so
		// we evaluate them in one level only for now
		// TODO: create a new WalkDir that walks symlinks
		// and refactor this method
		if d.Type()&fs.ModeSymlink != 0 {
			log.Debug("found a symlink: ", path)
			target, err := os.Readlink(path)
			if err != nil {
				log.Warn("error reading symlink ", path, " error: ", err)
				return nil
			}
			evaluated := filepath.Join(filepath.Dir(path), target)
			log.Debug("symlink evaluated: ", evaluated, " target: ", target, " path: ", path)
			entries, err := os.ReadDir(evaluated)
			if err != nil {
				log.Warn("error reading directory ", evaluated, " error: ", err)
				return nil
			}
			log.Debug("directory read: ", evaluated, "entries: ", entries)
			for _, entry := range entries {
				log.Debug("found entry through symlink: ", entry, " path: ", path)

				info, err := entry.Info()
				if err != nil {
					log.Warn("error reading file ", path, " error: ", err)
					continue
				}
				filesToEvaluate = append(filesToEvaluate, fileInfoImp{path: filepath.Join(path, entry.Name()), FileInfo: info})
			}
		}
		for _, fileInfo := range filesToEvaluate {
			allowed := false
			for _, filter := range filters {
				if walker, ok := filter.(FileTreeOperator); ok {
					if err := walker.WalkDirFunc(ctx, path, d, err); err != nil {
						log.Debug("error walking path ", path, " error: ", err)
					}
				}
				if allowed {
					continue
				}
				fileAllowed, err := filter.IsFileAllowed(ctx, fileInfo)
				if err != nil {
					log.Warn("error check if file is allowed in path ", path, " error: ", err)
					continue
				}
				if fileAllowed {
					allowed = true
				}
			}
			if allowed {
				matchedFiles = append(matchedFiles, fileInfo)
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return matchedFiles, nil
}
