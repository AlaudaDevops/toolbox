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
	"strings"
	"syscall"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/api"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/contributions"
	ghclient "github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/github"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jirasync"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/calculators"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/storage"
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

	if cfg.Metrics.Enabled {
		logger.Info("Initializing metrics system")
		err = initMetrics(ctx, router, cfg)
		if err != nil {
			logger.Error("Failed to initialize metrics system", zap.Error(err))
		}
	}

	// Initialize team-analytics storage + GitHub sync if enabled.
	//
	// We open the store once and let it close on process exit; the
	// GitHub syncer runs on a configurable interval in its own goroutine.
	// Both are no-ops when the corresponding config blocks are off, so
	// existing deployments stay unchanged.
	if cfg.Storage.Enabled {
		if err := initTeamAnalytics(ctx, router, cfg); err != nil {
			logger.Error("Failed to initialize team analytics", zap.Error(err))
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

// initMetrics initializes the metrics system if enabled in config
func initMetrics(ctx context.Context, router *gin.Engine, cfg *config.Config) error {
	if cfg.Jira.BaseURL == "" || cfg.Jira.Username == "" || cfg.Jira.Password == "" {
		logger.Warn("Metrics enabled but Jira credentials not configured in config file")
		return nil
	}

	jiraClient, err := jira.NewClient(
		cfg.Jira.BaseURL,
		cfg.Jira.Username,
		cfg.Jira.Password,
		cfg.Jira.Project,
	)
	if err != nil {
		logger.Error("Failed to create Jira client for metrics", zap.Error(err))
		return err
	}
	// Create collector and service
	collector := metrics.NewCollector(jiraClient, &cfg.Metrics)
	metricsService := metrics.NewService(&cfg.Metrics, collector)

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
	return nil
}

// initTeamAnalytics opens the persistent store, runs migrations, mounts
// the contributions REST endpoints, and (if configured) starts the
// GitHub sync loop. Failures here are non-fatal — the rest of the app
// continues to serve roadmap + metrics.
func initTeamAnalytics(ctx context.Context, router *gin.Engine, cfg *config.Config) error {
	store, err := openStore(cfg.Storage)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info("Team analytics store ready",
		zap.String("type", cfg.Storage.Type),
		zap.Int("backfill_days", cfg.Storage.BackfillDays))

	service := contributions.NewService(store)
	aggregator := contributions.NewAggregator(store)
	api.AddContributionsRoutes(router, store, service, aggregator)
	logger.Info("Contributions API routes added")

	// Optional Jira sync goroutine.
	//
	// Jira credentials are required (we reuse the same Basic auth that
	// the metrics collector uses). On the first run with no
	// `collection_runs` row for source=jira, jirasync.Run does the
	// 3-pass JQL backfill described in PROPOSAL.md §6.5. After each
	// successful run the aggregator rebuilds the rollup window so the
	// dashboards see the new data.
	if cfg.Jira.SyncEnabled {
		if cfg.Jira.BaseURL != "" && cfg.Jira.Username != "" && cfg.Jira.Password != "" {
			jc, err := jira.NewClient(cfg.Jira.BaseURL, cfg.Jira.Username, cfg.Jira.Password, cfg.Jira.Project)
			if err != nil {
				logger.Error("jira sync: failed to build client", zap.Error(err))
			} else {
				backfill := cfg.Jira.BackfillDays
				if backfill <= 0 {
					backfill = cfg.Storage.BackfillDays
				}
				syncer := jirasync.NewSyncer(jc, store, jirasync.Config{
					Project:          cfg.Jira.Project,
					BackfillDays:     backfill,
					StoryPointsField: cfg.Jira.StoryPointsField,
					SprintField:      cfg.Jira.SprintField,
				})
				interval, err := time.ParseDuration(cfg.Jira.SyncInterval)
				if err != nil {
					interval = 30 * time.Minute
				}
				go runJiraSync(ctx, syncer, aggregator, interval)
				logger.Info("Jira sync started", zap.Duration("interval", interval), zap.Int("backfill_days", backfill))
			}
		} else {
			logger.Warn("jira.sync_enabled but credentials missing; skipping")
		}
	}

	// Optional GitHub sync goroutine.
	if cfg.GitHub.Enabled {
		startGitHubSync(ctx, cfg, store, aggregator)
	}

	go func() {
		<-ctx.Done()
		_ = store.Close()
	}()
	return nil
}

// startGitHubSync wires up the GitHub side: client (PAT or App auth),
// Syncer, and the goroutine that drives it on the configured interval.
//
// All failures are non-fatal — if GitHub auth or repo parsing breaks,
// we log and continue without a GitHub syncer rather than refusing to
// start the whole server. The roadmap UI keeps working.
func startGitHubSync(ctx context.Context, cfg *config.Config, store storage.Store, aggregator *contributions.Aggregator) {
	repos := parseRepos(cfg.GitHub.Repos)
	if len(repos) == 0 {
		logger.Warn("github.enabled but github.repos is empty; skipping sync")
		return
	}
	projectKey := cfg.GitHub.ProjectKey
	if projectKey == "" {
		projectKey = cfg.Jira.Project
	}
	client, authMode, err := buildGitHubClient(&cfg.GitHub)
	if err != nil {
		logger.Error("github client init failed", zap.Error(err))
		return
	}
	logger.Info("GitHub auth ready", zap.String("mode", authMode))

	backfill := cfg.GitHub.BackfillDays
	if backfill <= 0 {
		backfill = cfg.Storage.BackfillDays
	}
	syncer := ghclient.NewSyncer(client, store, repos, ghclient.DefaultLinker(projectKey), backfill)

	interval, err := time.ParseDuration(cfg.GitHub.SyncInterval)
	if err != nil {
		interval = 30 * time.Minute
	}
	go runGitHubSync(ctx, syncer, aggregator, interval)
	logger.Info("GitHub sync started",
		zap.Duration("interval", interval),
		zap.Int("repos", len(repos)),
		zap.Int("backfill_days", backfill))
}

// buildGitHubClient picks the auth path. App config wins when present
// (production deployments should run with an App; Token is for local
// dev / one-off scripts).
//
// Returns (client, modeLabel, error) so the caller can log which path
// is in effect.
func buildGitHubClient(cfg *config.GitHub) (*ghclient.Client, string, error) {
	if cfg.App.Configured() {
		ts, err := ghclient.NewAppTokenSource(ghclient.AppCredentials{
			AppID:          cfg.App.AppID,
			InstallationID: cfg.App.InstallationID,
			PrivateKeyPath: cfg.App.PrivateKeyPath,
			PrivateKeyPEM:  []byte(cfg.App.PrivateKeyPEM),
			BaseURL:        cfg.BaseURL,
		})
		if err != nil {
			return nil, "", fmt.Errorf("github app: %w", err)
		}
		return ghclient.NewWithTokenSource(cfg.BaseURL, ts, nil), "app", nil
	}
	if cfg.Token == "" {
		// Allow unauthenticated for public-only smoke tests, but warn:
		// every real install should be authenticated to lift the
		// 60-req/h ceiling.
		logger.Warn("github auth: no token and no app configured — running unauthenticated (60 req/h cap)")
		return ghclient.New(cfg.BaseURL, "", nil), "unauthenticated", nil
	}
	return ghclient.New(cfg.BaseURL, cfg.Token, nil), "pat", nil
}

// openStore selects the right storage backend based on cfg.Type. We
// keep this here in main rather than in the storage package so that
// adding new backends doesn't pull every driver into every binary.
func openStore(cfg config.Storage) (storage.Store, error) {
	t := cfg.Type
	if t == "" {
		t = "sqlite"
	}
	switch t {
	case "sqlite":
		return storage.OpenSQLite(cfg.Path)
	case "postgres", "postgresql", "pg":
		dsn := cfg.DSN
		if dsn == "" {
			dsn = os.Getenv("STORAGE_DSN")
		}
		if dsn == "" {
			return nil, fmt.Errorf("storage.type=postgres but storage.dsn / STORAGE_DSN is empty")
		}
		return storage.OpenPostgres(dsn)
	default:
		return nil, fmt.Errorf("unknown storage.type %q", t)
	}
}

func parseRepos(specs []string) []ghclient.RepoConfig {
	out := make([]ghclient.RepoConfig, 0, len(specs))
	for _, raw := range specs {
		// "owner/name", "owner/name:component", or "owner/*" for
		// wildcard expansion (resolved at Sync time against
		// /orgs/{owner}/repos). Component labels are not allowed on
		// wildcard specs — write `OWNER/NAME:component` separately if
		// you want a per-repo label override.
		spec, comp, _ := strings.Cut(raw, ":")
		owner, name, ok := strings.Cut(spec, "/")
		if !ok || owner == "" || name == "" {
			logger.Warn("malformed github repo spec, skipping", zap.String("spec", raw))
			continue
		}
		if name == "*" && comp != "" {
			logger.Warn("github repo wildcard cannot carry a component label, dropping label",
				zap.String("spec", raw))
			comp = ""
		}
		out = append(out, ghclient.RepoConfig{
			Owner: owner, Name: name, Component: comp,
		})
	}
	return out
}

func runGitHubSync(ctx context.Context, s *ghclient.Syncer, agg *contributions.Aggregator, interval time.Duration) {
	doOne := func(label string) {
		if err := s.Sync(ctx); err != nil {
			logger.Error(label+" GitHub sync failed", zap.Error(err))
			return
		}
		if err := agg.RebuildRecent(ctx, 0); err != nil {
			logger.Warn("aggregator rebuild after GitHub sync failed", zap.Error(err))
		}
	}
	doOne("Initial")
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			doOne("Periodic")
		}
	}
}

// runJiraSync mirrors runGitHubSync: one initial cycle, then a ticker.
// Each successful run triggers an aggregator rebuild so the rollups
// stay in step with the snapshot data.
func runJiraSync(ctx context.Context, s *jirasync.Syncer, agg *contributions.Aggregator, interval time.Duration) {
	doOne := func(label string) {
		res, err := s.Run(ctx)
		if err != nil {
			logger.Error(label+" Jira sync failed", zap.Error(err))
			return
		}
		logger.Info(label+" Jira sync complete",
			zap.String("mode", res.Mode),
			zap.Int("issues", res.IssuesWritten),
			zap.Int("members", res.MembersSeen),
			zap.Int64("duration_ms", res.DurationMs))
		if err := agg.RebuildRecent(ctx, 0); err != nil {
			logger.Warn("aggregator rebuild after Jira sync failed", zap.Error(err))
		}
	}
	doOne("Initial")
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			doOne("Periodic")
		}
	}
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
