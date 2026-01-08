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

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
)

// MetricCalculator is the core interface for all metric calculators.
// Implement this interface to add new metrics to the system.
type MetricCalculator interface {
	// Name returns the unique identifier for this metric
	Name() string

	// Description returns a human-readable description of what this metric measures
	Description() string

	// Unit returns the unit of measurement (e.g., "days", "releases/month", "ratio")
	Unit() string

	// AvailableFilters returns the list of filters this metric supports
	AvailableFilters() []string

	// Calculate computes the metric value(s) based on the provided context
	Calculate(ctx context.Context, data *models.CalculationContext) ([]models.MetricResult, error)

	// PrometheusMetrics returns the Prometheus metric descriptors for this metric
	PrometheusMetrics() []models.PrometheusMetricDesc
}

// BaseCalculator provides common functionality for all calculators
type BaseCalculator struct {
	name        string
	description string
	unit        string
	filters     []string
	options     map[string]interface{}
}

// NewBaseCalculator creates a new base calculator
func NewBaseCalculator(name, description, unit string, filters []string, options map[string]interface{}) BaseCalculator {
	if options == nil {
		options = make(map[string]interface{})
	}
	return BaseCalculator{
		name:        name,
		description: description,
		unit:        unit,
		filters:     filters,
		options:     options,
	}
}

// Name returns the metric name
func (b *BaseCalculator) Name() string {
	return b.name
}

// Description returns the metric description
func (b *BaseCalculator) Description() string {
	return b.description
}

// Unit returns the metric unit
func (b *BaseCalculator) Unit() string {
	return b.unit
}

// AvailableFilters returns the available filters
func (b *BaseCalculator) AvailableFilters() []string {
	return b.filters
}

// GetOption retrieves an option value with a default fallback
func (b *BaseCalculator) GetOption(key string, defaultValue interface{}) interface{} {
	if val, exists := b.options[key]; exists {
		return val
	}
	return defaultValue
}

// GetIntOption retrieves an int option with a default fallback
func (b *BaseCalculator) GetIntOption(key string, defaultValue int) int {
	val := b.GetOption(key, defaultValue)
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetFloat64Option retrieves a float64 option with a default fallback
func (b *BaseCalculator) GetFloat64Option(key string, defaultValue float64) float64 {
	val := b.GetOption(key, defaultValue)
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultValue
	}
}

// GetStringOption retrieves a string option with a default fallback
func (b *BaseCalculator) GetStringOption(key string, defaultValue string) string {
	val := b.GetOption(key, defaultValue)
	if s, ok := val.(string); ok {
		return s
	}
	return defaultValue
}

// GetStringSliceOption retrieves a string slice option with a default fallback
func (b *BaseCalculator) GetStringSliceOption(key string, defaultValue []string) []string {
	val := b.GetOption(key, nil)
	if val == nil {
		return defaultValue
	}
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return defaultValue
	}
}

// FilterByComponent checks if an item matches the component filter
func FilterByComponent(components []string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, comp := range components {
		for _, f := range filter {
			if comp == f {
				return true
			}
		}
	}
	return false
}

// Contains checks if a slice contains a specific string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
