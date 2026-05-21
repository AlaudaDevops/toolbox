---
title: DORA Optimization — Engineering Delivery Dashboard
type: feat
status: handoff
date: 2026-05-21
companion: docs/team-analytics/dora-optimization-mockup.html
brainstorm_partial_merge: docs/brainstorms/2026-05-21-dora-drilldown-brainstorm.md
---

# DORA Optimization — Engineering Delivery Dashboard

> **本文档目的**：把 DORA Lead Time 优化的**决策、数据现状、mock 形态、未完成项**整理成可交接的 plan。接手者拿这份文档 + 配套 mock 即可进入实施阶段。
>
> **本文档是逻辑 spec**（算法、字段、数据源、UI 形态、文件位置）；配套 mock 是视觉 spec。

## Overview

把 `MetricsDashboard.jsx` 的 **Lead Time for Changes** 卡片从单一数字升级到 **Dev → Review → Release** 3 段拆分 + 瓶颈高亮 + hover worst_issues，再加一个独立 9 月趋势 panel。严守 DORA 经典语义（first commit → release，不含 backlog）。

> Deployment Frequency / Change Failure Rate / Mean Time to Recovery / Cycle Time 4 个指标 **calculator + UI 完全不动**（见 §Decisions D2）。本 plan 的实施范围聚焦在 Lead Time 一个指标。

**前置事实**（决策依赖）：
- 上架 AC 时间 == Jira version releaseDate（团队纪律保证）
- DEVOPS Jira project 版本命名规范：`{component}-v{semver}`（如 `tektoncd-operator-v4.6.3`）
- 排除 v3 旧插件：katanomi / knative / jenkins（不在 v4 团队关注范围）

## Problem Statement

`MetricsDashboard.jsx` 的 Lead Time 现状偏差：

| 维度 | 现状 | 目标 |
|---|---|---|
| 起点 | `Epic.created`（含 backlog 等待） | first commit author date（纯工程交付） |
| 展示形态 | **单一数字** | **3 段拆分（Dev/Review/Release）+ 瓶颈高亮 + 9 月趋势 + MoM/QoQ** |
| 行动信号 | 数字变化时看不到瓶颈在哪段、是否在恶化 | 卡片直接指向需要改进的 stage + worst issues |

`Epic.created` 起点被"epic 早建但晚开始"扭曲；单一数字让团队"看到指标动了但不知道改哪儿"。

## Ground truth (2026-05-21)

实施前已验证的项目实际状态，作为算法 / 数据流的事实输入。

### DEVOPS issuetype 分布（挂 fix_version 的 issue）

```
206 Story
151 Bug
 98 Technical Debt
 21 Job
 11 Epic           ← 仅 2.2%
 11 Document
  1 Sub-task
  1 Improvement
```

**结论**：`Epic` 几乎不挂 fix_version，主流是 `Story / Bug / Technical Debt`（共 91%）。**计算粒度必须用通用 "issue"，不是 "epic"**。Lead Time worst_issues 应按 issue_type 分组展示。

### Probe A · fix_version 命名实证

| Pattern | 示例 | 处置 |
|---|---|---|
| `tektoncd-operator-v4.X.X` | `tektoncd-operator-v4.6.3` | v4 主流，**计入**统计 |
| `tekton-operator-v3.X.X` | `tekton-operator-v3.20.0` | v3 旧系列，**排除**（D6） |
| 简单数字 / `v2.X` | `0.3`, `v2.1`, `1.0` | 老命名（早期 issue），component filter 上不显示 |

Component filter 解析正则：`^([a-z][a-z0-9-]+)-v(\d+(\.\d+)*)$` 提取 component 名。

### Probe B · `pull_requests.jira_key` 字段实证

- **来源**：`backend/internal/storage/migrations/0006_rename_epic_key_to_jira_key.sql`（**W5**，2026-05-19）。**不是 W2**——plan 之前所有"W2 jira_key"应理解为 W5
- **schema**：单值 `TEXT`（从 `epic_key` RENAME 来；W5 commit message 解释了原因 = 关联的不只 Epic）
- **索引**：`idx_pr_jira ON pull_requests(jira_key)`
- **多 issue 关联**：schema 不支持，单 PR 只能关联 1 个 jira_key → 详细设计 E2"1 PR 引用多 issue"实际**几乎不发生**，简化为退化场景

### Probe B · Lead Time 数据流（3 跳 join，已验证全部就位）

```
pull_requests.first_commit_at     ──┐
  └─ Phase 2 新增字段（唯一新增）   │
pull_requests.jira_key (W5)       ──┼──→ T0..T3 计算
  └─ TEXT 单值                       │
issue_snapshots.versions (0001)   ──┤
  └─ JSON array of version names    │
data.Releases (in-memory cache)   ──┘
  └─ Jira API pull · Version{ReleaseDate, Released, Archived}
```

**关键事实**：
- `issue_snapshots` 表已存 fix_version names（`versions` 列，JSON array），无需新增表
- Jira version 的 `ReleaseDate` **不持久化 SQL**——存在 collector 内存 cache `data.Releases` 里，calculator 通过 `CalculationContext` 读取（参考 `release_frequency.go:50-65`）
- Lead Time calculator 拼接：取 issue_snapshots 最新 run 的 issue → versions[0] 名字 → 在 data.Releases 里查同名 version 的 ReleaseDate
- 注意 `models.Version.Archieved` 字段（typo 在源码中）

### Probe B · pull_requests 现有 schema

```sql
-- 0001_init.sql + 0002-0008 演化结果
pull_requests (
    id, repo_id, number, title, state, author_id,
    head_branch, base_branch, additions, deletions, changed_files,
    jira_key,                       -- W5 改名自 epic_key
    created_at, first_review_at, merged_at, closed_at, fetched_at,
    first_human_review_at           -- 0005 (W2)
    -- ↓ Phase 2 新增
    -- first_commit_at TIMESTAMP NULL
)
```

### Probe 结论给 Phase 2 的影响

| 之前假设 | Probe 验证后 |
|---|---|
| 计算单位 = "epic" | **改为 "issue"**（详细设计已用，与 Probe A 一致）|
| W2 jira_key 字段 | **W5（0006 migration）** |
| jira_key 多值 | **单值 TEXT**（E2 简化）|
| 需要新增 issue/version 表 | **不需要**——issue_snapshots + data.Releases 已就位 |
| 唯一新增字段 | **`pull_requests.first_commit_at`** 确认 |
| Phase 2 schema migration 编号 | **0009_pr_first_commit_at.sql**（沿用 0001-0008 命名约定）|

## Scope

### What's in

- **Lead Time 起点改**：从 `Epic.created` 改为 PR first commit author date（涉及 GitHub PR API + W5 `jira_key` 字段，见 §Ground truth）
- **Lead Time stage 3 段拆分**：Dev → Review → Release（严守 D3 起点 = first commit，**不含 Backlog**），calculator 输出 3 个 duration，UI 在 Lead Time 卡片内嵌 3 段水平 bar，瓶颈段高亮（D15），hover 展开 worst_issues（每 stage top-1）
- **Lead Time Trend Panel**：KPI grid **下方**独立 panel（不在卡片内）· 全宽 sparkline + 当月点高亮 + 端点日期标签 + 右上方向箭头 MoM/QoQ chip · 新建 `LeadTimeTrendPanel.jsx`
- **Lead Time 卡片 transparency footer**：单行显示 `linked-PR coverage X% · N excluded`
- **v3 plugin 排除**：katanomi / knative / jenkins 在数据采集层过滤

### What's out (explicit defer)

- ❌ **其他 4 指标**（Deployment Frequency / Change Failure Rate / Mean Time to Recovery / Cycle Time）：**calculator + UI 完全不动**（见 §Decisions D2）
- ❌ **Dashboard chrome 重塑**：顶部 masthead / tabs / filter bar 保持项目原状；不加 contributors / window 切换 chip；不新建 `MetricsTrend.jsx` / `MetricsNarrative.jsx`；不加底部 footnote
- ⚠️ **Drill-down 完整详情页**（部分 defer）：本次仅吸收 Lead Time stage 3 段 + hover worst_issues。以下仍 defer 到独立 plan：Backlog 段拆分（brainstorm §4 的 4 段方案，跟 D3 不自洽）/ per-epic 全表 / 7d 滚动 leading signals / notable months 自动 flag / `/metric/<name>` 独立路由
- ❌ **Trend panel 内嵌 worst_issues / detail view**：panel 只展示 sparkline + chip，不混入其他信息密度


## Decisions (frozen)

记录所有已敲定的决策，避免走回头路。

| ID | Decision | Why |
|---|---|---|
| D1 | "上架 AC 时间 == Jira version releaseDate" 作为 Lead Time 终点的事实依据 | 团队纪律保证 |
| D2 | 其他 4 指标（Deployment Frequency / Change Failure Rate / Mean Time to Recovery / Cycle Time）**calculator + UI 完全不动** | 本 plan 范围明确聚焦 Lead Time；其他指标改造需要独立 plan |
| D3 | Lead Time 起点改为 PR first commit author date | Epic.created 含 backlog 等待，不反工程效能 |
| D6 | 数据采集层 + UI 层 **排除 v3 插件**：katanomi / knative / jenkins | v4 团队不关注，避免历史产物稀释水位 |
| D7 | UI 默认 **human only**，可切换含 bot | bot 占 71%，含 bot 数字漂亮但 actionable 弱 |
| D8 | UI 文本**全英文** + 全称（Deployment Frequency 等），禁止 DORA 缩写 | 团队读者不熟悉 DORA 缩写；跟 `MetricsDashboard.jsx` 原始 displayName 风格一致 |
| D14 | Lead Time **不含 Backlog 段**——严守 D3 起点（first commit），只输出 Dev / Review / Release 3 段；brainstorm §4 的 4 段方案（含 Backlog）跟 D3 矛盾，保留在 brainstorm 留待未来 drill-down plan | 方向 A 决议：按 DORA 经典语义严守 commit-centric 起点，Backlog 决策延迟可见度让位给指标自洽 |
| D15 | **瓶颈高亮规则**：某 stage 同时满足「占总时长 > 40%」**且**「该 stage duration ≥ 全员该 stage P75」时标红；只满足一项不标 | 单一阈值要么噪音多要么漏检；占比阈值找"主导段"，P75 阈值过滤"整体都快但比例高"的伪瓶颈 |
| D16 | Stage 数字 / 瓶颈高亮跟随顶部 human/include-bots chip 切换；卡片左下角带小字 `human only` / `incl. bots` 标识当前模式 | bot PR 占 71%，自动 merge 会把 Review stage 大幅压低；不绑定 chip 会让 stage 数字跟头部 KPI 数字脱节 |
| D17 | Lead Time 的「9-month trend」放在 **KPI grid 下方独立 trend panel**（不嵌入卡片）：全宽 sparkline（每月 total p50 一个点）+ 右上 **MoM / QoQ chip**（含方向箭头 + 百分比）；不展示 stage 分趋势；不开 detail page；Lead Time 卡片本身**保持瘦**（不显示 trend section） | (1) 痛点 2「看不到月份环比」是 brainstorm §1 明确写的；(2) sparkline 在卡片内信息密度过高，2026-05-21 mockup 实测后决定外移；(3) 分趋势 / 详情页留给 future drill-down plan；(4) YoY 因窗口 9 月不计算 |

## Dashboard layout (final spec)

> **范围收紧** · 仅改造 Lead Time 卡片，**不动 dashboard chrome、不动其他卡片、不新增 panel**。Canonical 视觉态：`docs/team-analytics/dora-optimization-mockup.html`（mockup 用灰盒占位标识"unchanged · existing"区域，唯一渲染细节的是 Lead Time 卡片）。

```
┌─────────────────────────────────────────────────────────────────────┐
│ App chrome (existing · unchanged):                                  │
│   masthead · tabs · filter bar                                      │
│   ⇒ 跟主分支 baseline 像素级 diff 为零                              │
├─────────────────────────────────────────────────────────────────────┤
│ KPI cards (5 columns; 现有 grid 不动):                              │
│  ┌────────┐ ┌────────────┐ ┌────────┐ ┌────────┐ ┌────────┐         │
│  │ Deploy │ │ Lead Time  │ │Change  │ │ MTTR   │ │ Cycle  │         │
│  │ Freq   │ │ ──(NEW)─── │ │Failure │ │unchang.│ │ Time   │         │
│  │unchang.│ │ 18d Medium │ │unchang.│ │existing│ │unchang.│         │
│  │existing│ │ ─────────  │ │existing│ │        │ │existing│         │
│  │        │ │ ▰▰▰▰▰▰▰▰  │ │        │ │        │ │        │         │
│  │        │ │ Dev/Rev⚠/  │ │        │ │        │ │        │         │
│  │        │ │  Release   │ │        │ │        │ │        │         │
│  │        │ │ hover→worst│ │        │ │        │ │        │         │
│  │        │ │ coverage96%│ │        │ │        │ │        │         │
│  └────────┘ └────────────┘ └────────┘ └────────┘ └────────┘         │
├─────────────────────────────────────────────────────────────────────┤
│ Lead Time Trend (NEW panel · full-width · D17):                     │
│   ┌───────────────────────────────────────────────────────────────┐ │
│   │ Lead Time Trend · 9mo · total p50      ▼5% MoM   ▲13% QoQ    │ │
│   │                                                               │ │
│   │   ╱╲___╱╲                                                     │ │
│   │  ╱      ╲___╱╲___╱╲      ╱╲                                  │ │
│   │ ╱             ╲___╱  ╲___╱  ●  ← current month               │ │
│   │                                                               │ │
│   │ 2025-09  10  11  12  01  02  03  04  2026-05                  │ │
│   └───────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘

⇒ Lead Time 卡片瘦身（不含 trend），独立 Trend Panel 单独占一行。
⇒ 其他 4 卡 + 上下 chrome 全部保持项目原有渲染路径。
⇒ 文件层：新建 LeadTimeTrendPanel.jsx；MetricsDashboard.jsx 仅 1 行 JSX 改动（append panel）；MetricCard.jsx 按 metric.name 分支渲染。
```

**Lead Time 卡片内部信息层级**（自上而下，**瘦身后 8 slot**）：

| Slot | 内容 | Spec 锚点 |
|---|---|---|
| 1 | Title `Lead Time for Changes` | existing |
| 2 | Value `18d` | existing |
| 3 | Unit `first commit → released · p50` + DORA band | D3 + D9 |
| 4 | separator | new |
| 5 | 3-stage horizontal bar (Dev / Review ⚠ / Release) | D14 + D15 + 详细设计 §6 |
| 6 | Stage legend (per-stage p50 hours + bottleneck flag) | D15 |
| 7 | "hover → worst issues per stage" cue | 详细设计 §7 |
| 8 | Footer `linked-PR coverage X% · N excluded` | Q1 a + 详细设计 §4 |
| (`:hover`) | worst_issues popup · 每 stage top-1 | 详细设计 §7 |

**Lead Time Trend Panel 信息层级**（D17 修订 · 独立 panel）：

| Slot | 内容 | Spec 锚点 |
|---|---|---|
| 1 | Header 左：title `Lead Time Trend · 9 months · total p50` | D17 |
| 2 | Header 右：MoM chip + QoQ chip (▼/▲ + % + period label) | D17 |
| 3 | Main：inline SVG sparkline · 9 个点连线 · null 月虚线 · 当月点 accent 高亮 | §11 |
| 4 | Axis：左 `2025-09` 右 `2026-05 ●` | §11 |
| 5 | Footnote（可选）：`Lead Time 越短越好 · 负值方向 (▼) = 改善` | §11 |

**其他 4 卡片** · 完全不动：
- Deploy Freq · Change Failure · MTTR · Cycle Time 渲染逻辑、字段、文案、CSS 跟主分支 baseline 像素级一致
- 不加 band 渲染、不英文化 displayName、不加 placeholder/proxy/unchanged 视觉标识（mockup 里灰盒占位**只是文档说明**，不是 React 渲染目标）

## Phase split

### Phase 0 · backend 启动 + Jira sync + PR 数据采集

**目的**：让 Lead Time 计算依赖的所有数据源真的能流通——Jira issue/version 同步 + GitHub/GitLab PR 同步 + 新字段 `first_commit_at` 填充。

**改动**：
1. Backend `.env`：填入 `JIRA_BASE_URL=https://jira.alauda.cn`、`JIRA_USERNAME=systembot`、`JIRA_PASSWORD=<from ~/.config/devops-release/config.json>`、`JIRA_PROJECT=DEVOPS`
2. `backend/internal/config/config.go`：把 `storage.enabled` 默认改为 `true`（或新建 production yaml）
3. `backend/internal/metrics/collector.go`：确认 collector 启动时跑一次 Jira + PR sync
4. **新增 plug exclusion filter**（D6）：`metrics.exclude_plugins: [katanomi, knative, jenkins]`，collector 在拉 versions 时按 name regex 过滤
5. **数据归档**：把采集的 Tekton 14 仓库 PR + Jira version 样本归档到 `backend/testdata/`（作为离线测试数据）

**验收**：
- `curl /api/metrics/lead_time_to_release?component=tektoncd-operator` 返回非空 `metadata.stages`
- `SELECT COUNT(*) FROM pull_requests WHERE first_commit_at IS NULL` 在一轮 sync 后保持小数（< 5% 全量）

### Phase 1 · Lead Time 卡片改造 + 独立 Trend Panel

**目的**：在现有 `MetricsDashboard.jsx` 上：(1) **升级 Lead Time 卡片**（stage bar + 瓶颈高亮 + hover worst_issues + transparency footer，**不含** trend）；(2) **在 KPI grid 下方插入独立 `LeadTimeTrendPanel`**（sparkline + MoM/QoQ chip）。**不动其他卡片、不动 dashboard chrome / masthead / tabs / filter bar**。

**范围红线**（与 Q1 a / Q2 b / D17 修订一致）：
- ❌ 不重塑顶部 dash-head / masthead / tabs / filter bar
- ❌ 不加 contributors `human only / include bots` 切换 chip
- ❌ 不加 window 切换 chip
- ❌ **不新建** `MetricsTrend.jsx`（泛化柱图）；仅新建专用 `LeadTimeTrendPanel.jsx`
- ❌ **不新建** `MetricsNarrative.jsx`（叙事 panel 不做）
- ❌ Change Failure / MTTR / Cycle Time / **Deploy Freq** 四卡视觉零改动
- ❌ Component filter 形态、CSS 不改

**改动**（2 个文件修改 + 1 个新建）：

**1. 修改 `frontend/src/components/metrics/MetricCard.jsx`**——按 metric name 分支渲染：

```
if (metric.name === "lead_time"):
  ┌─ title (existing)
  ├─ value + band (existing)
  ├─ ─────── (新增 separator)
  ├─ stagebar (3 段 · 瓶颈高亮 ⚠)
  ├─ stage legend (Dev / Review / Release · 各段 p50)
  ├─ "hover → worst issues per stage" cue
  ├─ footer · "linked-PR coverage X% · {N} excluded"  ← Q1 a
  └─ :hover → worst_issues popup (每 stage top-1)
  ⇒ 卡片瘦身：不含 trend section（trend 移到独立 panel）

else:
  return <ExistingMetricCard {...metric} />     ← 完全不动
```

**2. 新建 `frontend/src/components/metrics/LeadTimeTrendPanel.jsx`**：

```
┌─ panel header (横向 flex)
│  ├─ 左：title "Lead Time Trend · 9 months · total p50"
│  └─ 右：MoM chip + QoQ chip
├─ <svg> 全宽 sparkline · 9 个点 · 当月点 accent 高亮
├─ axis labels: 2025-09 ... 2026-05
└─ optional footnote: "Lead Time 越短越好 · 负值 = 改善"
```

**3. 修改 `frontend/src/components/MetricsDashboard.jsx`**——**仅 1 行 JSX 改动**：在现有 KPI grid 下方 append `<LeadTimeTrendPanel data={leadTimeMetricResult} />`。dashboard 顶部 / grid 渲染 / 其他卡片传 prop 全部不动。

UI 细节：
- **stage bar**：3 段 flex 比例按 `p50_hours`；瓶颈段按 D15 高亮（红边框 + ⚠ 角标）
- **MoM / QoQ chip**（panel 内）：方向箭头 + 百分比；负数 (▼) = improved 绿色；正数 (▲) = degraded 红色；null = 灰色 "n/a"
- **sparkline**：inline SVG `<polyline>`，9 月点连线，null 月画虚线，当月圆点放大 + accent 色，高度 80-100px
- **hover preview**：纯 CSS `:hover`（无 JS）；浮层显示 worst_issues 每 stage top-1
- **footer**：单行 mono 文本，coverage % 来自 API `coverage.coverage_pct`

**API 调用变化**：
- Lead Time 单独请求 `?include_bots=false&component=...&with_trend=true`（trend 字段按 D17 §11）
- `MetricCard` (Lead Time) 用同一 API 响应的 `stages` + `worst_issues` + `coverage` 部分
- `LeadTimeTrendPanel` 用同一 API 响应的 `trend` 部分（无需独立请求 → 复用 cache）
- 其他 4 个 metric 走原有 API endpoint，不变

**验收**：
- Lead Time **卡片**显示 3 段 bar + stage legend + footer + hover preview；**不**含 trend
- KPI grid **下方**显示独立 `Lead Time Trend` panel：sparkline + 当月点高亮 + MoM/QoQ chip + 端点日期
- Deploy Freq / Change Failure / MTTR / Cycle Time 四卡跟主分支 baseline **像素级 diff 为零**（除 grid 高度自适应）
- dashboard chrome（masthead、tabs、filter bar）跟主分支 baseline **像素级 diff 为零**
- 浏览器访问 `/metrics` 第一屏跟 `dora-optimization-mockup.html` 视觉一致

### Phase 2 · Lead Time 起点改正 + 3 段 stage 拆分

**目的**：把 Lead Time 从 "Epic.created → releaseDate · 单一数字" 改为 "first commit → releaseDate · Dev / Review / Release 3 段拆分"。

**3 段定义**（方向 A · 严守 D3）：

| Stage | 起点 | 终点 | 来源字段 |
|---|---|---|---|
| Dev | `pull_requests.first_commit_at` | `pull_requests.created_at`（first PR opened on linked epic） | GitHub PR commits API |
| Review | `pull_requests.created_at`（first PR） | `pull_requests.merged_at`（last PR on linked epic） | 现有 PR sync |
| Release | `pull_requests.merged_at`（last PR） | Jira version `releaseDate` | 现有 Jira sync |

**核心数据洞察**：3 段所需字段中 first_commit_at 是唯一新字段，其他全部已有。**不需要** Jira changelog 提取（Backlog 段在方向 A 下不存在，省掉 brainstorm 决策 7 关联的实现成本）。

**改动**：

1. `backend/internal/metrics/calculators/lead_time.go`：
   - 算法起点改为 `pull_requests.first_commit_at`（D3）
   - 输出从单一 duration 改为 `{ total_hours, stages: [{name: "dev"|"review"|"release", duration_hours, p50, p75}], worst_issues: [...] }`
   - 瓶颈识别按 D15：占比 >40% **且** ≥ P75 才标 bottleneck=true
   - human/include_bots 参数透传，影响 worst_issues 排序和 stage 数字（D16）

2. `backend/internal/github/`：新增 `ListPRCommits(repo, prNumber)` 客户端方法，调用 `GET /repos/{owner}/{repo}/pulls/{n}/commits` 取 first commit author date

3. `backend/internal/gitlab/`：同上，GitLab MR 的 commits API（`GET /projects/:id/merge_requests/:iid/commits`）

4. 数据模型扩展：
   - **schema migration**：新建 `backend/internal/storage/migrations/0009_pr_first_commit_at.sql`，内容 = `ALTER TABLE pull_requests ADD COLUMN first_commit_at TIMESTAMP NULL`（SQLite + Postgres 同一份，沿用 0001-0008 命名约定）
   - sync 时调 PR commits API 填充该字段；**retroactive backfill 复用现有 `BackfillDays` 配置**（不需要新建一次性脚本）——启动 sync 时以 `Since=now-BackfillDays` 拉 PR，每个 PR 触发一次 `ListPRCommits`/`ListMRCommits` 填字段；`COALESCE` UPSERT 保护已有值

5. PR ↔ Jira issue 关联（**已验证**，见 §Probe results）：
   - **3 跳 join**：`pull_requests.jira_key (W5, TEXT 单值)` → `issue_snapshots.versions[0]` (取最新 run 的 issue 快照；JSON array 第一个 name) → `data.Releases` (in-memory cache，含 ReleaseDate)
   - **不需要新增表**：`issue_snapshots`（0001）已存 versions，`data.Releases` 由现有 collector cache
   - **前置依赖**：`docs/plans/2026-05-20-feat-pr-jira-link-audit-plan.md` 的覆盖率 ≥ 70%（见 Risks R2）

6. **Fallback 规则**（与详细设计 §4 矩阵一致，此处仅列实施要点；详细决策见详细设计 §4 C1-C6）：
   - 单 PR 缺 first_commit_at → 该 PR 标 NULL，calculator 在该 issue 内**取剩余 PR 的 min(first_commit_at)** 作 T0；该 issue 仍计入 3 段统计（C2）
   - 该 issue 全部 PR 都缺 first_commit_at → 排除 Dev 段，Review 段起点改用 T1（C3）
   - issue 无 linked PR → **完全排除统计**（不引入非 commit-centric 起点，严守 D14；C4）
   - issue 无 released fix_version → 完全排除（C5）
   - 任一 stage < 0 → 完全排除 + 记 `data_anomaly`（C6）
   - 透明度：API 返回 `coverage: { full, partial, dev_missing, no_prs, unreleased, data_anomaly, coverage_pct }`

7. **趋势计算**（D17 + §11）：calculator 同时输出 `trend.points[9]` + `trend.mom_pct` + `trend.qoq_pct` + `trend.mom_direction` + `trend.qoq_direction`；month bucket 按 `fix_version.releaseDate` 分组；sample < 3 的月 `p50 = null`

8. UI 数据交付：
   - MetricCard Lead Time 卡片读 `stages` + `worst_issues` + `trend` + `coverage` prop（Phase 1 渲染逻辑已收 prop）
   - 卡片 unit 文案 "first commit → released · p50"
   - 卡片 footer transparency "linked-PR coverage X% · {N} excluded"（来自 `coverage.coverage_pct` + 排除 issue 数）

**验收**：
- `curl /api/metrics/lead_time?include_bots=false&component=tektoncd-operator&with_trend=true` 返回 `{ total, stages: [3], worst_issues: [top 3 per stage], coverage: {...}, trend: { points: [9 月], mom_pct, qoq_pct, mom_direction, qoq_direction }, consistency_warning: ... }`
- Lead Time API 输出的 `total.p50_hours` 中位数明显高于 PR merge lead time（含 release 等待）
- `sum(stages[].p50_hours)` 与 `total.p50_hours` 的偏差 < 5%（详细设计 §4 一致性约束）
- 切换 `include_bots=true` 后 Review stage 数字下降（bot auto-merge 拉低）
- worst_issues 按每 stage top-1 凑齐 3 个，bot PR 携带 `is_bot_majority: true` 标识
- Bot PR 在 UI hover preview 上带 🤖 徽章
- `trend.points` 数组长度严格等于 `window_months`，按时间升序
- `mom_pct` / `qoq_pct` 在数据不足时返回 `null`（不是 0）；UI chip 显示 `n/a`

### Phase 3 · 扩展到全部 67 活跃 repo

**目的**：从 Tekton 14 个 repo 验证形态，扩到 AlaudaDevops org 全部 67 个活跃 repo（每个 plugin 一个独立视图）。

**改动**：
1. `backend/internal/github/sync.go`：repos 配置从手动 list 改成 `OWNER/*` 通配符 + 排除规则（archived / `_pac-quota-pad-*` / v3 plugins）
2. Backend 跑 first-run backfill，拉所有 active repo PR 数据
3. 仪表盘 Component selector 显示全部 plugin
4. 默认选中 "tektoncd-operator"（高频 plugin），保留"留空 = 全部合并"模式

**验收**：
- 切换 Component 能看到 connectors-operator / harbor-ce-operator / gitlab-ce-operator / sonarqube-ce-operator / nexus-ce-operator 各自的 DORA 数字
- 留空时显示组织整体水位

## Lead Time 3 段计算 · 详细设计

> Phase 1 / Phase 2 / Dashboard layout / Risks R7-R8 都引用本节。落到 `backend/internal/metrics/calculators/lead_time.go`。

### 1 · 输入数据

| 来源 | 字段 | 备注 |
|---|---|---|
| `pull_requests` 表 | `id, jira_key, author_id, head_branch, additions, deletions, changed_files, first_commit_at, created_at, first_review_at, first_human_review_at, merged_at, closed_at, fetched_at` | **Probe B 已验证**：`jira_key` 来源 W5/0006 migration，TEXT 单值；`first_commit_at` 为 Phase 2 新增字段（0009 migration），PR commits API 填充；其他字段全部已有 |
| `issue_snapshots` 表 | `run_id, issue_key, issue_type, status, versions (JSON array of fix_version names), created_at, resolved_at` | **Probe B 已验证**：fix_version names 以 JSON array 存 `versions` 列；按 `(MAX(run_id), issue_key)` 拿最新快照 |
| `data.Releases` (in-memory) | `models.Version{ID, Name, ReleaseDate, UserReleaseDate, Released, Archieved}` | **Probe B 已验证**：Jira version 不持久化 SQL，collector cache 在内存，calculator 通过 `CalculationContext` 读取（参考 `release_frequency.go:50-65`）。注意源码 typo：字段名 `Archieved` |

**单位**：所有时间戳 UTC；duration 用小时（`hours`）存储和返回。

**术语澄清**：本节后续用 **"issue"** 作计算单位（Probe A 实证：Epic 仅占 2.2%，主流为 Story 41% + Bug 30% + Technical Debt 20%）。UI 文案 / worst_issues 显示 issue 的实际 issuetype（Story / Task / Bug / Technical Debt）。

### 2 · Issue 级 3 段公式

对每个**目标 issue** s（定义：fix_version 在统计窗口内 released = true 且 fix_version.name 匹配 component filter），计算 4 个时间锚点：

```
T0 = min(p.first_commit_at)  for p in s.linked_prs   // 第一次写代码
T1 = min(p.created_at)        for p in s.linked_prs   // 第一次开 PR (ready-for-review)
T2 = max(p.merged_at)         for p in s.linked_prs   // 最后一个 PR merge
T3 = s.fix_version.releaseDate (snapped to end-of-day) // 上架时刻
```

3 段定义（**保证 dev + review + release = total，无 gap**）：

```
dev_s     = T1 − T0       // 编码到首次 review 提交（含独立写代码时间）
review_s  = T2 − T1       // 首次 review 到全部 merge（含多 PR 串联 / 反复修改 / review 等待）
release_s = T3 − T2       // 全部 merge 到上架（含发版纪律 / 集成测试等待）
total_s   = T3 − T0       // = dev_s + review_s + release_s
```

**关键设计取舍**：Review 段是"剩余时间"，吸收所有 PR 间 gap、多 PR 串联、reopen、rebase 等情形，使得 3 段在 issue 维度严格可加。代价是 Review 段语义比 brainstorm 4 段方案宽——包含"中间编码"和"等待复评"。在 footnote 用一句话向团队解释。

### 3 · linked_prs 解析规则

- `pull_requests.jira_key` 为来源（待 Probe B 验证字段细节）
- 单 PR 引用多 jira_key → 该 PR 计入每个 issue 的 linked_prs 各一次（重复计入符合"该 issue 的贡献 PR"语义；worst_N 展示侧按 jira_key 去重避免视觉重复，见 P3-4）
- `include_bots=false` 模式下：在求 T0/T1/T2 之前**先过滤掉** `is_bot=true` 的 PR；过滤后无 PR 的 issue 走 Case 3 fallback
- PR 的 `merged_at > T3`（先 release 后 hotfix）→ 该 PR **排除**在当前 issue 的 T0/T1/T2 计算外（hotfix 自然归到下一个 release 的 issue）

### 4 · Fallback 矩阵

| Case | 触发 | 处理 |
|---|---|---|
| C1 数据齐全 | issue 有 PR · 所有 PR 有 first_commit_at · fix_version 有 releaseDate | 正常 3 段 |
| C2 部分 PR 缺 first_commit_at | 至少一个 linked PR 有 first_commit_at（其他 NULL） | 用 available PR 的 min(first_commit_at) 作 T0；该 issue 计入 3 段统计但 `coverage.partial += 1` |
| C3 issue 全部 PR 都缺 first_commit_at | 所有 linked PR 的 first_commit_at NULL | issue 排除 Dev 段；Review 段起点改为 T1 = min(created_at)，Release 段不变；total = T3 − T1；`coverage.dev_missing += 1` |
| C4 issue 无 linked PR | jira_key 关联失败或无 PR | **完全排除**（KPI + stages 都不计，避免引入非 commit-centric 起点违反 D14）；`coverage.no_prs += 1` 仅用于 transparency footnote 暴露 |
| C5 issue 无 fix_version / fix_version 未 released | fix_version.released=false 或 releaseDate=NULL | issue 完全**排除**（KPI + stages 都不计）；`coverage.unreleased += 1` |
| C6 任一 stage duration < 0 | 时区错乱 / squash 重写 / draft PR 占位等导致 T0 > T1 或 T1 > T2 或 T2 > T3 | issue 完全**排除**；`coverage.data_anomaly += 1`，API 在 `consistency_warning` 提示 |

**一致性约束**：calculator 强制校验 `sum(stages.p50) / total.p50 ∈ [0.95, 1.05]`，否则 API 返回 `consistency_warning`（见 R8）。

### 5 · 聚合到团队

对所有符合 C1/C2/C3 的 issue，计算各 stage duration 的分布统计：

```
For each stage s in [dev, review, release]:
  s.p25, s.p50, s.p75, s.p90 = percentile(issue_s for issue in qualified_issues)
  s.sample_count = len(issue_s)
```

⚠️ **中位数不可加性**：`p50(dev) + p50(review) + p50(release) ≠ p50(total)` —— 团队读图时必须知道这一点。

**UI 处理**：
- Stage bar 长度按 **stage p50 的相对比例**渲染（归一化为 100%），不按绝对值
- bar 旁标注每段 p50 绝对小时数（如 `Dev 18h · Review 96h · Release 168h`）
- 顶部 KPI 数字仍是 `total.p50`
- Footnote 一句话：「Stage 数字为各段中位数；因中位数不可加，stage 之和 ≠ 顶部 Lead Time 数字」

### 6 · 瓶颈识别（D15 落地）

对 stage s 标 `bottleneck=true` 当且仅当：

```
s.p50 / sum(stages.p50) > 0.40           // 占主导比例
  AND
s.p50 ≥ team_baseline.p75 for stage s    // 该段绝对值偏高
```

`team_baseline.p75` 来源：过去 12 个月所有 issue 的 stage p75（rolling baseline，存于内存或单独表，初次启动用 9 个月可用数据；后续按需引入 `metric_period_values` 表持久化，见 brainstorm §5.1，本 plan 内**不实施**）。

UI 高亮：bar 红色边框 + ⚠ 图标 + tooltip 文案 `45% of total · >= team baseline P75 (210h)`（英文，D8/D9）。

### 7 · worst_issues 排序

**每个 stage 各 top-1，凑齐 3 个**（避免 release 段绝对值大压倒 review/dev 异常的 visibility 问题）：

```
For each stage in [dev, review, release]:
  candidates = qualified_issues where argmax(stage.duration_in_issue / total_in_issue) == stage
  worst_per_stage[stage] = candidates sorted by stage.duration DESC, take 1

worst_issues = [worst_per_stage["dev"], worst_per_stage["review"], worst_per_stage["release"]]
                filter out None  // stage 无 candidate 时省略
```

**取舍说明**：原 plan 写"按 bottleneck_duration_hours 全局 DESC top 3" 会导致 worst_3 全是 release 段瓶颈（release 段绝对值通常远大于其他段）。每 stage top-1 让团队同时看到 3 段各自的最差案例，可指认改进点更平衡。

每条返回 `{ jira_key, issuetype, summary, bottleneck_stage, bottleneck_duration_hours, total_lead_time_hours, pr_count, is_bot_majority }`。

`is_bot_majority`：issue 关联 PR 中 bot PR 占比 > 50%（用于 UI 显示 🤖 徽章）。

### 8 · API 返回结构

`MetricResult.Value` 字段为 **days**（向后兼容现有 `MetricCard` / `MetricBreakdown` / Prometheus exporter `lead_time_days` 的 day 阈值消费）。Metadata 含两层：
- **新的 hour-precision payload**：`total` / `stages` / `worst_issues` / `coverage` / `trend` 等（Phase 1 frontend 用）
- **legacy days 兼容字段**：`min` / `max` / `average` / `count` / `sample_size` / `percentile`，供 `MetricBreakdown.jsx` 直接读旧路径

```json
GET /api/metrics/lead_time?component=tektoncd-operator&window_months=9&include_bots=false

{
  "metric": "lead_time",
  "value": 18,        // ← p50 in days, backward-compat with MetricCard
  "unit": "days",
  "window": { "start": "2025-09-01", "end": "2026-05-21" },
  "filters": { "component": "tektoncd-operator", "include_bots": false },
  "total": {
    "p25_hours": 56, "p50_hours": 432, "p75_hours": 1320, "p90_hours": 4080,
    "sample_count": 47
  },
  "stages": [
    {
      "name": "dev",  "label": "Dev",
      "definition": "first_commit_at → first_pr_opened",
      "p50_hours": 18, "p75_hours": 72, "sample_count": 45,
      "bottleneck": false
    },
    {
      "name": "review", "label": "Review",
      "definition": "first_pr_opened → last_pr_merged",
      "p50_hours": 96, "p75_hours": 240, "sample_count": 45,
      "bottleneck": true,
      "bottleneck_reason": "45% of total Lead Time · >= team baseline P75 (210h)"
    },
    {
      "name": "release", "label": "Release",
      "definition": "last_pr_merged → fix_version.release_date",
      "p50_hours": 168, "p75_hours": 480, "sample_count": 47,
      "bottleneck": false
    }
  ],
  "worst_issues": [
    {
      "jira_key": "DEVOPS-12345",
      "issuetype": "Story",
      "summary": "Tekton operator: support tier overrides",
      "bottleneck_stage": "review",
      "bottleneck_duration_hours": 720,
      "total_lead_time_hours": 1100,
      "pr_count": 5,
      "is_bot_majority": false
    }
    // ...one entry per stage (dev / review / release), 3 total
  ],
  "coverage": {
    "issues_total": 50,
    "issues_full":     45,   // C1
    "issues_partial":   3,   // C2
    "issues_dev_missing": 1, // C3
    "issues_no_prs":    1,   // C4
    "issues_unreleased": 0,  // C5 (excluded, count only)
    "issues_data_anomaly": 0,// C6 (excluded, count only)
    "coverage_pct": 96       // (full + partial + dev_missing) / (total - unreleased - data_anomaly)
  },
  "consistency_warning": null  // string when sum(stages.p50) vs total.p50 偏差 > 5%
}
```

### 9 · Edge cases 清单（实施时逐条验证）

| # | 场景 | 处理 |
|---|---|---|
| E1 | 1 个 issue 关联多 fix_version | **按 component filter 范围筛选**：先 filter `fix_version.name` 与 component prefix 匹配的子集，再取最早 released=true 的 releaseDate 作 T3；如全集都 released=true（极少），取最早 |
| E2 | 1 个 PR 引用多 issue | 该 PR 计入每个 issue 的 linked_prs 各一次（重复但正确）；**worst_issues 展示侧按 jira_key 去重**——同一 PR 让多个 issue 同时进 top-N 时仅显示第一个并标 `cross_referenced: true` |
| E3 | PR.merged_at 晚于 release day（calendar day 比较）| 该 PR 不算入当前 issue（视为 hotfix，归下一 release）。**Jira release date 是 calendar date，calculator 在 release 当日 EOD 之前 merge 的 PR 不算 hotfix**——避免丢失 release day 当天的 last PR |
| E4 | PR.first_commit_at < issue.created（提前编码）| 仍以 PR.first_commit_at 为 T0（commit-centric 严守 D3）|
| E5 | issue.fix_version 名称不匹配 `{component}-v{semver}`（如 `argo-cd-2.9.0` 无 `v` 前缀） | calculator 直接复用 `EnrichedRelease.Component`（collector 已 parsed，跟 release_frequency 同源），不重新解析；component filter 仍可命中 |
| E6 | 窗口边界：first_commit 在窗口外但 release 在窗口内 | 计入（窗口判定按 release 日期，对齐 DORA 惯例）|
| E7 | issue 跨 release（早 fix_version → 又重新 fix_version）| 用最新 released=true 的 fix_version；如有多个，记录 `multiple_releases=true` |
| E8 | 任一 stage duration < 0（dev/review/release 中任一为负，常见于时区错乱 / squash 重写 / draft PR 占位早于 first commit）| 该 issue 完全排除，记入 `coverage.data_anomaly`（与详细设计 §4 C6 一致）|
| E9 | 超长 dev 段：T0 早于 window.start 超过 180d（first_commit 在 2024 写，2026 才 release）| issue 完全排除该窗口的 Lead Time 统计 + 记入 `coverage.excluded_long_dev`；transparency footnote 暴露占比。避免长尾拉飞 p50 |

### 10 · 数据维护与刷新

- calculator 是**纯函数**，不写表；按需调用、按需聚合
- 数据源刷新走现有 collector 链：Jira sync（按 fixVersion / changelog） + GitHub PR sync（含新增 commits API）
- **PR fetch lookback**：collector 拉 PR 时用 `since = now - (HistoricalDays + 180d)`。Lead Time 按 release date 判定 window，但关联 PR 可能在 release window 起点前 merge；buffer 跟 E9 long-dev 阈值（180d）对齐，超出 180d 的 issue 已经被 E9 排除，所以更早的 PR 不需要 fetch
- 性能预算：9 月 ~50 issue × 平均 3 PR ≈ 150 PR 维度查询，单次 calculator 调用 < 200ms（含 percentile 计算）
- 如未来 issue 数量级 > 1000，再考虑预聚合（brainstorm §5.1 的 `metric_period_values` 表可启用，本 plan 内不实施）

**依赖**：
- percentile 计算：Go 标准库无；用 `gonum.org/v1/gonum/stat`（如项目已 vendor 则复用，否则 Phase 2 一并加入 go.mod）。fallback：自实现 `quickselect` 算法（< 30 行，无外部依赖）

**缓存策略**：
- 同一 dashboard 加载触发 5 个 metric 并发请求 → 5 × 200ms = 1s+ 用户感知
- calculator 结果按 `(metric, component, window_months, include_bots)` 作 cache key，TTL = **1 小时**（与现有 Jira sync 间隔对齐）
- 缓存 invalidate 触发器：collector 完成一次 sync 后，pub `metrics.cache_invalidate` 事件清空整个 namespace
- 本地开发模式可用 `?no_cache=1` 绕过

**Future enhancement**（不在本 plan 内）：
- **Review 段细粒度拆分**：当前 Review 段含 "PR open → first review comment / first approve / last merge" 整体；团队反馈"看不清是 review 等待慢还是 merge 等待慢"时，可拆为 `review_wait → approve_wait → merge_wait` 3 子段。**前置数据**：需采集 PR review event（comments / approvals）—— W2/W8 当前不含，需新增 GitHub events sync
- **Backlog 段重新引入**：如 Probe A 确认 Story 普遍走 backlog→in_progress→resolution 状态流，且团队明确要求看 backlog 决策延迟，再开独立 plan 实施 brainstorm §4 的 4 段方案（含 Backlog）
- **Stage 分趋势 / 详情页**：当前 trend 只画 total p50；如团队要看"Review 段在哪个月恶化"则需 3 段叠加趋势 / 独立 detail page，留给 future drill-down plan

### 11 · 趋势计算（D17 落地）

> 卡片内 9-month sparkline + MoM / QoQ chip 的算法。沿用 `lead_time.go` calculator，不开新文件。

**Month bucketing 规则**：

```
For each month m in [window.start, window.start + 1mo, ..., window.end]:
  qualified_issues_m = filter(qualified_issues,
                              fix_version.releaseDate falls in [m.start, m.end])
  month_p50_m = p50(total_s for s in qualified_issues_m)
  month_sample_m = len(qualified_issues_m)
```

- 月份判定**按 `fix_version.releaseDate`**（与 release_frequency 一致），不按 issue.created
- `sample < 3` 的月份 `month_p50 = null`（点上标空心圆，sparkline 在该月画虚线）
- 月份桶不受 component / include_bots filter 影响逻辑，但 filter 参数会作用在 qualified_issues 上

**MoM / QoQ 公式**：

```
last_month = month_p50[-1]          // 当月 (e.g., 2026-05)
prev_month = month_p50[-2]          // 上月 (e.g., 2026-04)
mom_pct = (last_month - prev_month) / prev_month * 100

curr_q_avg = mean(month_p50[-3:])   // 最近 3 月
prev_q_avg = mean(month_p50[-6:-3]) // 再往前 3 月
qoq_pct = (curr_q_avg - prev_q_avg) / prev_q_avg * 100
```

- 任一参与的月份 `month_p50 = null` 时，对应 MoM / QoQ chip 显示 `n/a` 而不是 0
- Lead Time 是越短越好的指标：**正数 (▲) = 恶化（红色 chip）**；**负数 (▼) = 改善（绿色 chip）**

**API 返回结构扩展**（详细设计 §8 基础上）：

```jsonc
{
  "metric": "lead_time",
  // ...existing fields (window, filters, total, stages, worst_issues, coverage)
  "trend": {
    "granularity": "month",
    "points": [
      { "period": "2025-09", "p50_hours": 576, "sample_count": 5 },
      { "period": "2025-10", "p50_hours": 504, "sample_count": 6 },
      { "period": "2025-11", "p50_hours": null,"sample_count": 1 },  // < 3 samples
      // ... 9 月共 9 个 entries
      { "period": "2026-05", "p50_hours": 432, "sample_count": 5, "is_current": true }
    ],
    "mom_pct": -5.3,    // last month vs prev month
    "qoq_pct": 12.7,    // last 3 mo avg vs prev 3 mo avg
    "mom_direction": "improved",   // "improved" (negative for lead time) | "degraded" | "n/a"
    "qoq_direction": "degraded"
  }
}
```

**性能与缓存**：
- Month 聚合用 SQL 一次取出（`GROUP BY strftime('%Y-%m', releaseDate)`），无 N+1
- 复用 §10 cache key + 增加 `granularity=month` 维度 → 同一 dashboard 加载一次 trend 计算服务整个会话
- 性能预算 +30ms（9 个 percentile × 10ms 上限）

**UI 渲染规则**（Phase 1 落地 · D17 修订后）：
- **位置**：KPI grid 下方独立 panel（**不嵌 Lead Time 卡片**），新文件 `LeadTimeTrendPanel.jsx`
- **panel 宽度**：跟随 dashboard 容器（full-width），不强求 5-column 对齐
- panel 顶部 header：左侧标题 `Lead Time Trend · 9 months · total p50`；右侧 MoM / QoQ chip
- panel 主体：inline SVG `<polyline>` 9 个点连线，高度 80-100px（比卡片内更高，信息更舒展）
- null 点 → 段画虚线
- 当月点（`is_current=true`）画放大圆点 + accent color
- panel 底部：端点日期标签 (`2025-09 ... 2026-05`)；可选最小化 footnote `Lead Time 越短越好 · 负值方向 = 改善`
- panel 自身不带 hover preview（保持纯展示），worst_issues hover 仍在 Lead Time 卡片
- panel 跟随 component / include_bots filter 联动（与 Lead Time 卡片同源 API 调用，复用 cache）

## Open Questions

实施阶段的 sub-task 决定（每个有 default，plan 不阻塞）。

| Q | Question | Default assumption |
|---|---|---|
| Q1 | Bot 识别规则：仅按 GitHub `is_bot` 还是按 login pattern？ | **混合**：is_bot OR login 含 bot/renovate/dependabot/alaudaa 关键字。Sub-task of Phase 1 |
| Q2 | 单 plugin 视图 vs 合并视图的优先级 | **单 plugin 优先**（默认 tektoncd-operator），合并视图 filter 留空触发。Sub-task of Phase 3 |
| Q3 | DORA band 阈值：用 DORA 2024 报告标准 vs 内部 calibrate？ | **DORA 2024 标准**作为 v1，UI 旁标 "DORA 2024 bands"，团队自行解读 |
| Q4 | Lead Time stage bar 窄屏（< 768px）降级 | **垂直堆叠**（3 段横→竖）；hover preview 改为 tap 展开。Sub-task of Phase 1 |

## Files referenced

### 配套 mock（视觉 spec）
- `docs/team-analytics/dora-optimization-mockup.html` — 静态 HTML，无依赖，打开即可

### Brainstorm 配套文档
- `docs/brainstorms/2026-05-21-dora-drilldown-brainstorm.md` — drill-down 完整方案；本 plan 仅吸收 **Lead Time 3 段拆分子集**（方向 A），其余（per-epic 全表 / leading signals / notable months / 独立路由 / 4 段含 Backlog 方案）保留在 brainstorm 留待未来独立 plan

### Backend 关键文件（Probe B 已验证存在性 + 内容）
- `backend/internal/config/config.go` — Storage / Jira / GitHub / GitLab 配置入口
- `backend/internal/metrics/collector.go` — Jira sync 入口
- `backend/internal/metrics/calculators/release_frequency.go` — **完全不动**（D2），但其 `CalculationContext.Releases` 消费模式被 Lead Time calculator 复用（一样从 in-memory cache 读 Jira version）
- `backend/internal/metrics/calculators/lead_time.go` — **Phase 2 重点改造对象**（起点改 + 输出 3 个 stage duration + worst_issues + coverage）
- `backend/internal/metrics/calculators/{cycle_time,patch_ratio,time_to_patch}.go` — **完全不动**（D4 + D5）
- `backend/internal/github/{client,sync}.go` — 新增 `ListPRCommits` 方法 + `pull_requests.first_commit_at` 字段填充
- `backend/internal/gitlab/` — 同上，MR commits API
- `backend/internal/jirasync/sync.go` + `backend/internal/jira/snapshot_search.go` — issue_snapshots.versions 已经存 fix_version names（JSON array，`snapshot_search.go:278-279`）；**不需要新增 changelog 提取**（方向 A 下 Backlog 段不在范围）
- `backend/internal/jira/models.go` — `ConvertJiraVersionToVersion`（line 311）已实现 Jira API → `models.Version{ReleaseDate, Released, Archieved}` 转换；calculator 读 `data.Releases` 即可
- `backend/internal/storage/migrations/0001_init.sql` — `pull_requests` + `issue_snapshots` 基础 schema
- `backend/internal/storage/migrations/0006_rename_epic_key_to_jira_key.sql` — **W5**，`jira_key` 字段来源（不是 W2）
- `backend/internal/storage/migrations/0009_pr_first_commit_at.sql` — **Phase 2 新建**，唯一 schema 变更
- `backend/testdata/` — 持久化 `/tmp/dora/*.json{,l}` 到这里

### Frontend 关键文件（Lead Time 卡片 + 独立 Trend Panel 范围）
- `frontend/src/components/MetricsDashboard.jsx` — **仅 1 行 JSX 改动**：KPI grid 下方 append `<LeadTimeTrendPanel data={leadTimeMetricResult} />`；顶层 chrome、grid、其他卡片渲染路径完全不动
- `frontend/src/components/metrics/MetricCard.jsx` — 按 `metric.name === "lead_time"` 分支渲染，新增 stage bar / 瓶颈高亮 / footer / hover worst_issues popup（**不含** trend section）；其他 4 卡（Deploy Freq / Change Failure / MTTR / Cycle Time）渲染路径**完全不动**
- `frontend/src/components/metrics/LeadTimeTrendPanel.jsx` — **新建**：full-width panel 渲染 sparkline + MoM/QoQ chip；消费 `metric.trend` API 子结构；纯展示（无 hover preview / drill-down）
- `frontend/src/components/MetricsDashboard.css` — 新增 Lead Time 卡片 CSS（stage bar、瓶颈红框、popup）+ TrendPanel CSS（sparkline、chip）；不改现有规则
- ❌ ~~`frontend/src/components/MetricsTrend.jsx`~~ — **不新建**（泛化版本不做，仅做专用 LeadTimeTrendPanel）
- ❌ ~~`frontend/src/components/MetricsNarrative.jsx`~~ — **不新建**（范围收紧后取消）

## Risks

| R | Risk | Mitigation |
|---|---|---|
| R1 | Jira version 命名不规范（漏填 `releaseDate` 或不用 `{component}-v{semver}` 格式） | Phase 0 启动时加纪律监控指标：检测 `released=true` 但 `releaseDate=null` 的版本，仪表盘上暴露给团队 |
| R2 | PR ↔ Jira issue 关联率低（**W5** 的 `jira_key` 字段没覆盖到所有 PR） | Phase 2 启动前先跑 `docs/plans/2026-05-20-feat-pr-jira-link-audit-plan.md` 的审计，确认覆盖率 ≥ 70% |
| R3 | squash merge 后 author date 重写，影响 Lead Time 准确性 | Phase 2 改用 GitHub PR API 的 `/pulls/{n}/commits` 拿原始 first commit time（不依赖 git log） |
| R4 | Bot 识别遗漏（如 `alauda-github-idpbot` 不以 `bot$` 结尾） | Phase 1 实现时维护一份 known-bots config，定期 review。Sub-task of Q2. |
| R5 | UI 默认 human only 让"包含 bot 工作量"的工程视角隐形 | 切换 chip 设计为显眼的 toggle，保留 1-click 切换到 include bots |
| R6 | 67 repo 全拉对 GitHub API rate limit 压力大 | 使用 GitHub App 而不是 PAT（rate limit 高一倍），按 repo 串行 + 失败重试 |
| R7 | first_commit_at 抽取失败率高（PR API 404 / squash 后无原始 commit / 私有 fork 不可达），导致 Dev 段统计样本不足 | Phase 2 改动 6 已定义 fallback 规则：单 PR 缺失 → epic 排除 Dev 段（其他段仍计）；epic 完全无 PR → 该 epic 仅计 KPI 总数不进 stage bar。API 返回 `coverage` 字段，UI footnote 暴露 `linked-PR coverage X%`；< 60% 时仪表盘顶部加 banner 提示 stage 数据可信度低 |
| R8 | Lead Time stage 数字与顶部 KPI 总数不一致（fallback epic 排除带来误差） | calculator 内做一致性校验：stages 之和 / total 偏差 > 5% 时 API 返回 `consistency_warning`，UI 展示 ⓘ 图标可点开看哪些 epic 被排除及原因 |

## Handoff checklist

接手者按这个清单逐项确认：

- [x] Brainstorm 决策清单冻结（见 §Decisions · D1-D16）
- [x] 方向 A 决议（Lead Time 拆 3 段 · 不含 Backlog · 严守 D3）
- [x] Cycle Time / Patch Ratio / Time to Patch 三者**完全不动**（D4 + D5）
- [x] 真实数据已采集并验证（19 Jira releases + 2,457 GitHub PRs）
- [x] Mock 文件英文化 + react-select 形态最终态 (`docs/team-analytics/dora-optimization-mockup.html`)
- [x] Open Questions 都有 default assumption（可直接进 Phase 0-3）
- [x] **Probe A · Jira 探针**（2026-05-21 已跑，见 §Probe results）：
  ```bash
  # 验证 DEVOPS project 里哪些 issuetype 真正挂 fix_version
  curl -sk -u "$JIRA_USER:$JIRA_PASS" \
    "$JIRA_URL/rest/api/2/search?jql=project=DEVOPS+AND+fixVersion+is+not+EMPTY&fields=issuetype&maxResults=200" | \
    jq -r '.issues[].fields.issuetype.name' | sort | uniq -c | sort -rn
  ```
  - 预期产出：issuetype 分布（如 `Story 180 / Task 45 / Bug 30 / Epic 5`），决定计算单位
  - 如果 Epic 占比 < 10% → 确认按 "issue" 通用计算（当前详细设计假设）
  - 如果 Epic 占主导 → 维持 "epic" 语义，无需改设计
  - 如果 fixVersion 在多 issuetype 都分布 → 维持通用 "issue" 计算 + UI 按 issuetype 分组展示
- [x] **Probe B · jira_key 字段探针**（2026-05-21 已跑，见 §Probe results）：
  ```bash
  # 在 roadmap-planner 项目里 grep schema 定义
  grep -rn "jira_key\|JiraKey" backend/internal/db/ backend/internal/ | grep -i "schema\|migration\|struct"
  ```
  - 验证 `pull_requests.jira_key` 字段实际存在 + schema 形态（TEXT 单值 / TEXT[] / 关联表）
  - 如不存在 → Phase 2 前置依赖增加"先实现 jira_key 关联"（参考 W2 / `2026-05-20-feat-pr-jira-link-audit-plan.md`）
- [x] **mockup 视觉补充**：Lead Time 卡片 3 段水平 stage bar + 瓶颈高亮（Review 段 ⚠）+ hover worst_issues（每 stage top-1）已落地到 `docs/team-analytics/dora-optimization-mockup.html`（2026-05-21）
- [ ] **数据从 `/tmp/dora/` 持久化到 `backend/testdata/`**（Phase 0 第 1 步）
- [ ] Phase 0 启动：backend 启动 Jira + PR sync，让 Lead Time 数据流通（`first_commit_at` 字段在一轮 sync 后填充到位）
- [ ] Phase 1 启动：UI 改造（英文 + 卡片 + 趋势图 + Lead Time stage bar）
- [x] **Phase 2 完成**（2026-05-21，feature 分支 `feat/dora-lead-time-phase2`）：
  - migration 0009 · `pull_requests.first_commit_at`
  - PR/MR commits API client + sync 集成（COALESCE-保护 UPSERT）
  - `lead_time.go` 算法重写：3 段 + fallback C1-C6 + worst_issues 每 stage top-1 + coverage + trend (MoM/QoQ) + consistency_warning
  - `/api/metrics/lead_time_to_release` 加 `include_bots` / `with_trend` query params
  - 11 个 unit test 全部 PASS
- [ ] Phase 3 启动：扩展到 67 repo
