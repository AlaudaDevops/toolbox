/*
Copyright 2026 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package contributions

import (
	"strings"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
)

// Lane is the four-bucket sprint classification used by the sprint card
// and (eventually) any other panel that wants a coarser "what state is
// this issue in?" answer than Jira's per-issue-type workflow.
type Lane int

const (
	LaneTodo Lane = iota
	LaneInProgress
	LaneDone
	LaneCancelled
)

// String returns the lower-case lane name as it appears on the wire and
// in config. Useful for warning logs when a status name doesn't match
// any lane.
func (l Lane) String() string {
	switch l {
	case LaneTodo:
		return "todo"
	case LaneInProgress:
		return "in_progress"
	case LaneDone:
		return "done"
	case LaneCancelled:
		return "cancelled"
	}
	return "in_progress"
}

// StatusClassifier maps a Jira `status.name` to a Lane. Lookups are
// case-folded. A name that doesn't match any of the four explicit
// lists falls back to in_progress and is recorded in Unknown so the
// caller can log a warning once.
//
// Construction is via NewStatusClassifier; the default cover-set comes
// from DefaultStatusLanes and is merged with any operator-supplied
// overrides (config wins per lane — empty lane in config means "use
// default for that lane").
type StatusClassifier struct {
	bucket map[string]Lane
}

// NewStatusClassifier builds a classifier from the configured lanes.
// Any lane left empty in cfg inherits from DefaultStatusLanes so the
// operator can override one lane without re-declaring the others.
func NewStatusClassifier(cfg config.StatusLanes) StatusClassifier {
	d := DefaultStatusLanes()
	merge := func(configured, fallback []string) []string {
		if len(configured) > 0 {
			return configured
		}
		return fallback
	}
	cfg.Todo = merge(cfg.Todo, d.Todo)
	cfg.InProgress = merge(cfg.InProgress, d.InProgress)
	cfg.Done = merge(cfg.Done, d.Done)
	cfg.Cancelled = merge(cfg.Cancelled, d.Cancelled)

	b := make(map[string]Lane, len(cfg.Todo)+len(cfg.InProgress)+len(cfg.Done)+len(cfg.Cancelled))
	for _, s := range cfg.Todo {
		b[normalizeStatus(s)] = LaneTodo
	}
	for _, s := range cfg.InProgress {
		b[normalizeStatus(s)] = LaneInProgress
	}
	for _, s := range cfg.Done {
		b[normalizeStatus(s)] = LaneDone
	}
	for _, s := range cfg.Cancelled {
		b[normalizeStatus(s)] = LaneCancelled
	}
	return StatusClassifier{bucket: b}
}

// Classify returns the lane for the given status name. Unknown
// statuses fall back to in_progress with `known=false` so the caller
// can record the miss and surface it to the operator.
func (c StatusClassifier) Classify(status string) (lane Lane, known bool) {
	if c.bucket == nil {
		return LaneInProgress, false
	}
	l, ok := c.bucket[normalizeStatus(status)]
	if !ok {
		return LaneInProgress, false
	}
	return l, true
}

func normalizeStatus(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// DefaultStatusLanes returns the W4 (2026-05-19) baseline mapping
// derived from a live Jira query against the DEVOPS project. The lists
// cover all 37 statuses across 17 issue types (Bug, Story, Tech-debt,
// Improvement, Vulnerability, Task, Sub-task, Pillar, Milestone,
// Document, Epic, Job, Customer Reported Incident, Platform
// application, Components, Components-sub-task). Operators can
// override one lane in the ConfigMap without re-declaring the others.
func DefaultStatusLanes() config.StatusLanes {
	return config.StatusLanes{
		Todo: []string{
			"Backlog",
			"Blocked",
			"Open",
			"待处理",
			"阻塞中",
		},
		InProgress: []string{
			"Acceptance Testing",
			"CONFIRM RELEASE",
			"Components-test",
			"DEPLOY",
			"Designing",
			"Developing",
			"Doc Reviewing",
			"In Progress",
			"In Testing",
			"Mitigated",
			"Ready for Delivery",
			"Ready for Doc Review",
			"Ready for QA",
			"Review/Test Failed",
			"Signed Off",
			"Test Failed",
			"Testing",
			"Under Review",
			"Verify",
			"Wait for Verify",
			"调研中",
			"调研完成",
			"设计完成",
			"开发完成",
			"测试完成",
			"验收完成",
		},
		Done: []string{
			"Done",
			"Resolved",
			"using",
			"已完成",
		},
		Cancelled: []string{
			"Cancelled",
			"已取消",
		},
	}
}
