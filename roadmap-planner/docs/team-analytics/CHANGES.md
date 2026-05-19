# Team Analytics — Behaviour Changes

This file is the running log of behaviour changes to the team-analytics
pipeline. Each entry maps a workstream from the 2026-05-19 audit follow-up
plan (`docs/team-analytics/audit-2026-05-19/PLAN.md`) to the concrete
config knobs, defaults, and visible effects of the change. Operators
should consult this when upgrading.

Newest entries first.

## W7 — 12-month window + release-cadence "quarter" buckets

**What changed (delivered in this PR)**

- `storage.backfill_days` default flips from `180` → `365`. The
  first-run JQL pulls a full year of Jira history.
- `aggregator.RebuildRecent`'s default rebuild window flips from
  `187` days → `400` days (cap raised to `730`).
- New migration `0008_quarter_assignments.sql` adds a small
  `(issue_key, quarter_label, source)` table. Empty on install.
- New `contributions.QuarterResolver` (period.go).
- `MemberSummary.PeriodTotals []PeriodBucket` populated when the API
  gets `?period=release_quarter`. Read-time fold over the existing
  weekly rollup via `FoldWeeksByCalendarQuarter`.
- `parseQuery` default window expands to 364 days when
  `?period=release_quarter` requests so the chart shows a full year
  without explicit `from`/`to`.

**Deferred to a follow-up PR**

- The Jira sync pass that walks every Milestone, parses
  `^(\d{4}Q[1-4])[：:]` (full-width Chinese colon!), follows the
  Blocks inward link to the Epic, and writes
  `(epic_key, quarter_label)` into `quarter_assignments`.
- The Story / Bug → Epic Link walk that picks up parent-Epic
  milestone-quarter.
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
