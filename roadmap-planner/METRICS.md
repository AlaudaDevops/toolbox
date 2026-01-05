# Metrics System Implementation Plan

## Overview

Add a pluggable metrics system to the roadmap-planner that calculates adapted DORA metrics from Jira data, exposes them via REST API, and integrates with Prometheus for storage/alerting.

## User Requirements

- **Data Source**: Jira only (Releases/Versions with release dates, Epics)
- **Storage**: Prometheus (external time-series DB)
- **Display**: REST API only (JSON) - no frontend dashboard
- **Granularity**: Both plugin/component AND team/pillar level
- **Flexibility**: Easy to change metric calculations in the future

## Metrics to Implement

| Metric | Description | Data Source |
|--------|-------------|-------------|
| Release Frequency | Releases published per month | Jira Versions (Released=true) |
| Lead Time to Release | Epic creation to release date | Epic created date + Version releaseDate |
| Cycle Time | Epic "In Progress" to "Done" | Epic status changes (via changelog) |
| Patch Ratio | Patch releases / total releases | Version naming pattern analysis |
| Time to Patch | Bug report to patch release | Bug/Vulnerability created + patch Version releaseDate |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Jira Server                              │
│         (Versions, Epics, Issues, Changelogs)               │
└──────────────────────┬──────────────────────────────────────┘
                       │ Fetch on interval
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Metrics Collector                          │
│    (Background worker, caches enriched Jira data)           │
└──────────────────────┬──────────────────────────────────────┘
                       │ Provides data
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Metrics Service                            │
│         (Registry of pluggable calculators)                 │
└────────────┬─────────────────────────────┬──────────────────┘
             │                             │
             ▼                             ▼
┌────────────────────────┐    ┌───────────────────────────────┐
│  REST API Handlers     │    │  Prometheus Exporter          │
│  /api/metrics/*        │    │  /metrics                     │
│  (JSON responses)      │    │  (Prometheus format)          │
└────────────────────────┘    └───────────────────────────────┘
```

## Package Structure

```
backend/internal/
├── metrics/
│   ├── service.go           # MetricsService - orchestrates everything
│   ├── registry.go          # Calculator registry (pluggable)
│   ├── collector.go         # Jira data collector (background worker)
│   ├── prometheus.go        # Prometheus exporter
│   ├── calculators/
│   │   ├── interface.go     # MetricCalculator interface
│   │   ├── release_frequency.go
│   │   ├── lead_time.go
│   │   ├── cycle_time.go
│   │   ├── patch_ratio.go
│   │   └── time_to_patch.go
│   └── models/
│       └── metrics.go       # MetricResult, EnrichedRelease, EnrichedEpic
├── api/handlers/
│   └── metrics.go           # REST API handlers (NEW)
└── config/
    └── config.go            # Extended with Metrics config
```

## Key Interface (Pluggable Calculators)

```go
// MetricCalculator - implement this to add new metrics
type MetricCalculator interface {
    Name() string
    Description() string
    Calculate(ctx context.Context, data *CalculationContext) ([]MetricResult, error)
    PrometheusMetrics() []PrometheusMetricDesc
}

// Register new calculators in service.go:
registry.Register(NewReleaseFrequencyCalculator(cfg))
registry.Register(NewLeadTimeCalculator(cfg))
// To add a new metric: implement interface + register
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/metrics` | List available metrics with descriptions |
| GET | `/api/metrics/:name` | Get specific metric (supports filters) |
| GET | `/api/metrics/summary` | Get all metrics aggregated |
| GET | `/metrics` | Prometheus scrape endpoint (no auth) |

**Query Parameters**:
- `component`: Filter by plugin/component name
- `pillar`: Filter by pillar/team ID
- `from`, `to`: Time range (ISO 8601)

**Example Response** (`GET /api/metrics/release_frequency?component=argo-cd`):
```json
{
  "name": "release_frequency",
  "value": 2.5,
  "unit": "releases/month",
  "labels": {"component": "argo-cd"},
  "breakdown": [
    {"dimension": "quarter", "key": "2025Q3", "value": 3.0},
    {"dimension": "quarter", "key": "2025Q4", "value": 2.0}
  ],
  "metadata": {
    "total_releases": 15,
    "major_releases": 3,
    "patch_releases": 4
  }
}
```

## Configuration Schema

```yaml
# Addition to config.yaml
metrics:
  enabled: true
  collection_interval: "5m"      # How often to fetch from Jira
  historical_days: 365           # How far back to query

  prometheus:
    enabled: true
    path: "/metrics"
    namespace: "roadmap"         # Metric prefix: roadmap_dora_*

  calculators:
    - name: "release_frequency"
      enabled: true
      options:
        window_months: 3         # Rolling window for calculation

    - name: "lead_time_to_release"
      enabled: true
      options:
        percentile: 50           # Use median (P50)

    - name: "cycle_time"
      enabled: true
      options:
        in_progress_statuses: ["In Progress", "In Development"]
        done_statuses: ["Done", "Closed", "Released"]

    - name: "patch_ratio"
      enabled: true
      options:
        patch_pattern: "^(.+)-(\\d+\\.\\d+\\.\\d+)$"  # component-X.Y.Z

    - name: "time_to_patch"
      enabled: true
      options:
        bug_types: ["Bug", "Vulnerability", "Security"]
```

## Files to Create

| File | Purpose |
|------|---------|
| `backend/internal/metrics/service.go` | Main service, lifecycle management |
| `backend/internal/metrics/registry.go` | Calculator registry |
| `backend/internal/metrics/collector.go` | Jira data fetcher (background) |
| `backend/internal/metrics/prometheus.go` | Prometheus exporter |
| `backend/internal/metrics/models/metrics.go` | Data models |
| `backend/internal/metrics/calculators/interface.go` | Calculator interface |
| `backend/internal/metrics/calculators/release_frequency.go` | Release frequency metric |
| `backend/internal/metrics/calculators/lead_time.go` | Lead time metric |
| `backend/internal/metrics/calculators/cycle_time.go` | Cycle time metric |
| `backend/internal/metrics/calculators/patch_ratio.go` | Patch ratio metric |
| `backend/internal/metrics/calculators/time_to_patch.go` | Time to patch metric |
| `backend/internal/api/handlers/metrics.go` | REST API handlers |

## Files to Modify

| File | Changes |
|------|---------|
| `backend/internal/config/config.go` | Add `Metrics` config struct |
| `backend/internal/api/routes.go` | Add metrics routes |
| `backend/internal/jira/client.go` | Add `GetIssueChangelog()` method for cycle time |
| `backend/cmd/server/main.go` | Start metrics collector background worker |
| `backend/go.mod` | Add `prometheus/client_golang` dependency |

## Implementation Phases

### Phase 1: Core Infrastructure
1. Add metrics config to `config.go`
2. Create `metrics/models/metrics.go` with data structures
3. Create `metrics/calculators/interface.go` with `MetricCalculator` interface
4. Create `metrics/registry.go` for calculator registration
5. Create `metrics/service.go` to orchestrate

### Phase 2: Data Collection
1. Create `metrics/collector.go` - background worker
2. Extend `jira/client.go` with `GetIssueChangelog()` for status history
3. Create enriched data models (`EnrichedRelease`, `EnrichedEpic`)
4. Integrate collector startup in `main.go`

### Phase 3: Metric Calculators
1. Implement `release_frequency.go`
2. Implement `lead_time.go`
3. Implement `cycle_time.go`
4. Implement `patch_ratio.go`
5. Implement `time_to_patch.go`

### Phase 4: API & Prometheus
1. Create `api/handlers/metrics.go`
2. Add routes to `routes.go`
3. Create `metrics/prometheus.go`
4. Add `/metrics` endpoint (public, no auth)

### Phase 5: Testing
1. Unit tests for each calculator
2. Integration tests for API endpoints
3. Test Prometheus scraping

## Key Existing Patterns to Follow

- **Handler pattern**: Follow `handlers/roadmap.go` structure
- **Config pattern**: Extend `Config` struct with `Metrics Metrics`
- **Jira client**: Methods return `(result, error)`, use `c.handleError()`
- **Logging**: Use `logger.WithComponent("metrics")`
- **Routes**: Add to protected group for API, public for `/metrics`

## Existing Code to Leverage

- `models.Version` already has `ReleaseDate` field (line 162-168)
- `jira.Client.GetProjectDetails()` returns versions (line 551-561)
- `ConvertJiraVersionToVersion()` handles conversion (line 260-277)
- Version naming already contains component prefix (line 541-547)

## Dependencies to Add

```
github.com/prometheus/client_golang v1.19.0
```
