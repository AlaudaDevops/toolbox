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

package cmd_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/AlaudaDevops/toolbox/syncfiles/cmd"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
)

// Test data:
// testdata
// ├── config.yaml
// ├── copy-directly.yaml
// └── source-folder
//
//	├── file1.txt
//	├── file2.txt
//	├── file3.txt
//	└── subfolder
//		├── file4.txt
//		├── file5.next
//		├── file6.txt
//		└── thirdlevel
//			├── included.txt
//			└── my.txt
//
// ----
// Expected result:
// testdata/copied-directly
// ├── deep
// │	└── included.txt
// └── shallow
//
//	├── file4.txt
//	├── file6.txt
//	└── thirdlevel
//		└── included.txt
func Test_RunCopyDirectly(t *testing.T) {
	os.RemoveAll("testdata/copied-directly")
	t.SkipNow()
	ctx, _ := testLoggerContext()
	err := cmd.RunCopyDirectly(ctx, nil, nil, "testdata/copy-directly.yaml")
	if err != nil {
		t.Error("error from RunCopyDirecly: ", err)
	}
	defer os.RemoveAll("testdata/copied-directly")
	paths := []string{}
	filepath.WalkDir("testdata/copied-directly", func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)
		return nil
	})
	expected := []string{
		"testdata/copied-directly",
		"testdata/copied-directly/deep",
		"testdata/copied-directly/deep/included.txt",
		"testdata/copied-directly/deep/my.txt",
		"testdata/copied-directly/shallow",
		"testdata/copied-directly/shallow/file4.txt",
		"testdata/copied-directly/shallow/file5.next",
		"testdata/copied-directly/shallow/file6.txt",
		"testdata/copied-directly/shallow/thirdlevel",
		"testdata/copied-directly/shallow/thirdlevel/included.txt",
		"testdata/copied-directly/shallow/thirdlevel/my.txt",
	}
	if diff := cmp.Diff(paths, expected); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func testLoggerContext() (context.Context, *zap.SugaredLogger) {
	ctx := context.Background()
	log := logger.NewLoggerFromContext(ctx, logger.LogLeveler{Level: "debug"})
	ctx = logger.WithLogger(ctx, log)
	return ctx, log
}
