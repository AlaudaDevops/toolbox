# Team Analytics — Behaviour Changes

This file is the running log of behaviour changes to the team-analytics
pipeline. Each entry maps a workstream from the 2026-05-19 audit follow-up
plan (`docs/team-analytics/audit-2026-05-19/PLAN.md`) to the concrete
config knobs, defaults, and visible effects of the change. Operators
should consult this when upgrading.

Newest entries first.

## W5 — `pull_requests.epic_key` → `jira_key` rename

**What changed**

- Migration `0006_rename_epic_key_to_jira_key.sql` renames the column
  via `ALTER TABLE pull_requests RENAME COLUMN epic_key TO jira_key`
  (portable in both SQLite ≥3.25 and Postgres), drops the
  `idx_pr_epic` index, and re-creates it as `idx_pr_jira`.
- Go struct field `PullRequest.EpicKey` → `PullRequest.JiraKey`, with
  no alias (hard rename).
- Every SQL reference and assignment in the github / gitlab syncers,
  the storage upsert, and the `sprintCounts` join updates in lock-step.

**Why**

The column was misnamed at birth: the linker pulls *any* Jira key out
of the source branch / title via `[A-Z]+-\d+`, so the value is mostly
a Story or Bug, never an Epic. Audit on prod (4411 linked PRs) showed
**zero** rows pointing at an actual Epic. The misnomer wasted reader
time and was load-bearing in one place — `sprintCounts` happened to be
correct only because most sprint members are Stories.

**Backward compatibility**

The JSON tag `epic_key` flips to `jira_key`. We grep'd `frontend/` and
no API consumer reads it (the PR list endpoint never exposed it; only
counts are surfaced). Safe one-shot rename.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B8.

## W6 — GitLab parity: `hydrate_diff` default flips to `true`

**What changed**

- `gitlab.NewSyncer` constructs with `HydrateDiff: true` (was `false`).
- viper default for `gitlab.hydrate_diff` is now `true`.

**Why**

Merged-MR additions / deletions / changed_files are needed by the
Dashboard tab and the GitHub side already collects them on every cycle
(GitHub returns the counts on the list endpoint; GitLab requires a
per-MR show call to populate them). The extra call is ~+2% of the
GitLab API budget per cycle. Per maintainer (Q12 / 2026-05-19), the
one-shot backfill cost is fine to absorb.

**Already-on-main**

The "skip review fetch for draft MRs" mirror was already in place
(`gitlab/sync.go` checks `mr.Draft || mr.WorkInProg`), so this PR
only flips the diff default.

**Deferred**

The `changes_requested` review-state marker for GitLab (Q5) is
deferred. The `/lgtm` approval / commented classifier stays unchanged.

**Rollback**

`gitlab.hydrate_diff: false` in the ConfigMap returns to the pre-W6
behavior — additions / deletions / changed_files stay at zero on
merged MRs.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B12.

## W2 — Bot consolidation under one synthetic `bot` member

**What changed**

- Two new columns (migration `0005_first_human_review.sql`):
  - `pr_reviews.is_bot INTEGER NOT NULL DEFAULT 0` — tagged when the
    `reviewer_login` matches `team_analytics.bot_logins`.
  - `pull_requests.first_human_review_at TIMESTAMP` — `MIN(submitted_at)`
    over non-bot reviews on the PR. Lets the NetworkDensity panel
    read human review latency with one column instead of a window
    over `pr_reviews` on every panel load.
- New `team_analytics.bot_logins []string` config knob — explicit list
  of GitHub / GitLab logins that should be folded into the synthetic
  `bot` member. Match is exact + case-folded; no suffix magic
  (confirmed by maintainer 2026-05-19 to avoid false positives).
- Github + GitLab syncers now consult the bot predicate per PR / per
  review: bot author → `author_id = "bot"`, bot reviewer →
  `reviewer_id = "bot"` + `is_bot = 1`. `first_human_review_at` is
  set live during sync.
- `contributions.ConsolidateBots` runs once on every startup as a
  retroactive backfill so rows ingested before W2 deployed get the
  same treatment without a re-sync. Idempotent.
- `NetworkDensity` (orphan rate, first-review p50/p90, cross-pillar
  review %) now binds on `first_human_review_at` and joins with
  `AND rv.is_bot = 0`. The dashboard's "first review under 24h"
  metric reflects the human review experience and is no longer
  collapsed to ~0.1h by renovate's instant comments.

**Config additions** (default off):

```yaml
team_analytics:
  bot_logins:
    - alaudabot
    - alaudaa-renovate
    - edge-katanomi-app2[bot]
    - copilot-pull-request-reviewer[bot]
    - kilo-code-bot[bot]
    - copilot
```

**Visible effects when configured**

- A new dashboard row appears for the `bot` synthetic member (already
  in the W1 allowlist). Until W1 is enabled, the bot row sits next to
  every other member in the rollup; with W1 the bot row stays visible
  because the allowlist injects `"bot"` automatically.
- `prs_merged` and `prs_reviewed` totals for the bot row may be large
  (renovate alone is hundreds of PRs in any 6-month window). This is
  expected — it surfaces the volume of automated work without
  contaminating human metrics.
- First-review-latency p50 jumps from sub-1h to the real human value
  (typically several hours).

**Rollback**

- Empty `bot_logins: []` and restart — the predicate disables and new
  PR / review rows ingest under their raw login. The two new columns
  remain (additive migration), but `is_bot` falls back to 0 on new
  rows.
- The retroactive author / reviewer reassignment is harder to undo
  cleanly. If a full reset is required, manual SQL: `UPDATE pull_requests
  SET author_id = '' WHERE author_id = 'bot'` and similar for reviews.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
finding B4 (56% of review rows are bots).

## W1 — Member allowlist + GitLab `with_shared=false` + Pass B instance sweep

**What changed**

- **`gitlab/client.go::ListGroupProjects`** now sends `with_shared=false`
  unconditionally. The GitLab default is `true`, which made
  `gitlab.groups=["devops/**"]` return every shared project the
  group had access to (the root cause of the audit's B1 finding —
  59 % of prod MRs came from outside `devops/**`).
- **W1 allowlist.** New helper `contributions.BuildAllowlist(cfg)`
  computes the effective set from
  `team_analytics.github_login_prefills ∪ team_analytics.gitlab_username_prefills`
  minus `team_analytics.member_denylist`. The synthetic `bot` member id
  is always included so W2's bot-consolidation row stays visible.
- **Aggregator filter.** When the allowlist is enabled, every INSERT in
  `Aggregator.Rebuild` carries an
  `AND COALESCE(m.id, pr.author_id) IN (?, …)` clause. A post-rebuild
  cleanup deletes `member_week_metrics` rows for members no longer in
  the set.
- **API filter.** `GET /api/contributions/members` and
  `GET /api/contributions/team` drop members outside the allowlist, on
  top of the existing `include_inactive` filter.
- **GitLab Pass B.** A new `Syncer.MemberInstanceSweep` field — driven
  by `gitlab.member_instance_sweep` — turns on a second sync pass that
  fetches `/api/v4/merge_requests?scope=all&author_username=<u>` for
  every allowlisted member with a configured `gitlab_username`. MRs
  filed by our team in projects outside `gitlab.groups` (e.g.
  `container-platform/*`, `alauda/artifacts`) now land in
  `pull_requests` with their proper `repo_id`. Dedupes against Pass A
  via the existing `<group/proj>!<iid>` primary key.

**Config additions** (all default to off / empty — no behaviour change
on upgrade until the operator opts in):

```yaml
gitlab:
  member_instance_sweep: false  # true to enable Pass B
team_analytics:
  member_denylist: []           # e.g. ["gxjiao", "lmhe", "zhwang", "chaozhou"]
```

**Allowlist activation rules**

- Empty prefills + empty denylist → allowlist disabled; the aggregator
  and the API behave exactly like before W1.
- Any prefill configured → allowlist activated; denylist subtracts.
- Denylist only (no prefills) → allowlist stays disabled (we can't
  derive who counts, so we refuse to filter rather than collapse to
  `{bot}` and silently empty the dashboard).

**Visible effects when enabled**

- Members outside the allowlist disappear from the team-overview list
  and from every per-member chart.
- With Pass B on, allowlisted members' counts may rise as MRs from
  out-of-`devops/**` projects fold in.
- One synthetic `bot` row appears in `members.id="bot"`. Until W2
  ships, it stays at zero counts because no PR rows are reassigned to
  it yet.

**Rollback**

- Flip `gitlab.member_instance_sweep: false` to disable Pass B without
  touching the allowlist.
- Empty the prefill maps to disable the allowlist filter entirely
  (reverts the aggregator to its pre-W1 SQL shape).
- Drop the denylist entries to re-include suppressed members without
  rebuilding the rollup table.

**Audit cross-reference:** `docs/team-analytics/audit-2026-05-19/REPORT.md`
findings B1, B10, B11, B6.
