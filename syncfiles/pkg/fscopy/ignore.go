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

	goignore "github.com/monochromegane/go-gitignore"
)

// IgnoreNode represents a node in a ignore file filter, and can be used directly
// to check if a given path is ignored
type IgnoreNode struct {
	path     string
	matcher  goignore.IgnoreMatcher
	children map[string]*IgnoreNode
}

var _ FileFilter = &IgnoreNode{}

// IsFileAllowed implements the FileFilter interface returning true if the file is allowed
// and false if it should be ignored
func (n *IgnoreNode) IsFileAllowed(ctx context.Context, file FileInfo) (bool, error) {
	matchers := n.ListMatchers(file.GetPath())
	for _, matcher := range matchers {
		if matcher.Match(file.GetPath(), false) {
			return false, nil
		}
	}
	return true, nil
}

// ListMatchers returns a map of matchers for the given path
// matching the path and all its ancestors
func (n *IgnoreNode) ListMatchers(path string) (result map[string]goignore.IgnoreMatcher) {
	if !strings.HasPrefix(path, n.path) {
		return
	}
	result = map[string]goignore.IgnoreMatcher{n.path: n.matcher}
	for _, child := range n.children {
		if childResult := child.ListMatchers(path); len(childResult) > 0 {
			for childKey, matcher := range childResult {
				result[childKey] = matcher
			}
		}
	}
	return
}

// AddChild tries to add the child to an existing child
// verifying if it should be added down the tree
// otherwise will add to its own children
func (n *IgnoreNode) AddChild(path string, matcher goignore.IgnoreMatcher) bool {
	if !strings.HasPrefix(path, n.path) {
		return false
	}
	for _, child := range n.children {
		if child.AddChild(path, matcher) {
			return true
		}
	}
	if n.children == nil {
		n.children = make(map[string]*IgnoreNode)
	}
	n.children[path] = &IgnoreNode{
		path:     path,
		matcher:  matcher,
		children: nil,
	}
	return true
}
