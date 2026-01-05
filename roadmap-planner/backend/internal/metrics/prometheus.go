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
	"net/http"
	"sync"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// PrometheusExporter exposes metrics in Prometheus format
type PrometheusExporter struct {
	config    *config.PrometheusConfig
	service   *Service
	logger    *zap.Logger
	registry  *prometheus.Registry

	mu sync.RWMutex

	// Prometheus metrics
	releaseFrequency *prometheus.GaugeVec
	leadTime         *prometheus.GaugeVec
	cycleTime        *prometheus.GaugeVec
	patchRatio       *prometheus.GaugeVec
	timeToPatch      *prometheus.GaugeVec

	// Meta metrics
	lastCollectionTime prometheus.Gauge
	collectionErrors   prometheus.Counter
	releasesTotal      prometheus.Gauge
	epicsTotal         prometheus.Gauge
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(cfg *config.PrometheusConfig, service *Service) *PrometheusExporter {
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "roadmap"
	}

	registry := prometheus.NewRegistry()

	e := &PrometheusExporter{
		config:   cfg,
		service:  service,
		logger:   logger.WithComponent("prometheus-exporter"),
		registry: registry,

		releaseFrequency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "dora",
				Name:      "release_frequency",
				Help:      "Number of releases per month",
			},
			[]string{"component"},
		),

		leadTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "dora",
				Name:      "lead_time_days",
				Help:      "Lead time from epic creation to release in days",
			},
			[]string{"component"},
		),

		cycleTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "dora",
				Name:      "cycle_time_days",
				Help:      "Cycle time from epic in-progress to done in days",
			},
			[]string{"component"},
		),

		patchRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "dora",
				Name:      "patch_ratio",
				Help:      "Ratio of patch releases to total releases",
			},
			[]string{"component"},
		),

		timeToPatch: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "dora",
				Name:      "time_to_patch_days",
				Help:      "Time from bug report to patch release in days",
			},
			[]string{"component"},
		),

		lastCollectionTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "collector",
				Name:      "last_collection_timestamp",
				Help:      "Unix timestamp of last successful data collection",
			},
		),

		collectionErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "collector",
				Name:      "errors_total",
				Help:      "Total number of collection errors",
			},
		),

		releasesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "collector",
				Name:      "releases_total",
				Help:      "Total number of releases in cache",
			},
		),

		epicsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "collector",
				Name:      "epics_total",
				Help:      "Total number of epics in cache",
			},
		),
	}

	// Register metrics
	registry.MustRegister(
		e.releaseFrequency,
		e.leadTime,
		e.cycleTime,
		e.patchRatio,
		e.timeToPatch,
		e.lastCollectionTime,
		e.collectionErrors,
		e.releasesTotal,
		e.epicsTotal,
	)

	return e
}

// Handler returns the HTTP handler for the /metrics endpoint
func (e *PrometheusExporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// GinHandler returns a Gin handler for the /metrics endpoint
func (e *PrometheusExporter) GinHandler() gin.HandlerFunc {
	h := e.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Update refreshes all Prometheus metrics from calculated values
func (e *PrometheusExporter) Update(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update collector meta metrics
	collector := e.service.Collector()
	if collector != nil {
		lastCollected := collector.LastCollected()
		if !lastCollected.IsZero() {
			e.lastCollectionTime.Set(float64(lastCollected.Unix()))
		}
		e.releasesTotal.Set(float64(collector.ReleaseCount()))
		e.epicsTotal.Set(float64(collector.EpicCount()))
	}

	// Calculate and update each metric
	filters := models.MetricFilters{}

	// Update release frequency
	if results, err := e.service.CalculateMetric(ctx, "release_frequency", filters, models.TimeRange{}); err == nil {
		for _, r := range results {
			component := r.Labels["component"]
			if component == "" {
				component = "unknown"
			}
			e.releaseFrequency.With(prometheus.Labels{"component": component}).Set(r.Value)
		}
	} else {
		e.logger.Warn("Failed to calculate release_frequency", zap.Error(err))
		e.collectionErrors.Inc()
	}

	// Update lead time
	if results, err := e.service.CalculateMetric(ctx, "lead_time_to_release", filters, models.TimeRange{}); err == nil {
		for _, r := range results {
			component := r.Labels["component"]
			if component == "" {
				component = "unknown"
			}
			e.leadTime.With(prometheus.Labels{"component": component}).Set(r.Value)
		}
	} else {
		e.logger.Warn("Failed to calculate lead_time_to_release", zap.Error(err))
		e.collectionErrors.Inc()
	}

	// Update cycle time
	if results, err := e.service.CalculateMetric(ctx, "cycle_time", filters, models.TimeRange{}); err == nil {
		for _, r := range results {
			component := r.Labels["component"]
			if component == "" {
				component = "unknown"
			}
			e.cycleTime.With(prometheus.Labels{"component": component}).Set(r.Value)
		}
	} else {
		e.logger.Warn("Failed to calculate cycle_time", zap.Error(err))
		e.collectionErrors.Inc()
	}

	// Update patch ratio
	if results, err := e.service.CalculateMetric(ctx, "patch_ratio", filters, models.TimeRange{}); err == nil {
		for _, r := range results {
			component := r.Labels["component"]
			if component == "" {
				component = "unknown"
			}
			e.patchRatio.With(prometheus.Labels{"component": component}).Set(r.Value)
		}
	} else {
		e.logger.Warn("Failed to calculate patch_ratio", zap.Error(err))
		e.collectionErrors.Inc()
	}

	// Update time to patch
	if results, err := e.service.CalculateMetric(ctx, "time_to_patch", filters, models.TimeRange{}); err == nil {
		for _, r := range results {
			component := r.Labels["component"]
			if component == "" {
				component = "unknown"
			}
			e.timeToPatch.With(prometheus.Labels{"component": component}).Set(r.Value)
		}
	} else {
		e.logger.Warn("Failed to calculate time_to_patch", zap.Error(err))
		e.collectionErrors.Inc()
	}

	e.logger.Debug("Prometheus metrics updated")
	return nil
}

// StartUpdater starts a background goroutine that periodically updates Prometheus metrics
func (e *PrometheusExporter) StartUpdater(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial update
	if err := e.Update(ctx); err != nil {
		e.logger.Error("Initial Prometheus metrics update failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Prometheus updater stopping")
			return
		case <-ticker.C:
			if err := e.Update(ctx); err != nil {
				e.logger.Error("Prometheus metrics update failed", zap.Error(err))
			}
		}
	}
}
