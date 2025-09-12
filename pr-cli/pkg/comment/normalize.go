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

package comment

import "strings"

// Normalize normalizes a comment for consistent processing by trimming whitespace
// and handling common escape sequences that might appear in command line inputs
func Normalize(comment string) string {
	// Trim whitespace first
	comment = strings.TrimSpace(comment)

	// Only handle escaped newlines at the end of commands (common in CLI contexts)
	// This prevents unintended replacements in the middle of user content
	// Loop to handle multiple consecutive escape sequences at the end
	for {
		trimmed := false
		if strings.HasSuffix(comment, "\\n") {
			comment = strings.TrimSuffix(comment, "\\n")
			trimmed = true
		}
		if strings.HasSuffix(comment, "\\r") {
			comment = strings.TrimSuffix(comment, "\\r")
			trimmed = true
		}
		if !trimmed {
			break
		}
	}

	// Trim again after suffix removal
	comment = strings.TrimSpace(comment)

	return comment
}
