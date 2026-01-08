# Metrics System Documentation

## Overview

This document describes the metrics system for the roadmap-planner, which calculates adapted DORA metrics from Jira data. Since our team delivers plugins to customers who deploy them in their own infrastructure (no production access), we've adapted traditional DORA metrics to measure what we can observe: our release and development processes.

## Implementation Status

### Phase 1: Core Infrastructure - **COMPLETED**
- [x] Add metrics config to `config.go`
- [x] Create `metrics/models/metrics.go` with data structures
- [x] Create `metrics/calculators/interface.go` with `MetricCalculator` interface
- [x] Create `metrics/registry.go` for calculator registration
- [x] Create `metrics/service.go` to orchestrate

### Phase 2: Data Collection - **COMPLETED**
- [x] Create `metrics/collector.go` - background worker
- [x] Extend `jira/client.go` with `GetIssueChangelog()` for status history
- [x] Create enriched data models (`EnrichedRelease`, `EnrichedEpic`)
- [x] Integrate collector startup in `main.go`
- [x] Add filter support for releases (`filterReleases` with regex)

### Phase 3: Metric Calculators - **COMPLETED**
- [x] Implement `release_frequency.go`
- [x] Implement `lead_time.go`
- [x] Implement `cycle_time.go`
- [x] Implement `patch_ratio.go`
- [x] Implement `time_to_patch.go`

### Phase 4: API & Prometheus - **COMPLETED**
- [x] Create `api/handlers/metrics.go`
- [x] Add routes to `routes.go`
- [x] Create `metrics/prometheus.go`
- [x] Add `/metrics` endpoint (public, no auth)

### Phase 5: Testing - **PENDING**
- [ ] Unit tests for each calculator
- [ ] Integration tests for API endpoints
- [ ] Test Prometheus scraping

---

## Metrics Explained

### 1. Release Frequency

#### Rationale

**Why this metric matters:**
Release Frequency is a proxy for DORA's "Deployment Frequency". Since customers deploy our plugins themselves, we measure how often we *publish* releases rather than how often they're deployed. High release frequency indicates:
- Smaller, incremental changes (lower risk)
- Faster feedback loops
- Continuous delivery capability
- Team velocity and health

**What it tells you:**
- How often each component/plugin receives updates
- Whether release cadence matches expectations (monthly feature releases, bi-monthly patches)
- Components that may be stagnating or receiving too many releases

#### Calculation

**Formula:**
```
Release Frequency = Count of Released Versions / Window Period (months)
```

**Algorithm:**
1. Filter Jira Versions where `Released = true`
2. Filter by time range (`releaseDate` within window)
3. Group by component (extracted from version name pattern)
4. Count releases in the rolling window (default: 3 months)
5. Divide by window months to get releases/month

**Example:**
- Component `argo-cd` has 6 releases in the last 3 months
- Release Frequency = 6 / 3 = **2.0 releases/month**

**Breakdown provided:**
- Total releases count
- Major releases count (X.0.0)
- Minor releases count (X.Y.0)
- Patch releases count (X.Y.Z where Z > 0)

#### Configuration Options

```yaml
calculators:
  - name: "release_frequency"
    enabled: true
    options:
      window_months: 3  # Rolling window for calculation
```

---

### 2. Lead Time to Release

#### Rationale

**Why this metric matters:**
Lead Time measures the time from when work is identified (Epic created) to when it's available to customers (release published). This is our adaptation of DORA's "Lead Time for Changes". It indicates:
- How quickly we can deliver value
- Process efficiency from ideation to delivery
- Predictability for stakeholders

**What it tells you:**
- Average time from Epic creation to release
- Whether delivery is speeding up or slowing down
- Components with delivery bottlenecks

#### Calculation

**Formula:**
```
Lead Time = Release Date - Epic Created Date (in days)
```

**Algorithm:**
1. For each Epic with a Fix Version that has been released:
   - Get Epic's `created` date from Jira
   - Get the `releaseDate` from the linked Fix Version
   - Calculate difference in days
2. Group by component (from Epic's component field)
3. Calculate the Pth percentile (default: P50/median)
4. Also calculate min, max, and average for context

**Example:**
- Epic "Add SSO support" created on Jan 1
- Released in version `argo-cd-2.9.0` on Jan 45
- Lead Time = **45 days**

**Why percentile instead of average:**
Outliers (like long-running features) can skew averages. The median (P50) gives a more representative "typical" lead time.

#### Configuration Options

```yaml
calculators:
  - name: "lead_time_to_release"
    enabled: true
    options:
      percentile: 50  # 50 = median, 90 = P90
```

---

### 3. Cycle Time

#### Rationale

**Why this metric matters:**
Cycle Time measures actual development time - from when work starts ("In Progress") to when it's complete ("Done"). This differs from Lead Time which includes waiting time before work begins. Cycle Time indicates:
- Development efficiency
- Team capacity utilization
- Work item complexity patterns

**What it tells you:**
- How long active development takes
- Whether work items are appropriately sized
- Process bottlenecks during development

#### Calculation

**Formula:**
```
Cycle Time = Done Date - In Progress Date (in days)
```

**Algorithm:**
1. For each resolved Epic:
   - Fetch issue changelog via Jira API
   - Find first transition TO an "In Progress" status
   - Find last transition TO a "Done" status
   - Calculate difference in days
2. Fallback: If no status changes found, use `created` → `resolved` dates
3. Group by component
4. Calculate the Pth percentile (default: P50)

**Example:**
- Epic moved to "In Progress" on Feb 1
- Epic moved to "Done" on Feb 15
- Cycle Time = **14 days**

#### Configuration Options

```yaml
calculators:
  - name: "cycle_time"
    enabled: true
    options:
      percentile: 50
      in_progress_statuses:
        - "In Progress"
        - "In Development"
        - "开发中"  # Chinese locale
      done_statuses:
        - "Done"
        - "Closed"
        - "Released"
        - "已完成"  # Chinese locale
      # Optional: Regex pattern to only calculate matching versions
      component_pattern: '^[a-zA-Z0-9\-]+\-[v]*\d+.\d+.\d+$'
```

---

### 4. Patch Ratio

#### Rationale

**Why this metric matters:**
Patch Ratio is our proxy for DORA's "Change Failure Rate". Since we can't measure production failures directly, we measure how often we need to release patches (bug fixes) relative to feature releases. A high patch ratio may indicate:
- Quality issues in feature releases
- Inadequate testing
- Technical debt accumulation

**What it tells you:**
- Quality of initial releases
- Whether components need more testing investment
- Release stability trends

**Interpretation:**
- 0.0-0.2: Excellent - Few patches needed
- 0.2-0.4: Good - Normal maintenance level
- 0.4-0.6: Concerning - Many bug fixes
- 0.6+: Critical - More patches than features

#### Calculation

**Formula:**
```
Patch Ratio = Patch Releases / Total Releases
```

**Algorithm:**
1. Filter released versions within the time window
2. Classify each version by type:
   - **Major**: Version X.0.0 (or first number changes)
   - **Minor**: Version X.Y.0 (second number changes)
   - **Patch**: Version X.Y.Z where Z > 0 (third number changes)
3. Group by component
4. Calculate ratio: patches / total

**Example:**
- Component has 10 total releases
- 3 major, 4 minor, 3 patches
- Patch Ratio = 3 / 10 = **0.30**

#### Configuration Options

```yaml
calculators:
  - name: "patch_ratio"
    enabled: true
    options:
      window_months: 6  # Longer window for ratio stability
```

---

### 5. Time to Patch

#### Rationale

**Why this metric matters:**
Time to Patch is our proxy for DORA's "Mean Time to Recovery" (MTTR). It measures how quickly we can fix bugs and vulnerabilities once reported. This is critical for:
- Security response capability
- Customer trust
- Compliance requirements (SLAs for security fixes)

**What it tells you:**
- How quickly the team responds to issues
- Whether security vulnerabilities are prioritized
- Capacity to handle urgent fixes

#### Calculation

**Formula:**
```
Time to Patch = Patch Release Date - Bug Created Date (in days)
```

**Algorithm:**
1. Filter issues by type (Bug, Vulnerability, Security)
2. For each bug/vulnerability with a Fix Version that's released:
   - Get issue `created` date
   - Get `releaseDate` from linked Fix Version
   - Calculate difference in days
3. Group by component
4. Calculate the Pth percentile (default: P50)

**Example:**
- Security vulnerability reported on Mar 1
- Patch version `argo-cd-2.9.1` released on Mar 5
- Time to Patch = **4 days**

#### Configuration Options

```yaml
calculators:
  - name: "time_to_patch"
    enabled: true
    options:
      percentile: 50
      bug_types:
        - "Bug"
        - "Vulnerability"
        - "Security"
        - "缺陷"  # Chinese locale
```

---

## Data Requirements

For metrics to calculate correctly, your Jira data must meet these requirements:

### For Release Frequency

| Requirement | Field | How to Verify |
|-------------|-------|---------------|
| Versions exist | Project → Versions | Check Jira project has versions defined |
| Released flag set | Version.Released = true | Ensure released versions are marked as "Released" |
| Release date populated | Version.ReleaseDate | **CRITICAL**: Set release date on all released versions |
| Semantic versioning | Version.Name | Format: `component-X.Y.Z` (e.g., `argo-cd-2.9.0`) |

**Common Issues:**
- ❌ Versions without release dates are excluded from calculations
- ❌ Versions not marked as "Released" are ignored
- ❌ Version names without semver pattern cannot determine major/minor/patch

**Fix in Jira:**
1. Go to Project Settings → Versions
2. For each version: set Release Date and mark as Released
3. Use consistent naming: `{component}-{major}.{minor}.{patch}`

### For Lead Time to Release

| Requirement | Field | How to Verify |
|-------------|-------|---------------|
| Epics have Fix Version | Epic.FixVersion | Link Epics to versions they're included in |
| Fix Version is released | Version.Released = true | Version must be released with date |
| Epic has created date | Issue.Created | Automatic in Jira |
| Epic has component | Epic.Component | Assign component(s) to Epics |

**Common Issues:**
- ❌ Epics without Fix Version won't have a release date
- ❌ Epics without components grouped as "unknown"

**Fix in Jira:**
1. Before releasing, ensure all included Epics have Fix Version set
2. Assign components to Epics when creating them

### For Cycle Time

| Requirement | Field | How to Verify |
|-------------|-------|---------------|
| Status transitions recorded | Issue.Changelog | Automatic in Jira |
| Epics are resolved | Resolution != null | Close/resolve completed Epics |
| Use standard statuses | Status names match config | Configure matching status names |

**Common Issues:**
- ❌ Custom status names not in config → cycle time may use fallback dates
- ❌ Epics not resolved → excluded from calculation

**Fix in Jira:**
1. Ensure workflow uses recognizable status names
2. Configure `in_progress_statuses` and `done_statuses` in config
3. Resolve/close Epics when complete

### For Patch Ratio

| Requirement | Field | How to Verify |
|-------------|-------|---------------|
| Semantic versioning | Version.Name | Format: `component-X.Y.Z` |
| All releases have dates | Version.ReleaseDate | Every version needs release date |

**Common Issues:**
- ❌ Non-semver names → cannot classify as major/minor/patch
- ❌ Missing patch versions → ratio will be artificially low

**Version Name Patterns Recognized:**
- `argo-cd-2.9.0` → Component: `argo-cd`, Version: 2.9.0 (minor)
- `argo-cd-v2.9.1` → Component: `argo-cd`, Version: 2.9.1 (patch)
- `argo-cd 2.10.0` → Component: `argo-cd`, Version: 2.10.0 (minor)

### For Time to Patch

| Requirement | Field | How to Verify |
|-------------|-------|---------------|
| Issue type matches config | Issue.Type | Bugs use "Bug", "Vulnerability", etc. |
| Bugs have Fix Version | Bug.FixVersion | Link bugs to patch versions |
| Fix Version released | Version.Released = true | Mark patch versions as released |

**Common Issues:**
- ❌ Bugs without Fix Version → excluded from calculation
- ❌ Custom issue types not in config → excluded

**Fix in Jira:**
1. Configure `bug_types` in config to match your issue types
2. Always link bugs/vulnerabilities to the version that fixes them

---

## Filtering Data

You can filter which releases are included in calculations using the `filters` config:

```yaml
metrics:
  enabled: true
  filters:
    - name: "releases"
      enabled: true
      options:
        name_regex: "^(argo-cd|tekton|harbor)-.*$"  # Only these components
```

This is useful when your Jira project contains versions for multiple products and you only want to measure specific ones.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Jira Server                              │
│         (Versions, Epics, Issues, Changelogs)               │
└──────────────────────┬──────────────────────────────────────┘
                       │ Fetch on interval (default: 5m)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Metrics Collector                          │
│    (Background worker, caches enriched Jira data)           │
│    - Fetches all versions from project                      │
│    - Fetches all Epics (with filters)                       │
│    - Applies release filters (name_regex, etc.)             │
└──────────────────────┬──────────────────────────────────────┘
                       │ Provides CalculationContext
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Metrics Service                            │
│         (Registry of pluggable calculators)                 │
│    - release_frequency                                      │
│    - lead_time_to_release                                   │
│    - cycle_time                                             │
│    - patch_ratio                                            │
│    - time_to_patch                                          │
└────────────┬─────────────────────────────┬──────────────────┘
             │                             │
             ▼                             ▼
┌────────────────────────┐    ┌───────────────────────────────┐
│  REST API Handlers     │    │  Prometheus Exporter          │
│  /api/metrics/*        │    │  /metrics                     │
│  (JSON responses)      │    │  (Prometheus format)          │
│  Requires auth         │    │  No auth required             │
└────────────────────────┘    └───────────────────────────────┘
```

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/metrics` | Yes | List available metrics |
| GET | `/api/metrics/:name` | Yes | Get specific metric (with filters) |
| GET | `/api/metrics/summary` | Yes | Get all metrics aggregated |
| GET | `/api/metrics/status` | Yes | Collector status |
| GET | `/metrics` | No | Prometheus scrape endpoint |

**Query Parameters:**
- `component`: Filter by plugin/component name
- `pillar`: Filter by pillar/team ID
- `from`, `to`: Time range (ISO 8601 or YYYY-MM-DD)

**Example Response** (`GET /api/metrics/release_frequency?component=argo-cd`):
```json
{
  "name": "release_frequency",
  "value": 2.5,
  "unit": "releases/month",
  "labels": {"component": "argo-cd"},
  "timestamp": "2025-01-06T10:00:00Z",
  "metadata": {
    "total_releases": 8,
    "major_releases": 1,
    "minor_releases": 4,
    "patch_releases": 3,
    "window_months": 3
  }
}
```

---

## Configuration Reference

```yaml
metrics:
  enabled: true                    # Enable/disable metrics system
  collection_interval: "5m"        # How often to fetch from Jira
  historical_days: 365             # How far back to query

  prometheus:
    enabled: true
    path: "/metrics"               # Prometheus scrape endpoint
    namespace: "roadmap"           # Metric prefix: roadmap_dora_*

  filters:
    - name: "releases"
      enabled: true
      options:
        name_regex: "^.*$"         # Regex to filter version names

    - name: "issues"
      enabled: true
      options:
        issuetypes: ["Bug"]        # Issue types to include

  calculators:
    - name: "release_frequency"
      enabled: true
      options:
        window_months: 3

    - name: "lead_time_to_release"
      enabled: true
      options:
        percentile: 50

    - name: "cycle_time"
      enabled: true
      options:
        percentile: 50
        in_progress_statuses: ["In Progress", "In Development"]
        done_statuses: ["Done", "Closed", "Released"]

    - name: "patch_ratio"
      enabled: true
      options:
        window_months: 6

    - name: "time_to_patch"
      enabled: true
      options:
        percentile: 50
        bug_types: ["Bug", "Vulnerability", "Security"]
```

---

## Adding New Metrics

To add a new metric calculator:

1. Create a new file in `backend/internal/metrics/calculators/`
2. Implement the `MetricCalculator` interface:

```go
type MetricCalculator interface {
    Name() string
    Description() string
    Unit() string
    AvailableFilters() []string
    Calculate(ctx context.Context, data *CalculationContext) ([]MetricResult, error)
    PrometheusMetrics() []PrometheusMetricDesc
}
```

3. Register in `main.go`:

```go
svc.RegisterCalculator(calculators.NewMyNewCalculator(getOptions("my_new_metric")))
```

4. Add Prometheus gauges in `prometheus.go` if needed

---

## Troubleshooting

### Metrics show zero or no data

1. **Check collector status**: `GET /api/metrics/status`
2. **Verify Jira credentials** are in config (not just headers)
3. **Check release dates** are set on versions
4. **Check Fix Versions** are linked to Epics

### Release frequency not matching expectations

1. **Verify version naming** follows `component-X.Y.Z` pattern
2. **Check filters** aren't excluding too many versions
3. **Verify window period** in config

### Cycle time using fallback dates

1. **Check status names** match config
2. **Verify workflow** has status transitions recorded
3. **Check changelog** is accessible via Jira API

### Time to patch shows no data

1. **Verify issue types** match `bug_types` config
2. **Check bugs have Fix Version** set
3. **Verify Fix Version is released**
