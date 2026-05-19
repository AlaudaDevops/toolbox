# Metrics Audit — Follow-up Plan
**Source audit:** `REPORT.md` (this directory)
**Branch:** `plan/metrics-audit-followups`
**Owner:** TBD
**Status:** **DRAFT — not yet executed.** Several decisions are gated on the open questions at the bottom of this file.

This document maps every finding in the audit to a concrete deliverable. It is sized to be picked up by one engineer in roughly the order listed; later items inherit context from earlier ones.

---

## Scope and shape

- One PR per workstream (W1 … W7), each a tight diff with a backfill / migration story.
- All workstreams land behind config — no surprise behaviour changes on existing prod data.
- Each workstream ships:
  - Code change
  - Migration (where the schema moves)
  - Aggregator rebuild on startup so the dashboard is correct on the first read after deploy
  - At least one Go unit test that covers the new behaviour
  - A short note in `docs/team-analytics/CHANGES.md`
- The audit report (`REPORT.md`) stays in the repo as the historical "before" snapshot.

---

## W1 — Restrict member metrics to DEVOPS team contributions (covers B1, B10, B11)

**Problem.** Today the GitLab sync over-ingests because `with_shared=true` is the GitLab default, and the aggregator credits MRs solely on `LOWER(m.gitlab_username) = pr.author_login`. Result: zhwang's whole rollup is from `container-platform/*` / `ops/*` / `alauda/artifacts`.

**Direction agreed with maintainer.** Keep the GitLab scan instance-wide (it's a feature, not a bug — we want to capture our members' contributions to other teams' repos), but restrict **calculation** to members of the DEVOPS team. Add an explicit deny list for members we no longer want represented at all (Zonghang, Chaozhou, others).

**Design.**

1. **`team_analytics.members` becomes the source of truth for "who is a tracked member".** Move the today-implicit set (anyone the Jira sync ingested under DEVOPS) onto an explicit allowlist driven by config. Three possible knobs:
   - `team_analytics.member_allowlist: ["daniel","qingliu",…]` — opt-in.
   - `team_analytics.member_denylist: ["zhwang","chaozhou","gxjiao","lmhe"]` — opt-out.
   - **Recommended:** allowlist by default (derived from `github_login_prefills`/`gitlab_username_prefills` union, since those are already curated) **plus** an explicit `member_denylist` field for one-off removals.
2. **Aggregator** gains a single `WHERE member_id IN (…allowlist…)` predicate after the JOIN. Everything outside the allowlist drops from `member_week_metrics` and from every downstream API.
3. **`/api/contributions/members`** filters to the allowlist. The existing `include_inactive` flag stays for the `members.active` Jira-truth dimension, but the allowlist is the harder gate.
4. **`PillarThroughput`** keeps counting every in-scope MR / Jira issue (it's a team-level chart, not a per-member chart), but with the GitLab scope tightened it will naturally drop the noise from `alauda/artifacts` etc. — see W1b.

**W1b. GitLab fetch — keep instance-wide, drop shared.** Set `with_shared=false` on the `/groups/:id/projects` call. Rationale: "scan the whole instance" should mean "scan the `devops/**` subtree we control", not "scan every group that has ever shared a project with us." Maintainer agreed B1 should be re-framed as "instance-wide" but in the GitLab sense this means *traverse subgroups recursively*, not *follow shared-project links*. **OPEN QUESTION (Q1) below.**

**Files**
- `backend/internal/gitlab/client.go` — add `with_shared=false`.
- `backend/internal/config/config.go` — add `TeamAnalytics.MemberAllowlist []string`, `TeamAnalytics.MemberDenylist []string`.
- `backend/internal/contributions/allowlist.go` — new helper that resolves the effective set (allowlist − denylist).
- `backend/internal/contributions/aggregator.go` — wire allowlist into the four INSERT SELECT statements.
- `backend/internal/api/handlers/contributions.go::ListMembers/TeamOverview` — also filter on the same set.

**Migration**
- No schema change. On rollout, run `aggregator.RebuildRecent(0)` once so the rollup discards non-allowlisted members.
- Optionally: `DELETE FROM member_week_metrics WHERE member_id NOT IN (…allowlist…)` for cleanliness.

**Rollback.** Empty allowlist == "trust the data, no filter" — same behaviour as today. Safe to revert by clearing the config.

**Tests**
- Aggregator: write 3 PRs by 3 logins, only one in the allowlist; rebuild; assert 1 row in rollup.
- Handler: with denylist, the team overview hides the denylisted member even if they have rollup rows.

**Risk.** Low. The blast radius is "previously-shown members get hidden". Visible to operators on next reload.

---

## W2 — Bot classification (covers B4)

**Problem.** 1450 of 3079 PRs in the 12-week window have `first_review_at` inside 6 minutes; 1659 of those first reviews are by `alaudabot`. `pr_reviews` is 56 % bot rows. The "first-review latency" KPI is unusable.

**Design.**

1. Introduce a single bot predicate in one place. Two stacked filters:
   - **Suffix rule.** Anyone matching `*[bot]` or `*-bot` or `*-app` is a bot.
   - **Explicit deny set** in config: `team_analytics.bot_reviewers: ["alaudabot","alaudaa-renovate","edge-katanomi-app2","copilot-pull-request-reviewer","kilo-code-bot",…]`.
2. **`first_review_at` computation** skips bot reviewers when picking `min(submitted_at)`.
3. **`pr_reviews` row write** still happens — we want the audit trail — but they're tagged with a new `is_bot` column (migration `0005_pr_reviews_is_bot.sql`) and the aggregator filters on it for `prs_reviewed` and any future latency metric.
4. **`NetworkDensity`** queries already key off `first_review_at` and `pr_reviews`; once both sides are bot-aware, p50 / orphan / under-24h all start measuring **humans only**.

**Files**
- `backend/internal/contributions/bot.go` — new pure function `IsBotLogin(login string, cfg BotConfig) bool`.
- `backend/internal/storage/migrations/0005_pr_reviews_is_bot.sql` — add `is_bot INTEGER NOT NULL DEFAULT 0`.
- `backend/internal/github/sync.go`, `backend/internal/gitlab/sync.go` — set `is_bot` when writing reviews and when computing `FirstReviewAt`.
- `backend/internal/contributions/aggregator.go` — `WHERE NOT rv.is_bot`.
- `backend/internal/contributions/service_extras.go::NetworkDensity` — `WHERE NOT pr.first_review_by_bot` (or recompute on the fly).
- `backend/internal/config/config.go` — `TeamAnalytics.BotReviewers []string`.

**Migration**
- Add column with default `0`.
- On startup: one-time backfill task `UPDATE pr_reviews SET is_bot=1 WHERE …`.
- Re-derive `pull_requests.first_review_at` for affected PRs (one CTE: pick the earliest non-bot review per PR).

**Rollback.** Drop the column; revert config.

**Risk.** Headline numbers change visibly — first-review p50 will jump from 0.1 h to "real value, probably 4–10 h". Surface this in the release notes.

---

## W3 — Member exclusion / inactive handling (covers B6, B11)

**Problem.** `Guangxin Jiao[X]` and `Limei He[X]` are flagged as active in `members` because Jira's `active` field still says so. Maintainer wants both removed from the view and all calculations.

**Design.** Reuse W1's denylist:

```yaml
team_analytics:
  member_denylist:
    - gxjiao
    - lmhe
    - zhwang     # also wanted
    - chaozhou   # also wanted
```

The aggregator filters them out; the `/members` list filters them out; the `team` overview filters them out. No additional schema needed beyond W1.

In addition, optionally:

- **Heuristic recommendation.** Log a warning when a member's `display_name` ends in `[X]` or `[已离职]` so the operator sees a candidate for the denylist on the next reload. *Don't* auto-deny — false positives are too disruptive.

**Files** — same as W1.

**Tests** — covered by W1's tests.

---

## W4 — Sprint WIP / In-Progress / Done split (covers B7)

**Problem.** `SprintStats.WIP` lumps Backlog / Selected for Development / Blocked / In Progress together because the classifier only checks `resolved_at IS NULL`.

**Design.** Split into three buckets using the existing `status` column:

```go
type SprintStats struct {
    Name        string `json:"name,omitempty"`
    Todo        int    `json:"todo"`         // Backlog, Selected, Blocked, To Do, …
    InProgress  int    `json:"in_progress"`  // matches in_progress_statuses
    Done        int    `json:"done"`         // matches done_statuses OR resolved_at not null
    PRsOpen     int    `json:"prs_open"`
    PRsMerged   int    `json:"prs_merged"`
    PRsTotal    int    `json:"prs_total,omitempty"` // convenience
}
```

Status lists land under `team_analytics.statuses` so they can grow independently of the metrics-block config (which is DORA-focused):

```yaml
team_analytics:
  statuses:
    todo:        ["Backlog", "Selected for Development", "Blocked", "To Do", "Open", "待办", "已阻塞"]
    in_progress: ["In Progress", "In Development", "调研中", "调研完成", "设计完成", "开发完成", "测试完成", "验收完成"]
    done:        ["Done", "Closed", "Released", "已完成"]
```

Anything unmatched falls under `in_progress` (least-bad default — operator gets pinged to extend the lists). **OPEN QUESTION (Q2) on exact status names.**

**Files**
- `backend/internal/contributions/service_extras.go::sprintCounts` — use the new helper.
- `backend/internal/config/config.go` — add `TeamAnalytics.Statuses` struct.
- `frontend/...` — Member-profile sprint card needs a three-row layout; coordinate with whoever owns the React component (see Q3 — do we extend `prototype.html` first, or jump straight to JSX?).

**Risk.** Frontend coupling. If the frontend reads `wip` today, ship a graceful alias (`wip = todo + in_progress`) for one release.

---

## W5 — `pull_requests.epic_key` → `jira_key` rename (covers B8)

**Problem.** The column stores any `DEVOPS-\d+` mentioned in the source branch or title, not just epic keys. 0 of 4411 linked Jira issues in prod are actual Epics; the rest are Stories, Bugs, Jobs, Technical Debt.

**Proposed change.**

1. **Schema rename** — `0006_rename_epic_key_to_jira_key.sql`:
   ```sql
   ALTER TABLE pull_requests RENAME COLUMN epic_key TO jira_key;
   CREATE INDEX IF NOT EXISTS idx_pull_requests_jira_key ON pull_requests(jira_key);
   ```
   (SQLite's `ALTER TABLE RENAME COLUMN` is 3.25+ — fine.)
2. **Code rename** — `storage.PullRequest.EpicKey` → `JiraKey`; `Linker.Link` keeps its signature, just stores into the new field.
3. **API/JSON** — current JSON shape only exposes `jira_key` in member-profile sprint queries; existing callers read top-level summary fields, so we don't break wire format. **OPEN QUESTION (Q4)** — do we keep the JSON tag as `epic_key` for backward compat for one release?

**Why this matters.** Anyone reading the code today reasonably assumes `epic_key` filters to Epics. The sprint-PR query (`service_extras.go:535-546`) is correct by accident — it relies on the *misnamed* column matching *story* issue keys.

**Files**
- `backend/internal/storage/migrations/0006_rename_epic_key_to_jira_key.sql`
- `backend/internal/storage/store.go` — struct field rename + struct tag.
- `backend/internal/github/sync.go`, `backend/internal/gitlab/sync.go` — field rename.
- `backend/internal/contributions/service_extras.go` — SQL rename.
- `frontend/...` — if it reads `epic_key` anywhere.

**Risk.** Medium — touches every PR-row read site. Cover with `make test` and a sample profile query.

---

## W6 — GitHub / GitLab parity (covers B12)

**Maintainer asked: what can we do here?**

Three asymmetries, each with a choice:

1. **MR additions / deletions / changed_files.** Today only filled when `HydrateDiff=true` (off by default). Recommendation: **flip the default to `true`** and add a `team_analytics.gitlab_hydrate_diff: false` opt-out. Cost is ~1 extra API call per merged MR — at the current sync size (42 records/cycle) this is negligible. The benefit is that PR-size charts (planned for Phase 3) work for GitLab on day one.
2. **`changes_requested` review state.** GitLab has no first-class API for this. Closest analogues:
   - `/needs-rework`, `/changes-requested`, `/not-lgtm` magic comments in alauda's prow conventions. **OPEN QUESTION (Q5)** — do we agree on a marker?
   - GitLab approval rules — but those are project-config-dependent.
   - **Recommendation:** add a regex pattern in config (`team_analytics.review_classifiers.changes_requested: '^\s*/(needs-rework|not-lgtm)\s*$'`) and let teams adopt it; default empty. Until then we accept the asymmetry.
3. **Draft MR exclusion from review fetch.** GitHub has `pr.Draft`, GitLab has `mr.Draft` (also called WIP). The GitHub branch skips reviews for drafts; the GitLab branch doesn't. Trivial — mirror the github skip in `gitlab/sync.go:Sync`.

**Files**
- `backend/internal/gitlab/sync.go` — `s.HydrateDiff = true` default; draft skip.
- `backend/internal/gitlab/sync.go::classifyNote` — extend with optional regex.
- `backend/internal/config/config.go` — new fields.

**Risk.** GitLab API budget. Monitor `record_count` in the next sync cycle.

---

## W7 — Window: 12 months, bucketed by quarter (covers B13)

**Maintainer ask:** "I want to change the whole collection to span last 12 months, but split by quarter — feasible?"

**Yes. Two pieces:**

### W7a — Collection (ingest) — extend to 365 days

- `backend/internal/storage/config.go::Storage.BackfillDays = 365` (today 180).
- `github.backfill_days`, `gitlab.backfill_days`, `jira.backfill_days` either inherit `storage.backfill_days` (today's behaviour when set to `0`) or get overridden — recommend leaving the inherits in place and bumping `storage.backfill_days` only.
- `aggregator.RebuildRecent` default window — bump from 187 to **400** so the first rebuild after deploy fully populates the year.

Storage cost: prod DB is 16 MB after ~6 months. Doubling the window → roughly 30-35 MB. Negligible.

API cost: backfill is a one-shot. Subsequent syncs remain incremental from the last `collection_run`.

### W7b — Aggregation (read) — quarter buckets

Two ways to do this; pick one (see Q6):

**Option A — Aggregate at read time.** Keep `member_week_metrics` weekly; the API endpoint accepts `?bucket=quarter` (default `week`). The handler folds weekly rows into quarters in Go before returning. Trivial code, no new schema, but every query does the fold.

**Option B — Add `member_quarter_metrics`.** A second rollup, written by the aggregator in parallel with the weekly one. Faster reads, but two writers to keep in sync.

**Recommendation: Option A** for now, promote to Option B if a quarter-level query is ever slow enough to matter. Profile after rollout.

**Quarter boundary.**

- Calendar quarters: Q1 = Jan-Mar, etc.
- Alauda fiscal year? **OPEN QUESTION (Q7)** — confirm calendar vs. fiscal.
- Default to calendar quarters unless we hear otherwise.

**API shape.** Extend `MemberWeekQuery` → `MemberPeriodQuery` (or add a `Bucket` field) and let the handler decide. The frontend gets:

```json
{
  "members": [
    {
      "member_id": "daniel",
      "quarter_totals": [
        {"quarter": "2025-Q2", "jira_done": 32, "points": 27, "prs_merged": 180, ...},
        {"quarter": "2025-Q3", ...},
        {"quarter": "2025-Q4", ...},
        {"quarter": "2026-Q1", ...},
        {"quarter": "2026-Q2", ...}
      ],
      ...
    }
  ],
  "from": "2025-05-25T00:00:00Z",
  "to":   "2026-05-25T00:00:00Z"
}
```

(week_totals stays the default; quarter_totals only on `?bucket=quarter`.)

**Files**
- `backend/internal/storage/config.go`
- `backend/internal/contributions/aggregator.go` — RebuildRecent default
- `backend/internal/contributions/service.go` — add `bucketByQuarter` helper
- `backend/internal/api/handlers/contributions.go` — parse `?bucket=`
- Frontend — new view tab for quarter; coordinate.

**Risk.** Medium — first deploy will re-run a full 365-day backfill. Run during off-hours. Watch `collection_runs.duration_ms`.

---

## W8 — `review_latency_p50_hours` (covers B3)

**Maintainer didn't comment on this finding.** Two options:

- **A.** Implement it (the TODO at `aggregator.go:246-248`). Once W2 lands, the underlying data is honest enough to be useful.
- **B.** Hide the field until A lands — the API and Member Profile drop the row so users don't think "0 h" is real.

**Recommendation:** **A**, after W2 ships, because W4-W7 don't unblock it and the field is already plumbed end-to-end. Cost is one calculator implementation, mirroring the `cycle_time` shape. **OPEN QUESTION (Q8) — confirm.**

---

## W9 — B2 / B5 pillar plumbing (HIGH severity in the audit, but design-dependent)

**Problem.** `member_week_metrics.pillar_id` and `.component` are always `''`. `cross_pillar_review_pct` is always `0`. `members.pillar_id` is `null` for everyone.

**Maintainer comment on B9:** "pillars[] derived from member work is by design — member work can traverse different pillars."

That makes the `pillars[]` field (B9) fine as-is. **But** the `members.pillar_id` column is then completely vestigial — nothing writes it, nothing reads it usefully, and the cross-pillar review % relies on it being populated.

**Recommendation, in order of preference:**

1. **Drop `members.pillar_id` from the schema and the PATCH endpoint.** Replace `cross_pillar_review_pct` with a different metric — say "cross-component review %" — derived from the same per-issue / per-repo pillar attribution that `PillarThroughput` already uses. The author of a PR is mapped to a pillar via the PR's repo; the reviewer is mapped via *their own* most-recent PR's repo (or via the same Jira-issue mapping if available). Subtle, but doesn't depend on a hand-curated `members.pillar_id` field.
2. **Keep `members.pillar_id` but populate it from a new `team_analytics.member_pillars` config block.** Operator declares each member's "home" pillar; PATCH overrides; cross-pillar review compares home pillars.
3. **Drop the cross-pillar review % metric entirely.** It hasn't been useful to anyone since launch.

**OPEN QUESTION (Q9) — which path?**

Once decided, the corresponding W9 lands in one PR.

---

## W10 — Window default label (covers B13, NIT)

`parseQuery` defaults to `[MondayOf(now-84d), MondayOf(now+7d))`. The label "last 12 weeks" is fine; the `to` being in the future is fine for SQL but reads weirdly in the API JSON. Two tiny tweaks:

- Cap `q.To = min(MondayOf(now+7d), MondayOf(now))` so the response never claims data through next Monday.
- Frontend label: align with the actual window.

Combine into the W7 PR.

---

## Order & sequencing

```
W1 ─────► W3 (denylist consumed by W1)
W1 ─────► W7b (quarter aggregation depends on the allowlisted member set)
W2 ─────► W8 (review latency wants the bot filter first)
W5 (rename) — independent
W6 (parity) — independent
W9 (pillars) — independent, gated on Q9
```

Suggested rollout sprints (2-week cadence):

1. **Sprint 1 (this sprint).** W1 + W3 — fix the "who counts" story. Single PR.
2. **Sprint 2.** W2 + W8 — fix the review story. Two PRs, second after first lands.
3. **Sprint 3.** W7 (12 months / quarter) + W10 — biggest visible change. One PR.
4. **Sprint 4.** W5 (rename) + W6 (parity) — janitorial.
5. **Sprint 5.** W9 — depends on Q9 answer.

---

## Open questions (please answer before execution)

**Q1 — B1 scope of "instance-wide".**
You said "GitLab instance-wide scan, focus on devops member contributions." Two interpretations of what we should ingest:
- **(a)** *Keep `with_shared=true` AND remove `devops/**` from the spec entirely → scan every project on `gitlab-ce.alauda.cn`.* Maximum coverage; expensive; lots of noise we filter at calculation time via the allowlist.
- **(b)** *Keep `groups: ["devops/**"]` but switch `with_shared=false`.* Plus W1's allowlist drops non-DEVOPS authors. The instance-wide-ness is "DEVOPS team's own subgroups, recursively". My recommendation.
- **(c)** *Open: scan a much wider but still-curated set, e.g. `devops/**`, plus members' personal namespaces (`mingfu/*`, `daniel/*`, …).*

Which?

**Q2 — Status name lists.** I drafted three buckets in W4. Are those names exhaustive for our workflows, or are there others (custom DEVOPS statuses we don't see in the audit sample)?

**Q3 — Frontend track.** This whole plan touches member-profile / team-overview JSX. Should I open a sister branch in `frontend/` per PR, or batch the frontend changes into a single follow-up PR?

**Q4 — `epic_key` rename.** Keep the JSON tag as `epic_key` for one release for backward compat, or hard-rename to `jira_key` immediately? (No external API consumers that I'm aware of, but worth confirming.)

**Q5 — GitLab `changes_requested` marker.** Do we already have a convention (`/needs-rework`, `/not-lgtm`, something else) we want to honour? If not, defer.

**Q6 — Quarter aggregation strategy.** Read-time fold (Option A, simpler) or a parallel rollup table (Option B, faster reads)? I lean A.

**Q7 — Quarter boundary.** Calendar (Q1 = Jan-Mar) or Alauda fiscal year? If fiscal, where does FY start?

**Q8 — `review_latency_p50_hours`.** Implement (after W2) or hide? I lean implement.

**Q9 — Pillar attribution path.** Drop `members.pillar_id`, populate via config, or drop the cross-pillar review % metric? I lean "drop members.pillar_id + redefine cross-pillar review % from PR-repo and issue-component attribution."

**Q10 — Bot deny set seed.** Confirm the initial list — `alaudabot`, `alaudaa-renovate`, `edge-katanomi-app2[bot]`, `copilot-pull-request-reviewer[bot]`, `kilo-code-bot[bot]`. Anything obvious I missed?

**Q11 — Member denylist seed.** Confirm: `gxjiao`, `lmhe`, `zhwang`, `chaozhou`. Any others?

**Q12 — Backfill cost.** W7's 365-day backfill will hammer GitHub + GitLab on first run. Acceptable to run during business hours, or do we want a manual gate / scheduled overnight kick?

---

*If the answers to Q1–Q12 land in a single Discord reply, I can convert each W block into a tracked Jira task and start executing in order.*
