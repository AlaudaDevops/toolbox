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
	"io"
	"os"
	"path/filepath"
	"strings"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
)

type FileSystemCopier struct {
}

var _ FileCopier = &FileSystemCopier{}

// Copy copies a list of files using a base and destination folders
// implements the FileCopier interface
func (s *FileSystemCopier) Copy(ctx context.Context, base, destination string, files ...ifs.FileInfo) error {
	log := logger.GetLogger(ctx)
	log.Debug("copying files from ", base, " to ", destination)
	for _, file := range files {
		log.Debug("copying file ", file.GetPath())
		if err := s.CopyFile(ctx, base, destination, file); err != nil {
			return err
		}
	}
	return nil
}

// CopyFile copies one file from base to destination
func (s *FileSystemCopier) CopyFile(ctx context.Context, base, destination string, file ifs.FileInfo) error {
	// Get relative path and construct destination path
	relativeFilePath, _ := strings.CutPrefix(file.GetPath(), base)
	desiredFilePath := filepath.Join(destination, relativeFilePath)
	desiredFolderPath := filepath.Dir(desiredFilePath)

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(desiredFolderPath, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(file.GetPath())
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(desiredFilePath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Preserve original file mode
	return os.Chmod(desiredFilePath, file.Mode())
}

const upperDir = ".."

// LinkCopier implements the FileCopier interface
func (s *FileSystemCopier) Link(ctx context.Context, base, destination string, links ...LinkRequest) error {
	log := logger.GetLogger(ctx)
	log.Debug("linking files from ", base, " to ", destination)

	for _, link := range links {
		targetPath := filepath.Join(destination, link.Destination)

		// split target to calculate how many levels there are
		// to navigate upper levels
		targetSplit := strings.Split(targetPath, "/")
		upperList := make([]string, len(targetSplit)-1)
		for i := range len(targetSplit) - 1 {
			upperList[i] = upperDir
		}
		// join upperList with destination
		sourcePath := filepath.Join(append(upperList, base, link.Source)...)

		// creating base dir for target
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil && os.IsExist(err) {
			log.Warn("error creating parent folder for ", targetPath, " err: ", err)
		}
		log.Debug("linking file ", sourcePath, " to ", targetPath)
		err := os.Symlink(sourcePath, targetPath)
		if err != nil {
			log.Error("error linking file from ", sourcePath, " to ", targetPath, " err: ", err)
			return err
		}
	}
	return nil
}
