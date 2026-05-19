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

## W1 — Member-author-driven ingest, with denylist (covers B1, B10, B11) — **REVISED 2026-05-19**

**Problem recap.** Today the GitLab sync over-ingests because `with_shared=true` is the GitLab default; the aggregator credits MRs solely on `LOWER(m.gitlab_username) = pr.author_login`, so zhwang's whole rollup is from `container-platform/*` / `ops/*` / `alauda/artifacts`.

**Maintainer answer (Q1).** "Our members can contribute in very different groups and repos, their contributions should be counted as well." → we do NOT want to restrict the ingest scope to `devops/**`; on the contrary, we want **wider** coverage so that, say, daniel's MRs in `alaudadevops/*` already work and qingliu's MRs in `container-platform/*` would also count if they ever happen.

**Revised design — member-driven ingest, not group-driven.**

1. **Two ingest passes, both honouring the allowlist** below:
   - **Pass A (existing).** Per-project sync of `devops/**` (subgroups recursive) — captures team review participation and PRs by non-members reviewing our members' work. **Set `with_shared=false`** so we don't pull in the shared-project graph (this was the root cause of B1, and we still don't want it). The `devops/**` scope here is just "the projects our team owns"; it's not a member-filter, it's a team-context capture.
   - **Pass B (new).** Per-member sweep of MRs across the entire GitLab instance. For each member in the allowlist:
     - `GET /api/v4/merge_requests?scope=all&author_username=<u>&updated_after=<since>&per_page=100` (page through). With the bot's PAT scope (`read_api`) this returns every MR they can see — for our members, that's the entire instance up to private-project visibility. Reviews on those MRs come from the same notes-as-reviews mechanism. The same `with_shared` flag does not apply at this endpoint.
   - **De-duplication** is automatic because `pull_requests.id = "<group/proj>!<iid>"` is unique across passes.
2. **`team_analytics.members` becomes the source of truth for "who counts".**
   - **Recommended:** *derive* the allowlist from the union of `github_login_prefills` ∪ `gitlab_username_prefills` keys (already curated, already declared once in the ConfigMap, no new config to maintain).
   - **Explicit knob:** `team_analytics.member_denylist: ["zhwang","chaozhou","gxjiao","lmhe"]` — confirmed by maintainer.
3. **Aggregator** gains a single `WHERE COALESCE(m.id, pr.author_id) IN (…allowlist…)` after the existing JOIN. Everything outside the allowlist drops from `member_week_metrics` and from every downstream API.
4. **`/api/contributions/members`** filters to the same set; `include_inactive` stays for the (less reliable) `members.active` dimension.
5. **`PillarThroughput`** keeps counting every ingested MR / issue (team-level chart), but the bot work goes into the synthetic `bot` member from W2 instead of staying anonymous; out-of-scope-author noise drops because the dominant contaminator (`alauda/artifacts`, `container-platform/*`) had no allowlisted human authors in our window.

**Files**
- `backend/internal/gitlab/client.go` — add `q.Set("with_shared", "false")` to `ListGroupProjects`. Add a new `ListInstanceMergeRequests(ctx, opts)` calling `/api/v4/merge_requests?scope=all&…`.
- `backend/internal/gitlab/sync.go` — second sweep loop after `resolveProjects`; reuse the same upsert path so the storage shape is identical.
- `backend/internal/config/config.go` — `TeamAnalytics.MemberDenylist []string` (allowlist is derived, no field).
- `backend/internal/contributions/allowlist.go` — new helper computing the effective set from the prefill maps minus the denylist; cached at startup, refreshed on config reload.
- `backend/internal/contributions/aggregator.go` — wire the allowlist into the four INSERT SELECT statements (one extra IN clause + a parameter slice).
- `backend/internal/api/handlers/contributions.go::ListMembers/TeamOverview` — also filter on the same set.

**Migration**
- No schema change.
- One-shot post-deploy: `DELETE FROM member_week_metrics WHERE member_id NOT IN (…allowlist…)` for cleanliness, then `aggregator.RebuildRecent(400)` (note the bumped window from W7).
- Out-of-scope `pull_requests` rows stay in the table — they're useful for repo-level metrics and they cost nothing.

**Rollback.** Empty denylist + skip Pass B = today's behaviour. Two boolean flags can gate the new sweep if we need to revert under live traffic.

**Tests**
- Aggregator unit test: 3 PRs by 3 logins, only one allowlisted; rebuild; assert 1 row in rollup.
- Aggregator unit test: denylist removes an allowlisted user.
- GitLab sync test (mock client): two members each have 2 MRs in different namespaces; sweep returns 4 rows; second sync run de-dups against the existing PR set.

**Risk.** Medium. New API surface (`/merge_requests?scope=all`) has different pagination semantics — verify the page-size + `X-Total-Pages` behaviour in a smoke test before relying on it in prod.

---

## W2 — Bot contributions consolidated under one synthetic `bot` member (covers B4) — **REVISED 2026-05-19**

**Maintainer answer (Q10).** "I lean towards merging bot contribution in one fictional Bot user." → instead of *hiding* bot rows, we **reassign** every bot author / reviewer to a single member id (`bot`, which already exists in the directory) and let the dashboard show "Bot did N reviews" as one row. Humans see honest first-review latency; the team still sees the volume of automated work.

**Design.**

1. **One bot predicate in one place.** Per Q10-B: **explicit list only, no suffix magic.**
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
   `IsBotLogin(login string) bool` is a flat lowercased-string-set lookup. New bot accounts get added to the ConfigMap and a hot reload picks them up.
2. **Member resolution change.** In `github/sync.go::byLogin` and `gitlab/sync.go::byUsername`, after the existing lookup, fall back to `"bot"` (a member that already exists with `id="bot"` and empty logins) when `IsBotLogin(login)` returns true. That means:
   - `pull_requests.author_id` becomes `"bot"` for renovate / scan-bots / etc.
   - `pr_reviews.reviewer_id` becomes `"bot"` for automated comments.
3. **`first_review_at` calculation** still selects `min(submitted_at)` across *all* reviews including bot ones — but the **NetworkDensity panel** computes its p50 from `pull_requests` where the first reviewer is not a bot. New helper column `pull_requests.first_human_review_at` (migration `0005_first_human_review.sql`) keeps the SQL straightforward.
4. **`pr_reviews.is_bot`** column joins the schema for analytical convenience even though resolver maps to `"bot"` member — this lets us answer "how much bot noise are we generating per pillar" without an extra lookup table.
5. **Allowlist (W1)** must **always** include `"bot"` so its rollup row stays visible in the team overview. The dashboard treats it as just another member.

**Files**
- `backend/internal/contributions/bot.go` — pure `IsBotLogin(login string, cfg BotConfig) bool`.
- `backend/internal/storage/migrations/0005_first_human_review.sql` — add `is_bot INTEGER NOT NULL DEFAULT 0` on `pr_reviews`; add `first_human_review_at TIMESTAMP` on `pull_requests`.
- `backend/internal/github/sync.go`, `backend/internal/gitlab/sync.go` — set `is_bot`; compute `first_human_review_at`; reassign `author_id` / `reviewer_id` to `"bot"` when the login matches.
- `backend/internal/contributions/service_extras.go::NetworkDensity` — bind on `first_human_review_at`; `JOIN ... AND rv.is_bot = 0` for cross-pillar.
- `backend/internal/config/config.go` — `TeamAnalytics.BotReviewers []string`.

**Migration**
- Two columns added with sensible defaults.
- One-shot backfill (Go, not raw SQL — needs the bot list): walk `pr_reviews`, set `is_bot` where `IsBotLogin(reviewer_login)`. Walk `pull_requests`, set `first_human_review_at = MIN(submitted_at) WHERE NOT is_bot`. Reassign `author_id` / `reviewer_id` to `"bot"` for the matching rows.
- `aggregator.RebuildRecent(400)` after.

**Rollback.** Two new columns are additive. Reverting the resolver shift requires re-running the backfill in reverse — easier said than done, so the safe rollback is "revert the resolver code but leave the bot rows assigned to `bot`" — the dashboard tolerates that.

**Risk.** Numbers will move visibly: first-human-review p50 jumps from 0.1 h to whatever the real number is, and the `bot` member will likely top the team table for `prs_merged` (renovate alone has hundreds in the window). Headline this in the release notes.

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

## W4 — Sprint TODO / In-Progress / Done split (covers B7) — **REVISED 2026-05-19**

**Problem.** `SprintStats.WIP` lumps Backlog / Blocked / In Progress together because the classifier only checks `resolved_at IS NULL`.

**Maintainer answer (Q2).** "Can you query Jira directly and cross-check?" → done. DEVOPS uses **different status sets per issue type**:

| issue type | statuses |
|---|---|
| Bug, Story, Technical Debt | Open, Designing, Developing, Testing, Acceptance Testing, Ready for Delivery, Done, Cancelled |
| Improvement, Vulnerability, Task, Sub-task, Pillar, Milestone, Document | Backlog, Blocked, In Progress, In Testing, Ready for QA, Resolved, Signed Off, Test Failed, Cancelled |
| Epic | 待处理, 调研中, 调研完成, 设计完成, 开发完成, 测试完成, 验收完成, 阻塞中, 已完成, 已取消 |
| Job | Open, In Progress, Under Review, Ready for Delivery, Done, Cancelled |
| Customer Reported Incident | Open, In Progress, Under Review, Verify, Wait for Verify, Mitigated, Resolved |
| Platform application | Open, In Progress, Under Review, CONFIRM RELEASE, DEPLOY, using, Done, Cancelled |
| Components / Components-sub-task | In Progress, Components-test, Ready for QA, Test Failed, Done, Cancelled |

**Proposed 3-bucket mapping** (status order matches what the Jira API returns — all checked names below are real statuses I pulled live on 2026-05-19):

```yaml
team_analytics:
  statuses:
    todo:
      - "Backlog"
      - "Blocked"
      - "Open"
      - "待处理"
      - "阻塞中"
    in_progress:
      - "Acceptance Testing"
      - "CONFIRM RELEASE"
      - "Components-test"
      - "DEPLOY"
      - "Designing"
      - "Developing"
      - "Doc Reviewing"
      - "In Progress"
      - "In Testing"
      - "Mitigated"
      - "Ready for Delivery"
      - "Ready for Doc Review"
      - "Ready for QA"
      - "Review/Test Failed"
      - "Signed Off"
      - "Test Failed"
      - "Testing"
      - "Under Review"
      - "Verify"
      - "Wait for Verify"
      - "调研中"
      - "调研完成"
      - "设计完成"
      - "开发完成"
      - "测试完成"
      - "验收完成"
    done:
      - "Done"
      - "Resolved"
      - "Cancelled"
      - "using"
      - "已完成"
      - "已取消"
```

Open mapping calls:
- **"Cancelled"** put under `done` because the work no longer takes capacity, even though it wasn't delivered. **Q2-follow-up A:** should we add a fourth bucket `cancelled` so velocity charts can exclude it? Default is `done`.
- **"Ready for Delivery"** put under `in_progress` because it's still QA / release work in flight. **Q2-follow-up B:** confirm.

```go
type SprintStats struct {
    Name        string `json:"name,omitempty"`
    Todo        int    `json:"todo"`
    InProgress  int    `json:"in_progress"`
    Done        int    `json:"done"`
    PRsOpen     int    `json:"prs_open"`
    PRsMerged   int    `json:"prs_merged"`
    PRsTotal    int    `json:"prs_total,omitempty"`
}
```

Anything that doesn't match any list falls back to `in_progress` and the operator gets a warning log so they can extend the config.

**Files**
- `backend/internal/contributions/service_extras.go::sprintCounts` — use a status-classification helper.
- `backend/internal/config/config.go` — add `TeamAnalytics.Statuses` struct + defaults (the lists above, so the config can stay empty in most installs).
- `frontend/...` — Member-profile sprint card needs a three-row layout. Per Q3 below — coordinate inside the same PR.

**Risk.** Backward compat: if any caller reads `wip`, we keep it as a derived alias (`wip = todo + in_progress`) for one release tagged for removal.

---

## W5 — `pull_requests.epic_key` → `jira_key` rename (covers B8) — **CLARIFICATION 2026-05-19**

**Maintainer asked (Q4):** "don't understand the issue here, elaborate."

**The issue, in detail.**

1. The schema column is named **`epic_key`** (migration `0001_init.sql`):
   ```sql
   CREATE TABLE pull_requests (
       ...
       epic_key TEXT,
       ...
   );
   ```
2. The Go struct field is named **`EpicKey`** with JSON tag `epic_key` (`backend/internal/storage/store.go`):
   ```go
   type PullRequest struct {
       ...
       EpicKey string `json:"epic_key,omitempty"`
       ...
   }
   ```
3. The PR-linker in `backend/internal/gitlab/sync.go:48` (and the github one) sets this column via a regex that matches **any** Jira key in the source branch or title:
   ```go
   regexp.MustCompile(`(?i)\b(` + regexp.QuoteMeta(projectKey) + `-\d+)\b`)
   ```
4. In prod (4411 PRs with a linked key), the issue types of the matched Jira issues are:

   | issue type | count |
   |---|---|
   | Story | 2817 |
   | Bug | 996 |
   | Job | 199 |
   | Technical Debt | 195 |
   | Document | 118 |
   | Sub-task | 57 |
   | Task | 25 |
   | Improvement | 4 |
   | **Epic** | **0** |

   Zero of the rows ever point at an actual Epic.

**Why this is a bug worth a rename.**

- A maintainer reading `pull_requests.epic_key` reasonably assumes "filter PRs to only those that landed in an Epic". The query `WHERE epic_key IS NOT NULL` does NOT mean that.
- The sprint stats query (`service_extras.go:535-546`) joins `pr.epic_key = issue_snapshots.issue_key WHERE sprint_id = ?` — this happens to be **correct on accident** because most sprint members are Stories, and `epic_key` happens to store Stories. If anyone later writes a similar query expecting actual-Epic semantics, they'll silently get wrong numbers.
- Future readers grepping for the right column name will look for `jira_key` first; the misnomer wastes their time.

**Why I asked Q4 specifically.**

The rename is internal — DB column + Go field + SQL queries — but the **JSON tag** is visible to any API client that reads PR metadata. If something outside the repo (a script, the frontend) consumes `"epic_key"` from a response, renaming the tag breaks it.

Audit: the JSON tag `epic_key` is **not** in the response of any currently shipped endpoint. The PR list is not exposed; only counts are. So the JSON-tag rename is safe.

**Decision: hard-rename both DB column and JSON tag in the same migration.** No alias.

**Files** (unchanged from the draft above):
- `backend/internal/storage/migrations/0006_rename_epic_key_to_jira_key.sql`
- `backend/internal/storage/store.go` — struct field + tag rename.
- `backend/internal/github/sync.go`, `backend/internal/gitlab/sync.go` — field rename.
- `backend/internal/contributions/service_extras.go` — SQL rename.
- Frontend grep — currently no hits for `epic_key`; safe.

**Risk.** Medium — many compile sites, but the failure mode is loud (build error or missing column). Covered by `make test`.

---

## W6 — GitHub / GitLab parity (covers B12) — **REVISED 2026-05-19**

**Maintainer answers.**
- **Q5 (changes_requested marker):** "defer, we generally don't do that in gitlab." → dropped from this workstream.
- **Q12 (extra API cost):** "backfill can be schedule overnight." → fine to flip `HydrateDiff` on without worrying about the one-shot spike.

**Final scope of W6:**

1. **MR additions / deletions / changed_files** — flip `gitlab/sync.go:110` default `HydrateDiff: false` → `true`. Add config opt-out `team_analytics.gitlab_hydrate_diff: false` for installs that hit rate limits. Cost is ~1 extra API call per merged MR; the current sync ingests ~50 MRs per cycle, so this is roughly +2 % of GL API budget per cycle and ~30 min one-shot extra during the W7 backfill (runs overnight per Q12).
2. **Draft MR review-fetch skip** — `gitlab/sync.go:Sync` mirrors the github branch's `if pr.Draft { skipReviews = true }` pattern. Saves a per-draft API call and prevents drafts polluting `first_review_at`.
3. ~~`changes_requested` marker~~ — **deferred per Q5.**

**Files**
- `backend/internal/gitlab/sync.go` — default flip; draft skip.
- `backend/internal/config/config.go` — `TeamAnalytics.GitLabHydrateDiff *bool` (pointer so absent ≠ false).

**Risk.** Negligible after Q12 confirmation.

---

## W7 — 12-month window, release-cadence "quarter" buckets (covers B13) — **REVISED 2026-05-19**

**Maintainer answers.**
- **Q6 (aggregation strategy):** "start with read time fold, we can split if necessary later" → keep `member_week_metrics` weekly; fold in handler.
- **Q7 (quarter boundary):** "actually our quarters are not real quarters it is a version cadence, it can be split by the current quarters split already in jira (2026Q1 2026Q2 2026Q4) which directly relates to our release dates (quite weird, I know)" → quarters track release cadence, not calendar.
- **Q12 (timing):** "backfill can be schedule overnight." → fine.

### W7a — Collection (ingest) — extend to 365 days
- `Storage.BackfillDays = 365` (today 180).
- `github.backfill_days`, `gitlab.backfill_days`, `jira.backfill_days` already inherit `storage.backfill_days` when set to `0`, leave as-is.
- `aggregator.RebuildRecent` default window: bump from 187 to **400** so the first rebuild after deploy fully populates the year.

Storage cost: prod DB is 16 MB after ~6 months → ~30-35 MB after a year. Negligible.

API cost: one-shot backfill scheduled overnight per Q12 above. Incremental syncs from `collection_runs.captured_at` thereafter.

### W7b — Quarter source: Milestone-prefix chain (confirmed 2026-05-19)

**Maintainer answer (Q7):** "They are not labels, they are written inside a Milestone Jira issue (automatically added as a prefix in the milestone name) and the epics are linked with a blocking link to each of the milestones. Only issue will be dangling issues or epics that are not inside a milestone, from that we can use resolution_date or other date fields to infer."

**Verified on live Jira (2026-05-19).**

- Milestones in DEVOPS look like:
  - `DEVOPS-43598 — 2026Q2：Tekton Pipelines - Restricted 安全策略适配/Tekton Hub下线`
  - `DEVOPS-42014 — 2026Q1：Security patches in 2026Q1`
  - `DEVOPS-43568 — 2026Q4：Connectors final adjustments`

  Note the **full-width Chinese colon `：` (U+FF1A)** — not the regular ASCII `:`. The regex has to accept both.
- Each Milestone has `issuelinks` of `type.name = "Blocks"`; the inward issue is an Epic. So the chain is **Epic ←Blocks— Milestone**, and the quarter prefix lives on the Milestone summary.
- Dangling cases:
  - Issue has no Epic Link → fall back to its own `resolution_date` (or `merged_at` for the PR side) and bucket into the calendar quarter the event landed in.
  - Epic has no Milestone blocker → same fallback using Epic's `resolution_date`.

**Resolver design.**

1. **Jira sync extension.**
   - Existing pass already fetches Epics and Stories/Bugs/etc.
   - New pass: fetch every issue with `issuetype = Milestone` updated in the window. Parse the prefix `^(\d{4}Q[1-4])[：:]` from `summary` → `milestone.quarter_label`. Walk `issuelinks` for `type=Blocks, inwardIssue.type=Epic` → record `(epic_key, quarter_label)` pairs.
   - Store in a new table `quarter_assignments(issue_key TEXT PRIMARY KEY, quarter_label TEXT, source TEXT)`. `source ∈ {"milestone","fallback"}`. Migration `0008_quarter_assignments.sql`.
   - On each Jira sync, refresh the rows touched in the window.
2. **Epic → Quarter** lookup is O(1) once `quarter_assignments` is populated.
3. **Story/Bug/Task → Quarter** lookup walks the issue's Epic Link (`customfield_10005`) → consult `quarter_assignments[epic_key]`. If missing, fall back to `2026Q<calendar-quarter-of-resolved_at>`.
4. **PR → Quarter** lookup walks `pull_requests.jira_key` (W5) → issue → quarter (as above). PRs without a Jira key fall back to `2026Q<calendar-quarter-of-merged_at>`.

Quarter labels are returned literally from the milestone summaries — no need to invent a calendar mapping. Calendar-quarter fallbacks use the obvious Jan-Mar=Q1 etc. mapping so dangling items still land somewhere.

### W7c — API + frontend shape

Add `?period=week|release_quarter` (default `week`). On `period=release_quarter`, fold weekly rows: each row's PR set → resolve quarter via the chain above → sum into the per-quarter bucket. The output still includes a `dangling: int` count per period so the UI can show "and N items without a quarter assignment".

```json
{
  "members": [
    {
      "member_id": "daniel",
      "period_totals": [
        {"period": "2026Q1", "from": "...", "to": "...", "jira_done": 32, "points": 27, "prs_merged": 180, ...},
        {"period": "2026Q2", ...},
        {"period": "2026Q3", ...},
        {"period": "2026Q4", ...}
      ],
      ...
    }
  ],
  "from": "2025-05-25T00:00:00Z",
  "to":   "2026-05-25T00:00:00Z"
}
```

**Files**
- `backend/internal/storage/config.go` — `BackfillDays = 365`.
- `backend/internal/storage/migrations/0008_quarter_assignments.sql` — new table `(issue_key, quarter_label, source)`.
- `backend/internal/jirasync/sync.go` — milestone pass + `quarter_assignments` upserts. Regex `^(\d{4}Q[1-4])[：:]\s*` for the prefix.
- `backend/internal/contributions/aggregator.go` — `RebuildRecent` default 400 days; quarter folding helper.
- `backend/internal/contributions/period.go` — `QuarterFor(jiraKey, fallbackTime) (string, source)` looking up `quarter_assignments` with the dangling fallback.
- `backend/internal/contributions/service.go` — `bucketByPeriod` that swaps `WeekTotals` for `PeriodTotals`.
- `backend/internal/api/handlers/contributions.go` — parse `?period=`.
- Frontend — new tab "Quarter" inside each chart; same PR.

**Risk.** Medium. New table + new Jira sync pass; quarter fallback path needs unit-test coverage so dangling items don't silently land in the wrong bucket.

---

## W8 — `review_latency_p50_hours` (covers B3) — **CONFIRMED 2026-05-19**

**Maintainer answer (Q8):** "implement."

**Design.** Per-(reviewer, week) p50 of `(first_human_review_at - created_at)` — depends on W2 having shipped `first_human_review_at` so the metric measures human review time. Aggregator query mirrors the existing `prs_reviewed` insert, but uses Go-side sort+percentile (same trick `NetworkDensity` uses, see `service_extras.go:104-137`).

**Files**
- `backend/internal/contributions/aggregator.go` — new `reviewLatency` step, INSERT into the existing `review_latency_p50_hours` column.
- `backend/internal/contributions/aggregator_test.go` — table-driven test for known input.

**Risk.** Low. Schema and API already accept the field.

**Depends on:** W2.

---

## W9 — Pillar plumbing redesign (covers B2, B5; B9 stays as-is) — **CONFIRMED 2026-05-19**

**Maintainer answer (Q9): option (i)** — drop `members.pillar_id`; redefine cross-pillar review % using PR-repo + Jira-component attribution.

**Design.**

1. **Schema migration `0007_drop_members_pillar.sql`:**
   ```sql
   ALTER TABLE members DROP COLUMN pillar_id;
   -- member_week_metrics.pillar_id column kept for now (always ''); the
   -- aggregator-write change to make it useful is out of scope of this W9.
   ```
2. **API surface:**
   - `PATCH /api/contributions/members/:id` drops `pillar_id` from the accepted body.
   - The `Member` JSON shape drops `pillar_id` (or returns it always `null` for one release).
3. **`cross_pillar_review_pct` redefined.** New helper `pillarsForPR(pr) []string`:
   - Primary: `PillarMap.PillarsForRepo(pr.repo_id)`.
   - Fallback: if the PR has a `jira_key` (W5), the Jira issue's `components` / `versions` map via `PillarMap.PillarsFor(comps, vers)`.
   - Else: synthetic "Unassigned" — excluded from the denominator (same as today).
   - For a review row, the reviewer's "side" is the union of pillars across the reviewer's *own* recent PRs in the same window (cheap to compute: one extra subquery per panel load). A review is "cross-pillar" when the PR's pillar set and the reviewer's pillar set are disjoint AND both are non-empty.
4. The frontend's "this member's pillars" card already pulls from B9's per-member-attribution map; no change needed there.

**Why this is non-trivial.** Today's metric (`service_extras.go:139-162`) compares `members.pillar_id` directly. The new metric needs:
- A per-PR pillar resolver (cheap; reuses existing config).
- A per-reviewer "what pillars are they active in" resolver (one extra SQL window per panel render).

We can cache the reviewer→pillar resolution per-window inside the handler to avoid repeated SQL.

**Files**
- `backend/internal/storage/migrations/0007_drop_members_pillar.sql`.
- `backend/internal/storage/store.go::Member`, `MemberIdentity` — drop `PillarID`.
- `backend/internal/api/handlers/contributions.go::UpdateMember` — drop `pillar_id`.
- `backend/internal/contributions/service_extras.go::NetworkDensity` — rewrite the cross-pillar query as described.
- `backend/internal/contributions/prefills.go` — drop the (unused) write path.

**Risk.** Medium. SQL change in a hot path. Cover with a unit test where 2 PRs land in distinct pillars, 1 review is by an author in the other pillar → 50 %.

**Depends on:** W5 (uses `jira_key`).

---

## W10 — Window default label (covers B13, NIT)

`parseQuery` defaults to `[MondayOf(now-84d), MondayOf(now+7d))`. The label "last 12 weeks" is fine; the `to` being in the future is fine for SQL but reads weirdly in the API JSON. Two tiny tweaks:

- Cap `q.To = min(MondayOf(now+7d), MondayOf(now))` so the response never claims data through next Monday.
- Frontend label: align with the actual window.

Combine into the W7 PR.

---

## Order & sequencing — **REVISED 2026-05-19**

```
W1 (allowlist + instance-wide MR sweep) ───► W2 (bot consolidation)
                                              │
                                              ├──► W8 (review latency p50, needs first_human_review_at)
                                              └──► W9 (cross-pillar % rewrite, needs W5 for jira_key)
W5 (epic_key→jira_key) ───────────────────────┘
W4 (sprint TODO/IP/Done) — independent of all
W6 (gitlab parity) — independent
W7 (12-month + release-quarter buckets) — gated on Q7-follow-up
W10 (window bound + label tidy) — folded into W7
W3 — folded into W1 (same denylist)
```

Suggested rollout, two-week cadence, one PR per workstream unless noted:

1. **Sprint 1 (this sprint, after answers below land):** W1 (single PR, includes W3) — "who counts" + GitLab `with_shared=false` + per-member sweep. **Visible effect:** zhwang/chaozhou/gxjiao/lmhe disappear from the dashboard; allowlisted members' counts may rise slightly if they had MRs in `container-platform/*` etc.
2. **Sprint 2:** W2 (PR2) + W4 (PR3). **Visible effect:** team's first-review p50 jumps from 0.1 h to something realistic; a new `bot` row tops the team table; sprint card shows three columns.
3. **Sprint 3:** W5 (PR4) + W6 (PR5). Janitorial.
4. **Sprint 4:** W7 (PR6, big), W10 folded in; W8 (PR7); W9 (PR8) — these three can land in parallel once Sprint 3 ships.

---

## Open questions — all resolved (final 2026-05-19)

| | original Q | resolution |
|---|---|---|
| Q1 | scope of "instance-wide" | members can contribute everywhere → ingest instance-wide, filter by author at calc time (W1 rewritten) |
| Q2 | sprint status lists | pulled live from Jira; full mapping in W4 |
| Q2-A | Cancelled bucket | stays in `done` (per maintainer) |
| Q2-B | Ready for Delivery | `in_progress` (no objection raised) |
| Q3 | frontend track | one PR per workstream, backend + frontend together (no objection raised) |
| Q4 | `epic_key` rename | confirmed hard-rename DB + Go + JSON in one migration |
| Q5 | GL `changes_requested` | deferred |
| Q6 | quarter aggregation | read-time fold |
| Q7 | quarter source | Milestone-prefix chain, verified live (W7b) |
| Q8 | review_latency_p50_hours | implement after W2 |
| Q9 | pillar attribution | option (i) — drop `members.pillar_id`, redefine cross-pillar % from PR-repo + Jira-component |
| Q10 | bot handling | reassign to synthetic `bot` member |
| Q10-B | bot seed | `alaudabot, alaudaa-renovate, edge-katanomi-app2[bot], copilot-pull-request-reviewer[bot], kilo-code-bot[bot], copilot` — **explicit list only, no suffix magic** (per maintainer) |
| Q11 | member denylist | `gxjiao, lmhe, zhwang, chaozhou` |
| Q12 | backfill timing | overnight |

**Plan is fully specified.** Next step: convert each W block into a Jira task and start with W1.
