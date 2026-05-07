/*
 * Team Analytics — production view, faithful to docs/team-analytics/prototype.html.
 *
 * Three tabs:
 *   1. Team overview   — sortable table with deltas, pillar filter pills,
 *                        throughput-by-pillar stack chart, review network panel.
 *   2. Member profile  — sidebar identity + KPI grid · multi-line weekly chart
 *                        · components-touched bars · this-sprint stats. Has
 *                        an inline edit form for github_login + pillar.
 *   3. Slice explorer  — pivot of (member|pillar) × (week|quarter)
 *                        on (PRs merged|Jira done|Story pts) with heat tint.
 *
 * Tab + selected member are reflected into ?tab=…&member=… so the view
 * is deep-linkable and the browser's back button works as expected.
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { contributionsAPI, handleAPIError } from '../services/api';
import './TeamAnalytics.css';

/* -------------------------------------------------------------------- */
/*  Generic helpers                                                      */
/* -------------------------------------------------------------------- */

const PILLAR_COLORS = {
  Unassigned: '#a39a8a',
};
const SERIES_PALETTE = ['#b8443c', '#2c5d75', '#b87a1a', '#3a6d4b', '#7a4f9c', '#9c4f4f', '#4f7f9c'];

const formatHours = (h) => (h == null ? '—' : `${(+h).toFixed(1)}h`);
const formatPct = (p) => (p == null || isNaN(p) ? '—' : `${Math.round(p)}%`);
const formatPctOne = (p) => (p == null || isNaN(p) ? '—' : `${(+p).toFixed(1)}%`);
const formatDate = (s) => {
  if (!s) return '—';
  try { return new Date(s).toISOString().slice(0, 16).replace('T', ' '); }
  catch { return s; }
};

const pick = (obj, ...keys) => {
  for (const k of keys) {
    if (obj && obj[k] !== undefined && obj[k] !== null && obj[k] !== '') return obj[k];
  }
  return '';
};

const initialsOf = (name) =>
  String(name || '?')
    .split(/\s+/)
    .filter(Boolean)
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase() || '?';

const colorForPillar = (p, fallbackIdx = 0) => {
  if (!p) return PILLAR_COLORS.Unassigned;
  if (PILLAR_COLORS[p]) return PILLAR_COLORS[p];
  let h = 0;
  for (let i = 0; i < p.length; i++) h = (h * 31 + p.charCodeAt(i)) >>> 0;
  return SERIES_PALETTE[(h + fallbackIdx) % SERIES_PALETTE.length];
};

/* -------------------------------------------------------------------- */
/*  URL state — ?tab=&member=                                            */
/* -------------------------------------------------------------------- */

const VALID_TABS = new Set(['team', 'member', 'slice']);
const readURLState = () => {
  if (typeof window === 'undefined') return { tab: 'team', id: '' };
  try {
    const p = new URLSearchParams(window.location.search);
    let tab = p.get('tab') || '';
    const id = p.get('member') || p.get('id') || '';
    if (!VALID_TABS.has(tab)) tab = id ? 'member' : 'team';
    return { tab, id };
  } catch {
    return { tab: 'team', id: '' };
  }
};
const writeURLState = ({ tab, id }) => {
  if (typeof window === 'undefined') return;
  try {
    const url = new URL(window.location.href);
    if (tab && tab !== 'team') url.searchParams.set('tab', tab);
    else url.searchParams.delete('tab');
    if (id) url.searchParams.set('member', id);
    else url.searchParams.delete('member');
    window.history.replaceState({}, '', url.toString());
  } catch {
    /* SecurityError on file:// — ignore */
  }
};

/* -------------------------------------------------------------------- */
/*  Delta indicator (last-4 vs prior-4 weeks)                            */
/* -------------------------------------------------------------------- */

const computeDelta = (series, key) => {
  if (!series || series.length < 8) return { kind: 'flat', pct: 0, ok: false };
  const recent = series.slice(-4).reduce((a, s) => a + (s[key] || 0), 0);
  const prior  = series.slice(-8, -4).reduce((a, s) => a + (s[key] || 0), 0);
  if (prior === 0) {
    if (recent > 0) return { kind: 'up', pct: 100, ok: true, novel: true };
    return { kind: 'flat', pct: 0, ok: false };
  }
  const pct = Math.round(((recent - prior) / prior) * 100);
  if (Math.abs(pct) < 5) return { kind: 'flat', pct: Math.abs(pct), ok: true };
  return { kind: pct > 0 ? 'up' : 'down', pct: Math.abs(pct), ok: true };
};

function DeltaPill({ d }) {
  if (!d || !d.ok) return <span className="ta-delta flat">·</span>;
  if (d.kind === 'flat') return <span className="ta-delta flat">±{d.pct}%</span>;
  return <span className={`ta-delta ${d.kind}`}>{d.kind === 'up' ? '▲' : '▼'} {d.pct}%</span>;
}

/* -------------------------------------------------------------------- */
/*  Sparkline (table cell)                                               */
/* -------------------------------------------------------------------- */

function Spark({ values, color = 'currentColor', width = 110, height = 28 }) {
  if (!values || values.length === 0) {
    return <span className="ta-meta">—</span>;
  }
  const max = Math.max(1, ...values);
  const step = (width - 4) / Math.max(1, values.length - 1);
  const pts = values.map((v, i) => `${2 + i * step},${height - 2 - (v / max) * (height - 4)}`);
  const last = pts[pts.length - 1].split(',');
  return (
    <svg className="ta-spark" width={width} height={height} viewBox={`0 0 ${width} ${height}`}>
      <polyline fill="none" stroke={color} strokeWidth="1.4" points={pts.join(' ')} />
      <circle cx={last[0]} cy={last[1]} r="2.2" fill={color} />
    </svg>
  );
}

/* -------------------------------------------------------------------- */
/*  Pillar throughput stack chart (Team tab)                             */
/* -------------------------------------------------------------------- */

function PillarStack({ buckets }) {
  if (!buckets || buckets.length === 0) {
    return <p className="ta-meta">No throughput data in window.</p>;
  }
  const byWeek = new Map();
  const pillars = new Set();
  buckets.forEach((b) => {
    pillars.add(b.pillar);
    const k = b.week_start;
    if (!byWeek.has(k)) byWeek.set(k, {});
    byWeek.get(k)[b.pillar] = b.prs_merged || 0;
  });
  const weekKeys = [...byWeek.keys()].sort();
  const pillarList = [...pillars].sort();
  const stack = weekKeys.map((wk) => {
    const cell = byWeek.get(wk);
    return { week: wk, ...Object.fromEntries(pillarList.map((p) => [p, cell[p] || 0])) };
  });
  const max = Math.max(1, ...stack.map((s) => pillarList.reduce((a, p) => a + s[p], 0)));

  const W = 600, H = 240, pad = { l: 36, r: 12, t: 14, b: 36 };
  const xStep = (W - pad.l - pad.r) / Math.max(1, stack.length);

  const colors = Object.fromEntries(pillarList.map((p, i) => [p, colorForPillar(p, i)]));

  return (
    <>
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
        {stack.map((s, i) => {
          let yCur = H - pad.b;
          return (
            <g key={s.week}>
              {pillarList.map((p) => {
                const h = (s[p] / max) * (H - pad.t - pad.b);
                yCur -= h;
                return (
                  <rect key={p}
                    x={pad.l + i * xStep + 2}
                    y={yCur}
                    width={Math.max(1, xStep - 4)}
                    height={Math.max(0, h)}
                    fill={colors[p]}
                    opacity="0.85" />
                );
              })}
            </g>
          );
        })}
        {stack.map((s, i) => {
          if (i % 2 !== 0 && i !== stack.length - 1) return null;
          const lab = (s.week || '').slice(5, 10);
          return (
            <text key={i} className="ta-chart-axis"
                  x={pad.l + i * xStep + xStep / 2}
                  y={H - pad.b + 14} textAnchor="middle">{lab}</text>
          );
        })}
      </svg>
      <div className="ta-legend" style={{ marginTop: 10 }}>
        {pillarList.map((p) => (
          <span key={p}><i style={{ background: colors[p] }} />{p}</span>
        ))}
      </div>
    </>
  );
}

/* -------------------------------------------------------------------- */
/*  Network density panel (Team tab)                                     */
/* -------------------------------------------------------------------- */

function NetworkPanel({ network }) {
  if (!network || !network.prs_considered) {
    return (
      <div className="ta-chartwrap--small">
        <p className="ta-fact">
          Review-network metrics need at least one PR + review in the window.
          Once GitHub sync has populated history (and members have <code>github_login</code>
          set on their profile), this panel surfaces orphan rate, first-review p50/p90,
          and cross-pillar review percentage.
        </p>
      </div>
    );
  }
  const under24 = formatPct(network.first_review_under_24h_pct);
  const cross   = formatPct(network.cross_pillar_review_pct);
  return (
    <div className="ta-chartwrap--small">
      <p className="ta-fact">
        Last 12 weeks: <strong>{under24}</strong> of PRs received their first review within 24 hours.
        Cross-pillar review participation is <strong>{cross}</strong> — that's a
        leading indicator of knowledge sharing, and you can slice it on the Slice tab.
      </p>
      <div className="ta-bullets">
        <div>→ p50 first-review latency · <b>{formatHours(network.first_review_p50_hours)}</b></div>
        <div>→ p90 first-review latency · <b>{formatHours(network.first_review_p90_hours)}</b></div>
        <div>→ orphan PRs (no review)   · <b>{formatPctOne(network.orphan_pct)}</b> of {network.prs_considered}</div>
      </div>
    </div>
  );
}

/* -------------------------------------------------------------------- */
/*  Multi-line weekly trend (Member profile)                             */
/* -------------------------------------------------------------------- */

function TrendChart({ weeks }) {
  if (!weeks || weeks.length === 0) {
    return <p className="ta-meta">No activity in window.</p>;
  }
  const W = 720, H = 220, pad = { l: 40, r: 12, t: 14, b: 30 };
  const allMax = Math.max(1, ...weeks.flatMap((w) => [w.jira_done || 0, w.prs_merged || 0, w.reviews || 0]));
  const xStep = (W - pad.l - pad.r) / Math.max(1, weeks.length - 1);
  const yOf = (v) => H - pad.b - (v / allMax) * (H - pad.t - pad.b);
  const xOf = (i) => pad.l + i * xStep;

  const lines = [
    { key: 'jira_done', color: '#2c5d75' },
    { key: 'reviews',   color: '#b87a1a' },
    { key: 'prs_merged',color: 'var(--ta-accent, #b8443c)' },
  ];

  return (
    <>
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
        {weeks.map((w, i) => (
          <text key={i} className="ta-chart-axis"
                x={xOf(i)} y={H - 10} textAnchor="middle">w{i + 1}</text>
        ))}
        {lines.map((ln) => {
          const pts = weeks.map((w, i) => `${xOf(i)},${yOf(w[ln.key] || 0)}`);
          return (
            <g key={ln.key}>
              <polyline fill="none" stroke={ln.color} strokeWidth="1.7" points={pts.join(' ')} />
              {pts.map((p, i) => {
                const [cx, cy] = p.split(',');
                return <circle key={i} cx={cx} cy={cy} r="2.5" fill={ln.color} />;
              })}
            </g>
          );
        })}
      </svg>
      <div className="ta-legend" style={{ marginTop: 6 }}>
        <span><i style={{ background: 'var(--ta-accent, #b8443c)' }} />PRs merged</span>
        <span><i style={{ background: '#2c5d75' }} />Jira done</span>
        <span><i style={{ background: '#b87a1a' }} />Reviews</span>
      </div>
    </>
  );
}

/* -------------------------------------------------------------------- */
/*  Status badge (last-sync indicator)                                   */
/* -------------------------------------------------------------------- */

function StatusBadge({ source, info }) {
  let label, kind;
  if (!info || info.status === 'never_run') {
    label = `${source} · never synced`;
    kind = 'warn';
  } else if (info.error) {
    label = `${source} · error`;
    kind = 'bad';
  } else {
    label = `${source} · ${formatDate(info.last_run_at)}`;
    kind = 'ok';
  }
  return <span className={`ta-badge ta-badge--${kind}`}>{label}</span>;
}

/* -------------------------------------------------------------------- */
/*  Top-level component                                                  */
/* -------------------------------------------------------------------- */

const TABLE_COLS = [
  { key: 'name',    label: 'Member',     align: 'left',  sortable: true },
  { key: 'pillar',  label: 'Pillar',     align: 'left',  sortable: true },
  { key: 'jira',    label: 'Jira done',  align: 'right', sortable: true, withDelta: 'jira_done' },
  { key: 'points',  label: 'Story pts',  align: 'right', sortable: true },
  { key: 'prs',     label: 'PRs merged', align: 'right', sortable: true, withDelta: 'prs_merged' },
  { key: 'reviews', label: 'Reviews',    align: 'right', sortable: true, withDelta: 'reviews' },
  { key: 'latency', label: 'Review p50', align: 'right', sortable: true },
  { key: 'spark',   label: 'PRs / wk',   align: 'left',  sortable: false },
];

export default function TeamAnalytics() {
  const initial = readURLState();
  const [tab, setTab] = useState(initial.tab);
  const [selectedID, setSelectedID] = useState(initial.id);

  const [members, setMembers] = useState([]);
  const [team, setTeam] = useState([]);
  const [status, setStatus] = useState(null);
  const [network, setNetwork] = useState(null);
  const [pillars, setPillars] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [pillarFilter, setPillarFilter] = useState('all');

  const [sortKey, setSortKey] = useState('name');
  const [sortDir, setSortDir] = useState('asc');

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        setLoading(true);
        const [membersResp, teamResp, statusResp, netResp, pillResp] = await Promise.all([
          contributionsAPI.listMembers().catch(() => ({ members: [] })),
          contributionsAPI.team().catch(() => ({ members: [] })),
          contributionsAPI.status().catch(() => null),
          contributionsAPI.network().catch(() => null),
          contributionsAPI.pillars().catch(() => ({ buckets: [] })),
        ]);
        if (cancelled) return;
        setMembers(membersResp.members || []);
        setTeam(teamResp.members || []);
        setStatus(statusResp);
        setNetwork(netResp);
        setPillars(pillResp.buckets || []);
      } catch (e) {
        if (cancelled) return;
        const err = handleAPIError(e);
        setError(err.message);
        toast.error(err.message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [reloadTick]);

  const rows = useMemo(() => {
    const byID = new Map((team || []).map((t) => [t.member_id, t]));
    const merged = (members || []).map((m) => {
      const id = pick(m, 'id', 'ID');
      const t = byID.get(id) || {};
      const week_totals = t.week_totals || [];
      return {
        id,
        name: pick(m, 'display_name', 'DisplayName') || id || 'unknown',
        github: pick(m, 'github_login', 'GitHubLogin'),
        email:  pick(m, 'email', 'Email'),
        jira_account_id: pick(m, 'jira_account_id', 'JiraAccountID'),
        pillar: pick(m, 'pillar_id', 'PillarID'),
        active: pick(m, 'active', 'Active') !== false,
        jira:    t.jira_issues_done || 0,
        points:  t.jira_points_done || 0,
        prs:     t.prs_merged || 0,
        reviews: t.prs_reviewed || 0,
        latency: t.review_latency_p50_hours,
        week_totals,
      };
    });
    (team || []).forEach((t) => {
      if (!members.find((m) => pick(m, 'id', 'ID') === t.member_id)) {
        merged.push({
          id: t.member_id, name: t.member_id,
          github: '', email: '', jira_account_id: '',
          pillar: '', active: true,
          jira:    t.jira_issues_done || 0,
          points:  t.jira_points_done || 0,
          prs:     t.prs_merged || 0,
          reviews: t.prs_reviewed || 0,
          latency: t.review_latency_p50_hours,
          week_totals: t.week_totals || [],
        });
      }
    });
    return merged;
  }, [members, team]);

  const pillarNames = useMemo(() => {
    const set = new Set();
    rows.forEach((r) => { if (r.pillar) set.add(r.pillar); });
    return [...set].sort();
  }, [rows]);

  const filteredRows = useMemo(() => {
    if (pillarFilter === 'all') return rows;
    if (pillarFilter === '__unassigned') return rows.filter((r) => !r.pillar);
    return rows.filter((r) => r.pillar === pillarFilter);
  }, [rows, pillarFilter]);

  const sortedRows = useMemo(() => {
    const arr = [...filteredRows];
    arr.sort((a, b) => {
      const av = a[sortKey] ?? 0;
      const bv = b[sortKey] ?? 0;
      if (av < bv) return sortDir === 'asc' ? -1 : 1;
      if (av > bv) return sortDir === 'asc' ? 1 : -1;
      return 0;
    });
    return arr;
  }, [filteredRows, sortKey, sortDir]);

  const handleSort = (k) => {
    if (k === sortKey) setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    else {
      setSortKey(k);
      setSortDir(k === 'name' || k === 'pillar' ? 'asc' : 'desc');
    }
  };

  const goToTab = useCallback((next, id = selectedID) => {
    if (next === 'member' && !id) {
      setTab('team');
      writeURLState({ tab: 'team', id: '' });
      return;
    }
    setTab(next);
    writeURLState({ tab: next, id: next === 'member' ? id : '' });
  }, [selectedID]);

  const openMember = useCallback((id) => {
    setSelectedID(id);
    setTab('member');
    writeURLState({ tab: 'member', id });
  }, []);

  const onSavedMember = useCallback(() => setReloadTick((t) => t + 1), []);

  const empty = !loading && rows.length === 0;

  return (
    <div className="ta-root">
      <header className="ta-mast">
        <div>
          <span className="ta-mast__bracket">N° 03 · TEAM ANALYTICS</span>
          <h1 className="ta-mast__title">Velocity, <em>contributions</em>, evolution.</h1>
          <p className="ta-mast__sub">Roadmap Planner · Last 12 weeks · Jira completions + GitHub PRs + reviews</p>
        </div>
        <div>
          <p className="ta-mast__meta">
            Source <strong>Jira + GitHub</strong> · Window <strong>12 weeks</strong>
          </p>
          <div className="ta-status">
            <StatusBadge source="jira"   info={status?.jira} />
            <StatusBadge source="github" info={status?.github} />
          </div>
        </div>
      </header>

      <nav className="ta-tabs" role="tablist">
        <button
          className={`ta-tab${tab === 'team' ? ' is-active' : ''}`}
          onClick={() => goToTab('team')}
          role="tab"
        >
          <span className="ta-tab__num">01</span>Team overview
        </button>
        <button
          className={`ta-tab${tab === 'member' ? ' is-active' : ''}`}
          onClick={() => goToTab('member')}
          disabled={!selectedID}
          role="tab"
          title={selectedID ? '' : 'Select a member from the team table first'}
        >
          <span className="ta-tab__num">02</span>Member profile
        </button>
        <button
          className={`ta-tab${tab === 'slice' ? ' is-active' : ''}`}
          onClick={() => goToTab('slice')}
          role="tab"
        >
          <span className="ta-tab__num">03</span>Slice explorer
        </button>
      </nav>

      {error && <div className="ta-banner ta-banner--bad">Failed to load: {error}</div>}

      {empty && (
        <div className="ta-empty">
          <h3>No analytics data yet</h3>
          <p>
            This view shows once <code>storage.enabled</code> is on and the collector has
            captured at least one cycle.
          </p>
        </div>
      )}

      {!empty && tab === 'team' && (
        <TeamView
          rows={sortedRows}
          pillarNames={pillarNames}
          pillarFilter={pillarFilter}
          onPillarFilter={setPillarFilter}
          sortKey={sortKey}
          sortDir={sortDir}
          onSort={handleSort}
          onRowClick={openMember}
          loading={loading}
          pillars={pillars}
          network={network}
        />
      )}

      {!empty && tab === 'member' && (
        <MemberView
          memberRow={rows.find((r) => r.id === selectedID)}
          onBack={() => goToTab('team')}
          onSaved={onSavedMember}
          allRows={rows}
          onSwitchMember={openMember}
        />
      )}

      {!empty && tab === 'slice' && (
        <SliceView rows={rows} />
      )}
    </div>
  );
}

/* -------------------------------------------------------------------- */
/*  Team Overview tab                                                    */
/* -------------------------------------------------------------------- */

function TeamView({ rows, pillarNames, pillarFilter, onPillarFilter, sortKey, sortDir, onSort, onRowClick, loading, pillars, network }) {
  return (
    <>
      <div className="ta-panel">
        <header className="ta-panel__head">
          <div>
            <div className="ta-panel__title">All members · last 12 weeks</div>
            <div className="ta-panel__sub">Sortable. Click a row for the member profile.</div>
          </div>
          <div className="ta-filters">
            <span className="ta-filters__lbl">Pillar</span>
            <button
              type="button"
              className={`ta-pill${pillarFilter === 'all' ? ' is-active' : ''}`}
              onClick={() => onPillarFilter('all')}
            >All</button>
            {pillarNames.map((p) => (
              <button key={p} type="button"
                className={`ta-pill${pillarFilter === p ? ' is-active' : ''}`}
                onClick={() => onPillarFilter(p)}
              >{p}</button>
            ))}
            <button
              type="button"
              className={`ta-pill${pillarFilter === '__unassigned' ? ' is-active' : ''}`}
              onClick={() => onPillarFilter('__unassigned')}
            >Unassigned</button>
          </div>
        </header>
        <table className="ta-tbl">
          <thead>
            <tr>
              {TABLE_COLS.map((c) => (
                <th key={c.key}
                    className={[
                      c.align === 'right' ? 'right' : '',
                      sortKey === c.key ? 'is-sorted' : '',
                      c.sortable ? '' : 'no-sort',
                    ].join(' ').trim()}
                    onClick={() => c.sortable && onSort(c.key)}>
                  {c.label}
                  {sortKey === c.key && c.sortable && (
                    <span style={{ marginLeft: 4 }}>{sortDir === 'asc' ? '↑' : '↓'}</span>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((m) => {
              const sparkValues = (m.week_totals || []).map((w) => w.prs_merged || 0);
              const dJira    = computeDelta(m.week_totals, 'jira_done');
              const dPRs     = computeDelta(m.week_totals, 'prs_merged');
              const dReviews = computeDelta(m.week_totals, 'reviews');
              const points = Number.isFinite(m.points) ? (+m.points).toFixed(1) : '0.0';
              return (
                <tr key={m.id}
                    className={`row-link${m.active === false ? ' is-inactive' : ''}`}
                    onClick={() => onRowClick(m.id)}>
                  <td>
                    <div className="ta-cell-member">
                      <span className="ta-avatar">{initialsOf(m.name)}</span>
                      <div>
                        <div className="ta-name">{m.name}</div>
                        {m.github && <div className="ta-meta">@{m.github}</div>}
                      </div>
                    </div>
                  </td>
                  <td><span className="ta-meta">{m.pillar || '—'}</span></td>
                  <td className="ta-right"><span className="ta-num">{m.jira}</span><DeltaPill d={dJira} /></td>
                  <td className="ta-right"><span className="ta-num">{points}</span></td>
                  <td className="ta-right"><span className="ta-num">{m.prs}</span><DeltaPill d={dPRs} /></td>
                  <td className="ta-right"><span className="ta-num">{m.reviews}</span><DeltaPill d={dReviews} /></td>
                  <td className="ta-right"><span className="ta-num">{formatHours(m.latency)}</span></td>
                  <td><Spark values={sparkValues} color="var(--ta-accent, #b8443c)" /></td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {loading && <div className="ta-loading">Loading…</div>}
      </div>

      <div className="ta-row">
        <div className="ta-panel">
          <header className="ta-panel__head">
            <div className="ta-panel__title">Throughput, by pillar</div>
            <div className="ta-panel__sub">PRs merged · 12-week stack</div>
          </header>
          <div className="ta-chartwrap">
            <PillarStack buckets={pillars} />
          </div>
        </div>
        <div className="ta-panel">
          <header className="ta-panel__head">
            <div className="ta-panel__title">Review network density</div>
            <div className="ta-panel__sub">Who reviews whom</div>
          </header>
          <NetworkPanel network={network} />
        </div>
      </div>
    </>
  );
}

/* -------------------------------------------------------------------- */
/*  Member Profile tab                                                   */
/* -------------------------------------------------------------------- */

function MemberView({ memberRow, onBack, onSaved, allRows, onSwitchMember }) {
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [editGh, setEditGh] = useState('');
  const [editPillar, setEditPillar] = useState('');
  const [saving, setSaving] = useState(false);

  const id = memberRow?.id;

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const d = await contributionsAPI.member(id);
        if (cancelled) return;
        setDetail(d);
        setEditGh(pick(d.info, 'github_login', 'GitHubLogin') || memberRow.github || '');
        setEditPillar(pick(d.info, 'pillar_id', 'PillarID') || memberRow.pillar || '');
      } catch (e) {
        if (cancelled) return;
        toast.error(`Profile load: ${handleAPIError(e).message}`);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [id, memberRow?.github, memberRow?.pillar]);

  if (!memberRow) {
    return (
      <div className="ta-panel">
        <div className="ta-empty">
          <h3>No member selected</h3>
          <p>Pick a row from the Team Overview tab to drill in.</p>
          <p className="ta-meta">
            <button type="button" className="ta-btn" onClick={onBack}>← back to team</button>
          </p>
        </div>
      </div>
    );
  }

  const info = detail?.info || {};
  const totals = {
    jira:    detail?.jira_issues_done ?? memberRow.jira ?? 0,
    points:  detail?.jira_points_done ?? memberRow.points ?? 0,
    prs:     detail?.prs_merged ?? memberRow.prs ?? 0,
    reviews: detail?.prs_reviewed ?? memberRow.reviews ?? 0,
  };
  const weeks = detail?.week_totals || memberRow.week_totals || [];
  const components = detail?.components_touched || [];
  const sprint = detail?.sprint || null;

  const dirty =
    String(editGh || '').toLowerCase() !== String(pick(info, 'github_login', 'GitHubLogin') || '').toLowerCase() ||
    String(editPillar || '') !== String(pick(info, 'pillar_id', 'PillarID') || '');

  const onSave = async () => {
    setSaving(true);
    try {
      const updated = await contributionsAPI.updateMember(id, {
        github_login: (editGh || '').trim(),
        pillar_id:    (editPillar || '').trim(),
      });
      toast.success('Saved · rebuilding rollups in background');
      setDetail((prev) => ({ ...(prev || {}), info: updated }));
      onSaved?.(updated);
    } catch (e) {
      toast.error(`Save failed: ${handleAPIError(e).message}`);
    } finally {
      setSaving(false);
    }
  };

  const compMax = Math.max(1, ...components.map((c) => c.issues || 0));
  const sortedRoster = [...allRows].sort((a, b) => a.name.localeCompare(b.name));

  return (
    <>
      <button type="button" className="ta-backlink" onClick={onBack}>
        ← Back to team overview
      </button>

      <div className="ta-profile-grid">
        <aside className="ta-profile-card">
          <div className="ta-profile-avatar">{initialsOf(memberRow.name)}</div>
          <div className="ta-profile-name">{pick(info, 'display_name', 'DisplayName') || memberRow.name}</div>
          <div className="ta-profile-pillar">{(memberRow.pillar || 'unassigned').toUpperCase()}</div>
          <div className="ta-profile-handles">
            <div>github · {memberRow.github ? `@${memberRow.github}` : <em style={{ color: 'var(--ta-fg-muted)' }}>not linked</em>}</div>
            <div>jira · {memberRow.email || pick(info, 'email', 'Email') || '—'}</div>
          </div>
          <div className="ta-kpi-grid">
            <div className="ta-kpi"><div className="ta-kpi__lbl">Jira done</div><div className="ta-kpi__val">{totals.jira}</div></div>
            <div className="ta-kpi"><div className="ta-kpi__lbl">Story pts</div><div className="ta-kpi__val">{Number.isFinite(totals.points) ? (+totals.points).toFixed(0) : '0'}</div></div>
            <div className="ta-kpi"><div className="ta-kpi__lbl">PRs merged</div><div className="ta-kpi__val">{totals.prs}</div></div>
            <div className="ta-kpi"><div className="ta-kpi__lbl">Reviews</div><div className="ta-kpi__val">{totals.reviews}</div></div>
          </div>
        </aside>

        <div>
          <div className="ta-card">
            <h3 className="ta-card__title">Weekly contribution
              <span className="ta-legend">
                <span><i style={{ background: 'var(--ta-accent, #b8443c)' }} />PRs merged</span>
                <span><i style={{ background: '#2c5d75' }} />Jira done</span>
                <span><i style={{ background: '#b87a1a' }} />Reviews</span>
              </span>
            </h3>
            <div className="ta-card__sub">Last {weeks.length || 12} weeks · counts per ISO week</div>
            <TrendChart weeks={weeks} />
          </div>

          <div className="ta-card">
            <h3 className="ta-card__title">Components touched
              <span className="ta-legend"><span style={{ fontFamily: 'var(--ta-mono)' }}>Each row = one component · width = issues</span></span>
            </h3>
            <div className="ta-card__sub">Where this member spent their work in the window</div>
            {components.length === 0 ? (
              <p className="ta-meta">No components recorded for this member yet — Jira issues need a Component value.</p>
            ) : (
              components.map((c) => (
                <div key={c.component} className="ta-bar-row">
                  <div className="ta-bar-label">{c.component}</div>
                  <div className="ta-bar-track">
                    <div className="ta-bar-fill" style={{ width: `${(c.issues / compMax) * 100}%` }} />
                  </div>
                  <div className="ta-bar-num">{c.issues}</div>
                </div>
              ))
            )}
          </div>

          <div className="ta-card">
            <h3 className="ta-card__title">This sprint
              <span className="ta-pill" style={{ background: 'var(--ta-bg-soft)' }}>
                {sprint?.name || '—'}
              </span>
            </h3>
            <div className="ta-card__sub">Live · refreshes with the next collection cycle</div>
            <div className="ta-sprint-grid">
              <div><div className="ta-kpi__lbl">In progress</div><div className="ta-num ta-num--lg">{sprint?.wip ?? '—'}</div></div>
              <div><div className="ta-kpi__lbl">Done</div><div className="ta-num ta-num--lg">{sprint?.done ?? '—'}</div></div>
              <div><div className="ta-kpi__lbl">PRs open</div><div className="ta-num ta-num--lg">{sprint?.prs_open ?? '—'}</div></div>
              <div><div className="ta-kpi__lbl">PRs merged</div><div className="ta-num ta-num--lg">{sprint?.prs_merged ?? '—'}</div></div>
            </div>
          </div>

          <div className="ta-card">
            <h3 className="ta-card__title">Identity
              <span className="ta-meta">edit · save triggers an aggregator rebuild</span>
            </h3>
            <div className="ta-card__sub">Map this member to a GitHub login + pillar so PRs and review counts show up</div>
            <div className="ta-edit-grid">
              <label className="ta-field">
                <span className="ta-field-label">Member ID</span>
                <input className="ta-input ta-input--ro" value={memberRow.id} readOnly />
                <span className="ta-field-help">Stable internal slug — derived from email at first sync.</span>
              </label>
              <label className="ta-field">
                <span className="ta-field-label">Email (Jira)</span>
                <input className="ta-input ta-input--ro" value={pick(info, 'email', 'Email') || memberRow.email || ''} readOnly />
              </label>
              <label className="ta-field">
                <span className="ta-field-label">Jira Account ID</span>
                <input className="ta-input ta-input--ro" value={pick(info, 'jira_account_id', 'JiraAccountID') || memberRow.jira_account_id || ''} readOnly />
              </label>
              <label className="ta-field">
                <span className="ta-field-label">GitHub login</span>
                <input className="ta-input"
                       value={editGh}
                       onChange={(e) => setEditGh(e.target.value)}
                       placeholder="alicetan"
                       autoComplete="off"
                       spellCheck={false} />
                <span className="ta-field-help">Empty = no GitHub link.</span>
              </label>
              <label className="ta-field">
                <span className="ta-field-label">Pillar</span>
                <input className="ta-input"
                       value={editPillar}
                       onChange={(e) => setEditPillar(e.target.value)}
                       placeholder="essentials" />
                <span className="ta-field-help">Free-form. Drives the Team filter pills + pillar stack chart.</span>
              </label>
            </div>
            <div className="ta-form-actions">
              {loading && <span className="ta-field-help">Loading…</span>}
              {saving && <span className="ta-field-help">Saving…</span>}
              <button type="button" className="ta-btn" onClick={onBack}>Cancel</button>
              <button type="button" className="ta-btn ta-btn--primary" onClick={onSave}
                      disabled={!dirty || saving || loading}>Save</button>
            </div>
          </div>

          <div className="ta-card">
            <h3 className="ta-card__title">Switch member</h3>
            <div className="ta-card__sub">Jump straight to another profile without going back</div>
            <select className="ta-input" value={memberRow.id} onChange={(e) => onSwitchMember(e.target.value)}>
              {sortedRoster.map((r) => (
                <option key={r.id} value={r.id}>{r.name}{r.pillar ? ` · ${r.pillar}` : ''}</option>
              ))}
            </select>
          </div>
        </div>
      </div>
    </>
  );
}

/* -------------------------------------------------------------------- */
/*  Slice Explorer tab                                                   */
/* -------------------------------------------------------------------- */

const SLICE_METRICS = [
  { val: 'prs',     label: 'PRs merged' },
  { val: 'jira',    label: 'Jira done'  },
  { val: 'points',  label: 'Story pts'  },
];
const SLICE_ROW_AXES = [
  { val: 'pillar', label: 'Pillar' },
  { val: 'member', label: 'Member' },
];
const SLICE_COL_AXES = [
  { val: 'week',    label: 'Week'    },
  { val: 'quarter', label: 'Quarter' },
];

function SliceView({ rows }) {
  const [rowsAxis, setRowsAxis] = useState('pillar');
  const [colsAxis, setColsAxis] = useState('week');
  const [metric,   setMetric]   = useState('prs');

  const allWeeks = useMemo(() => {
    const set = new Map();
    rows.forEach((r) => (r.week_totals || []).forEach((w) => set.set(w.week_start, true)));
    return [...set.keys()].sort();
  }, [rows]);

  const seriesByMember = useMemo(() => {
    const out = new Map();
    rows.forEach((r) => {
      const byWeek = new Map((r.week_totals || []).map((w) => [w.week_start, w]));
      out.set(r.id, allWeeks.map((wk) => byWeek.get(wk) || { jira_done: 0, prs_merged: 0, reviews: 0 }));
    });
    return out;
  }, [rows, allWeeks]);

  const metricGetter = useCallback((b) => {
    if (metric === 'jira')   return b.jira_done   || 0;
    if (metric === 'points') return 0;
    return b.prs_merged || 0;
  }, [metric]);

  const rowKeys = useMemo(() => {
    if (rowsAxis === 'member') {
      return rows.slice().sort((a, b) => a.name.localeCompare(b.name)).map((r) => r.id);
    }
    const set = new Set();
    rows.forEach((r) => set.add(r.pillar || 'Unassigned'));
    return [...set].sort();
  }, [rows, rowsAxis]);

  const cols = useMemo(() => {
    if (colsAxis === 'week') {
      return allWeeks.map((wk, i) => ({ key: wk, label: `w${i + 1}` }));
    }
    const chunkSize = Math.max(1, Math.ceil(allWeeks.length / 4));
    const out = [];
    for (let i = 0; i < 4; i++) {
      const start = i * chunkSize;
      const end = Math.min(allWeeks.length, start + chunkSize);
      if (start >= end) break;
      out.push({ key: `q${i}`, label: `Q${i + 1}`, range: [start, end] });
    }
    return out;
  }, [allWeeks, colsAxis]);

  const matrix = useMemo(() => rowKeys.map((rk) => {
    const matches = rowsAxis === 'member'
      ? rows.filter((r) => r.id === rk)
      : rows.filter((r) => (r.pillar || 'Unassigned') === rk);
    const cells = cols.map((col) => {
      if (colsAxis === 'week') {
        if (metric === 'points') return 0;
        const idx = allWeeks.indexOf(col.key);
        if (idx < 0) return 0;
        return matches.reduce((acc, r) => {
          const series = seriesByMember.get(r.id) || [];
          return acc + metricGetter(series[idx] || {});
        }, 0);
      }
      const [start, end] = col.range;
      if (metric === 'points') {
        const perQuarter = matches.reduce((acc, r) => acc + (r.points || 0), 0) / Math.max(1, cols.length);
        return Math.round(perQuarter);
      }
      return matches.reduce((acc, r) => {
        const series = seriesByMember.get(r.id) || [];
        return acc + series.slice(start, end).reduce((s, b) => s + metricGetter(b), 0);
      }, 0);
    });
    const max = Math.max(1, ...cells);
    const total = cells.reduce((a, b) => a + b, 0);
    return { rowKey: rk, cells, max, total };
  }), [rowKeys, cols, rows, rowsAxis, colsAxis, metric, seriesByMember, allWeeks, metricGetter]);

  const labelOfRow = (rk) => {
    if (rowsAxis === 'member') {
      const r = rows.find((x) => x.id === rk);
      return r ? r.name : rk;
    }
    return rk;
  };

  return (
    <div className="ta-panel">
      <header className="ta-panel__head">
        <div>
          <div className="ta-panel__title">Slice explorer</div>
          <div className="ta-panel__sub">2 axes + 1 metric. Always.</div>
        </div>
      </header>

      <div className="ta-slice-filters">
        <span className="ta-filters__lbl">Rows</span>
        <div className="ta-seg">
          {SLICE_ROW_AXES.map((o) => (
            <button key={o.val}
                    className={rowsAxis === o.val ? 'is-active' : ''}
                    onClick={() => setRowsAxis(o.val)}>{o.label}</button>
          ))}
        </div>
        <span className="ta-filters__lbl" style={{ marginLeft: 12 }}>Cols</span>
        <div className="ta-seg">
          {SLICE_COL_AXES.map((o) => (
            <button key={o.val}
                    className={colsAxis === o.val ? 'is-active' : ''}
                    onClick={() => setColsAxis(o.val)}>{o.label}</button>
          ))}
        </div>
        <span className="ta-filters__lbl" style={{ marginLeft: 12 }}>Metric</span>
        <div className="ta-seg">
          {SLICE_METRICS.map((o) => (
            <button key={o.val}
                    className={metric === o.val ? 'is-active' : ''}
                    onClick={() => setMetric(o.val)}>{o.label}</button>
          ))}
        </div>
      </div>

      <div className="ta-pivot-wrap">
        <table className="ta-pivot">
          <thead>
            <tr>
              <th>{rowsAxis.toUpperCase()}</th>
              {cols.map((c) => <th key={c.key}>{c.label}</th>)}
              <th>Σ</th>
            </tr>
          </thead>
          <tbody>
            {matrix.map((row) => (
              <tr key={row.rowKey}>
                <td>{labelOfRow(row.rowKey)}</td>
                {row.cells.map((v, i) => (
                  <td key={i}
                      className={`heat${v === 0 ? ' heat-zero' : ''}`}
                      style={{ '--h': (v / row.max).toFixed(2) }}>
                    <span>{v}</span>
                  </td>
                ))}
                <td style={{ fontWeight: 600 }}>{row.total}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="ta-pivot-foot">
        Heat is per-row max. Cells are absolute counts. Story-points by week is left
        empty because the rollup keeps total points only — switch to "Quarter" for
        an even split, or use the Team Overview column.
      </div>
    </div>
  );
}
