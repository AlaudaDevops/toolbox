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
	"fmt"
	"sync"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/calculators"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
)

// Registry manages metric calculators
type Registry struct {
	mu          sync.RWMutex
	calculators map[string]calculators.MetricCalculator
}

// NewRegistry creates a new metric registry
func NewRegistry() *Registry {
	return &Registry{
		calculators: make(map[string]calculators.MetricCalculator),
	}
}

// Register adds a calculator to the registry
func (r *Registry) Register(calc calculators.MetricCalculator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := calc.Name()
	if _, exists := r.calculators[name]; exists {
		return fmt.Errorf("calculator %s already registered", name)
	}

	r.calculators[name] = calc
	return nil
}

// Get retrieves a calculator by name
func (r *Registry) Get(name string) (calculators.MetricCalculator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	calc, exists := r.calculators[name]
	return calc, exists
}

// All returns all registered calculators
func (r *Registry) All() []calculators.MetricCalculator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	calcs := make([]calculators.MetricCalculator, 0, len(r.calculators))
	for _, calc := range r.calculators {
		calcs = append(calcs, calc)
	}
	return calcs
}

// Names returns all registered calculator names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.calculators))
	for name := range r.calculators {
		names = append(names, name)
	}
	return names
}

// ListMetricInfo returns information about all registered metrics
func (r *Registry) ListMetricInfo() []models.MetricInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]models.MetricInfo, 0, len(r.calculators))
	for _, calc := range r.calculators {
		infos = append(infos, models.MetricInfo{
			Name:             calc.Name(),
			Description:      calc.Description(),
			Unit:             calc.Unit(),
			AvailableFilters: calc.AvailableFilters(),
		})
	}
	return infos
}

// Unregister removes a calculator from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.calculators, name)
}

// Count returns the number of registered calculators
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.calculators)
}
