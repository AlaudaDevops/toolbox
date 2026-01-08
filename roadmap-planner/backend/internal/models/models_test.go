/*
Copyright 2024 The AlaudaDevops Authors.

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

package models

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestSortEpics(t *testing.T) {
	table := map[string]struct {
		epics    []Epic
		expected []Epic
	}{
		"sort by priority": {
			epics: []Epic{
				{Priority: "L1 - High"},
				{Priority: "L0 - Critical"},
				{Priority: "L2 - Medium"},
			},
			expected: []Epic{
				{Priority: "L0 - Critical"},
				{Priority: "L1 - High"},
				{Priority: "L2 - Medium"},
			},
		},
		"sort by name": {
			epics: []Epic{
				{Name: "L1 - High"},
				{Name: "L0 - Critical"},
				{Name: "L2 - Medium"},
			},
			expected: []Epic{
				{Name: "L0 - Critical"},
				{Name: "L1 - High"},
				{Name: "L2 - Medium"},
			},
		},
		"sort by priority and name": {
			epics: []Epic{
				{Priority: "L0 - Critical", Name: "A"},
				{Priority: "L0 - Critical", Name: "BCD"},
				{Priority: "L1 - High", Name: "B"},
				{Priority: "L1 - High", Name: "A"},
				{Priority: "L2 - Medium", Name: "C"},
			},
			expected: []Epic{
				{Priority: "L0 - Critical", Name: "A"},
				{Priority: "L0 - Critical", Name: "BCD"},
				{Priority: "L1 - High", Name: "A"},
				{Priority: "L1 - High", Name: "B"},
				{Priority: "L2 - Medium", Name: "C"},
			},
		},
	}

	for name, tc := range table {
		t.Run(name, func(t *testing.T) {
			SortEpics(tc.epics)
			if diff := cmp.Diff(tc.expected, tc.epics); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
