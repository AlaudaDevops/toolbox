# Pillar attribution fallback plan

> Working doc — not committed yet. Created 2026-05-09 after slice-explorer
> story-points fix shipped (image `v0.2.0-g2a824c6`) but pillar buckets
> still show all 341.5 points under "Unassigned" because most issues have
> no `components` set.

## Pickup prompt (paste into a fresh session)

> Resume the pillar-attribution fallback work in
> `/workspaces/daniel-pod/github/alaudadevops/toolbox/roadmap-planner`.
> Start by reading `PILLAR_FALLBACK_PLAN.md` at the repo root —
> it has the data sample, the proposed phases, the file pointers, and
> the open questions. We've decided on Phase 1 (fixVersion-prefix
> matching). Implement it on a new branch, ship via `/build-image`,
> bump edge integration-test, and verify the `/api/contributions/pillars`
> response shows the previously-unassigned points routed to real pillars.
> Mapping decisions are in the "Open questions" section — if any are
> still unresolved, propose defaults and ask before coding.

## Why we're here

The slice explorer's pillar axis surfaces story points correctly now
(commit `2a824c6` on main / shipped to edge-int as `v0.2.0-g2a824c6`),
but on real DEVOPS Jira data 97% of recently-resolved issues have **no**
`components` field set, so almost all points land under the synthetic
`"Unassigned"` pillar bucket.

`PillarsForComponents()` in `backend/internal/contributions/pillar_mapping.go`
does case-insensitive **exact** match against the configured component
list, with no fallback if components are absent.

## Data sample (590 issues resolved in last 120 days, sampled
via `https://jira.alauda.cn/rest/api/2/search` on 2026-05-09)

| Bucket | Count | % | Notes |
|---|---:|---:|---|
| Has `components` | 24 | 4% | direct match (current path) |
| No comp, has `fixVersions` | 361 | 61% | **fixVersion-prefix fallback** |
| No comp, no version, has Epic Link | 101 | 17% | reach into parent epic |
| No comp, no version, no epic | 104 | 18% | mostly auto-generated `Job` tickets (128/205 of this bucket are issuetype=Job) |

fixVersion names are clean and 99% follow `<component>-v<X.Y.Z>`:

| Prefix | n | suggested pillar |
|---|---:|---|
| `tektoncd-operator-*` | 110 | CI/CD |
| `connectors-operator-*` | 80 | Tool Integration |
| `gitlab-ce-operator-*` | 48 | Tool Deployment |
| `harbor-ce-operator-*` | 32 | Tool Deployment |
| `sonarqube-ce-operator-*` | 30 | CI/CD or Tool Deployment? |
| `sre-agent-*` | 17 | HyperFlux? new pillar? |
| `nexus-ce-operator-*` | 17 | Tool Deployment |
| `katanomi-operator-*` | 4 | CI/CD |
| `thanos-*` | 2 | skip or new pillar |
| `tekton-operator-*` | 1 | CI/CD |
| `v4.x` meta-only | few | unrouteable, skip |
| `无` (junk) | rare | skip |

Epic Link is `customfield_10002` (confirmed via
`/rest/api/2/field` lookup; schema
`com.pyxis.greenhopper.jira:gh-epic-link`). We currently do **not**
pull this field — see file pointer below.

## Plan (3 phases, ship one at a time)

### Phase 1 — fixVersion-prefix matching (covers ~61%)

1. Extend the pillar config with a `version_prefixes` list per pillar:
   ```yaml
   team_analytics:
     pillars:
       "CI/CD":
         components: [...]
         version_prefixes: ["tektoncd-operator", "katanomi-operator", "tekton-operator"]
       "Tool Deployment":
         version_prefixes: ["gitlab-ce-operator", "harbor-ce-operator", "nexus-ce-operator"]
       "Tool Integration":
         version_prefixes: ["connectors-operator"]
   ```
   Glob form (`tektoncd-*`) is fine if simpler — the matching code
   already has a `path.Match` style helper for repos.
2. In `PillarMap`, add a `versionPrefixes` field (lowercased, kept in
   insertion order); accept comma-or-array forms.
3. Add `PillarsForVersions(versions []string) []string` that fans out
   the same way `PillarsForComponents` does. Match: a version
   `name` is checked against each pillar's prefixes; if the version
   string starts with `<prefix>-` (or matches the glob) the pillar is
   credited. Multi-version issues credit each matching pillar.
4. In `PillarThroughput` (the `/api/contributions/pillars` endpoint),
   when an issue has zero matching pillars from `PillarsForComponents`,
   try `PillarsForVersions` next. Only fall to `"Unassigned"` if both
   miss.
5. Same fallback in `componentsByMember` if needed for member-level
   surfacing — TBD whether the team page also shows version-derived
   pillars per member; check if `MemberSummary.Components` should
   include version-derived component prefixes too.
6. Update `PublicConfig()` so the frontend keeps showing pillar names
   in the configured order — version_prefixes don't need to leak to UI.
7. Bonus: in the `Unassigned` bucket caption on the chart, hint that
   adding a version_prefix in config would route those points.

### Phase 2 — Epic-link fallback (covers ~17% more)

1. Schema: add `parent_key TEXT` to `issue_snapshots` (new migration
   under `backend/internal/storage/migrations/`).
2. `SearchSnapshots` opts: include `customfield_10002` in the fields
   list, extract into a new `SnapshotIssue.ParentKey` (and probably
   `SnapshotIssue.ParentSummary` for debugging).
3. Wire through `jirasync` and the storage writer so `parent_key`
   lands in the table.
4. In `PillarThroughput`, after components and versions miss, look up
   the parent epic's row in `issue_snapshots` (latest snapshot per key,
   same window-function pattern as the main query) and try
   `PillarsForComponents` + `PillarsForVersions` against that. Cache by
   epic to avoid repeated lookups in a single rollup pass.

### Phase 3 — Exclude `Job` issuetype (covers ~22% of remaining noise)

1. Add `team_analytics.exclude_issue_types: ["Job"]` to config.
2. Filter at query time: `WHERE issue_type NOT IN (?, ...)` in both
   the team rollup and the pillar rollup queries.
3. Validate with the user before flipping — they may want
   `Milestone`/`Epic` excluded too (or handled separately as parents only).

## File pointers

| Concern | Path |
|---|---|
| Pillar config schema | `backend/internal/config/config.go` (`TeamAnalytics`, `PillarMapping`) |
| Pillar matcher | `backend/internal/contributions/pillar_mapping.go` |
| Pillar throughput SQL | `backend/internal/contributions/service_extras.go` (`PillarThroughput`) |
| Snapshot writer / fields | `backend/internal/jira/snapshot_search.go` (fields list + `SnapshotIssue`) |
| Snapshot storage | `backend/internal/storage/migrations/0001_init.sql` (+ a new migration for Phase 2) and `backend/internal/storage/generic.go` (`INSERT INTO issue_snapshots`) |
| Sync orchestrator | `backend/internal/jirasync/sync.go` |
| Team service / member surface | `backend/internal/contributions/service.go` |
| Frontend slice axis | `frontend/src/components/TeamAnalytics.jsx` (`SliceView`, `seriesByPillar`) |
| Live config (edge) | `gitlab/devops/edge cluster/integration-test/templates/devops-tooling/roadmap-planner.yaml` (block `team_analytics.pillars`) |

## Open questions for the user

1. `sonarqube-ce-operator` — CI/CD or Tool Deployment?
2. `sre-agent` — does this go under HyperFlux (need new pillar) or skip?
3. `thanos` — skip or new pillar?
4. Phase 3: confirm we want to exclude `Job` issuetype outright. Any
   other issuetypes to drop? (`Milestone`, `Epic` themselves?)
5. Should the version-prefix-derived pillar also light up the
   per-member "components touched" list, or stay pillar-only?

## How to ship (mirrors the established loop)

1. Branch `feat/roadmap-planner-pillar-version-fallback` off main.
2. Implement Phase 1 with tests under
   `backend/internal/contributions/pillar_mapping_test.go`.
3. Push, open PR, `/build-image roadmap-planner`. Image tag will be
   `v0.2.0-g<short>`.
4. Bump edge: edit
   `gitlab/devops/edge/cluster/integration-test/templates/devops-tooling/roadmap-planner.yaml`,
   open MR, approve + squash-merge.
5. Wait for ArgoCD reconcile, port-forward the pod, hit
   `/api/contributions/pillars?from=...&to=...` with the four
   `X-Jira-*` headers, and check that the previously-Unassigned
   points have moved into real pillars.
6. WeCom ping per memory `feedback_wecom_progress.md` once verified.

## Reusable verification snippet

```bash
JP=$(KUBECONFIG=~/.kube/config.yaml kubectl --context edge-int -n devops-tooling \
  get secret roadmap-planner-secrets -o jsonpath='{.data.JIRA_PASSWORD}' | base64 -d)
JU=devopsbot; JB=https://jira.alauda.cn; JPR=DEVOPS
POD=$(KUBECONFIG=~/.kube/config.yaml kubectl --context edge-int -n devops-tooling \
  get pod -l app=roadmap-planner -o name | head -1)
(KUBECONFIG=~/.kube/config.yaml kubectl --context edge-int -n devops-tooling \
  port-forward "$POD" 18080:8080 >/tmp/pf.log 2>&1 &)
sleep 4
FROM=$(date -u -d "$(date -u -d 'last monday' '+%Y-%m-%d') -84 days" '+%Y-%m-%d')
TO=$(date -u -d "$(date -u -d 'next monday' '+%Y-%m-%d')" '+%Y-%m-%d')
curl -sS -H "X-Jira-Username: $JU" -H "X-Jira-Password: $JP" \
        -H "X-Jira-BaseURL: $JB"   -H "X-Jira-Project: $JPR" \
  "http://127.0.0.1:18080/api/contributions/pillars?from=$FROM&to=$TO" \
  | jq '[.buckets[] | {pillar, points}] | group_by(.pillar) | map({pillar: .[0].pillar, points: (map(.points)|add)}) | sort_by(-.points)'
pkill -f "port-forward $POD" 2>/dev/null
```

## Sample fixtures (saved during the 2026-05-09 investigation)

| Issue | Type | Components | fixVersions |
|---|---|---|---|
| DEVOPS-43909 | Task | (none) | `["sonarqube-ce-operator-v2026.1.2"]` |
| DEVOPS-43743 | Bug  | (none) | `["connectors-operator-v1.11.0"]` |
| DEVOPS-43871 | Story | (none) | `["gitlab-ce-operator-v18.8.0", "harbor-ce-operator-v2.14.3", "sonarqube-ce-operator-v2026.1.2", "nexus-ce-operator-v3.76.12"]` |
| DEVOPS-43651 | Bug  | (none) | `["tektoncd-operator-v4.11.0"]` |
| DEVOPS-43951 | Job  | (none) | (none) — Epic Link `DEVOPS-43559` |

The multi-version issue (DEVOPS-43871) is the cleanest test for fan-out:
once Phase 1 is in, its story points should fully credit Tool Deployment
(gitlab/harbor/nexus) **and** CI/CD (sonarqube-ce-operator if that's
the chosen pillar).

## Memories worth re-reading before resuming

- `reference_rp_pillar_attribution.md` — current attribution model
- `reference_viper_lowercase_keys.md` — viper destroys map-key case;
  use yaml.v3 re-parse for case-sensitive blocks (already done for
  `team_analytics` — extend the carveout if you add a new map block)
- `reference_pac_manual_pipelinerun.md` — only if PaC is down again
- `reference_modernc_sqlite_wal.md` — only if you need direct DB edits
