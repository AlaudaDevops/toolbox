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

package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/calculators"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
	"go.uber.org/zap"
)

// Service orchestrates metric collection and calculation
type Service struct {
	config    *config.Metrics
	registry  *Registry
	collector *Collector
	logger    *zap.Logger
}

// NewService creates a new metrics service
func NewService(cfg *config.Metrics, collector *Collector) *Service {
	return &Service{
		config:    cfg,
		registry:  NewRegistry(),
		collector: collector,
		logger:    logger.WithComponent("metrics-service"),
	}
}

// Registry returns the calculator registry
func (s *Service) Registry() *Registry {
	return s.registry
}

// Collector returns the data collector
func (s *Service) Collector() *Collector {
	return s.collector
}

// RegisterCalculator adds a calculator to the registry
func (s *Service) RegisterCalculator(calc calculators.MetricCalculator) error {
	// Check if this calculator is enabled in config
	for _, calcCfg := range s.config.Calculators {
		if calcCfg.Name == calc.Name() && !calcCfg.Enabled {
			s.logger.Info("Calculator disabled by config, skipping registration",
				zap.String("calculator", calc.Name()))
			return nil
		}
	}

	if err := s.registry.Register(calc); err != nil {
		return fmt.Errorf("failed to register calculator %s: %w", calc.Name(), err)
	}

	s.logger.Info("Registered calculator", zap.String("calculator", calc.Name()))
	return nil
}

// ListAvailableMetrics returns information about all available metrics
func (s *Service) ListAvailableMetrics() []models.MetricInfo {
	return s.registry.ListMetricInfo()
}

// CalculateMetric calculates a specific metric with the given filters
func (s *Service) CalculateMetric(ctx context.Context, name string, filters models.MetricFilters, timeRange models.TimeRange) ([]models.MetricResult, error) {
	calc, exists := s.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("metric %s not found", name)
	}

	// Get data from collector
	data, err := s.collector.GetData()
	if err != nil {
		return nil, fmt.Errorf("failed to get data from collector: %w", err)
	}

	// Apply filters and time range
	data.Filters = filters
	if !timeRange.Start.IsZero() || !timeRange.End.IsZero() {
		data.TimeRange = timeRange
	}

	// Calculate metric
	results, err := calc.Calculate(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate metric %s: %w", name, err)
	}

	return results, nil
}

// CalculateAllMetrics calculates all registered metrics with the given filters
func (s *Service) CalculateAllMetrics(ctx context.Context, filters models.MetricFilters) (*models.MetricSummary, error) {
	// Get data from collector
	data, err := s.collector.GetData()
	if err != nil {
		return nil, fmt.Errorf("failed to get data from collector: %w", err)
	}

	// Apply filters
	data.Filters = filters

	summary := &models.MetricSummary{
		Timestamp: time.Now(),
		Filters:   filters,
		Metrics:   make(map[string]models.MetricSummaryItem),
	}

	// Calculate each metric
	for _, calc := range s.registry.All() {
		results, err := calc.Calculate(ctx, data)
		if err != nil {
			s.logger.Warn("Failed to calculate metric",
				zap.String("metric", calc.Name()),
				zap.Error(err))
			continue
		}

		// Use the first result as the summary value
		if len(results) > 0 {
			// Aggregate if multiple results
			var totalValue float64
			for _, r := range results {
				totalValue += r.Value
			}
			avgValue := totalValue / float64(len(results))

			summary.Metrics[calc.Name()] = models.MetricSummaryItem{
				Value: avgValue,
				Unit:  calc.Unit(),
			}
		}
	}

	return summary, nil
}

// GetCalculatorOptions returns the options for a specific calculator from config
func (s *Service) GetCalculatorOptions(name string) map[string]interface{} {
	for _, calcCfg := range s.config.Calculators {
		if calcCfg.Name == name {
			return calcCfg.Options
		}
	}
	return nil
}

// IsCalculatorEnabled checks if a calculator is enabled in config
func (s *Service) IsCalculatorEnabled(name string) bool {
	for _, calcCfg := range s.config.Calculators {
		if calcCfg.Name == name {
			return calcCfg.Enabled
		}
	}
	// Default to enabled if not explicitly configured
	return true
}
