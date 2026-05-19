# Team Analytics — Behaviour Changes

This file is the running log of behaviour changes to the team-analytics
pipeline. Each entry maps a workstream from the 2026-05-19 audit follow-up
plan (`docs/team-analytics/audit-2026-05-19/PLAN.md`) to the concrete
config knobs, defaults, and visible effects of the change. Operators
should consult this when upgrading.

Newest entries first.

## W1 — Member allowlist + GitLab `with_shared=false` + Pass B instance sweep

**What changed**

- `gitlab/client.go::ListGroupProjects` sends `with_shared=false`.
- `contributions.BuildAllowlist(cfg)` derives the W1 allowlist from
  the prefill maps minus `team_analytics.member_denylist`. The
  synthetic `bot` member id is always included.
- Aggregator INSERTs carry `AND COALESCE(m.id, pr.author_id) IN (?, …)`
  when the allowlist is enabled. Post-rebuild cleanup deletes rollup
  rows for members no longer in the set.
- `GET /api/contributions/members` and `/team` drop members outside
  the allowlist.
- New GitLab Pass B (`gitlab.member_instance_sweep: true`) sweeps
  `/api/v4/merge_requests?scope=all&author_username=<u>` per allowlisted
  member.

**Config additions** (all default off):

```yaml
gitlab:
  member_instance_sweep: false
team_analytics:
  member_denylist: []
```

**Activation rules**

| prefills | denylist | allowlist |
|---|---|---|
| empty | empty | disabled (pre-W1) |
| populated | any | enabled, denylist subtracts |
| empty | populated | disabled (refuse to collapse to `{bot}`) |

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
findings B1, B10, B11, B6.

## W4 — Sprint card: 4 lanes (todo / in_progress / done / cancelled)

**What changed**

- `SprintStats` JSON gains `todo`, `in_progress`, `cancelled`. `wip`
  stays for one release as derived alias.
- New `team_analytics.statuses` config block with W4 default mapping
  pulled live from DEVOPS Jira.
- `contributions/status_lanes.go::StatusClassifier` does the lookup.
- `sprintCounts` swaps `resolved_at IS NULL` for the classifier.
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
- Window expands to 364 days when `?period=release_quarter`.

**Deferred to follow-up PR**

- Jira sync pass that walks Milestone summaries to populate
  `quarter_assignments` from the `^(\d{4}Q[1-4])[：:]` prefix.
- Frontend `?period=` tab.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B13.

## W5 — `pull_requests.epic_key` → `jira_key` rename

**What changed**

- Migration `0006_rename_epic_key_to_jira_key.sql`.
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
