# Team Analytics — Behaviour Changes

This file is the running log of behaviour changes to the team-analytics
pipeline. Each entry maps a workstream from the 2026-05-19 audit follow-up
plan (`docs/team-analytics/audit-2026-05-19/PLAN.md`) to the concrete
config knobs, defaults, and visible effects of the change. Operators
should consult this when upgrading.

Newest entries first.

## W2 — Bot consolidation under one synthetic `bot` member

**What changed**

- Migration `0005_first_human_review.sql` adds
  `pr_reviews.is_bot` and `pull_requests.first_human_review_at`.
- New `team_analytics.bot_logins []string` config knob — explicit
  list, case-folded, no suffix magic.
- Github + GitLab syncers reassign bot author/reviewer to the `"bot"`
  member id, tag `is_bot`, and compute `first_human_review_at` live.
- `contributions.ConsolidateBots` runs once on every startup as a
  retroactive backfill (idempotent).
- `NetworkDensity` reads `first_human_review_at` and joins with
  `AND rv.is_bot = 0`.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B4 (56% of review rows are bots).

## W1 — Member allowlist + GitLab `with_shared=false` + Pass B instance sweep

**What changed**

- `gitlab/client.go::ListGroupProjects` sends `with_shared=false`.
- `contributions.BuildAllowlist(cfg)` derives the W1 allowlist from
  the prefill maps minus `team_analytics.member_denylist`.
- Aggregator filters on the allowlist when enabled; cleanup deletes
  rollup rows for members no longer in the set.
- New GitLab Pass B per-allowlisted-member instance-wide MR sweep.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
findings B1, B10, B11, B6.

## W4 — Sprint card: 4 lanes (todo / in_progress / done / cancelled)

**What changed**

- `SprintStats` JSON gains `todo`, `in_progress`, `cancelled`.
- New `team_analytics.statuses` config block.
- `contributions/status_lanes.go::StatusClassifier`.
- Member-profile sprint card renders 6 KPI tiles.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B7.

## W7 — 12-month window + release-cadence "quarter" buckets

**What changed (delivered)**

- `storage.backfill_days` default flips from 180 → 365.
- `aggregator.RebuildRecent`'s default rebuild window 187 → 400 days.
- Migration `0008_quarter_assignments.sql`.
- New `contributions.QuarterResolver` (period.go).
- `MemberSummary.PeriodTotals` populated on `?period=release_quarter`.

**Deferred**

- Jira sync Milestone-prefix pass.
- Frontend `?period=` tab.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B13.

## W5 — `pull_requests.epic_key` → `jira_key` rename

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B8.

## W6 — GitLab parity: `hydrate_diff` default flips to `true`

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B12.
