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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/api"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/calculators"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerCfg := &logger.Config{
		Level:       cfg.Logger.Level,
		Development: cfg.Logger.Development || cfg.Debug,
		Encoding:    cfg.Logger.Encoding,
	}

	if err := logger.Initialize(loggerCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Ensure logger is synced on exit
	defer func() {
		_ = logger.Sync()
	}()

	// Set gin mode
	if cfg.Debug || cfg.Logger.Development {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := api.NewRouter(cfg)

	// Create context for background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize metrics system if enabled
	var metricsService *metrics.Service
	_ = metricsService // Silence unused warning when metrics disabled

	if cfg.Metrics.Enabled {
		logger.Info("Initializing metrics system")

		// Create Jira client for metrics collector using config credentials
		if cfg.Jira.BaseURL != "" && cfg.Jira.Username != "" && cfg.Jira.Password != "" {
			jiraClient, err := jira.NewClient(
				cfg.Jira.BaseURL,
				cfg.Jira.Username,
				cfg.Jira.Password,
				cfg.Jira.Project,
			)
			if err != nil {
				logger.Error("Failed to create Jira client for metrics", zap.Error(err))
			} else {
				// Create collector and service
				collector := metrics.NewCollector(jiraClient, &cfg.Metrics)
				metricsService = metrics.NewService(&cfg.Metrics, collector)

				// Register calculators
				registerCalculators(metricsService, &cfg.Metrics)

				// Start collector in background
				go func() {
					if err := collector.Start(ctx); err != nil && err != context.Canceled {
						logger.Error("Metrics collector stopped with error", zap.Error(err))
					}
				}()

				// Add metrics API routes
				api.AddMetricsRoutes(router, cfg, metricsService)
				logger.Info("Metrics API routes added")

				// Initialize Prometheus exporter if enabled
				if cfg.Metrics.Prometheus.Enabled {
					prometheusExporter := metrics.NewPrometheusExporter(&cfg.Metrics.Prometheus, metricsService)

					// Add Prometheus endpoint (no auth required)
					prometheusPath := cfg.Metrics.Prometheus.Path
					if prometheusPath == "" {
						prometheusPath = "/metrics"
					}
					router.GET(prometheusPath, prometheusExporter.GinHandler())
					logger.Info("Prometheus metrics endpoint added", zap.String("path", prometheusPath))

					// Start Prometheus updater
					go prometheusExporter.StartUpdater(ctx, 1*time.Minute)
				}
			}
		} else {
			logger.Warn("Metrics enabled but Jira credentials not configured in config file")
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Cancel background workers (metrics collector, prometheus updater)
	cancel()

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// registerCalculators registers all metric calculators with the service
func registerCalculators(svc *metrics.Service, cfg *config.Metrics) {
	// Get options for each calculator from config
	getOptions := func(name string) map[string]interface{} {
		for _, calc := range cfg.Calculators {
			if calc.Name == name {
				return calc.Options
			}
		}
		return nil
	}

	// Register release frequency calculator
	if err := svc.RegisterCalculator(calculators.NewReleaseFrequencyCalculator(getOptions("release_frequency"))); err != nil {
		logger.Warn("Failed to register release_frequency calculator", zap.Error(err))
	}

	// Register lead time calculator
	if err := svc.RegisterCalculator(calculators.NewLeadTimeCalculator(getOptions("lead_time_to_release"))); err != nil {
		logger.Warn("Failed to register lead_time_to_release calculator", zap.Error(err))
	}

	// Register cycle time calculator
	if err := svc.RegisterCalculator(calculators.NewCycleTimeCalculator(getOptions("cycle_time"))); err != nil {
		logger.Warn("Failed to register cycle_time calculator", zap.Error(err))
	}

	// Register patch ratio calculator
	if err := svc.RegisterCalculator(calculators.NewPatchRatioCalculator(getOptions("patch_ratio"))); err != nil {
		logger.Warn("Failed to register patch_ratio calculator", zap.Error(err))
	}

	// Register time to patch calculator
	if err := svc.RegisterCalculator(calculators.NewTimeToPatchCalculator(getOptions("time_to_patch"))); err != nil {
		logger.Warn("Failed to register time_to_patch calculator", zap.Error(err))
	}

	logger.Info("Metric calculators registered", zap.Int("count", svc.Registry().Count()))
}
