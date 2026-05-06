# Team Analytics — Proposal

> **Status**: Draft for review · **Author**: roadmap-planner team · **Date**: 2026-05-06
> **Companion**: see `prototype.html` in this folder for an interactive mock of the proposed UI.

---

## 1. The ask, in one paragraph

We have a working roadmap-planner that visualises Jira pillars / milestones /
epics on a Kanban board and computes five DORA-style metrics in memory. We want
to grow it into a **team performance system**: pull historical data from Jira
(issues completed, by sprint / type / user) and GitHub (PRs, reviews,
contributions), persist it, and give us evolving views of *how a person* and
*how a team* are doing — sliced by pillar, component, or member.

The current design has two structural gaps that block this:

1. **No persistence.** The collector holds everything in memory and re-fetches
   on restart, so we cannot compare *this week* to *last quarter*.
2. **No second data source.** The system only knows about Jira. PRs, reviews,
   merge frequency — invisible.

This document compares four ways to close the gap, picks one as the
recommendation, and ships an initial implementation under
`backend/internal/storage` + `backend/internal/github` so the recommendation
is not just words.

---

## 2. What's already in the box

A short reminder before we propose changes.

| Layer | What we have | Where |
|---|---|---|
| Backend | Go 1.25 + Gin, Jira REST client (`andygrunwald/go-jira`), DORA collector running every 5 min in memory | `backend/internal/{jira,metrics}` |
| Storage | None. Everything is `sync.RWMutex`-guarded slices in `Collector` | `backend/internal/metrics/collector.go:42` |
| Metrics | Release Frequency, Lead Time to Release, Cycle Time, Patch Ratio, Time to Patch — all break-downable by component | `backend/internal/metrics/calculators/` |
| Auth | Per-request Jira basic auth via `X-Jira-*` headers; session cookie on backend | `backend/internal/api/middleware/` |
| Frontend | React 18 + Vite + Recharts; Kanban + Metrics dashboards | `frontend/src/components/` |
| Identity model | Pillar → Milestone → Epic → Issue, with components & assignees on every level | `backend/internal/models/` |

**The good news**: the calculator + breakdown abstraction
(`MetricBreakdown{Dimension, Key, Value}` at
`backend/internal/metrics/models/metrics.go:37`) already supports the
"slice by pillar / component / member" question. We just don't have history
or non-Jira inputs feeding it yet.

---

## 3. Reference: what middlewarehq/middleware does

The colleague's pointer is good. Middleware is an Apache-2.0 DORA platform
(Python + Flask + Postgres + Redis + Next.js) with a clean ETL → ORM → REST →
React separation. Highlights worth borrowing from:

- **Event-sourced PR timeline.** `PullRequestEvent` rows store every state
  change (OPENED / REVIEW / MERGED…) so timings can be re-derived without
  re-fetching from GitHub.
- **Settings table** (polymorphic per org/team) for prod-branch filters,
  excluded PRs, incident regexes — the same pattern we'd want for our
  pillar/component mapping rules.
- **Per-org Integration table** with chunked encrypted tokens; both GitHub
  and GitLab speak through a `CodeProviderETLHandler` interface.

What it **doesn't** do: any Jira ingestion. That's a deliberate philosophical
choice (DORA from PRs only). For us, Jira is the source of truth for "what
work was completed, by whom, in which sprint" — so we cannot adopt
middleware as-is.

---

## 4. The four options

The axes that matter:

- **Build vs. adopt** — how much new code do we own?
- **Stack drift** — does it stay Go-native or do we bolt on a Python service?
- **Time to first useful chart** — when does a person see *their* PR
  velocity next to *their* Jira completions?

| | A. Minimal extension | B. Native historian *(recommended)* | C. Middleware sidecar | D. Warehouse + thin BI |
|---|---|---|---|---|
| New services | 0 | 0 | +1 (middleware container) | +2 (DuckDB/ClickHouse + metabase) |
| Storage | SQLite file mounted alongside backend | SQLite (dev) / Postgres (prod) — same code path | Middleware's Postgres + ours | Warehouse columnar store |
| Jira ingestion | Reuse existing collector | Existing collector + snapshot writer | Build a `JiraETLHandler` for middleware | dbt / Airbyte connector |
| GitHub ingestion | Tiny client, PR list only | New `internal/github` package, paginated PR + review fetcher | Middleware does it | Airbyte / Fivetran connector |
| Linking PR↔Epic | Branch-name regex only | Branch + PR title + commit-message scan | Branch regex (middleware native) | SQL join in warehouse |
| Frontend | Extend existing dashboard | New "Team" view + extend Metrics | Two UIs (ours + middleware's) | Metabase or Superset |
| Effort | ~1 week | ~3 weeks for v1 | ~2 weeks integration + ongoing sync overhead | ~6 weeks |
| Risk | Underpowered for "evolution over time" | Schema design has to be right early | Two data planes drift | Big up-front investment |

### 4.1 Option A — Minimal extension

Bolt SQLite onto the existing `Collector` as a write-behind cache: every time
`Collect()` finishes, dump the current `EnrichedRelease` / `EnrichedIssue`
slices as a row in a `snapshots(captured_at, payload_json)` table. Add a
500-line GitHub client that lists merged PRs in a configured org/repo set
nightly, joins them to epics by branch-name regex (`DEVOPS-12345-*`), and adds
two extra metrics (`pr_throughput_per_member`, `review_latency_p50`).

**Why pick it**: shipped in a week. Anyone scared of DB migrations is happy.

**Why it bites later**: the JSON-blob snapshot makes "show me Alice's PR
count by week for the last year" require deserialising every blob in Go —
i.e., we built a database with no indexes. We will end up rewriting it as
Option B in three months.

### 4.2 Option B — Native historian *(recommended)*

A new `backend/internal/storage` package that owns:

- a small relational schema (~8 tables),
- a `Store` interface with one in-tree implementation (`sqliteStore`) and
  one easy-to-add (`postgresStore` — same SQL, swap driver), and
- a `Snapshotter` that the existing collector calls at the end of every
  `Collect()`.

A new `backend/internal/github` package mirrors the Jira client: an
incremental fetcher of PRs, reviews, and (optional) commits scoped to an
allow-listed set of repos. PR↔Epic linking is a pluggable `Linker` with
three built-in strategies (branch regex, PR title regex, commit-message
scan) — config picks which apply.

A new `backend/internal/contributions` package does the actual rollups:
`(member, week, pillar, component) → {jira_done, story_points, prs_merged,
prs_reviewed, review_latency_p50}` — directly fronting the new
`/api/contributions/*` endpoints.

The frontend gets a new **Team** tab next to **Roadmap** and **Metrics**
with three sub-views:

1. **Member profile** — sparklines for one engineer over time, with toggle
   between Jira completions, PRs merged, reviews given.
2. **Team overview** — every member as a row, each KPI as a column, sortable.
3. **Slice explorer** — pivot-table-ish: pick (pillar | component | sprint)
   on rows, (member | week | quarter) on columns, metric in cells.

**Why pick it**: it's the shape the existing code already wants. The
calculator/breakdown plumbing is there; we're just adding a real backing
store and a second source. Ports cleanly to Postgres when we outgrow
SQLite. Stays a single Go binary.

**The hard part**: deciding the schema once, well. Section 6 does this.

### 4.3 Option C — Middleware as a sidecar

Run middleware in a second container behind the same docker-compose, point
its GitHub ETL at our org, then either:

- iframe its DORA dashboard inside our `MetricsDashboard.jsx`, or
- have our backend hit middleware's REST `/code-analytics`, `/incidents`
  endpoints and fold the results in.

Add a new `JiraETLHandler` to middleware so it learns about issues / sprints /
story points (~600 lines of Python following the existing
`etl_github_handler.py` pattern).

**Why pick it**: their event-sourced PR timeline is genuinely better than
what we'd build in a month, and we get GitLab support for free if we ever
need it.

**Why I'd avoid it**: two backends, two languages (Go + Python), two DB
schemas, two deployment stories, two auth models. Our team is Go-native;
the maintenance cost compounds. Also middleware's identity model is *team
→ repo*, not *pillar → component → epic* — we'd have to bend it.

### 4.4 Option D — Warehouse + thin BI

Pull Jira and GitHub via Airbyte (or hand-rolled extractors) into a
columnar store (DuckDB for laptop, ClickHouse / Postgres for prod). Model
with dbt:

- `stg_jira_issues`, `stg_github_prs`, `stg_github_reviews`
- `fct_member_week`, `fct_pillar_week`, `dim_member`, `dim_pillar`

Frontend becomes Metabase or Superset embedded; roadmap-planner stays a
roadmap tool, and analytics is a sibling product.

**Why pick it**: this is what you do at ~50 engineers if Alauda decides
this is a serious internal tool. Slicing/dicing is unbounded.

**Why not now**: 6+ weeks for the first chart vs. 3 for Option B. Way too
much infrastructure for a tool with one user (us) and a clear, narrow set
of questions ("how is our team doing?"). Re-visit when the questions
outgrow what Option B's API can answer.

---

## 5. Recommendation

**Option B**, executed in three slices:

| Slice | What ships | Effort |
|---|---|---|
| B1 — Storage | SQLite store, schema, snapshotter wired into the existing collector. No new UI. Just: every metrics collection writes a snapshot row. | 5 days |
| B2 — GitHub | `internal/github` client (PRs + reviews) on a 30-min cron, basic Linker (branch regex), `/api/contributions/members` endpoint, **Team** tab with member overview table. | 8 days |
| B3 — Slicing | Slice explorer view, sprint awareness (Jira sprint custom field), historical sparklines on member profile, CSV export. | 7 days |

Slice B1 is the schema-defining moment; once shipped, B2 and B3 are
incremental.

This proposal includes a *partial* B1 + scaffolding for B2 already in the
branch, so you can `make build` and see something compile rather than read
about it. See section 8.

---

## 6. Schema for B1 / B2

Eight tables. Identifiers are stable strings where possible (Jira keys,
GitHub login, repo full name) so we never have to reconcile after a re-sync.

```sql
-- -----------------------------------------------------------------------
-- Identity
-- -----------------------------------------------------------------------
CREATE TABLE members (
    id              TEXT PRIMARY KEY,        -- our internal stable id (slug)
    display_name    TEXT NOT NULL,
    email           TEXT,
    jira_account_id TEXT UNIQUE,
    github_login    TEXT UNIQUE,
    pillar_id       TEXT,                    -- primary pillar (optional)
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
CREATE INDEX idx_members_pillar  ON members(pillar_id);
CREATE INDEX idx_members_github  ON members(github_login);

-- -----------------------------------------------------------------------
-- Roadmap snapshots — one row per metrics collection cycle
-- -----------------------------------------------------------------------
CREATE TABLE collection_runs (
    id             TEXT PRIMARY KEY,        -- ULID
    captured_at    TIMESTAMP NOT NULL,
    source         TEXT NOT NULL,           -- "jira" | "github"
    duration_ms    INTEGER NOT NULL,
    record_count   INTEGER NOT NULL,
    error          TEXT
);
CREATE INDEX idx_runs_captured ON collection_runs(captured_at DESC);

CREATE TABLE issue_snapshots (
    run_id         TEXT NOT NULL REFERENCES collection_runs(id),
    issue_key      TEXT NOT NULL,
    issue_type     TEXT NOT NULL,           -- "Epic" | "Story" | "Bug" | ...
    status         TEXT NOT NULL,
    assignee_id    TEXT REFERENCES members(id),
    pillar_id      TEXT,
    components     TEXT,                    -- JSON array
    versions       TEXT,                    -- JSON array
    sprint_id      TEXT,
    story_points   REAL,
    created_at     TIMESTAMP NOT NULL,
    resolved_at    TIMESTAMP,
    PRIMARY KEY (run_id, issue_key)
);
CREATE INDEX idx_issues_assignee ON issue_snapshots(assignee_id, run_id);
CREATE INDEX idx_issues_pillar   ON issue_snapshots(pillar_id, run_id);
CREATE INDEX idx_issues_sprint   ON issue_snapshots(sprint_id);

-- -----------------------------------------------------------------------
-- GitHub
-- -----------------------------------------------------------------------
CREATE TABLE repos (
    id            TEXT PRIMARY KEY,         -- "org/name"
    org           TEXT NOT NULL,
    name          TEXT NOT NULL,
    pillar_id     TEXT,                     -- optional, for grouping
    component     TEXT,                     -- canonical component name
    active        INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE pull_requests (
    id              TEXT PRIMARY KEY,       -- "org/name#number"
    repo_id         TEXT NOT NULL REFERENCES repos(id),
    number          INTEGER NOT NULL,
    title           TEXT NOT NULL,
    state           TEXT NOT NULL,          -- "open" | "merged" | "closed"
    author_id       TEXT REFERENCES members(id),
    head_branch     TEXT,
    base_branch     TEXT,
    additions       INTEGER,
    deletions       INTEGER,
    changed_files   INTEGER,
    epic_key        TEXT,                   -- linked Jira epic, may be NULL
    created_at      TIMESTAMP NOT NULL,
    first_review_at TIMESTAMP,
    merged_at       TIMESTAMP,
    closed_at       TIMESTAMP,
    fetched_at      TIMESTAMP NOT NULL
);
CREATE INDEX idx_pr_author  ON pull_requests(author_id, merged_at);
CREATE INDEX idx_pr_repo    ON pull_requests(repo_id, merged_at);
CREATE INDEX idx_pr_epic    ON pull_requests(epic_key);

CREATE TABLE pr_reviews (
    id          TEXT PRIMARY KEY,           -- repo + PR + review id
    pr_id       TEXT NOT NULL REFERENCES pull_requests(id),
    reviewer_id TEXT REFERENCES members(id),
    state       TEXT NOT NULL,              -- approved | changes_requested | commented
    submitted_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_review_reviewer ON pr_reviews(reviewer_id, submitted_at);

-- -----------------------------------------------------------------------
-- Pre-computed rollups (refreshed by aggregator)
-- -----------------------------------------------------------------------
CREATE TABLE member_week_metrics (
    member_id        TEXT NOT NULL REFERENCES members(id),
    week_start       DATE NOT NULL,         -- Monday
    pillar_id        TEXT,                  -- NULL = "all pillars"
    component        TEXT,                  -- NULL = "all components"
    jira_issues_done INTEGER NOT NULL DEFAULT 0,
    jira_points_done REAL    NOT NULL DEFAULT 0,
    prs_merged       INTEGER NOT NULL DEFAULT 0,
    prs_reviewed     INTEGER NOT NULL DEFAULT 0,
    review_latency_p50_hours REAL,
    PRIMARY KEY (member_id, week_start, pillar_id, component)
);
CREATE INDEX idx_mwm_week ON member_week_metrics(week_start, pillar_id);
```

Notes worth calling out:

- **Snapshots are immutable.** `issue_snapshots` is keyed on `run_id`, so we
  never mutate history; "current state" is the latest run. This is the same
  bet middleware made with `PullRequestEvent` and it pays off for any "show
  me what changed" question.
- **Rollups are derived.** `member_week_metrics` is a cache: we can drop and
  rebuild it from the snapshots and PRs at any time. It only exists because
  the dashboards query it constantly.
- **Pillar / component live on the issue snapshot, not just the member.**
  That's intentional: a member can contribute to multiple pillars, and
  filtering must work both directions ("Alice, by pillar" *and* "Pillar X,
  by member").

---

## 6.5 Initial backfill — first-run policy

Persistent storage is only useful if there is data to look at. Asking
the user to wait 12 weeks for the dashboards to fill is a non-starter,
so the **first** collection cycle on each source reaches back into
history before resuming incremental sync.

### Configuration

```yaml
storage:
  backfill_days: 180             # default — used by every source unless overridden
github:
  backfill_days: 0               # 0 = inherit storage.backfill_days
```

A single number controls the window. We deliberately keep it narrow:
one knob, two override points (storage + per-source), no `from`/`to`
date ranges. If someone needs surgical control they can run a one-off
import job; the auto-pilot stays simple.

### Mechanics, per source

**GitHub (B1, in this branch).**
The Syncer probes `collection_runs` for `source="github"`. If empty, it
treats the run as backfill and pages the GitHub PR list with
`since = now - backfill_days`. PR lists in `/repos/{o}/{r}/pulls?sort=updated`
are O(N) in window-PR count; for our scale (max ~2k PRs / repo / 6 mo)
that's a few minutes of API quota even with reviews fetched per PR.
Subsequent runs resume from `latest_run.captured_at - 1h`.

A status note is logged at run start so operators can tell which mode
they're in:

```
github sync starting  mode=backfill  since=2025-11-07T00:00:00Z  repos=4
```

**Jira (B2, snapshot writer not yet wired).**
Three flavours of historical fetch are needed; we'll execute them in
order on first run:

| Pass | JQL | Purpose |
|---|---|---|
| 1 | `project = X AND resolved >= -180d` | "Issues completed in window" — populates `issue_snapshots` with their *current* state, dated by `resolved_at`. |
| 2 | `project = X AND created >= -180d AND resolved is EMPTY` | "Open work that started in window" — captures live WIP without forcing a re-snapshot of every old open ticket. |
| 3 | for each issue from (1)+(2): `/issue/{key}/changelog` | Status timeline so cycle-time per resolved issue is computable. |

Pass 3 is the expensive one; we batch with the Jira REST `expand=changelog`
parameter on the search endpoint to avoid N+1 round-trips. For our
project (~3k issues / 6 mo), that's a single bulk fetch in < 1 minute.

We deliberately do **not** try to reconstruct snapshots at every
historical week boundary — only the resolution-time state. The rollup
table places the issue in the week of its `resolved_at`, which is the
right shape for "Jira done per week" without paying the
state-reconstruction tax.

### Failure modes & guardrails

- **Quota.** GitHub PAT is 5,000 req/hour. Backfill of 4 repos × ~500
  PRs each × 1 listing + 1 reviews call = ~4k requests — comfortable
  but tight. We log `X-RateLimit-Remaining` at end-of-batch, and if
  it dips under 200 we sleep until the reset.
- **Resumability.** Backfill writes a `collection_runs` row at the end,
  so a crashed backfill leaves no state and the next process attempt
  retries cleanly. We do not checkpoint mid-backfill — the API doesn't
  expose stable cursors for "PRs updated since X with offset Y" anyway.
- **No infinite scroll.** `MaxPages: 10` (≈1k PRs) caps any single
  repo's first run; partial-fill is preferable to a 30-min boot.
  Operators bump it manually for repos with deeper history.

---

## 6.6 Dialect abstraction & Postgres support

The branch ships SQLite as the default and Postgres as a first-class
peer. Both go through one `Dialect` interface; queries are written
once, with `?` placeholders, and `rebind()` shifts to `$N` at exec
time when the driver is `pgx`.

### Why Postgres now (not deferred)

The schema in §6 already runs unchanged on stock Postgres — DATE,
TIMESTAMP, REAL, INTEGER, TEXT, partial unique indexes, ON CONFLICT
DO UPDATE — none of it is SQLite-only. The two real differences are
small enough that paying the abstraction cost up-front is cheaper
than retrofitting it after the dashboards depend on the storage:

| Difference | SQLite | Postgres | How we handle it |
|---|---|---|---|
| Placeholders | `?` | `$1, $2, …` | `rebind()` rewrites at exec time |
| Week-start expr | `DATE(col, 'weekday 1', '-7 days')` | `date_trunc('week', col)::date` | `Dialect.WeekStart(col)` returns the right SQL fragment |
| Driver | `modernc.org/sqlite` (pure Go) | `github.com/jackc/pgx/v5/stdlib` | both registered with `database/sql` |

### When to use which

- **SQLite.** Local dev; demo / single-tenant deployments; "just give
  me the binary" installs. ~1MB of state for our workload, and the
  WAL pragma keeps the collector goroutine happy alongside HTTP reads.
- **Postgres.** Shared deployments, multi-replica pods, BI tooling
  (Metabase / Superset) needing query access, or any case where
  you'd back up the database separately from the binary. The DSN
  goes in `storage.dsn` or `STORAGE_DSN`.

The same migrations apply to both; new dialect-specific migrations
(if any ever land — JSONB indexes are the obvious candidate) go in
suffixed files (`0002_postgres_jsonb_indexes.sql`) gated by Dialect
in the migration loader.

### Code shape

```
backend/internal/storage/
├── store.go        — Store interface (unchanged surface)
├── dialect.go      — Dialect interface + sqlite & postgres impls
├── generic.go      — single Store implementation, parameterised by Dialect
├── sqlite.go       — OpenSQLite factory
├── postgres.go     — OpenPostgres factory
├── migrations.go   — dialect-aware migrations runner
└── migrations/
    └── 0001_init.sql   — shared schema
```

Tests cover the rebind logic (`dialect_test.go`) and the SQLite round
trip (`store_test.go`). A Postgres round-trip test lives behind a
`TEST_POSTGRES_DSN` env var so CI doesn't need a database — local
runs against a docker-compose pg are documented in `DEVELOPMENT.md`.

---

## 7. UI / UX direction

Companion file: `prototype.html` is a working static mock — open it in a
browser, no backend needed. It implements three of the proposed views:

1. **Team overview** — table of members with sparkline + KPI columns,
   sortable.
2. **Member profile** — drill-in view with weekly trend lines for Jira
   completions, PRs merged, reviews given, plus "current sprint" callout.
3. **Slice explorer** — pillar × week pivot, with a metric switcher.

Visual language stays consistent with the existing Atlas / Platform theming
(serif display, mono captions, restrained palette). Charts: keep using
Recharts in the real impl — the prototype uses inline SVG so it ships
without dependencies.

A few opinionated design calls baked into the prototype:

- **Sparklines over big charts.** The Team Overview shows 12-week sparklines
  per member, in-row. It's the difference between "30 numbers" and "30
  shapes" — the eye finds the outliers in milliseconds.
- **Two-axis filter, never three.** Pivot views always have 2 dimensions on
  axes and 1 in the cells. A 3-axis cube is unreadable.
- **No leaderboard framing.** Numbers next to names invite competitive
  reading; we deliberately label them as *throughput indicators* not
  *performance scores*, and add a "vs. team median" column instead of an
  absolute rank.

---

## 8. What's already in this branch

Branch: `feat/roadmap-planner-team-analytics`.

This proposal does not stop at words. The branch contains an initial cut
of Option B's foundation:

```
roadmap-planner/
├── docs/team-analytics/
│   ├── PROPOSAL.md              <- this document
│   └── prototype.html           <- interactive mock
├── backend/internal/storage/
│   ├── store.go                 <- Store interface
│   ├── sqlite.go                <- modernc.org/sqlite implementation
│   ├── migrations.go            <- embedded SQL migrations
│   ├── migrations/0001_init.sql <- schema from §6
│   └── store_test.go            <- round-trip test
├── backend/internal/github/
│   ├── client.go                <- minimal GitHub PR + review client
│   └── sync.go                  <- incremental fetcher + Linker hooks
├── backend/internal/contributions/
│   ├── aggregator.go            <- member_week_metrics builder
│   └── service.go               <- query layer behind the API
├── backend/internal/api/handlers/
│   └── contributions.go         <- /api/contributions/* endpoints
└── frontend/src/components/
    ├── TeamAnalytics.jsx        <- new "Team" view
    └── TeamAnalytics.css
```

What ships compiling and tested:

- ✅ `storage` package: dialect-aware `genericStore` + `OpenSQLite` +
  `OpenPostgres` + migrations runner + read-back, with round-trip and
  rebind tests.
- ✅ `github` package: `ListPullRequests(repo, since)`, `ListReviews(pr)`
  + Syncer with **first-run backfill** (default 180 days, configurable).
- ✅ Config wiring (`storage.{type,path,dsn,backfill_days}`,
  `github.{token,repos,backfill_days}`); env-var overrides.
- ✅ Contributions handler — `/api/contributions/{members,team,members/:id,status}`
  registered behind AuthMiddleware.
- ✅ Frontend **Team** tab and member-overview view, fed by the new endpoints.

What's intentionally stubbed (TODOs marked in code, see B2/B3):

- ⏳ Linker: only branch-name regex implemented; PR-title and commit-message
  strategies are interfaces with TODOs.
- ⏳ Aggregator: writes nothing yet — service queries hit the snapshot tables
  directly. B2 will materialise the rollup table.
- ⏳ Sprint awareness: Jira sprint custom-field reader is wired but not
  emitting `sprint_id` into snapshots yet.
- ⏳ Auth: GitHub token comes from config / env only; no per-user OAuth.

---

## 9. Risks & open questions

1. **PR↔Epic linking accuracy.** Branch regex is great when conventions are
   followed and useless otherwise. We need a sample run on the real `devops`
   org to measure hit rate before relying on it for member metrics.
2. **Member identity drift.** People rename their GitHub login or Jira
   email. The `members` table is the join key; we need a UI to merge
   identities and a `member_aliases` table if drift is real (deferred to
   B3).
3. **Sprint custom field stability.** The current code already parses
   `customfield_12242` (quarter) and a handful of sequence custom fields.
   Sprint is a new one; if the field id moves between Jira projects we
   need it config-driven.
4. **Performance.** SQLite handles our scale comfortably (~10k issues,
   ~2k PRs/year). If we ever ingest the whole DEVOPS history, expect
   `issue_snapshots` to grow ~50k rows/year — still fine for SQLite, but
   that's when the Postgres switch becomes attractive.
5. **Privacy framing.** Per-member metrics are sensitive. Default the new
   Team view to *opt-in by member* via a `members.metrics_visible` flag, so
   we don't accidentally turn this into a stack-rank tool.

---

## 10. Decision needed

To unblock B2:

- [ ] Approve schema in §6 (or specify changes).
- [ ] Confirm GitHub org(s) and repo allow-list to ingest first.
- [ ] Approve "opt-in per member" default for the Team view.
- [x] ~~Pick: SQLite-only forever (simpler) or Postgres-ready abstraction~~
      → **Postgres-ready abstraction shipped in this branch** (§6.6). SQLite
      stays the default; Postgres is one config flip. No change to
      callers.
- [x] ~~Initial backfill window~~ → **180 days** (configurable via
      `storage.backfill_days`; per-source override on `github.backfill_days`).
      See §6.5. Override below if a different window is preferred.

If approved as-is, B2 lands in ~8 working days.
