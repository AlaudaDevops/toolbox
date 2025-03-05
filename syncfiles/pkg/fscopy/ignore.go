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

func (n *IgnoreNode) IsFileAllowed(ctx context.Context, path string) (bool, error) {
	matchers := n.listMatchers(path)
	for key, matcher := range matchers {
		trimmedPath := strings.TrimPrefix(path, key)
		if matcher.Match(trimmedPath, false) {
			return false, nil
		}
	}
	return true, nil
}

func (n *IgnoreNode) listMatchers(path string) (result map[string]goignore.IgnoreMatcher) {
	if !strings.HasPrefix(path, n.path) {
		return
	}
	result = map[string]goignore.IgnoreMatcher{n.path: n.matcher}
	for _, child := range n.children {
		if childResult := child.listMatchers(path); len(childResult) > 0 {
			for childKey, matcher := range childResult {
				result[childKey] = matcher
			}
		}
	}
	return
}

func (n *IgnoreNode) addChild(path string, matcher goignore.IgnoreMatcher) bool {
	if !strings.HasPrefix(path, n.path) {
		return false
	}
	for _, child := range n.children {
		if child.addChild(path, matcher) {
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
