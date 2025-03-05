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
	"os"
	"testing"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy"
	"github.com/google/go-cmp/cmp"
)

// var _ os.FileInfo = &os.File{}

// Base test for ListFiles without using filters and not handling errors
func TestFileSystemSelector_ListFilesWithoutFilters(t *testing.T) {
	ctx := context.Background()
	s := &fscopy.FileSystemSelector{}

	table := map[string]struct {
		Path          string
		ExpectedFiles []os.FileInfo
	}{
		"basic dual folder case with ignore": {
			"testdata/basic_dual_folder_case_with_ignore",
			[]os.FileInfo{
				// testdata/basic_dual_folder_case_with_ignore
				// 	├── .syncignore
				// 	├── file1.txt
				// 	├── file2.txt
				// 	├── file3.txt (ignored)
				// 	└── subfolder
				// 		├── file4.txt
				//		├── file5.next (ignored)
				// 		└── file6.txt
				// os.NewFile()
			},
		},
	}

	for testName, row := range table {
		t.Run(testName, func(t *testing.T) {
			results, err := s.ListFiles(ctx, row.Path)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(row.ExpectedFiles, results); diff != "" {
				t.Error(diff)
			}
		})
	}
}
