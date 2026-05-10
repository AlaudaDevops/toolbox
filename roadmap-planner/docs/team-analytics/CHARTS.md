# Team Analytics — Charts Reference

> **Audience**: anyone reading the roadmap-planner Team Analytics UI.
> Each chart below lists what it shows, where the data comes from, and how to read it.

The Team Analytics feature has three tabs in the top nav (the **Metrics** tab is a separate, older DORA dashboard, also documented here):

- [Dashboard](#dashboard-tab) — team-wide rollups, the entry point
- [Member profile](#member-profile) — drill-in for one person
- [Team overview](#team-overview-tab) — sortable roster + pillar throughput
- [Metrics (DORA)](#metrics-tab-dora) — release-process metrics

---

## Filters that affect everything

Two controls at the top of the **Dashboard** and **Member profile** drive every chart on those pages:

| Filter | Values | Scope |
|---|---|---|
| **Window** | 4 / 12 / 26 weeks | All Dashboard charts; weekly trend on the Member profile |
| **Pillar** | All / each configured pillar / Unassigned | All Dashboard charts |
| **Active metric** | one of the 5 KPI tiles | Highlighted/selected metric in the trend, mix, donut, movers, and ranking charts |

The five metrics that drive the active-metric selector everywhere:

- **PRs merged** — PRs the member authored that landed
- **PRs opened** — PRs the member authored (regardless of outcome)
- **Reviews** — PRs the member reviewed (not authored)
- **Jira done** — Jira issues transitioned to a done state where the member is the assignee
- **Story points** — story points on those done issues

---

## Dashboard tab

Team-wide view. Top of the page shows the window + pillar filters and five KPI tiles. Clicking a KPI tile selects the active metric, which restyles every downstream chart.

### KPI tiles (×5)

- **Type**: KPI cards with up/down/flat delta badge
- **Shows**: total count over the selected window, plus the percentage change between the **prior half** and **recent half** of that window (e.g. for a 12-week window, weeks 1-6 vs. weeks 7-12)
- **Source**: `GET /api/contributions/team`
- **Reading it**: a green up-arrow means the team did more of this metric in the recent half than the prior half. The number on the tile is the total for the window.

### Throughput trend

- **Type**: multi-line chart, one line per metric, full panel width
- **Axes**: X = week (ISO week), Y = count
- **Source**: `GET /api/contributions/team`, weekly buckets
- **Reading it**: the **active metric** is drawn bold; the other four fade to ~18% opacity. Click another KPI tile or a legend chip to change which line is in focus. Useful for spotting a steady decline or a sudden burst on one metric while the others stay flat.

### Pillar mix over time

- **Type**: stacked area chart
- **Axes**: X = week, Y = total count of the active metric, stacked by pillar
- **Source**: `GET /api/contributions/team`, summed per pillar per week
- **Reading it**: each colored band is one pillar's contribution to the active metric that week. A band that grows over time means that pillar took over more of the team's work; a band that shrinks means the opposite. Pillars with zero contribution are not drawn.

### Inflow vs outflow

- **Type**: grouped bar chart, two bars per week
- **Axes**: X = week, Y = count; blue bar = PRs **opened**, red bar = PRs **merged**
- **Source**: `GET /api/contributions/team`
- **Reading it**: this chart **always shows PR inflow vs outflow**, regardless of the active metric. If blue consistently outpaces red, work-in-flight is accumulating. If red ≈ blue, the team is keeping pace with what it opens.

### Distribution by pillar

- **Type**: donut chart with legend table
- **Source**: `GET /api/contributions/team`, active metric summed by pillar
- **Reading it**: center label = total. Each slice is one pillar's share of the active metric. Legend table on the side lists each pillar with its raw count and percentage.

### Top movers

- **Type**: horizontal bar chart, top 8 by absolute delta
- **Axes**: each bar = one team member; bar length = signed delta (recent half − prior half) of the active metric
- **Source**: `GET /api/contributions/team`, member-level weekly buckets
- **Reading it**: bars to the right (positive, metric color) = picked up the pace; bars to the left (negative, muted) = slowed down. Click any bar to drill into that member's profile.

### Per-member ranking

- **Type**: horizontal bar chart, full panel width, sorted descending
- **Axes**: bar = total of the active metric for the member over the selected window
- **Source**: `GET /api/contributions/team`
- **Reading it**: bar color = the member's pillar (so you can see at a glance which pillars dominate the leaderboard). Click any bar to drill into that member's profile.

---

## Member profile

Per-member drill-in. Reachable by clicking a member in **Top movers**, **Per-member ranking**, or the **Team overview** table. Layout mirrors the Dashboard's panel grid so the brain doesn't have to re-orient.

### Header

Avatar (initials), name, "Switch member" dropdown, "Edit identity" button. Below: the member's pillar(s) (with a tag indicating whether the pillar was **derived** automatically or set as a manual **override**), GitHub login, GitLab username, email.

### KPI tiles (×5)

Same five metrics as the Dashboard, but scoped to this member. Clicking a tile selects the active metric for the panels below. Source: `GET /api/contributions/member/{id}`.

### Weekly contribution

- **Type**: multi-line chart, one line per metric
- **Axes**: X = week (`w1` … `w12`), Y = count
- **Source**: `detail.week_totals` from `/api/contributions/member/{id}`
- **Reading it**: same mechanic as the Dashboard's Throughput trend — the active metric is bold, others fade. Vertices are dotted on the active line so you can pin specific weeks.

### Components touched

- **Type**: horizontal bar chart, one bar per Jira component
- **Axes**: bar length = number of issues touched in that component
- **Source**: `detail.components_touched`
- **Reading it**: where this person actually shipped work. Bars are normalized to the maximum value, so the relative widths reflect *concentration*, not absolute volume — read the trailing count for the absolute number.
- **Note**: when a member's issues only carry a `fixVersion` like `acp-tekton-1.18.x` and no explicit component, the backend infers the component from the version prefix (Phase 1 fallback — see `PILLAR_FALLBACK_PLAN.md`).

### Rank vs team

- **Type**: horizontal bar chart, one bar per metric
- **Axes**: bar length = the member's count divided by the team max for that metric; right-side badge = rank (e.g. `#3/12`)
- **Source**: computed on the client from all members' `week_totals`
- **Reading it**: at a glance, where this person sits on each of the five metrics across the whole roster. A member can be top-of-roster on one metric and bottom on another — that's the point of showing all five.

### Week-over-week pulse

- **Type**: grouped bar chart, two bars per metric
- **Axes**: left bar (gray) = prior half of the window, right bar (metric color) = recent half
- **Source**: `detail.week_totals`, split in half
- **Reading it**: a per-metric snapshot of the same prior-vs-recent comparison the KPI tiles use, but laid out side-by-side so you can compare all five at once.

### This sprint

- **Type**: 2×2 KPI grid
- **Cells**: In progress, Done, PRs open, PRs merged
- **Source**: `detail.sprint`
- **Reading it**: live state for the *current* sprint (whatever the latest open sprint is on the configured Jira board). Refreshes on the next collection cycle (typically a few minutes).

---

## Team overview tab

The roster view. No window filter — fixed at last 12 weeks.

### All members table

- **Type**: sortable data table, one row per member
- **Columns**: Member (avatar + display name + gh/gl logins), Pillar (tag list), Jira done (count + delta), Story pts, PRs merged (count + delta), Reviews (count + delta), Review p50 (hours to first review), PRs/wk (12-week sparkline)
- **Source**: `/api/contributions/listMembers` joined with `/api/contributions/team`
- **Reading it**: click any column header to sort. Click a row to drill into the member profile. The sparkline is *not* sortable — it's a quick visual to spot streaks and lulls without leaving the table.

### Throughput, by pillar

- **Type**: stacked bar chart
- **Axes**: X = week (12-week fixed), Y = PRs merged, stacked by pillar
- **Source**: `GET /api/contributions/pillars`
- **Reading it**: this is the same shape as the Dashboard's Pillar mix, but always pinned to PRs merged and 12 weeks. Useful for the "who shipped what last quarter" question.

### Review network density

- **Type**: prose KPI panel
- **Headline metrics**: % of PRs that received their first review within 24h; % of reviews that crossed pillar boundaries
- **Bullets**: p50 / p90 review latency, orphan rate (PRs that never got a review)
- **Source**: `GET /api/contributions/network`
- **Reading it**: a pulse on the team's review culture. Low first-review-under-24h means PRs sit. High cross-pillar % means people read code outside their own area — usually a good sign for knowledge-sharing.

---

## Metrics tab (DORA)

This is the older DORA-metrics dashboard, kept alongside the Team Analytics tabs because it answers a different question: how is our **release process** behaving, not how individuals are contributing.

Top filter: **Component** multi-select (drawn from the roadmap data, with a Clear filters button).

### Status badge + meta

Health indicator (Healthy / Initializing / Stale / Disabled), last-updated timestamp, and counts (releases, epics, issues considered).

### Metric cards (×5)

One expandable card per DORA-style metric:

| Metric | What it measures |
|---|---|
| **Release frequency** | how often a product releases ship |
| **Lead time to release** | issue creation → release that contains it |
| **Cycle time** | issue moves to in-progress → moves to done |
| **Patch ratio** | proportion of releases that are patch-level |
| **Time to patch** | how quickly a hotfix lands after a base release |

Click a card to expand a `MetricBreakdown` panel underneath with the per-component / per-release breakdown.

For the full definitions and calculation logic, see `METRICS.md` in the repo root.

---

## Where the data comes from (quick map)

| Endpoint | Used by |
|---|---|
| `GET /api/contributions/team` | Dashboard KPI tiles, Throughput trend, Pillar mix, Inflow vs outflow, Donut, Top movers, Per-member ranking |
| `GET /api/contributions/member/{id}` | Member profile (header, KPIs, Weekly contribution, Components touched, Week-over-week pulse, This sprint) |
| `GET /api/contributions/listMembers` | Team overview table |
| `GET /api/contributions/pillars` | Throughput by pillar (Team overview) |
| `GET /api/contributions/network` | Review network density (Team overview) |
| `GET /api/contributions/status` | Sync indicator (header) |
| `GET /api/metrics` | Metrics tab (DORA) |

For backend internals, see `API.md`. For the metrics calculation rules, see `METRICS.md`.
