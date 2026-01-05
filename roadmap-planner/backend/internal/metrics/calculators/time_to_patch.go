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

package calculators

import (
	"context"
	"sort"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
)

// TimeToPatchCalculator calculates time from bug report to patch release
type TimeToPatchCalculator struct {
	BaseCalculator
}

// NewTimeToPatchCalculator creates a new time to patch calculator
func NewTimeToPatchCalculator(options map[string]interface{}) *TimeToPatchCalculator {
	return &TimeToPatchCalculator{
		BaseCalculator: NewBaseCalculator(
			"time_to_patch",
			"Time from bug/vulnerability report to patch release",
			"days",
			[]string{"component", "pillar"},
			options,
		),
	}
}

// Calculate computes the time to patch metric
func (c *TimeToPatchCalculator) Calculate(ctx context.Context, data *models.CalculationContext) ([]models.MetricResult, error) {
	percentile := c.GetIntOption("percentile", 50)
	bugTypes := c.GetStringSliceOption("bug_types", []string{"Bug", "Vulnerability", "Security"})

	// Group bugs by component
	componentBugs := make(map[string][]models.EnrichedEpic)
	for _, epic := range data.Epics {
		// Only include bugs/vulnerabilities that have been released
		if epic.ReleaseDate.IsZero() || epic.CreatedDate.IsZero() {
			continue
		}

		// Filter by issue type (bugs, vulnerabilities)
		if !Contains(bugTypes, epic.IssueType) {
			continue
		}

		// Apply time range filter on release date
		if epic.ReleaseDate.Before(data.TimeRange.Start) || epic.ReleaseDate.After(data.TimeRange.End) {
			continue
		}

		for _, comp := range epic.Components {
			if len(data.Filters.Components) > 0 && !Contains(data.Filters.Components, comp) {
				continue
			}
			componentBugs[comp] = append(componentBugs[comp], epic)
		}

		if len(epic.Components) == 0 {
			componentBugs["unknown"] = append(componentBugs["unknown"], epic)
		}
	}

	results := make([]models.MetricResult, 0, len(componentBugs))

	for component, bugs := range componentBugs {
		var patchTimes []float64

		for _, bug := range bugs {
			// Calculate time from creation to release (patch time)
			patchTime := bug.ReleaseDate.Sub(bug.CreatedDate).Hours() / 24 // Convert to days
			if patchTime >= 0 {
				patchTimes = append(patchTimes, patchTime)
			}
		}

		if len(patchTimes) == 0 {
			continue
		}

		// Calculate percentile
		sort.Float64s(patchTimes)
		idx := int(float64(len(patchTimes)-1) * float64(percentile) / 100)
		if idx >= len(patchTimes) {
			idx = len(patchTimes) - 1
		}
		value := patchTimes[idx]

		// Calculate additional stats
		var sum float64
		for _, pt := range patchTimes {
			sum += pt
		}
		avg := sum / float64(len(patchTimes))

		results = append(results, models.MetricResult{
			Name:  c.Name(),
			Value: value,
			Unit:  c.Unit(),
			Labels: map[string]string{
				"component": component,
			},
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"sample_size": len(patchTimes),
				"percentile":  percentile,
				"min":         patchTimes[0],
				"max":         patchTimes[len(patchTimes)-1],
				"average":     avg,
			},
		})
	}

	return results, nil
}

// PrometheusMetrics returns the Prometheus metric descriptors
func (c *TimeToPatchCalculator) PrometheusMetrics() []models.PrometheusMetricDesc {
	return []models.PrometheusMetricDesc{
		{
			Name:       "time_to_patch_days",
			Help:       "Time from bug report to patch release in days",
			Type:       "gauge",
			LabelNames: []string{"component", "pillar"},
		},
	}
}
