/*
 * Team Analytics — Dashboard tab.
 *
 * A single screen of pre-baked aggregations driven by three filters:
 *   • Window  (4 / 12 / 26 weeks)
 *   • Pillar  (All / each configured pillar / Unassigned)
 *   • Metric  (PRs merged / PRs opened / Reviews / Jira done / Story pts)
 *
 * The Metric filter is fronted by 5 clickable KPI tiles. The "active"
 * metric drives the Throughput trend's highlighted line, the Pillar mix
 * over time, the donut, the Top movers, and the per-member ranking.
 *
 * The Inflow vs outflow panel is intentionally fixed to opened-vs-merged
 * since that's the question it answers — flipping it to "Reviews opened
 * vs merged" makes no sense.
 *
 * Theming: every visual var comes from styles/theme.css semantic tokens
 * — no hex literals, no theme branching here. Light/Dark/Atlas swap
 * automatically.
 */
import React, { useEffect, useMemo, useState } from 'react';
import { contributionsAPI } from '../services/api';

const METRIC_OPTIONS = [
  { val: 'prs',     label: 'PRs merged' },
  { val: 'opened',  label: 'PRs opened' },
  { val: 'reviews', label: 'Reviews'    },
  { val: 'jira',    label: 'Jira done'  },
  { val: 'points',  label: 'Story pts'  },
];

// CSS-variable colors so light/dark/Atlas all resolve at paint time.
const METRIC_COLOR = {
  prs:     'var(--accent)',
  opened:  'var(--ocean)',
  reviews: 'var(--amber)',
  jira:    'var(--forest)',
  points:  'var(--crimson)',
};

const PILLAR_PALETTE = ['var(--accent)', 'var(--ocean)', 'var(--amber)', 'var(--forest)', 'var(--crimson)'];
const colorForPillar = (pillarName, allPillars) => {
  if (!pillarName || pillarName === 'Unassigned') return 'var(--fg-faint)';
  const idx = (allPillars || []).indexOf(pillarName);
  if (idx >= 0) return PILLAR_PALETTE[idx % PILLAR_PALETTE.length];
  let h = 0;
  for (let i = 0; i < pillarName.length; i++) h = (h * 31 + pillarName.charCodeAt(i)) >>> 0;
  return PILLAR_PALETTE[h % PILLAR_PALETTE.length];
};

const fmt = (v) => {
  const n = +v || 0;
  return Number.isInteger(n) ? n : n.toFixed(1);
};

// metricOf reads the configured metric off a Bucket from /api/contributions/team.
// Bucket fields use the JSON names: jira_done, points, prs_merged, prs_opened, reviews.
const metricOf = (bucket, metric) => {
  if (!bucket) return 0;
  if (metric === 'prs')     return bucket.prs_merged || 0;
  if (metric === 'opened')  return bucket.prs_opened || 0;
  if (metric === 'reviews') return bucket.reviews    || 0;
  if (metric === 'jira')    return bucket.jira_done  || 0;
  if (metric === 'points')  return bucket.points     || 0;
  return 0;
};

const memberSum = (member, weeks, metric) =>
  weeks.reduce((acc, w) => {
    const b = (member.week_totals || []).find((x) => x.week_start === w);
    return acc + metricOf(b, metric);
  }, 0);

const teamSum = (members, weeks, metric) =>
  members.reduce((acc, m) => acc + memberSum(m, weeks, metric), 0);

const pillarOf = (member) => (member.pillars && member.pillars[0]) || member.pillar || 'Unassigned';

/* ---------------- date helpers ---------------- */
function isoDate(d) { return new Date(d).toISOString().slice(0, 10); }
function mondayOf(d) {
  const x = new Date(d);
  const day = x.getUTCDay();
  const diff = (day === 0 ? -6 : 1 - day);
  x.setUTCDate(x.getUTCDate() + diff);
  x.setUTCHours(0, 0, 0, 0);
  return x;
}
function lastNWeeks(n) {
  const today = new Date();
  const lastMonday = mondayOf(today);
  const out = [];
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(lastMonday);
    d.setUTCDate(d.getUTCDate() - i * 7);
    out.push(isoDate(d));
  }
  return out;
}

/* ---------------- chart primitives ---------------- */

// LineChart — multi-line overlay, optional highlight key fades others.
function LineChart({ weeks, seriesByMetric, metricKeys, highlight, height = 220 }) {
  const W = 880, H = height, pad = { l: 44, r: 14, t: 14, b: 28 };
  const allMax = Math.max(1, ...metricKeys.flatMap((k) => seriesByMetric[k] || []));
  const xStep = (W - pad.l - pad.r) / Math.max(1, weeks.length - 1);
  return (
    <svg className="ta-chart-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
      {[0, 1, 2, 3, 4].map((g) => {
        const y = pad.t + (H - pad.t - pad.b) * (g / 4);
        const v = Math.round(allMax * (1 - g / 4));
        return (
          <g key={g}>
            <line className="ta-chart-grid" x1={pad.l} x2={W - pad.r} y1={y} y2={y} />
            <text className="ta-chart-axis" x={pad.l - 6} y={y + 3} textAnchor="end">{v}</text>
          </g>
        );
      })}
      {weeks.map((wk, i) => (
        (i % 4 === 0 || i === weeks.length - 1) && (
          <text key={i} className="ta-chart-axis"
                x={pad.l + i * xStep} y={H - 8} textAnchor="middle">{wk.slice(5)}</text>
        )
      ))}
      {metricKeys.map((k) => {
        const series = seriesByMetric[k] || [];
        const isHi = !highlight || k === highlight;
        const op = highlight && !isHi ? 0.18 : 0.95;
        const sw = isHi ? 2.0 : 1.2;
        const pts = series.map((v, i) => `${pad.l + i * xStep},${H - pad.b - (v / allMax) * (H - pad.t - pad.b)}`).join(' ');
        return (
          <g key={k}>
            <polyline fill="none" stroke={METRIC_COLOR[k]}
                      strokeOpacity={op} strokeWidth={sw} points={pts} />
            {isHi && series.map((v, i) => {
              const cx = pad.l + i * xStep;
              const cy = H - pad.b - (v / allMax) * (H - pad.t - pad.b);
              return <circle key={i} cx={cx} cy={cy} r="2.4" fill={METRIC_COLOR[k]} />;
            })}
          </g>
        );
      })}
    </svg>
  );
}

// StackedAreaChart — cumulative areas, one polygon per category.
function StackedAreaChart({ weeks, stackByCategory, categories, colorMap, height = 220 }) {
  const W = 600, H = height, pad = { l: 44, r: 14, t: 14, b: 28 };
  const sums = weeks.map((_, i) => categories.reduce((a, c) => a + ((stackByCategory[c] || [])[i] || 0), 0));
  const max = Math.max(1, ...sums);
  const xStep = (W - pad.l - pad.r) / Math.max(1, weeks.length - 1);
  const offsets = weeks.map(() => H - pad.b);
  return (
    <svg className="ta-chart-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
      {[0, 1, 2, 3, 4].map((g) => {
        const y = pad.t + (H - pad.t - pad.b) * (g / 4);
        const v = Math.round(max * (1 - g / 4));
        return (
          <g key={g}>
            <line className="ta-chart-grid" x1={pad.l} x2={W - pad.r} y1={y} y2={y} />
            <text className="ta-chart-axis" x={pad.l - 6} y={y + 3} textAnchor="end">{v}</text>
          </g>
        );
      })}
      {weeks.map((wk, i) => (
        (i % 4 === 0 || i === weeks.length - 1) && (
          <text key={i} className="ta-chart-axis"
                x={pad.l + i * xStep} y={H - 8} textAnchor="middle">{wk.slice(5)}</text>
        )
      ))}
      {categories.map((cat) => {
        const series = stackByCategory[cat] || weeks.map(() => 0);
        const top = series.map((v, i) => {
          const h = (v / max) * (H - pad.t - pad.b);
          offsets[i] -= h;
          return `${pad.l + i * xStep},${offsets[i]}`;
        });
        const bot = series.map((v, i) => `${pad.l + i * xStep},${offsets[i] + (v / max) * (H - pad.t - pad.b)}`).reverse();
        return <polygon key={cat} points={[...top, ...bot].join(' ')} fill={colorMap[cat]} opacity="0.85" />;
      })}
    </svg>
  );
}

// GroupedBarChart — pairs of bars per week.
function GroupedBarChart({ weeks, seriesA, seriesB, labelA, labelB, colorA, colorB, height = 220 }) {
  const W = 600, H = height, pad = { l: 44, r: 14, t: 14, b: 36 };
  const max = Math.max(1, ...seriesA, ...seriesB);
  const xStep = (W - pad.l - pad.r) / weeks.length;
  const bw = Math.max(2, (xStep - 4) / 2);
  return (
    <svg className="ta-chart-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
      {[0, 1, 2, 3, 4].map((g) => {
        const y = pad.t + (H - pad.t - pad.b) * (g / 4);
        const v = Math.round(max * (1 - g / 4));
        return (
          <g key={g}>
            <line className="ta-chart-grid" x1={pad.l} x2={W - pad.r} y1={y} y2={y} />
            <text className="ta-chart-axis" x={pad.l - 6} y={y + 3} textAnchor="end">{v}</text>
          </g>
        );
      })}
      {weeks.map((wk, i) => {
        const x0 = pad.l + i * xStep + 2;
        const ha = (seriesA[i] / max) * (H - pad.t - pad.b);
        const hb = (seriesB[i] / max) * (H - pad.t - pad.b);
        return (
          <g key={i}>
            <rect x={x0}      y={H - pad.b - ha} width={bw} height={ha} fill={colorA} opacity="0.9" />
            <rect x={x0 + bw} y={H - pad.b - hb} width={bw} height={hb} fill={colorB} opacity="0.9" />
            {(i % 4 === 0 || i === weeks.length - 1) && (
              <text className="ta-chart-axis"
                    x={x0 + bw} y={H - 16} textAnchor="middle">{wk.slice(5)}</text>
            )}
          </g>
        );
      })}
      <g>
        <rect x={pad.l}        y={H - 10} width="9" height="9" fill={colorA} />
        <text className="ta-chart-axis" x={pad.l + 14}  y={H - 2}>{labelA}</text>
        <rect x={pad.l + 110}  y={H - 10} width="9" height="9" fill={colorB} />
        <text className="ta-chart-axis" x={pad.l + 124} y={H - 2}>{labelB}</text>
      </g>
    </svg>
  );
}

// DonutChart — proportions of a single metric by pillar.
function DonutChart({ slices }) {
  const cx = 90, cy = 90, r = 70, ir = 42;
  const total = slices.reduce((a, s) => a + s.value, 0);
  if (total === 0) {
    return (
      <svg width="180" height="180" viewBox="0 0 180 180">
        <text x={cx} y={cy} textAnchor="middle"
              className="ta-chart-axis">no data</text>
      </svg>
    );
  }
  let acc = 0;
  return (
    <svg width="180" height="180" viewBox="0 0 180 180">
      {slices.map((s, i) => {
        const start = (acc / total) * Math.PI * 2 - Math.PI / 2;
        acc += s.value;
        const end = (acc / total) * Math.PI * 2 - Math.PI / 2;
        const large = (end - start) > Math.PI ? 1 : 0;
        const sx = cx + Math.cos(start) * r,  sy = cy + Math.sin(start) * r;
        const ex = cx + Math.cos(end)   * r,  ey = cy + Math.sin(end)   * r;
        const isx = cx + Math.cos(end)   * ir, isy = cy + Math.sin(end)   * ir;
        const iex = cx + Math.cos(start) * ir, iey = cy + Math.sin(start) * ir;
        return (
          <path key={i}
                d={`M ${sx} ${sy} A ${r} ${r} 0 ${large} 1 ${ex} ${ey} L ${isx} ${isy} A ${ir} ${ir} 0 ${large} 0 ${iex} ${iey} Z`}
                fill={s.color} opacity="0.9" />
        );
      })}
      <text x={cx} y={cy - 4}  textAnchor="middle"
            style={{ fontFamily: 'var(--font-display)', fontSize: 22, fill: 'var(--fg)' }}>{fmt(total)}</text>
      <text x={cx} y={cy + 14} textAnchor="middle"
            style={{ fontFamily: 'var(--font-mono)', fontSize: 9.5, letterSpacing: '0.14em', fill: 'var(--fg-muted)' }}>TOTAL</text>
    </svg>
  );
}

/* ---------------- main view ---------------- */

export default function DashboardView({ rows, orderedPillarNames, onMemberClick }) {
  const [windowWeeks, setWindowWeeks] = useState(26);
  const [metric, setMetric] = useState('prs');
  const [pillarFilter, setPillarFilter] = useState('all');

  // The default /api/contributions/team window is 12 weeks; for the
  // Dashboard's wider windows we re-fetch with explicit from/to.
  const [team, setTeam] = useState(rows);
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    let cancelled = false;
    const refetch = async () => {
      const today = new Date();
      const to = isoDate(today);
      const fromDate = new Date(today);
      fromDate.setUTCDate(fromDate.getUTCDate() - windowWeeks * 7);
      const from = isoDate(fromDate);
      setLoading(true);
      try {
        const resp = await contributionsAPI.team({ from, to });
        if (cancelled) return;
        // Merge week_totals from the new fetch onto the rows we already
        // have (which carry display fields like name/pillars/github).
        const byID = new Map((resp.members || []).map((t) => [t.member_id, t]));
        const merged = (rows || []).map((r) => {
          const t = byID.get(r.id);
          return t ? { ...r, week_totals: t.week_totals || [], pillars: t.pillars || r.pillars } : r;
        });
        // Surface members that exist in the rollup but not yet in the
        // members directory (rare — happens before first PATCH).
        (resp.members || []).forEach((t) => {
          if (!merged.find((r) => r.id === t.member_id)) {
            merged.push({
              id: t.member_id, name: t.member_id, github: '', gitlab: '',
              pillars: t.pillars || [], pillar: (t.pillars || [])[0] || '',
              week_totals: t.week_totals || [], active: true,
            });
          }
        });
        setTeam(merged);
      } catch {
        if (!cancelled) setTeam(rows); // fall back to props
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    refetch();
    return () => { cancelled = true; };
  }, [windowWeeks, rows]);

  const weeks = useMemo(() => lastNWeeks(windowWeeks), [windowWeeks]);

  const pillarKeys = useMemo(() => [...orderedPillarNames, 'Unassigned'], [orderedPillarNames]);
  const pillarColor = useMemo(() => {
    const map = {};
    pillarKeys.forEach((p) => { map[p] = colorForPillar(p, orderedPillarNames); });
    return map;
  }, [pillarKeys, orderedPillarNames]);

  const filteredMembers = useMemo(() => {
    if (pillarFilter === 'all') return team;
    if (pillarFilter === 'Unassigned') return team.filter((m) => !m.pillars || m.pillars.length === 0);
    return team.filter((m) => (m.pillars || []).includes(pillarFilter));
  }, [team, pillarFilter]);

  /* ---- KPIs (each metric vs prior half) ---- */
  const half = Math.floor(weeks.length / 2);
  const recentWks = weeks.slice(half);
  const priorWks  = weeks.slice(0, half);
  const kpis = METRIC_OPTIONS.map((opt) => {
    const tot = teamSum(filteredMembers, weeks, opt.val);
    const rec = teamSum(filteredMembers, recentWks, opt.val);
    const pri = teamSum(filteredMembers, priorWks,  opt.val);
    let delta = null;
    if (pri > 0) {
      const pct = Math.round(((rec - pri) / pri) * 100);
      delta = { kind: Math.abs(pct) < 5 ? 'flat' : (pct > 0 ? 'up' : 'down'), pct: Math.abs(pct) };
    } else if (rec > 0) {
      delta = { kind: 'up', pct: 100, novel: true };
    }
    return { ...opt, total: tot, delta };
  });

  /* ---- Trend (multi-line) ---- */
  const trendSeries = useMemo(() => {
    const out = {};
    METRIC_OPTIONS.forEach((opt) => {
      out[opt.val] = weeks.map((w) => teamSum(filteredMembers, [w], opt.val));
    });
    return out;
  }, [filteredMembers, weeks]);

  /* ---- Pillar mix over time (stacked area) ---- */
  const pillarStack = useMemo(() => {
    const stack = {};
    pillarKeys.forEach((p) => { stack[p] = weeks.map(() => 0); });
    filteredMembers.forEach((m) => {
      const p = pillarOf(m);
      weeks.forEach((wk, i) => {
        const b = (m.week_totals || []).find((x) => x.week_start === wk);
        stack[p][i] += metricOf(b, metric);
      });
    });
    const used = pillarKeys.filter((p) => stack[p].some((v) => v > 0));
    return { stack, used };
  }, [filteredMembers, weeks, metric, pillarKeys]);

  /* ---- Inflow vs Outflow (opened vs merged) ---- */
  const flowOpened = weeks.map((w) => teamSum(filteredMembers, [w], 'opened'));
  const flowMerged = weeks.map((w) => teamSum(filteredMembers, [w], 'prs'));

  /* ---- Donut (selected metric, by pillar) ---- */
  const donutSlices = useMemo(() => {
    const totals = {};
    pillarKeys.forEach((p) => { totals[p] = 0; });
    filteredMembers.forEach((m) => { totals[pillarOf(m)] += memberSum(m, weeks, metric); });
    return pillarKeys
      .map((p) => ({ label: p, value: totals[p], color: pillarColor[p] }))
      .filter((s) => s.value > 0)
      .sort((a, b) => b.value - a.value);
  }, [filteredMembers, weeks, metric, pillarKeys, pillarColor]);
  const donutTotal = donutSlices.reduce((a, s) => a + s.value, 0);

  /* ---- Top movers (recent half vs prior half) ---- */
  const movers = useMemo(() => {
    return filteredMembers
      .map((m) => {
        const r = memberSum(m, recentWks, metric);
        const p = memberSum(m, priorWks, metric);
        return { id: m.id, name: m.name, recent: r, prior: p, delta: r - p };
      })
      .filter((r) => r.recent + r.prior > 0)
      .sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta))
      .slice(0, 8);
  }, [filteredMembers, metric, recentWks, priorWks]);
  const moversMax = Math.max(1, ...movers.map((r) => Math.abs(r.delta)));

  /* ---- Per-member ranking ---- */
  const ranking = useMemo(() => {
    return filteredMembers
      .map((m) => ({ id: m.id, name: m.name, pillar: pillarOf(m), val: memberSum(m, weeks, metric) }))
      .filter((r) => r.val > 0)
      .sort((a, b) => b.val - a.val);
  }, [filteredMembers, weeks, metric]);
  const rankingMax = Math.max(1, ...ranking.map((r) => r.val));

  return (
    <>
      <div className="ta-toolbar">
        <span className="ta-filters__lbl">Window</span>
        <div className="ta-seg">
          {[4, 12, 26].map((w) => (
            <button key={w}
                    className={windowWeeks === w ? 'is-active' : ''}
                    onClick={() => setWindowWeeks(w)}>{w} weeks</button>
          ))}
        </div>
        <span className="ta-filters__lbl" style={{ marginLeft: 12 }}>Pillar</span>
        <div className="ta-filters">
          <button type="button"
                  className={`ta-pill${pillarFilter === 'all' ? ' is-active' : ''}`}
                  onClick={() => setPillarFilter('all')}>All</button>
          {orderedPillarNames.map((p) => (
            <button key={p} type="button"
                    className={`ta-pill${pillarFilter === p ? ' is-active' : ''}`}
                    onClick={() => setPillarFilter(p)}>{p}</button>
          ))}
          <button type="button"
                  className={`ta-pill${pillarFilter === 'Unassigned' ? ' is-active' : ''}`}
                  onClick={() => setPillarFilter('Unassigned')}>Unassigned</button>
        </div>
      </div>

      <div className="ta-kpis">
        {kpis.map((k) => (
          <button key={k.val}
                  className={`ta-kpi ta-kpi--btn${metric === k.val ? ' is-active' : ''}`}
                  onClick={() => setMetric(k.val)}
                  type="button">
            <div className="ta-kpi__lbl">{k.label}</div>
            <div className="ta-kpi__val">{fmt(k.total)}</div>
            {k.delta && (
              <div className={`ta-kpi__delta ta-delta ${k.delta.kind}`}>
                {k.delta.kind === 'flat'
                  ? `±${k.delta.pct}%`
                  : `${k.delta.kind === 'up' ? '▲' : '▼'} ${k.delta.pct}%${k.delta.novel ? ' (new)' : ''}`}
                {' vs prior half'}
              </div>
            )}
          </button>
        ))}
      </div>
      <p className="ta-kpis-hint">Click a metric tile to drive the panels below.</p>

      <div className="ta-dash-grid">
        <div className="ta-panel ta-dash-grid__span">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">
                Throughput trend · <span style={{ color: METRIC_COLOR[metric] }}>{METRIC_OPTIONS.find((o) => o.val === metric)?.label}</span>
              </div>
              <div className="ta-panel__sub">Team aggregate · weekly</div>
            </div>
            <div className="ta-legend">
              {METRIC_OPTIONS.map((o) => (
                <button key={o.val}
                        className={`ta-legend__item${metric === o.val ? ' is-active' : ''}`}
                        onClick={() => setMetric(o.val)} type="button">
                  <i style={{ background: METRIC_COLOR[o.val] }} />{o.label}
                </button>
              ))}
            </div>
          </header>
          <div className="ta-chartwrap">
            <LineChart weeks={weeks} seriesByMetric={trendSeries}
                       metricKeys={METRIC_OPTIONS.map((o) => o.val)} highlight={metric} />
          </div>
        </div>

        <div className="ta-panel">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">Pillar mix over time</div>
              <div className="ta-panel__sub">Stacked area · {METRIC_OPTIONS.find((o) => o.val === metric)?.label}</div>
            </div>
          </header>
          <div className="ta-chartwrap">
            <StackedAreaChart weeks={weeks}
                              stackByCategory={pillarStack.stack}
                              categories={pillarStack.used}
                              colorMap={pillarColor} />
            <div className="ta-legend" style={{ marginTop: 10 }}>
              {pillarStack.used.map((p) => (
                <span key={p}><i style={{ background: pillarColor[p] }} />{p}</span>
              ))}
            </div>
          </div>
        </div>

        <div className="ta-panel">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">Inflow vs outflow</div>
              <div className="ta-panel__sub">PRs opened (blue) vs PRs merged (red) · per week</div>
            </div>
          </header>
          <div className="ta-chartwrap">
            <GroupedBarChart weeks={weeks}
                             seriesA={flowOpened} seriesB={flowMerged}
                             labelA="Opened" labelB="Merged"
                             colorA="var(--ocean)" colorB="var(--accent)" />
          </div>
        </div>

        <div className="ta-panel">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">Distribution by pillar</div>
              <div className="ta-panel__sub">{METRIC_OPTIONS.find((o) => o.val === metric)?.label}</div>
            </div>
          </header>
          <div className="ta-chartwrap ta-donut-row">
            <DonutChart slices={donutSlices} />
            <div className="ta-donut-legend">
              {donutSlices.map((s) => (
                <div key={s.label} className="row">
                  <span className="swatch" style={{ background: s.color }} />
                  <span className="lbl">{s.label}</span>
                  <span className="v">{fmt(s.value)}</span>
                  <span className="pct">{donutTotal > 0 ? Math.round((s.value / donutTotal) * 100) : 0}%</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="ta-panel">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">Top movers</div>
              <div className="ta-panel__sub">Recent half vs prior half · per-member delta</div>
            </div>
          </header>
          <div className="ta-chartwrap">
            <div className="ta-bars">
              {movers.length === 0 ? (
                <p className="ta-meta">No mover data in this window.</p>
              ) : movers.map((r) => {
                const w = (Math.abs(r.delta) / moversMax) * 100;
                const sign = r.delta > 0 ? '+' : (r.delta < 0 ? '' : '±');
                const color = r.delta >= 0 ? METRIC_COLOR[metric] : 'var(--fg-muted)';
                return (
                  <button key={r.id} type="button" className="ta-bar-row ta-bar-row--btn"
                          onClick={() => onMemberClick(r.id)}>
                    <div className="ta-bar-label">{r.name}</div>
                    <div className="ta-bar-track"><div className="ta-bar-fill" style={{ width: `${w}%`, background: color }} /></div>
                    <div className="ta-bar-num">{sign}{fmt(r.delta)}</div>
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        <div className="ta-panel ta-dash-grid__span">
          <header className="ta-panel__head">
            <div>
              <div className="ta-panel__title">Per-member ranking</div>
              <div className="ta-panel__sub">Selected window · {METRIC_OPTIONS.find((o) => o.val === metric)?.label} · click to drill into profile</div>
            </div>
          </header>
          <div className="ta-chartwrap">
            <div className="ta-bars">
              {ranking.length === 0 ? (
                <p className="ta-meta">No activity for this metric in the window.</p>
              ) : ranking.map((r) => {
                const w = (r.val / rankingMax) * 100;
                return (
                  <button key={r.id} type="button" className="ta-bar-row ta-bar-row--btn"
                          onClick={() => onMemberClick(r.id)}>
                    <div className="ta-bar-label">
                      {r.name}<span style={{ color: 'var(--fg-muted)', marginLeft: 8, fontSize: 10 }}>{r.pillar}</span>
                    </div>
                    <div className="ta-bar-track"><div className="ta-bar-fill" style={{ width: `${w}%`, background: pillarColor[r.pillar] || 'var(--accent)' }} /></div>
                    <div className="ta-bar-num">{fmt(r.val)}</div>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {loading && <div className="ta-loading">Refreshing…</div>}
    </>
  );
}
