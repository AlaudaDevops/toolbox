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
	"time"

	baseModels "github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
)

// MetricResult represents a single metric calculation result
type MetricResult struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Breakdown []MetricBreakdown      `json:"breakdown,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricBreakdown provides detailed breakdown by dimension
type MetricBreakdown struct {
	Dimension string  `json:"dimension"` // "component", "pillar", "quarter"
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
}

// MetricInfo describes an available metric
type MetricInfo struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Unit             string   `json:"unit"`
	AvailableFilters []string `json:"available_filters"`
}

// MetricSummary represents all metrics aggregated
type MetricSummary struct {
	Timestamp time.Time                    `json:"timestamp"`
	Filters   MetricFilters                `json:"filters"`
	Metrics   map[string]MetricSummaryItem `json:"metrics"`
}

// MetricSummaryItem represents a single metric in the summary
type MetricSummaryItem struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Trend string  `json:"trend,omitempty"` // e.g., "+15%", "-10%"
}

// TimeRange specifies the time window for calculation
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// MetricFilters allows filtering metrics by various dimensions
type MetricFilters struct {
	Components []string `json:"components,omitempty"`
	Pillars    []string `json:"pillars,omitempty"`
	Quarters   []string `json:"quarters,omitempty"`
}

// CalculationContext provides all data needed for metric calculation
type CalculationContext struct {
	Releases  []EnrichedRelease
	Epics     []EnrichedEpic
	TimeRange TimeRange
	Filters   MetricFilters
}

// EnrichedRelease represents a Jira version/release with additional metadata
type EnrichedRelease struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Released    bool      `json:"released"`
	Archived    bool      `json:"archived"`
	ReleaseDate time.Time `json:"release_date"`
	Type        string    `json:"type"` // "major", "minor", "patch"
	Component   string    `json:"component"`
	// Parsed version parts
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// EnrichedEpic represents an Epic with additional metadata for metrics
type EnrichedEpic struct {
	ID            string                    `json:"id"`
	Key           string                    `json:"key"`
	Name          string                    `json:"name"`
	Components    []string                  `json:"components"`
	Versions      []string                  `json:"versions"`
	Status        string                    `json:"status"`
	Priority      string                    `json:"priority"`
	IssueType     string                    `json:"issue_type"` // "Epic", "Bug", "Vulnerability"
	CreatedDate   time.Time                 `json:"created_date"`
	ResolvedDate  time.Time                 `json:"resolved_date,omitempty"`
	ReleaseDate   time.Time                 `json:"release_date,omitempty"`
	StatusChanges []baseModels.StatusChange `json:"status_changes,omitempty"`
	PillarID      string                    `json:"pillar_id,omitempty"`
}

// PrometheusMetricDesc describes a Prometheus metric
type PrometheusMetricDesc struct {
	Name       string
	Help       string
	Type       string // "gauge", "counter", "histogram"
	LabelNames []string
}
