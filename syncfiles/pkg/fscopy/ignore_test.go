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
	"strings"
	"testing"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy/fake"
	"github.com/google/go-cmp/cmp"
	goignore "github.com/monochromegane/go-gitignore"
)

func TestIgnoreFileFilter_MatchingIgnoreFiles(t *testing.T) {
	f := &IgnoreNode{
		path:    "abc",
		matcher: goignore.DummyIgnoreMatcher(true),
		children: map[string]*IgnoreNode{
			"abc/def": {
				path:    "abc/def",
				matcher: goignore.DummyIgnoreMatcher(true),
				children: map[string]*IgnoreNode{
					"abc/def/ghi": {
						path:    "abc/def/ghi",
						matcher: goignore.DummyIgnoreMatcher(true),
					},
				},
			},
			"abc/xyz": {
				path:    "abc/xyz",
				matcher: goignore.DummyIgnoreMatcher(true),
			},
		},
	}

	expected := map[string]goignore.IgnoreMatcher{
		"abc":         goignore.DummyIgnoreMatcher(true),
		"abc/def":     goignore.DummyIgnoreMatcher(true),
		"abc/def/ghi": goignore.DummyIgnoreMatcher(true),
	}

	result := f.ListMatchers("abc/def/ghi")
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// Adds child node because the prefix matches parent and first child nodes
func TestIgnoreFileFilter_AddChildMatching(t *testing.T) {
	f := &IgnoreNode{
		path:    "abc",
		matcher: goignore.DummyIgnoreMatcher(true),
		children: map[string]*IgnoreNode{
			"abc/def": {
				path:    "abc/def",
				matcher: goignore.DummyIgnoreMatcher(true),
				children: map[string]*IgnoreNode{
					"abc/def/ghi": {
						path:    "abc/def/ghi",
						matcher: goignore.DummyIgnoreMatcher(true),
					},
				},
			},
			"abc/xyz": {
				path:    "abc/xyz",
				matcher: goignore.DummyIgnoreMatcher(true),
			},
		},
	}

	if !f.AddChild("abc/def/xyz", goignore.DummyIgnoreMatcher(true)) {
		t.Errorf("Expected true, got false")
	}
	if len(f.children) != 2 || len(f.children["abc/def"].children) != 2 {
		t.Errorf("Expected 2 child nodes and 2 grandchild nodes")
	}
}

// Will deny adding this child node because their prefixes do not match
// root: abc
// child: def

func TestIgnoreFileFilter_AddChildNotMatching(t *testing.T) {
	f := &IgnoreNode{
		path:    "abc",
		matcher: goignore.DummyIgnoreMatcher(true),
		children: map[string]*IgnoreNode{
			"abc/def": {
				path:    "abc/def",
				matcher: goignore.DummyIgnoreMatcher(true),
				children: map[string]*IgnoreNode{
					"abc/def/ghi": {
						path:    "abc/def/ghi",
						matcher: goignore.DummyIgnoreMatcher(true),
					},
				},
			},
			"abc/xyz": {
				path:    "abc/xyz",
				matcher: goignore.DummyIgnoreMatcher(true),
			},
		},
	}

	if f.AddChild("def", goignore.DummyIgnoreMatcher(true)) {
		t.Errorf("Expected false, got true")
	}
	if len(f.children) != 2 || len(f.children["abc/def"].children) != 1 || len(f.children["abc/xyz"].children) != 0 {
		t.Errorf("Expected 2 child nodes, only one grandchild node for abc/def and no grandchild node for abc/xyz")
	}
}

// Checks if really matches from root node and matching child/grandchild nodes
func TestIgnoreFileFilter_IsFileAllowed(t *testing.T) {
	f := &IgnoreNode{
		path: "abc",
		// should return false
		matcher: goignore.DummyIgnoreMatcher(false),
		children: map[string]*IgnoreNode{
			"abc/def": {
				path: "abc/def",
				// returns false
				matcher: goignore.DummyIgnoreMatcher(false),
				children: map[string]*IgnoreNode{
					"abc/def/ghi": {
						path: "abc/def/ghi",
						// matches here, making file not allowed
						matcher: goignore.DummyIgnoreMatcher(true),
					},
				},
			},
			"abc/xyz": {
				path:    "abc/xyz",
				matcher: goignore.DummyIgnoreMatcher(true),
			},
		},
	}
	fakeFileInfo := &fake.FakeFileInfo{Path: "abc/def/ghi/bef", FileName: "bef"}
	// not allowed because it matched in internal matcher
	ok, err := f.IsFileAllowed(context.Background(), fakeFileInfo)
	if err != nil {
		t.Error(err)
	}
	if ok {
		t.Errorf("Expected false, got true")
	}
}

func TestIgnoreFileFilter_IsFileAllowedWithTestdata(t *testing.T) {
	matcher, err := goignore.NewGitIgnore("testdata/basic_dual_folder_case_with_ignore/.syncignore", "testdata/basic_dual_folder_case_with_ignore")
	if err != nil {
		t.Error(err)
	}
	f := &IgnoreNode{
		path:    "testdata/basic_dual_folder_case_with_ignore",
		matcher: matcher,
	}

	table := map[string]bool{
		"testdata/basic_dual_folder_case_with_ignore/file1.txt":            true,
		"testdata/basic_dual_folder_case_with_ignore/file2.txt":            true,
		"testdata/basic_dual_folder_case_with_ignore/file3.txt":            false,
		"testdata/basic_dual_folder_case_with_ignore/subfolder/file4.txt":  true,
		"testdata/basic_dual_folder_case_with_ignore/subfolder/file5.next": false,
		"testdata/basic_dual_folder_case_with_ignore/subfolder/file6.txt":  true,
	}

	for path, expected := range table {
		t.Run(path, func(t *testing.T) {
			pathSplit := strings.SplitN(path, "/", -1)
			ok, err := f.IsFileAllowed(context.Background(), &fake.FakeFileInfo{Path: path, FileName: pathSplit[len(pathSplit)-1]})
			if err != nil {
				t.Error(err)
			}
			if ok != expected {
				t.Errorf("Expected %v, got %v for path %s", expected, ok, path)
			}
		})
	}
}
