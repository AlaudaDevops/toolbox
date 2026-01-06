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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/config"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/jira"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/metrics/models"
	baseModels "github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"go.uber.org/zap"
)

// Collector handles periodic data collection from Jira
type Collector struct {
	jiraClient *jira.Client
	config     *config.Metrics
	logger     *zap.Logger

	mu            sync.RWMutex
	releases      []models.EnrichedRelease
	epics         []models.EnrichedEpic
	lastCollected time.Time
}

// NewCollector creates a new Jira data collector
func NewCollector(jiraClient *jira.Client, cfg *config.Metrics) *Collector {
	return &Collector{
		jiraClient: jiraClient,
		config:     cfg,
		logger:     logger.WithComponent("metrics-collector"),
		releases:   []models.EnrichedRelease{},
		epics:      []models.EnrichedEpic{},
	}
}

// Start begins periodic data collection
func (c *Collector) Start(ctx context.Context) error {
	// Initial collection
	if err := c.Collect(ctx); err != nil {
		c.logger.Error("Initial collection failed", zap.Error(err))
		// Don't return error - allow service to start even if initial collection fails
	}

	// Parse interval
	interval, err := time.ParseDuration(c.config.CollectionInterval)
	if err != nil {
		interval = 5 * time.Minute
		c.logger.Warn("Invalid collection interval, using default",
			zap.String("configured", c.config.CollectionInterval),
			zap.Duration("default", interval))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.logger.Info("Collector started",
		zap.Duration("interval", interval),
		zap.Int("historical_days", c.config.HistoricalDays))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Collector stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := c.Collect(ctx); err != nil {
				c.logger.Error("Collection failed", zap.Error(err))
			}
		}
	}
}

// Collect fetches data from Jira and caches it
func (c *Collector) Collect(ctx context.Context) error {
	c.logger.Info("Starting data collection")
	startTime := time.Now()

	// Fetch releases (versions) from Jira
	releases, err := c.fetchReleases(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}

	// Fetch epics
	epics, err := c.fetchEpics(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch epics: %w", err)
	}

	// Update cached data
	c.mu.Lock()
	c.releases = releases
	c.epics = epics
	c.lastCollected = time.Now()
	c.mu.Unlock()

	c.logger.Info("Collection complete",
		zap.Int("releases", len(releases)),
		zap.Int("epics", len(epics)),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

// fetchReleases gets release/version data from Jira
func (c *Collector) fetchReleases(ctx context.Context) ([]models.EnrichedRelease, error) {
	// Get project details which includes versions
	project, err := c.jiraClient.GetProjectDetails(ctx, "")
	if err != nil {
		return nil, err
	}

	releases := make([]models.EnrichedRelease, 0, len(project.Versions))
	for _, v := range project.Versions {
		release := c.enrichRelease(v)
		releases = append(releases, release)
	}

	originalCount := len(releases)

	// TODO: optmize this code to pre-construct filters
	if filter := c.config.GetFilter("releases"); filter != nil && filter.Enabled && len(filter.Options) > 0 {
		releases = c.filterReleases(releases, filter.Options)
		c.logger.Debug("Filtered releases")
	}

	c.logger.Debug("Fetched releases", zap.Int("count", len(releases)), zap.Int("original", originalCount))
	return releases, nil
}

// enrichRelease converts a basic Version to an EnrichedRelease
func (c *Collector) enrichRelease(v baseModels.Version) models.EnrichedRelease {
	release := models.EnrichedRelease{
		ID:       v.ID,
		Name:     v.Name,
		Released: v.Released,
		Archived: v.Archieved,
	}

	// Parse release date
	if v.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", v.ReleaseDate); err == nil {
			release.ReleaseDate = t
		}
	}

	// Extract component and version parts from name
	// Expected format: component-X.Y.Z or component-vX.Y.Z
	release.Component, release.Major, release.Minor, release.Patch = parseVersionName(v.Name)
	release.Type = classifyReleaseType(release.Major, release.Minor, release.Patch, v.Name)

	return release
}

// parseVersionName extracts component and version parts from a version name
// Supports formats: component-X.Y.Z, component-vX.Y.Z, component X.Y.Z
func parseVersionName(name string) (component string, major, minor, patch int) {
	// Common patterns:
	// argo-cd-2.9.0, argo-cd-v2.9.0, argo-cd 2.9.0
	// component-X.Y.Z-suffix (e.g., -rc1, -beta)

	// Try to match component-vX.Y.Z or component-X.Y.Z
	versionRegex := regexp.MustCompile(`^(.+?)[-\s]v?(\d+)\.(\d+)\.(\d+)`)
	matches := versionRegex.FindStringSubmatch(name)
	if len(matches) >= 5 {
		component = matches[1]
		major, _ = strconv.Atoi(matches[2])
		minor, _ = strconv.Atoi(matches[3])
		patch, _ = strconv.Atoi(matches[4])
		return
	}

	// If no version pattern found, the whole name is the component
	component = name
	return
}

// classifyReleaseType determines if a release is major, minor, or patch
func classifyReleaseType(major, minor, patch int, name string) string {
	// Check for pre-release indicators
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "-rc") ||
		strings.Contains(lowerName, "-alpha") ||
		strings.Contains(lowerName, "-beta") {
		return "prerelease"
	}

	// If we couldn't parse version numbers, default to minor
	if major == 0 && minor == 0 && patch == 0 {
		return "unknown"
	}

	// Classify based on version numbers
	if patch > 0 && minor == 0 && major == 0 {
		return "patch"
	}
	if patch > 0 {
		return "patch"
	}
	if minor > 0 {
		return "minor"
	}
	return "major"
}

// fetchEpics gets epic data from Jira
func (c *Collector) fetchEpics(ctx context.Context) ([]models.EnrichedEpic, error) {
	// Use the existing GetEpicsWithFilter method
	rawEpics, err := c.jiraClient.GetEpicsWithFilter(ctx, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// Build a map of version names to release dates
	c.mu.RLock()
	versionDates := make(map[string]time.Time)
	for _, r := range c.releases {
		if !r.ReleaseDate.IsZero() {
			versionDates[r.Name] = r.ReleaseDate
		}
	}
	c.mu.RUnlock()

	epics := make([]models.EnrichedEpic, 0, len(rawEpics))
	for _, epic := range rawEpics {
		enriched := models.EnrichedEpic{
			ID:         epic.ID,
			Key:        epic.Key,
			Name:       epic.Name,
			Components: epic.Components,
			Versions:   epic.Versions,
			Status:     epic.Status,
			Priority:   epic.Priority,
			IssueType:  "Epic",
		}

		// Find the earliest release date from versions
		for _, versionName := range epic.Versions {
			if releaseDate, ok := versionDates[versionName]; ok {
				if enriched.ReleaseDate.IsZero() || releaseDate.Before(enriched.ReleaseDate) {
					enriched.ReleaseDate = releaseDate
				}
			}
		}

		epics = append(epics, enriched)
	}

	c.logger.Debug("Fetched epics", zap.Int("count", len(epics)))
	return epics, nil
}

// GetData returns the current cached data for metric calculation
func (c *Collector) GetData() (*models.CalculationContext, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Copy the data to avoid race conditions
	releases := make([]models.EnrichedRelease, len(c.releases))
	copy(releases, c.releases)


	epics := make([]models.EnrichedEpic, len(c.epics))
	copy(epics, c.epics)

	return &models.CalculationContext{
		Releases: releases,
		Epics:    epics,
		TimeRange: models.TimeRange{
			Start: time.Now().AddDate(0, 0, -c.config.HistoricalDays),
			End:   time.Now(),
		},
	}, nil
}

// LastCollected returns the time of the last successful collection
func (c *Collector) LastCollected() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastCollected
}

// ReleaseCount returns the number of cached releases
func (c *Collector) ReleaseCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.releases)
}

// EpicCount returns the number of cached epics
func (c *Collector) EpicCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.epics)
}

func (c *Collector) filterReleases(releases []models.EnrichedRelease, options map[string]interface{}) (filtered []models.EnrichedRelease) {
	filtered = make([]models.EnrichedRelease, 0, len(releases))
	filters := make([]func(models.EnrichedRelease) bool, 0, len(options))
	filteredOut := make([]string, 0, len(releases))
	if v, ok := options["name_regex"]; ok {
		logger.Debugf("Will filter release by name regex: %s", v)
		regexString := v.(string)
		regex := regexp.MustCompile(regexString)
		filters = append(filters, func(r models.EnrichedRelease) bool {

			return regex.MatchString(r.Name)
		})
	}
	for _, r := range releases {
		shouldAppend := true
		for _, f := range filters {
			if !f(r) {
				shouldAppend = false
				break
			}
		}
		if shouldAppend {
			filtered = append(filtered, r)
		} else {
			filteredOut = append(filteredOut, r.Name)
		}
	}
	logger.Debugf("Filtered out %d releases: %v", len(filteredOut), filteredOut)
	return
}
