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
	"testing"

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

	result := f.listMatchers("abc/def/ghi")
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

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

	if !f.addChild("abc/def/xyz", goignore.DummyIgnoreMatcher(true)) {
		t.Errorf("Expected true, got false")
	}
}

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

	if f.addChild("def", goignore.DummyIgnoreMatcher(true)) {
		t.Errorf("Expected false, got true")
	}
}
