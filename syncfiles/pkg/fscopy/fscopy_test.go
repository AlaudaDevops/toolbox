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
	"io/fs"
	"os"
	"testing"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy/fake"
)

func TestFileSystemCopier_CopyFile(t *testing.T) {
	ctx, _ := testLoggerContext()

	fsCopier := &fscopy.FileSystemCopier{}

	err := fsCopier.CopyFile(ctx, "testdata/basic_dual_folder_case_with_ignore", "testdata/dest_basic_dual_folder_case_with_ignore", &fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/file1.txt", FileName: "file1.txt"})
	if err != nil {
		t.Error("error copying file ", err)
	}
	defer os.RemoveAll("testdata/dest_basic_dual_folder_case_with_ignore")

	files, err := fs.ReadDir(os.DirFS("testdata/dest_basic_dual_folder_case_with_ignore"), ".")
	if err != nil || len(files) != 1 {
		t.Error("did not copy file ", err)
	}
}

func TestFileSystemCopier_Copy(t *testing.T) {
	ctx, _ := testLoggerContext()

	fsCopier := &fscopy.FileSystemCopier{}

	files := []ifs.FileInfo{
		&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/file1.txt", FileName: "file1.txt"},
		&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/file4.txt", FileName: "file4.txt"},
		&fake.FakeFileInfo{Path: "testdata/basic_dual_folder_case_with_ignore/subfolder/thirdlevel/included.txt", FileName: "included.txt"},
	}

	if err := fsCopier.Copy(ctx, "testdata/basic_dual_folder_case_with_ignore", "testdata/dest_basic_dual_folder_case_with_ignore", files...); err != nil {
		t.Error("error copying files ", err)
	}
	defer os.RemoveAll("testdata/dest_basic_dual_folder_case_with_ignore")

	copiedFiles, err := fs.ReadDir(os.DirFS("testdata/dest_basic_dual_folder_case_with_ignore"), ".")
	if err != nil || len(copiedFiles) != 2 || copiedFiles[0].Name() != "file1.txt" {
		t.Error("did not copy files ", err, "file list ", copiedFiles, "len ", len(copiedFiles))
	}
	copiedFiles, err = fs.ReadDir(os.DirFS("testdata/dest_basic_dual_folder_case_with_ignore/subfolder"), ".")
	if err != nil || len(copiedFiles) != 2 || copiedFiles[0].Name() != "file4.txt" {
		t.Error("did not copy files ", err, "file list ", copiedFiles, "len ", len(copiedFiles))
	}
	copiedFiles, err = fs.ReadDir(os.DirFS("testdata/dest_basic_dual_folder_case_with_ignore/subfolder/thirdlevel"), ".")
	if err != nil || len(copiedFiles) != 1 || copiedFiles[0].Name() != "included.txt" {
		t.Error("did not copy files ", err, "file list ", copiedFiles, "len ", len(copiedFiles))
	}
}

func TestFileSystemCopier_Link(t *testing.T) {
	os.RemoveAll("testdata/linked_test")
	ctx, _ := testLoggerContext()
	fsCopier := &fscopy.FileSystemCopier{}

	type linkTest struct {
		ShouldBeSkiped bool
		Link           ifs.LinkRequest
	}

	linkData := []linkTest{
		{ShouldBeSkiped: false, Link: ifs.LinkRequest{Source: "file1.txt", Destination: "pub/file1.txt"}},
		{ShouldBeSkiped: false, Link: ifs.LinkRequest{Source: "subfolder/file4.txt", Destination: "pub/subfolder/file4.txt"}},
		{ShouldBeSkiped: false, Link: ifs.LinkRequest{Source: "subfolder/thirdlevel", Destination: "pub/subfolder/thirdlevel"}},
		{ShouldBeSkiped: true, Link: ifs.LinkRequest{Source: "subfolder/non-existing-source", Destination: "pub/subfolder/non-existing-source"}}, // should be skipped
	}
	links := []ifs.LinkRequest{}
	for _, link := range linkData {
		links = append(links, link.Link)
	}

	if err := os.MkdirAll("testdata/linked_test/pub", 0755); err != nil {
		t.Error("error creating directory ", err)
	}
	defer func() {
		// leave for debugging purposes
		if !t.Failed() {
			os.RemoveAll("testdata/linked_test")
		}
	}()

	if err := fsCopier.Link(ctx, "testdata/basic_dual_folder_case_with_ignore", "testdata/linked_test", links...); err != nil {
		t.Error("error linking files ", err)
	}
	for _, link := range linkData {
		_, err := os.Lstat("testdata/linked_test/" + link.Link.Destination)
		if link.ShouldBeSkiped && err == nil {
			t.Error("linked file should be skipped ", "testdata/linked_test/"+link.Link.Destination)
		} else if !link.ShouldBeSkiped && err != nil {
			t.Error("error verifying linked file ", "testdata/linked_test/"+link.Link.Destination, " err: ", err)
		}
	}
}
