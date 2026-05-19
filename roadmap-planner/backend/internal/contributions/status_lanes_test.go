/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"testing"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
)

func TestStatusClassifier(t *testing.T) {
	c := NewStatusClassifier(config.StatusLanes{})

	cases := []struct {
		status string
		want   Lane
		known  bool
	}{
		// English defaults.
		{"Backlog", LaneTodo, true},
		{"Blocked", LaneTodo, true},
		{"In Progress", LaneInProgress, true},
		{"Acceptance Testing", LaneInProgress, true},
		{"Ready for Delivery", LaneInProgress, true},
		{"Done", LaneDone, true},
		{"Resolved", LaneDone, true},
		{"Cancelled", LaneCancelled, true},

		// Chinese (DEVOPS Epic workflow).
		{"待处理", LaneTodo, true},
		{"调研中", LaneInProgress, true},
		{"已完成", LaneDone, true},
		{"已取消", LaneCancelled, true},

		// Case + whitespace tolerance.
		{"  done  ", LaneDone, true},
		{"IN PROGRESS", LaneInProgress, true},

		// Unknown status falls back to in_progress and reports known=false.
		{"made up status", LaneInProgress, false},
		{"", LaneInProgress, false},
	}

	for _, tc := range cases {
		got, known := c.Classify(tc.status)
		if got != tc.want || known != tc.known {
			t.Errorf("Classify(%q) = (%s, %v), want (%s, %v)",
				tc.status, got, known, tc.want, tc.known)
		}
	}
}

func TestStatusClassifierConfigOverrideOneLane(t *testing.T) {
	// Operator overrides only `todo` — the other three lanes must fall
	// back to the defaults so the operator doesn't have to re-declare
	// them.
	c := NewStatusClassifier(config.StatusLanes{
		Todo: []string{"Inbox"},
	})
	if got, _ := c.Classify("Inbox"); got != LaneTodo {
		t.Errorf("Classify(Inbox) = %s, want todo", got)
	}
	if got, _ := c.Classify("Done"); got != LaneDone {
		t.Errorf("override should not drop default Done lane; got %s", got)
	}
	if got, _ := c.Classify("Cancelled"); got != LaneCancelled {
		t.Errorf("override should not drop default Cancelled lane; got %s", got)
	}
}
