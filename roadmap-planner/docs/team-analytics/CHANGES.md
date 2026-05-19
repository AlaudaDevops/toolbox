# Team Analytics — Behaviour Changes

This file is the running log of behaviour changes to the team-analytics
pipeline. Each entry maps a workstream from the 2026-05-19 audit follow-up
plan (`docs/team-analytics/audit-2026-05-19/PLAN.md`) to the concrete
config knobs, defaults, and visible effects of the change. Operators
should consult this when upgrading.

Newest entries first.

## W4 — Sprint card: 4 lanes (todo / in_progress / done / cancelled)

**What changed**

- `SprintStats` JSON gains `todo`, `in_progress`, `cancelled`. `wip`
  stays for one release as derived alias `todo + in_progress`.
- New `team_analytics.statuses` config block — four optional lists.
  Empty lane inherits the W4 default (37 statuses pulled live from
  DEVOPS Jira, English + Chinese mix).
- `contributions/status_lanes.go::StatusClassifier` does the lookup
  (case-folded; unknown → `in_progress`, reported `known=false`).
- `sprintCounts` swaps `resolved_at IS NULL` for the classifier.
- Member-profile sprint card renders 6 KPI tiles.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B7.

## W7 — 12-month window + release-cadence "quarter" buckets

**What changed (delivered)**

- `storage.backfill_days` default flips from 180 → 365.
- `aggregator.RebuildRecent`'s default rebuild window flips from
  187 → 400 days (cap 730).
- New migration `0008_quarter_assignments.sql` adds a small
  `(issue_key, quarter_label, source)` table. Empty on install.
- New `contributions.QuarterResolver` (period.go).
- `MemberSummary.PeriodTotals []PeriodBucket` populated on
  `?period=release_quarter`.
- `parseQuery` default window expands to 364 days when
  `?period=release_quarter` requests so the chart shows a full year
  without explicit `from`/`to`.

**Deferred to a follow-up PR**

- Jira sync pass that walks every Milestone, parses
  `^(\d{4}Q[1-4])[：:]` (full-width Chinese colon!), follows the
  Blocks inward link to the Epic, writes `(epic_key, quarter_label)`
  into `quarter_assignments`.
- Story / Bug → Epic Link walk for parent-Epic milestone-quarter.
- Frontend `?period=` tab on the Team Dashboard.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B13.

## W5 — `pull_requests.epic_key` → `jira_key` rename

**What changed**

- Migration `0006_rename_epic_key_to_jira_key.sql` renames the column.
- Go struct field `PullRequest.EpicKey` → `PullRequest.JiraKey`, no
  alias.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B8.

## W6 — GitLab parity: `hydrate_diff` default flips to `true`

**What changed**

- `gitlab.NewSyncer` constructs with `HydrateDiff: true` (was `false`).
- viper default for `gitlab.hydrate_diff` is now `true`.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B12.
