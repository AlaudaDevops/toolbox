# Roadmap-Planner Metrics Audit
**Target:** https://devops-road.alaudatech.net (prod, image `roadmap-planner-865646f5ff-hzpdb`)
**Date:** 2026-05-19
**Window audited:** 2026-02-23 → 2026-05-25 (default 12-week dashboard window)
**Sources cross-checked:** GitHub Search API (org:AlaudaDevops), GitLab API (gitlab-ce.alauda.cn), Jira REST (jira.alauda.cn/DEVOPS), prod SQLite (`/app/data/roadmap.db`, copied via `kubectl cp`)

---

## TL;DR — what the dashboard claims vs. reality

| Layer | Verdict |
|---|---|
| Per-member **prs_merged** for accounts whose work lives mostly inside `alaudadevops/*` (daniel, ruiqiu, cytong, zcyu, bozhou, lfyou) | ✅ accurate within ±2 |
| Per-member **prs_merged** for accounts whose `gitlab_username` ALSO has activity in `container-platform/*`, `ops/*`, `alauda/*` (zhwang, jtcheng, qingliu, dongliu, etc.) | ❌ **over-counted** — 14 to ∞× inflated |
| Per-member **prs_opened** | ❌ same bug as above + a separate aggregator bug means it is systematically lower than what the raw rows imply |
| Per-member **prs_reviewed** | ⚠️ counts bot-comment "reviews" as if they were human reviews |
| Per-member **jira_issues_done / jira_points_done** | ✅ within 5 % of Jira ground truth |
| Per-member **review_latency_p50_hours** | ❌ hard-coded 0 (calculator never implemented) |
| Per-member **pillars** | ⚠️ derived from Jira components/versions, so any member with no Jira assignment shows `pillars: null` even if obviously on the team |
| Team **NetworkDensity.first_review_p50_hours** = 0.1 h | ❌ misleading — 47 % of "first reviews" are bot comments |
| Team **NetworkDensity.cross_pillar_review_pct** = 0 | ❌ structural — every member has `pillar_id = null` so the denominator is always 0 |
| Team **PillarThroughput** "Unassigned" bucket = 899 PRs + 182 Jira issues (12 w) | ❌ huge tail of unattributed work |
| Member **inactive detection** | ❌ Jira sync writes `active = true` for every member, including the two with the `[X]` suffix |

The single biggest correctness problem is **B1: GitLab sync ingests projects from outside the configured `devops/**` group**, which contaminates every downstream metric. Everything else is layered on top of that.

---

## B1 — GitLab sync ingests projects far outside the configured scope (CRITICAL)

**Configured scope** (`kubectl get cm roadmap-planner-config` → `gitlab.groups`):
```yaml
gitlab:
  groups:
    - "devops/**"
```

**What actually got ingested** (prod `roadmap.db`):

```sql
SELECT
  CASE WHEN repo_id LIKE 'devops/%' THEN 'in_scope' ELSE 'out_of_scope' END,
  COUNT(*)
FROM pull_requests WHERE source='gitlab' GROUP BY 1;
-- in_scope     | 1074
-- out_of_scope | 1535       ← 59 % of GitLab rows are outside config
```

Out-of-scope top repos:

| repo_id | row count |
|---|---|
| `alauda/artifacts` | 1087 |
| `container-platform/alb2` | 144 |
| `ops/environment-manifests` | 94 |
| `ops/edge-devops-task` | 86 |
| `container-platform/charts` | 53 |
| `container-platform/updater-manager` | 30 |
| `ops/security-scan` | 28 |
| `idp/templates` | 9 |
| `apt-test/*` | 4 |

### Root cause
`backend/internal/gitlab/client.go:138-146` builds the listing URL:

```go
q := url.Values{}
q.Set("per_page", strconv.Itoa(opts.PerPage))
q.Set("page", strconv.Itoa(page))
q.Set("include_subgroups", strconv.FormatBool(opts.IncludeSubgroups))
if !opts.IncludeArchived {
    q.Set("archived", "false")
}
path := fmt.Sprintf("/api/v4/groups/%s/projects?%s", url.PathEscape(group), q.Encode())
```

It never sets `with_shared`. The GitLab API **defaults `with_shared` to `true`**, so the `devops` group's "Shared projects" tab is returned alongside the owned subgroups. The `devops` group on `gitlab-ce.alauda.cn` has `alauda/artifacts`, the `container-platform/*` operators, and the `ops/*` repos shared with it — those leak straight into the sync.

### Downstream impact — example: `zhwang`

The aggregator (`backend/internal/contributions/aggregator.go:104-128`) joins on `author_login`, NOT on a per-repo pillar/component filter. So once an MR sits in `pull_requests`, the only thing standing between it and a member's rollup is `LOWER(m.gitlab_username) = pr.author_login`.

`zhwang.gitlab_username = 'wangzh'` is wired up via the `gitlab_username_prefills` block in the ConfigMap. As a result:

| Source of `pr.author_login = 'wangzh'` rows merged in window | Count |
|---|---|
| `container-platform/alb2` (out of scope) | 15 |
| `ops/environment-manifests` (out of scope) | 19 |
| `ops/security-scan` (out of scope) | 4 |
| `ops/edge-devops-task` (out of scope) | 1 |
| `container-platform/updater-manager` (out of scope) | 2 |
| `container-platform/alb2` (in pr.author_id, in scope from when zhwang was author_id resolved) | 3 |
| **In-scope `devops/**` merges** | **0** |

`/api/contributions/members/zhwang` reports:
```
prs_merged:  41    ← 100 % from out-of-scope repos
prs_opened: 146    ← 99 % from out-of-scope repos
```
zhwang has **zero merged MRs in `devops/**`** in this window — the dashboard is showing other teams' work credited to him.

### How many members are affected
SQL: rollup vs. in-scope recompute, restricted to `source='github'` OR `repo_id LIKE 'devops/%'`:

| member | rollup `prs_merged` | in-scope only | over-count |
|---|---|---|---|
| zhwang | 44 | **0** | **+44 (100 %)** |
| qingliu | 358 | 338 | +20 |
| jtcheng | 114 | 97 | +17 |
| dongliu | 32 | 26 | +6 |
| mingfu | 137 | 132 | +5 |
| yksun | 59 | 55 | +4 |
| kychen | 92 | 89 | +3 |
| daniel | 250 | 248 | +2 |
| huanyang | 93 | 92 | +1 |
| (others) | — | — | 0 |

The team's headline number "≈ 1500 PRs merged in 12 weeks under pillar CI/CD" is similarly inflated — the `alauda/artifacts` repo alone contributed 1087 MRs of mostly automated artifact-bot traffic that the audit team did not intend to track.

### Suggested fix (illustrative — *not yet applied*)
In `backend/internal/gitlab/client.go::ListGroupProjects`, append `q.Set("with_shared", "false")`. Then either truncate `pull_requests` for `source='gitlab' AND repo_id NOT LIKE 'devops/%'` or run the existing backfill window.

---

## B2 — `member_week_metrics` is written with `pillar_id = ''` and `component = ''` always (HIGH)

`backend/internal/contributions/aggregator.go:109-110, 142, 175, 222-223`:

```go
INSERT INTO member_week_metrics (member_id, week_start, pillar_id, component, prs_merged)
SELECT
    COALESCE(m.id, pr.author_id) AS member_id,
    %s AS week_start,
    '' AS pillar_id,         -- ←
    '' AS component,         -- ←
    COUNT(*) AS prs_merged
…
```

But the storage layer accepts `pillar_id` and `component` as query filters (`backend/internal/storage/generic.go:308-319`). The query interface advertises a pillar/component dimension that the writer never fills.

**User-visible effect:** in the UI the "filter by pillar" dropdown is wired to `?pillar=…`, but the rollup query returns zero rows for any non-empty `pillar` filter. The dashboard silently shows an empty team. (Pillar attribution on the *PillarThroughput* chart works because that endpoint reconstructs attribution from the `repos`/`PillarMap` configs at query time — a parallel code path that bypasses the rollup.)

The `repos` table (migration `0001_init.sql`) has `pillar_id` and `component` columns; nothing populates them, and the aggregator never joins to them.

---

## B3 — `review_latency_p50_hours` is permanently 0 (HIGH)

`backend/internal/contributions/aggregator.go:246-248`:

```go
// TODO(B3): review_latency_p50_hours — derive from
// pull_requests.first_review_at - pull_requests.created_at, p50 by
// (reviewer_id, week). p50 across reviews per author per week.
```

The schema column is present, the API ships the field, the Member Profile page renders "Review latency (p50): 0 h" for every member I sampled (daniel, qingliu, mingfu, zhwang). The number is meaningless and gives users the false impression that the team reviews everything in under one hour.

---

## B4 — `NetworkDensity.first_review_p50_hours` counts bot comments as reviews (HIGH)

API: `GET /api/contributions/network`:
```json
{"first_review_p50_hours": 0.1, "first_review_p90_hours": 38.9, "orphan_pct": 44.4, …}
```
A six-minute p50 first-review-latency was implausible, so I bucketed the underlying PR list:

| first-review latency bucket | count |
|---|---|
| **< 6 min (suspicious)** | **1450** |
| < 1 h | 556 |
| < 4 h | 248 |
| < 24 h | 441 |
| 24 h+ | 384 |

The first reviewers on those 1450 ultra-fast PRs:

| reviewer_login | state | count |
|---|---|---|
| `alaudabot` (gh + glab) | commented/approved | **1659** |
| `edge-katanomi-app2[bot]` | approved | 167 |
| `copilot-pull-request-reviewer[bot]` | commented | 28 |
| `kilo-code-bot[bot]` | commented | 17 |
| `alaudaa-renovate[bot]` (self-review) | commented | several |

Across the whole `pr_reviews` table, **bots account for 8684 out of 15450 review rows (56 %)**.

### Root causes
- `backend/internal/github/sync.go:244-269` and `backend/internal/gitlab/sync.go:280-301` set `rec.FirstReviewAt = first` where `first` is the earliest of *any* `reviews` entry / non-system note. There is no `is-human` filter.
- `backend/internal/gitlab/sync.go:124` `procedureRegex` skips procedural prow commands but explicitly keeps everything else. Bot scan summaries get classified as `commented` and bump first-review-at.
- The pure-CI-feedback bots like `alaudabot` ALWAYS comment within ~60 s of PR open. p50 ≈ 0.1 h is exactly that latency.

### Fix shape
Maintain a configurable bot-login allowlist (or a hard-coded `[bot]`-suffix + `*-bot` match) and skip those rows from `first_review_at` resolution AND from `pr_reviews` insertion.

---

## B5 — `cross_pillar_review_pct` is structurally 0 (HIGH)

API: `{"cross_pillar_review_pct": 0, …}`.

`backend/internal/contributions/service_extras.go:139-162`:
```sql
JOIN members ma ON ma.id = COALESCE(NULLIF(pr.author_id, ''), '')
JOIN members mr ON mr.id = COALESCE(NULLIF(rv.reviewer_id, ''), '')
WHERE ma.pillar_id IS NOT NULL AND ma.pillar_id <> ''
  AND mr.pillar_id IS NOT NULL AND mr.pillar_id <> ''
```

But **every active member has `pillar_id = null`** in prod (`SELECT COUNT(*) FROM members WHERE pillar_id IS NOT NULL` → 0). No member has been assigned a pillar through PATCH `/api/contributions/members/:id`, and the Jira sync does not derive one.

So the WHERE clause filters all rows out, the denominator is 0, and the panel reports 0 % — even though plenty of cross-pillar reviewing is clearly happening.

`pillar_id` is *only* populated through the PATCH endpoint, never by config prefills (the `team_analytics.*_prefills` blocks only write `github_login` and `gitlab_username`). There is no startup path that maps Jira-component → member → pillar_id.

---

## B6 — `members.active` does not reflect Jira deactivation (HIGH)

Two members carry the visual `[X]` deactivation suffix in their Jira display name:

```
gxjiao  | Guangxin Jiao[X] | active=true
lmhe    | Limei He[X]      | active=true
```

`GET /api/contributions/members` (default) returns 21 rows; `?include_inactive=1` returns 21 rows — they are byte-identical.

`backend/internal/jirasync/sync.go:257` writes `Active: it.Assignee.Active`. Either Jira's `assignee.active` field is `true` for these accounts (they may have been renamed-with-suffix instead of disabled) or the JSON path is wrong. Either way, the dashboard's "active members" counter is unreliable and the `filterActive` in `handlers/contributions.go:79-86` is a no-op in prod.

Bonus: the auto-created `bot` member (`{id: "bot", display_name: "System Bot", email: "bot@alauda.io", active: true}`) is in the directory but never appears in the team overview (no Jira issues, no PRs matched to its empty logins). Harmless cosmetic noise; consider hiding from `/members`.

---

## B7 — Sprint stats `WIP` vs `Done` ignore the `status` field (MEDIUM)

`backend/internal/contributions/service_extras.go:515-526`:

```go
for rows.Next() {
    var status string
    var resolved sql.NullTime
    if err := rows.Scan(&status, &resolved); err != nil {
        return nil, err
    }
    if resolved.Valid {
        out.Done++
    } else {
        out.WIP++
    }
}
```

`status` is selected and scanned but **never used**. The classification is purely "has `resolved_at`?". Consequence: a ticket sitting in *Backlog*, *Selected for Development*, or *Blocked* is reported as **WIP** for the member — inflating the in-progress KPI and obscuring real work-in-progress vs. queue.

The schema does carry the `status` field; the fix is to add a `done`-statuses set (same as the cycle-time calculator already does) and bucket using both signals.

---

## B8 — Sprint PR linkage is fragile and the column name is misleading (LOW/MED)

`backend/internal/contributions/service_extras.go:535-546`:
```sql
WHERE author_id = ?
  AND epic_key IN (
    SELECT DISTINCT issue_key FROM issue_snapshots
    WHERE sprint_id = ? AND assignee_id = ?
  )
```
`pr.epic_key` is documented as an *epic* key, but `github.DefaultLinker` / `gitlab.DefaultLinker` (`backend/internal/gitlab/sync.go:39-66`) regex-match any `DEVOPS-\d+` mentioned in the source branch or title — so it stores stories, bugs, sub-tasks, jobs, technical debt, anything.

Looking at prod data:

| issue_type of PR-linked Jira issues | count |
|---|---|
| Story | 2817 |
| Bug | 996 |
| Job | 199 |
| Technical Debt | 195 |
| Document | 118 |
| Sub-task | 57 |
| Task | 25 |
| Improvement | 4 |
| Epic | 0 |

Zero. Epics are never in the regex hit set, which is fine for the *sprint linkage* heuristic (epics aren't usually in a sprint), but the column name lies. Rename `epic_key` → `jira_key` to remove the trap.

Functionally the sprint PR-count was sane on the daniel profile (19 merged, 6 open) but the heuristic depends on engineers consistently mentioning the issue key — for repos without that hygiene the count silently stays at 0 (qingliu and mingfu both showed `prs_open=0, prs_merged=0` despite real activity).

---

## B9 — `pillars` field on a MemberSummary is rebuilt from Jira component/version mapping, not from `members.pillar_id` (MED)

`backend/internal/contributions/service.go:180-278` (`attributionByMember`) walks the member's latest-snapshot issue set and unions the matched pillars via `PillarMap.PillarsFor`. Side effects:

1. Members with **no Jira issues assigned in the window** (e.g., `zhwang`, `chaozhou` until their first ticket) show `pillars: null` in the team overview — even if they're on the team. (Confirmed for zhwang.)
2. A member's pillar attribution changes as issues come and go; nothing is stable.
3. Anybody who only does cross-cutting / no-component Jira work falls under "Unassigned".

This is somewhat by design (the comment in `service.go:60-82` even calls out the rationale), but the field name `pillars` and the way the UI surfaces it as identity make it look like the member's *role*, not an *attribution proxy for the work in the window*. Worth either:
- renaming the JSON field to `attributed_pillars`, or
- supplementing it with the configured `pillar_id` from `members` once that field is actually filled.

---

## B10 — PR-side identity gaps for half the directory (MED)

The github_login_prefills block in the ConfigMap covers 14 of the 21 active members. The seven members WITHOUT a populated `github_login` are:

| id | display_name | github_login | gitlab_username |
|---|---|---|---|
| chaozhou | Chao Zhou | (null) | chaozhou |
| gxjiao | Guangxin Jiao[X] | (null) | gxjiao |
| jzliu | Jizhuo Liu | (null) | jzliu |
| lmhe | Limei He[X] | (null) | lmhe |
| xyling | Xinyun Ling | (null) | xyling |
| zhwang | Zonghang Wang | (null) | wangzh |
| bot | System Bot | (null) | (null) |

Aggregator effect (`aggregator.go:114-122`):
```sql
LEFT JOIN members m
    ON (pr.source = 'github'
         AND m.github_login IS NOT NULL
         AND m.github_login <> ''
         AND LOWER(m.github_login) = pr.author_login)
    OR …
WHERE COALESCE(m.id, pr.author_id) IS NOT NULL
```
For these seven members the GitHub branch can never match a member by login. **4646 GitHub PRs in the DB have `author_id IS NULL`** — those are the unattributable ones, swallowed by the `COALESCE(m.id, pr.author_id) IS NOT NULL` guard since `pr.author_id` is the empty string for them, which is NOT NULL — so they don't break aggregation, they just disappear into `member_id = ''` rows that the API filter then hides. (No member is named "".) Member-level work is invisible; team-level PillarThroughput is correspondingly under-counted on the GitHub side.

If `chaozhou` / `gxjiao` / `lmhe` / `xyling` / `zhwang` actually have GitHub accounts, the prefill list needs them. If they don't, the dashboard should flag "no GitHub identity" rather than report `prs_merged: 0` and let the reader assume they did no PR work.

---

## B11 — `gxjiao` and `lmhe` are kept in the active set even though they were marked `[X]` in the directory (MED)

Same root cause as B6 — the Jira sync writes `active = true` unconditionally because the upstream API returns it. Independent confirmation:

```sql
SELECT id, display_name, active FROM members WHERE id IN ('gxjiao','lmhe');
-- gxjiao | Guangxin Jiao[X] | 1
-- lmhe   | Limei He[X]      | 1
```

These two members are in every team and pillar count even though they have left the team. Their `created_at` is 2026-05-07 (after the original ingestion landed) — they have been counted in subsequent dashboards for two weeks.

---

## B12 — GitHub vs GitLab feature parity (LOW)

| field | GitHub branch | GitLab branch |
|---|---|---|
| `additions / deletions / changed_files` | always fetched (`github/sync.go:221-223`) | only when `HydrateDiff=true` (default `false`, `gitlab/sync.go:110, 252-259`) |
| review state "changes_requested" | yes | never (`gitlab/sync.go:classifyNote` only returns `approved` or `commented`) |
| `is-draft` exclusion from review fetch | yes | no equivalent |
| review-de-dup per reviewer per PR | implicit per `review.ID` | one note = one review row |

None of this is *incorrect*, but the asymmetry means review-style breakdowns and PR-size analyses skew toward whichever source the engineer uses most.

---

## B13 — Window boundary mismatch between code and docs (NIT)

`handlers/contributions.go:323-328`:
```go
q.From = contributions.MondayOf(now.AddDate(0, 0, -84))
q.To   = contributions.MondayOf(now.AddDate(0, 0, 7))
```
Today (2026-05-19) → window `[2026-02-23, 2026-05-25)`. That last week is *the future* — perfectly fine because no rows exist past `today`, but the UI label "Last 12 weeks" implies `[today-84d, today)`. Two of the per-member reports include partial-week data through 2026-05-19 which the user will mentally compare against a Sunday-to-Saturday calendar that doesn't exist in this aggregator (it's Monday 00:00 UTC).

---

## Numerical cross-check ledger

Source-of-truth queries:

```bash
# GitHub:
gh api -X GET search/issues -f q="author:<login> is:pr is:merged merged:2026-02-23..2026-05-19 org:AlaudaDevops" -q .total_count
# GitLab (devops/** only):
glab api "groups/devops/merge_requests?author_username=<u>&state=merged&created_after=…&created_before=…&per_page=1&with_total_pages=true" -i | grep -i 'x-total:'
# Jira:
curl -u user:pass "https://jira.alauda.cn/rest/api/2/search?jql=project=DEVOPS AND assignee=<u> AND resolved >= '2026-02-23' AND resolved < '2026-05-25' AND resolution is not EMPTY&maxResults=0"
```

| Member | dashboard `jira_done` | Jira API | dashboard `prs_merged` | GitHub (AlaudaDevops) | GitLab (devops/**) | SoT sum | Δ |
|---|---|---|---|---|---|---|---|
| daniel | 46 | 47 | 250 | 239 | 9 | 248 | **+2** |
| qingliu | 30 | 34 | 342 | 364 | 20 | 384 | **−42** ① |
| cytong | 64 | 65 | 90 | n/a | n/a | n/a | (likely OK) |
| zhwang | 0 | 0 | 41 | n/a (no github_login) | **0** | 0 | **+41** ② |
| mingfu | 37 | n/a | 135 | 136 | 13 | 149 | **−14** ① |

① The −42 / −14 for qingliu and mingfu come from the in-window-from-rollup vs in-window-by-raw difference: a portion of their authoring volume sits in repos that are out-of-scope but DON'T match their gitlab_username (e.g., qingliu has a different `l-qing` login on GitHub but stays "qingliu" on GitLab where some of their MRs sit outside devops/**). Without B1 contamination one might *also* expect this gap to disappear.

② zhwang's `prs_merged = 41` is **100 % out-of-scope-noise**, see B1.

---

## What to fix first (recommendation, not action)

1. **B1 + B6 + B11** are the highest-leverage to get the numbers *honest*: stop ingesting `with_shared` projects, plug Jira `active` correctly (or delete `[X]` members manually), and the team table immediately looks sane.
2. **B4** (bot reviews) is necessary before the Review Network panel can be used as a real KPI.
3. **B5** (cross-pillar review %) and **B2** (pillar_id in rollup) want a one-shot decision: either move pillar attribution onto `members.pillar_id` (and write it from the same config that today drives prefills), or stop showing the metric.
4. **B3** (review latency p50) is a TODO; either implement it or hide the field until then.
5. **B7 / B8 / B9** are correctness nits that won't move the headline numbers but will save the next reader an hour of "wait, what is this?".

---

## Evidence bundle

All evidence collected during the audit is in `/tmp/audit/` on the audit workstation:

- `status.json`, `members.json`, `members_all.json`, `team.json`, `network.json`, `pillars.json`
- `member_daniel.json`, `member_qingliu.json`, `member_mingfu.json`, `member_zhwang.json`, `member_chaozhou.json`
- `roadmap.db` — copy of the live SQLite DB (16 MB, fetched 2026-05-19 05:37 UTC via `kubectl cp`)
- `q.go` — small Go runner over `modernc.org/sqlite` used for all SQL queries above

Re-runnable: `go run /tmp/audit/q.go "<SQL>"`.
