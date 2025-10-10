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

// ToggleAllUncheckedCheckboxes returns a comment body where all unchecked checkboxes ([ ])
// are marked as checked ([x]). The second return value indicates how many checkboxes were toggled.
func ToggleAllUncheckedCheckboxes(body string) (string, int) {
	count := strings.Count(body, "[ ]")
	if count == 0 {
		return body, 0
	}

	updated := strings.ReplaceAll(body, "[ ]", "[x]")
	return updated, count
}

// HasUncheckedCheckbox reports whether the body contains at least one unchecked checkbox.
func HasUncheckedCheckbox(body string) bool {
	return strings.Contains(body, "[ ]")
}
