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

// LeadTimeCalculator calculates the time from Epic creation to release
type LeadTimeCalculator struct {
	BaseCalculator
}

// NewLeadTimeCalculator creates a new lead time calculator
func NewLeadTimeCalculator(options map[string]interface{}) *LeadTimeCalculator {
	return &LeadTimeCalculator{
		BaseCalculator: NewBaseCalculator(
			"lead_time_to_release",
			"Time from Epic creation to release date",
			"days",
			[]string{"component", "pillar"},
			options,
		),
	}
}

// Calculate computes the lead time metric
func (c *LeadTimeCalculator) Calculate(ctx context.Context, data *models.CalculationContext) ([]models.MetricResult, error) {
	percentile := c.GetIntOption("percentile", 50) // Default to median

	// Group epics by component
	componentEpics := make(map[string][]models.EnrichedIssue)
	for _, epic := range data.Epics {
		// Skip epics without release date
		if epic.ReleaseDate.IsZero() || epic.CreatedDate.IsZero() {
			continue
		}

		// Apply time range filter on release date
		if epic.ReleaseDate.Before(data.TimeRange.Start) || epic.ReleaseDate.After(data.TimeRange.End) {
			continue
		}

		for _, comp := range epic.Components {
			// Apply component filter if specified
			if len(data.Filters.Components) > 0 && !Contains(data.Filters.Components, comp) {
				continue
			}
			componentEpics[comp] = append(componentEpics[comp], epic)
		}

		// If epic has no components, use "unknown"
		if len(epic.Components) == 0 {
			componentEpics["unknown"] = append(componentEpics["unknown"], epic)
		}
	}

	results := make([]models.MetricResult, 0, len(componentEpics))

	for component, epics := range componentEpics {
		var leadTimes []float64
		for _, epic := range epics {
			leadTime := epic.ReleaseDate.Sub(epic.CreatedDate).Hours() / 24 // Convert to days
			if leadTime >= 0 {
				leadTimes = append(leadTimes, leadTime)
			}
		}

		if len(leadTimes) == 0 {
			continue
		}

		// Calculate percentile
		sort.Float64s(leadTimes)
		idx := int(float64(len(leadTimes)-1) * float64(percentile) / 100)
		if idx >= len(leadTimes) {
			idx = len(leadTimes) - 1
		}
		value := leadTimes[idx]

		// Calculate additional stats
		var sum float64
		for _, lt := range leadTimes {
			sum += lt
		}
		avg := sum / float64(len(leadTimes))

		results = append(results, models.MetricResult{
			Name:  c.Name(),
			Value: value,
			Unit:  c.Unit(),
			Labels: map[string]string{
				"component": component,
			},
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"sample_size": len(leadTimes),
				"percentile":  percentile,
				"min":         leadTimes[0],
				"max":         leadTimes[len(leadTimes)-1],
				"average":     avg,
			},
		})
	}

	return results, nil
}

// PrometheusMetrics returns the Prometheus metric descriptors
func (c *LeadTimeCalculator) PrometheusMetrics() []models.PrometheusMetricDesc {
	return []models.PrometheusMetricDesc{
		{
			Name:       "lead_time_days",
			Help:       "Lead time from epic creation to release in days",
			Type:       "gauge",
			LabelNames: []string{"component", "pillar"},
		},
	}
}
