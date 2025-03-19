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

package fscopy_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy/fake"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
)

func testLoggerContext() (context.Context, *zap.SugaredLogger) {
	ctx := context.Background()
	log := logger.NewLoggerFromContext(ctx, logger.LogLeveler{Level: "debug"})
	ctx = logger.WithLogger(ctx, log)
	return ctx, log
}

// Base test for ListFiles without using filters and not handling errors
func TestFileSystemSelector_ListFilesWithoutFilters(t *testing.T) {
	ctx, _ := testLoggerContext()
	s := &fscopy.FileSystemSelector{}

	table := map[string]struct {
		Path          string
		ExpectedFiles []ifs.FileInfo
	}{
		"basic dual folder case with ignore": {
			"testdata/basic_dual_folder_case_with_ignore",
			[]ifs.FileInfo{
				// testdata/basic_dual_folder_case_with_ignore
				// 	├── .syncignore
				// 	├── file1.txt
				// 	├── file2.txt
				// 	├── file3.txt (ignored)
				// 	└── subfolder
				// 		├── file4.txt
				//		├── file5.next (ignored)
				// 		├── file6.txt
				// 		└── thirdlevel
				// 			├── included.txt
				// 			└── my.txt (ignored)
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/.syncignore", FileName: ".syncignore"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/file1.txt", FileName: "file1.txt"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/file2.txt", FileName: "file2.txt"},
				//&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/file3.txt", FileName: "file3.txt"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/file4.txt", FileName: "file4.txt"},
				// &fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/file5.next", FileName: "file5.next"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/file6.txt", FileName: "file6.txt"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/thirdlevel/.syncignore", FileName: ".syncignore"},
				&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/thirdlevel/included.txt", FileName: "included.txt"},
				// &fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/thirdlevel/my.txt", FileName: "my.txt"},
			},
		},
		"selector with linked folder": {
			"testdata/linked_folder",
			[]ifs.FileInfo{
				// linked_folder
				// ├── abc
				// │   └── target.md
				// └── xyz -> abc
				&fake.FakeFileInfo{Path: "testdata/linked_folder/abc/target.md", FileName: "target.md"},
				&fake.FakeFileInfo{Path: "testdata/linked_folder/xyz/target.md", FileName: "target.md"},
			},
		},
	}
	for testName, row := range table {
		t.Run(testName, func(t *testing.T) {
			results, err := s.ListFiles(ctx, row.Path)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(row.ExpectedFiles, results, cmp.Comparer(comparePaths)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func comparePaths(x, y ifs.FileInfo) bool {
	return x.GetPath() == y.GetPath()
}

func TestFileSystemSelector_ListFilesNotExistingFolder(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLoggerFromContext(ctx, logger.LogLeveler{Level: "debug"})
	ctx = logger.WithLogger(ctx, log)
	s := &fscopy.FileSystemSelector{}

	_, err := s.ListFiles(ctx, "testdata/non_existing_dir")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFileSystemSelector_ListFilesFilterWalkAndCheckError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLoggerFromContext(ctx, logger.LogLeveler{Level: "debug"})
	ctx = logger.WithLogger(ctx, log)
	s := &fscopy.FileSystemSelector{}

	fakeFileFilter := &fake.FakeFileFilter{Err: errors.New("error filtering file"), FunWalkDir: func(ctx context.Context, path string, d fs.DirEntry, err error) error {
		return errors.New("error walking path")
	}}

	allowedFiles, err := s.ListFiles(ctx, "testdata/basic_dual_folder_case_with_ignore", fakeFileFilter)

	if err != nil {
		t.Error("walking does not return error in filters")
	}
	if len(allowedFiles) != 0 {
		t.Error("should not return any files")
	}
}
