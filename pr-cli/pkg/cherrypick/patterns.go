/*
Copyright 2025 The AlaudaDevops Authors.

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

package cherrypick

import "regexp"

// Pre-compiled regular expressions for cherry-pick commands to avoid repeated compilation
// These patterns support both single-line and multiline comments using the (?m) flag
var (
	// CherryPickPattern matches /cherry-pick or /cherrypick command with target branch at the start of a line
	CherryPickPattern = regexp.MustCompile(`(?m)^/cherry-?pick\s+(\S+)`)
)
