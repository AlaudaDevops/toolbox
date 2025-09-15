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

package lgtm

import "testing"

func TestLGTMPatterns(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		lgtm   bool
		remove bool
		cancel bool
	}{
		{
			name:   "simple lgtm",
			input:  "/lgtm",
			lgtm:   true,
			remove: false,
			cancel: false,
		},
		{
			name:   "simple remove-lgtm",
			input:  "/remove-lgtm",
			lgtm:   false,
			remove: true,
			cancel: false,
		},
		{
			name:   "simple lgtm cancel",
			input:  "/lgtm cancel",
			lgtm:   true, // LGTMRegexp will match "/lgtm" part
			remove: false,
			cancel: true, // LGTMCancelRegexp will also match
		},
		{
			name:   "multiline with lgtm",
			input:  "/rebase\n/lgtm\n/ready",
			lgtm:   true,
			remove: false,
			cancel: false,
		},
		{
			name:   "multiline with remove-lgtm",
			input:  "/rebase\n/remove-lgtm\n/ready",
			lgtm:   false,
			remove: true,
			cancel: false,
		},
		{
			name:   "multiline with lgtm cancel",
			input:  "/rebase\n/lgtm cancel\n/ready",
			lgtm:   true, // LGTMRegexp will match "/lgtm" part
			remove: false,
			cancel: true, // LGTMCancelRegexp will also match
		},
		{
			name:   "lgtm with extra text",
			input:  "/lgtm looks good",
			lgtm:   true,
			remove: false,
			cancel: false,
		},
		{
			name:   "not a command",
			input:  "This looks good to me",
			lgtm:   false,
			remove: false,
			cancel: false,
		},
		{
			name:   "middle of line should not match",
			input:  "I think /lgtm is good",
			lgtm:   false,
			remove: false,
			cancel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lgtmMatch := LGTMRegexp.MatchString(tt.input)
			removeMatch := RemoveLGTMRegexp.MatchString(tt.input)
			cancelMatch := LGTMCancelRegexp.MatchString(tt.input)

			if lgtmMatch != tt.lgtm {
				t.Errorf("LGTMRegexp.MatchString(%q) = %v, want %v", tt.input, lgtmMatch, tt.lgtm)
			}
			if removeMatch != tt.remove {
				t.Errorf("RemoveLGTMRegexp.MatchString(%q) = %v, want %v", tt.input, removeMatch, tt.remove)
			}
			if cancelMatch != tt.cancel {
				t.Errorf("LGTMCancelRegexp.MatchString(%q) = %v, want %v", tt.input, cancelMatch, tt.cancel)
			}
		})
	}
}
