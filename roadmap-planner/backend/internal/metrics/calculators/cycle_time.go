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

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
)

// CycleTimeCalculator calculates the time from "In Progress" to "Done"
type CycleTimeCalculator struct {
	BaseCalculator
}

// NewCycleTimeCalculator creates a new cycle time calculator
func NewCycleTimeCalculator(options map[string]interface{}) *CycleTimeCalculator {
	return &CycleTimeCalculator{
		BaseCalculator: NewBaseCalculator(
			"cycle_time",
			"Time from Epic In Progress to Done",
			"days",
			[]string{"component", "pillar"},
			options,
		),
	}
}

// Calculate computes the cycle time metric
func (c *CycleTimeCalculator) Calculate(ctx context.Context, data *models.CalculationContext) ([]models.MetricResult, error) {
	percentile := c.GetIntOption("percentile", 50)
	inProgressStatuses := c.GetStringSliceOption("in_progress_statuses", []string{"In Progress", "In Development"})
	doneStatuses := c.GetStringSliceOption("done_statuses", []string{"Done", "Closed", "Released", "已完成"})

	filteredEpics := make(map[string][]string)
	// Group epics by component
	componentEpics := make(map[string][]models.EnrichedEpic)
	for _, epic := range data.Epics {
		// Only include resolved epics
		if epic.ResolvedDate.IsZero() {
			filteredEpics["No resolved date"] = append(filteredEpics["No resolved date"], epic.Key)
			continue
		}

		// Apply time range filter on resolved date
		if epic.ResolvedDate.Before(data.TimeRange.Start) || epic.ResolvedDate.After(data.TimeRange.End) {
			filteredEpics["Not in range"] = append(filteredEpics["Not in range"], epic.Key)
			continue
		}

		// Apply component filter if specified
		if !FilterByComponent(epic.Components, data.Filters.Components) {
			filteredEpics["Not in component filter"] = append(filteredEpics["Not in component filter"], epic.Key)
			continue
		}
		for _, comp := range epic.Components {
			if len(data.Filters.Components) > 0 && !Contains(data.Filters.Components, comp) {
				filteredEpics["Not in component filter"] = append(filteredEpics["Not in component filter"], epic.Key)
				continue
			}
			componentEpics[comp] = append(componentEpics[comp], epic)
		}

		if len(epic.Components) == 0 {
			componentEpics["unknown"] = append(componentEpics["unknown"], epic)
			filteredEpics["No component"] = append(filteredEpics["No component"], epic.Key)
		}
	}

	logger.Debugf("Filtered epics from cycle time calculation: %v", filteredEpics)

	results := make([]models.MetricResult, 0, len(componentEpics))

	for component, epics := range componentEpics {
		var cycleTimes []float64

		for _, epic := range epics {
			cycleTime := c.calculateCycleTime(epic, inProgressStatuses, doneStatuses)
			if cycleTime > 0 {
				cycleTimes = append(cycleTimes, cycleTime)
			}
		}

		if len(cycleTimes) == 0 {
			logger.Debugf("No cycle times for component %s", component)
			continue
		}

		// Calculate percentile
		sort.Float64s(cycleTimes)
		idx := int(float64(len(cycleTimes)-1) * float64(percentile) / 100)
		if idx >= len(cycleTimes) {
			idx = len(cycleTimes) - 1
		}
		value := cycleTimes[idx]

		// Calculate additional stats
		var sum float64
		for _, ct := range cycleTimes {
			sum += ct
		}
		avg := sum / float64(len(cycleTimes))

		results = append(results, models.MetricResult{
			Name:  c.Name(),
			Value: value,
			Unit:  c.Unit(),
			Labels: map[string]string{
				"component": component,
			},
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"sample_size": len(cycleTimes),
				"percentile":  percentile,
				"min":         cycleTimes[0],
				"max":         cycleTimes[len(cycleTimes)-1],
				"average":     avg,
			},
		})
	}

	return results, nil
}

// calculateCycleTime calculates the cycle time for a single epic based on status changes
func (c *CycleTimeCalculator) calculateCycleTime(epic models.EnrichedEpic, inProgressStatuses, doneStatuses []string) float64 {
	var startTime, endTime time.Time

	// Find when the epic entered "In Progress" and when it became "Done"
	for _, change := range epic.StatusChanges {
		// Find the first time it entered an "in progress" status
		if startTime.IsZero() && Contains(inProgressStatuses, change.ToStatus) {
			startTime = change.ChangedAt
		}

		// Find the last time it entered a "done" status
		if Contains(doneStatuses, change.ToStatus) {
			endTime = change.ChangedAt
		}
	}

	// If we don't have status changes, use created and resolved dates
	if startTime.IsZero() && !epic.CreatedDate.IsZero() {
		startTime = epic.CreatedDate
	}
	if endTime.IsZero() && !epic.ResolvedDate.IsZero() {
		endTime = epic.ResolvedDate
	}

	if startTime.IsZero() || endTime.IsZero() {
		return 0
	}

	// Calculate cycle time in days
	duration := endTime.Sub(startTime)
	if duration < 0 {
		return 0
	}

	return duration.Hours() / 24
}

// PrometheusMetrics returns the Prometheus metric descriptors
func (c *CycleTimeCalculator) PrometheusMetrics() []models.PrometheusMetricDesc {
	return []models.PrometheusMetricDesc{
		{
			Name:       "cycle_time_days",
			Help:       "Cycle time from epic in-progress to done in days",
			Type:       "gauge",
			LabelNames: []string{"component", "pillar"},
		},
	}
}
