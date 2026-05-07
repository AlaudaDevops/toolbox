/*
 * Team Analytics — production view.
 *
 * Layout:
 *   - team-overview table (clickable rows)
 *   - inline Member Profile drawer with:
 *       · identity card + editable github_login (PATCH /members/:id)
 *       · KPI tiles
 *       · 3-line weekly trend (Jira / PRs / Reviews)
 *   - last-sync footer per source
 *
 * The Jira → GitHub identity link is intentionally manual: the user
 * fills in `github_login` here, the backend triggers an aggregator
 * rebuild, and PR / review counts repopulate on the next read. We do
 * not auto-match by email — too many false positives on profiles that
 * don't expose email publicly.
 *
 * Inactive Jira users (deactivated accounts) are filtered server-side
 * by default. Pass ?include_inactive=1 to the URL to see them.
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { contributionsAPI, handleAPIError } from '../services/api';
import './TeamAnalytics.css';

const formatHours = (h) => (h == null ? '—' : `${h.toFixed(1)}h`);
const formatDate = (s) => {
  if (!s) return '—';
  try { return new Date(s).toISOString().slice(0, 16).replace('T', ' '); }
  catch { return s; }
};

const initialsOf = (name) =>
  String(name || '?')
    .split(/\s+/)
    .filter(Boolean)
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase() || '?';

// pick walks an object for the first non-empty value across snake_case
// + PascalCase aliases. Defensive against API rollout windows.
const pick = (obj, ...keys) => {
  for (const k of keys) {
    if (obj && obj[k] !== undefined && obj[k] !== null && obj[k] !== '') return obj[k];
  }
  return '';
};

// readQueryMember is a read-only sniff of ?member=<id> from the URL so
// teammates can deep-link to a profile. Updates use replaceState
// instead of pushState to avoid pollluting browser history with every
// table click.
const readQueryMember = () => {
  if (typeof window === 'undefined') return '';
  try { return new URLSearchParams(window.location.search).get('member') || ''; }
  catch { return ''; }
};
const writeQueryMember = (id) => {
  if (typeof window === 'undefined') return;
  try {
    const url = new URL(window.location.href);
    if (id) url.searchParams.set('member', id);
    else url.searchParams.delete('member');
    window.history.replaceState({}, '', url.toString());
  } catch {
    /* SecurityError on file:// — ignore */
  }
};

// Tiny inline sparkline. Used in the table row.
function Spark({ values, color = 'currentColor', width = 96, height = 22 }) {
  if (!values || values.length === 0) {
    return <span className="ta-spark ta-spark--empty">—</span>;
  }
  const max = Math.max(1, ...values);
  const step = (width - 4) / Math.max(1, values.length - 1);
  const pts = values.map((v, i) => `${2 + i * step},${height - 2 - (v / max) * (height - 4)}`).join(' ');
  return (
    <svg className="ta-spark" width={width} height={height} viewBox={`0 0 ${width} ${height}`}>
      <polyline fill="none" stroke={color} strokeWidth="1.4" points={pts} />
    </svg>
  );
}

// Multi-series line chart used in the profile trend section. We render
// hand-written SVG instead of pulling Recharts here because we want
// total control over the visual language (mono labels, restrained
// palette) and the data shape is tiny.
function TrendChart({ series, height = 160 }) {
  const padding = { top: 8, right: 8, bottom: 22, left: 28 };
  const width = 720; // viewBox width; CSS scales it.
  const inner = { w: width - padding.left - padding.right, h: height - padding.top - padding.bottom };
  const lengths = series.map((s) => s.values.length);
  const n = Math.max(0, ...lengths);
  if (n === 0) {
    return <div className="ta-trend-axis" style={{ padding: '12px 0' }}>No activity in window.</div>;
  }
  const max = Math.max(1, ...series.flatMap((s) => s.values));
  const step = inner.w / Math.max(1, n - 1);
  const yOf = (v) => padding.top + inner.h - (v / max) * inner.h;
  const xOf = (i) => padding.left + i * step;

  // Y ticks: 0, max/2, max
  const yTicks = [0, max / 2, max];

  return (
    <>
      <svg className="ta-trend-svg" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none">
        {/* axis lines */}
        <line x1={padding.left} y1={padding.top} x2={padding.left} y2={padding.top + inner.h}
              stroke="var(--border, #d8d2c4)" strokeWidth="1" />
        <line x1={padding.left} y1={padding.top + inner.h} x2={padding.left + inner.w} y2={padding.top + inner.h}
              stroke="var(--border, #d8d2c4)" strokeWidth="1" />
        {/* y-tick gridlines + labels */}
        {yTicks.map((t, i) => (
          <g key={i}>
            <line x1={padding.left} y1={yOf(t)} x2={padding.left + inner.w} y2={yOf(t)}
                  stroke="var(--border-soft, #e6e2d6)" strokeWidth="1" strokeDasharray="2 3" />
            <text className="ta-trend-axis" x={padding.left - 4} y={yOf(t) + 3} textAnchor="end">
              {Math.round(t)}
            </text>
          </g>
        ))}
        {/* x-axis week labels (every other to avoid crowding) */}
        {(series[0]?.labels || []).map((lab, i) => {
          if (i % 2 !== 0 && i !== n - 1) return null;
          return (
            <text key={i} className="ta-trend-axis"
                  x={xOf(i)} y={padding.top + inner.h + 14} textAnchor="middle">
              {lab}
            </text>
          );
        })}
        {/* series */}
        {series.map((s) => {
          const pts = s.values.map((v, i) => `${xOf(i)},${yOf(v)}`).join(' ');
          return (
            <g key={s.label}>
              <polyline fill="none" stroke={s.color} strokeWidth="1.6" points={pts} />
              {s.values.map((v, i) => (
                <circle key={i} cx={xOf(i)} cy={yOf(v)} r="2.5" fill={s.color} />
              ))}
            </g>
          );
        })}
      </svg>
      <div className="ta-trend-legend">
        {series.map((s) => (
          <span key={s.label}><i style={{ background: s.color }} />{s.label}</span>
        ))}
      </div>
    </>
  );
}

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

const COLUMNS = [
  { key: 'name',      label: 'Member',      align: 'left',  sortable: true  },
  { key: 'jira',      label: 'Jira done',   align: 'right', sortable: true  },
  { key: 'points',    label: 'Story pts',   align: 'right', sortable: true  },
  { key: 'prs',       label: 'PRs merged',  align: 'right', sortable: true  },
  { key: 'reviews',   label: 'Reviews',     align: 'right', sortable: true  },
  { key: 'latency',   label: 'Review p50',  align: 'right', sortable: true  },
  { key: 'spark',     label: 'PRs / wk',    align: 'left',  sortable: false },
];

// --------------------------------------------------------------------
// MemberProfile — drawer that opens when a row is clicked. It owns its
// own data fetch (detail endpoint) and the PATCH form. The parent
// passes the row data only as a "first paint" placeholder so the panel
// is never empty while loading.
// --------------------------------------------------------------------
function MemberProfile({ row, onClose, onSaved }) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [github, setGithub] = useState('');
  const [pillar, setPillar] = useState('');

  // Reset state any time the selected member changes.
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const detail = await contributionsAPI.member(row.id);
        if (cancelled) return;
        setData(detail);
        setGithub(pick(detail.info, 'github_login', 'GitHubLogin') || row.github || '');
        setPillar(pick(detail.info, 'pillar_id', 'PillarID') || row.pillar || '');
      } catch (e) {
        if (cancelled) return;
        const err = handleAPIError(e);
        toast.error(`Profile load: ${err.message}`);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [row.id, row.github, row.pillar]);

  const info = data?.info || {};
  const totals = {
    jira: data?.jira_issues_done ?? row.jira ?? 0,
    points: data?.jira_points_done ?? row.points ?? 0,
    prs: data?.prs_merged ?? row.prs ?? 0,
    reviews: data?.prs_reviewed ?? row.reviews ?? 0,
    latency: data?.review_latency_p50_hours ?? row.latency,
  };

  const weeks = data?.week_totals || [];
  const labels = weeks.map((w) => {
    if (!w.week_start) return '';
    try { return new Date(w.week_start).toISOString().slice(5, 10); }
    catch { return ''; }
  });
  const series = [
    { label: 'Jira done',   color: '#b8443c', values: weeks.map((w) => w.jira_done || 0),  labels },
    { label: 'PRs merged',  color: '#4a7d9c', values: weeks.map((w) => w.prs_merged || 0), labels },
    { label: 'Reviews',     color: '#6cb073', values: weeks.map((w) => w.reviews || 0),    labels },
  ];

  const dirty =
    String(github || '').toLowerCase() !== String(pick(info, 'github_login', 'GitHubLogin') || '').toLowerCase()
    || String(pillar || '') !== String(pick(info, 'pillar_id', 'PillarID') || '');

  const onSave = useCallback(async () => {
    setSaving(true);
    try {
      const payload = {
        github_login: (github || '').trim(),
        pillar_id: (pillar || '').trim(),
      };
      const updated = await contributionsAPI.updateMember(row.id, payload);
      toast.success('Saved · rebuilding rollups in background');
      // Refresh the local detail with the updated identity, but keep
      // the previously fetched week_totals (the rebuild is async — the
      // numbers will update on the next reload).
      setData((prev) => ({ ...(prev || {}), info: updated }));
      onSaved?.(updated);
    } catch (e) {
      const err = handleAPIError(e);
      toast.error(`Save failed: ${err.message}`);
    } finally {
      setSaving(false);
    }
  }, [github, pillar, row.id, onSaved]);

  return (
    <section className="ta-profile" aria-label="Member profile">
      <div className="ta-profile-head">
        <div className="ta-profile-id">
          <span className="ta-profile-avatar">{initialsOf(pick(info, 'display_name', 'DisplayName') || row.name)}</span>
          <div>
            <h2 className="ta-profile-name">{pick(info, 'display_name', 'DisplayName') || row.name || row.id}</h2>
            <div className="ta-profile-handle">
              {pick(info, 'email', 'Email') || row.id}
              {pick(info, 'github_login', 'GitHubLogin') && (
                <> · <span style={{ color: 'var(--accent, #b8443c)' }}>@{pick(info, 'github_login', 'GitHubLogin')}</span></>
              )}
            </div>
          </div>
        </div>
        <button className="ta-profile-close" onClick={onClose} aria-label="Close profile">Close ✕</button>
      </div>

      <div className="ta-profile-kpis">
        <div className="ta-kpi"><div className="ta-kpi-label">Jira done</div><div className="ta-kpi-value">{totals.jira}</div></div>
        <div className="ta-kpi"><div className="ta-kpi-label">Story pts</div><div className="ta-kpi-value">{Number.isFinite(totals.points) ? totals.points.toFixed(1) : '0.0'}</div></div>
        <div className="ta-kpi"><div className="ta-kpi-label">PRs merged</div><div className="ta-kpi-value">{totals.prs}</div></div>
        <div className="ta-kpi"><div className="ta-kpi-label">Reviews</div><div className="ta-kpi-value">{totals.reviews}</div></div>
        <div className="ta-kpi"><div className="ta-kpi-label">Review p50</div><div className="ta-kpi-value">{formatHours(totals.latency)}</div></div>
      </div>

      <form className="ta-profile-form" onSubmit={(e) => { e.preventDefault(); onSave(); }}>
        <label className="ta-field">
          <span className="ta-field-label">Member ID</span>
          <input className="ta-input ta-input--ro" value={row.id} readOnly />
          <span className="ta-field-help">Stable internal slug — derived from email at first sync.</span>
        </label>
        <label className="ta-field">
          <span className="ta-field-label">Email (Jira)</span>
          <input className="ta-input ta-input--ro" value={pick(info, 'email', 'Email') || ''} readOnly />
        </label>
        <label className="ta-field">
          <span className="ta-field-label">Jira Account ID</span>
          <input className="ta-input ta-input--ro" value={pick(info, 'jira_account_id', 'JiraAccountID') || ''} readOnly />
        </label>
        <label className="ta-field">
          <span className="ta-field-label">GitHub login *</span>
          <input
            className="ta-input"
            value={github}
            onChange={(e) => setGithub(e.target.value)}
            placeholder="alicetan"
            autoComplete="off"
            spellCheck={false}
          />
          <span className="ta-field-help">Maps PRs / reviews to this person. Empty = no GitHub link.</span>
        </label>
        <label className="ta-field">
          <span className="ta-field-label">Pillar</span>
          <input
            className="ta-input"
            value={pillar}
            onChange={(e) => setPillar(e.target.value)}
            placeholder="essentials"
          />
        </label>
        <div className="ta-form-actions">
          {loading && <span className="ta-field-help">Loading…</span>}
          {saving && <span className="ta-field-help">Saving…</span>}
          <button type="button" className="ta-btn" onClick={onClose}>Cancel</button>
          <button type="submit" className="ta-btn ta-btn--primary" disabled={!dirty || saving || loading}>
            Save
          </button>
        </div>
      </form>

      <div className="ta-trend">
        <h3 className="ta-trend-title">Weekly activity · last {weeks.length || 12} weeks</h3>
        <TrendChart series={series} />
      </div>
    </section>
  );
}

// --------------------------------------------------------------------
// TeamAnalytics — root component
// --------------------------------------------------------------------
export default function TeamAnalytics() {
  const [members, setMembers] = useState([]);
  const [team, setTeam] = useState([]);
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [sortKey, setSortKey] = useState('name');
  const [sortDir, setSortDir] = useState('asc');
  const [selectedID, setSelectedID] = useState(readQueryMember());
  const [reloadTick, setReloadTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        setLoading(true);
        const [membersResp, teamResp, statusResp] = await Promise.all([
          contributionsAPI.listMembers().catch(() => ({ members: [] })),
          contributionsAPI.team().catch(() => ({ members: [] })),
          contributionsAPI.status().catch(() => null),
        ]);
        if (cancelled) return;
        setMembers(membersResp.members || []);
        setTeam(teamResp.members || []);
        setStatus(statusResp);
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
      const sparkValues = (t.week_totals || []).map((w) => w.prs_merged || 0);
      const display = pick(m, 'display_name', 'DisplayName');
      return {
        id,
        name: display || id || 'unknown',
        github: pick(m, 'github_login', 'GitHubLogin'),
        pillar: pick(m, 'pillar_id', 'PillarID'),
        active: pick(m, 'active', 'Active') !== false, // default true
        jira: t.jira_issues_done || 0,
        points: t.jira_points_done || 0,
        prs: t.prs_merged || 0,
        reviews: t.prs_reviewed || 0,
        latency: t.review_latency_p50_hours,
        spark: sparkValues,
      };
    });
    (team || []).forEach((t) => {
      if (!members.find((m) => (pick(m, 'id', 'ID') === t.member_id))) {
        merged.push({
          id: t.member_id, name: t.member_id, github: '', pillar: '', active: true,
          jira: t.jira_issues_done || 0,
          points: t.jira_points_done || 0,
          prs: t.prs_merged || 0,
          reviews: t.prs_reviewed || 0,
          latency: t.review_latency_p50_hours,
          spark: (t.week_totals || []).map((w) => w.prs_merged || 0),
        });
      }
    });
    return merged;
  }, [members, team]);

  const sorted = useMemo(() => {
    const arr = [...rows];
    arr.sort((a, b) => {
      const av = a[sortKey] ?? 0;
      const bv = b[sortKey] ?? 0;
      if (av < bv) return sortDir === 'asc' ? -1 : 1;
      if (av > bv) return sortDir === 'asc' ? 1 : -1;
      return 0;
    });
    return arr;
  }, [rows, sortKey, sortDir]);

  const handleSort = (k) => {
    if (k === sortKey) setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    else {
      setSortKey(k);
      setSortDir(k === 'name' || k === 'pillar' ? 'asc' : 'desc');
    }
  };

  const selectMember = useCallback((id) => {
    setSelectedID(id);
    writeQueryMember(id);
  }, []);

  const closeProfile = useCallback(() => selectMember(''), [selectMember]);

  // Resolve the row that backs the open profile. Falls back to a stub
  // if the id was deep-linked but isn't in the directory yet.
  const selectedRow = useMemo(() => {
    if (!selectedID) return null;
    return rows.find((r) => r.id === selectedID) || { id: selectedID, name: selectedID, github: '', pillar: '' };
  }, [selectedID, rows]);

  const onProfileSaved = useCallback(() => {
    // Re-fetch the team-overview + members directory so the table
    // reflects the updated github_login. Numbers may not have changed
    // yet (rebuild is async), but the avatar/login row will.
    setReloadTick((t) => t + 1);
  }, []);

  const empty = !loading && rows.length === 0;

  return (
    <div className="ta-root">
      <header className="ta-head">
        <div>
          <div className="ta-eyebrow">N° 03 · TEAM ANALYTICS</div>
          <h1 className="ta-title">Velocity, contributions, evolution.</h1>
          <p className="ta-sub">Last 12 weeks · Jira completions + GitHub PRs + reviews</p>
        </div>
        <div className="ta-status">
          <StatusBadge source="jira"   info={status?.jira} />
          <StatusBadge source="github" info={status?.github} />
        </div>
      </header>

      {error && <div className="ta-banner ta-banner--bad">Failed to load: {error}</div>}

      {empty && (
        <div className="ta-empty">
          <h3>No analytics data yet</h3>
          <p>
            This view shows once <code>storage.enabled</code> is on and the collector has
            captured at least one cycle. See{' '}
            <a href="/docs/team-analytics/PROPOSAL.md" target="_blank" rel="noreferrer">
              docs/team-analytics/PROPOSAL.md
            </a>
            {' '}for setup.
          </p>
          <p className="ta-sub">
            For a preview of what this view will look like with data, open{' '}
            <a href="/docs/team-analytics/prototype.html" target="_blank" rel="noreferrer">
              docs/team-analytics/prototype.html
            </a>.
          </p>
        </div>
      )}

      {!empty && (
        <div className="ta-panel">
          <table className="ta-table">
            <thead>
              <tr>
                {COLUMNS.map((c) => (
                  <th
                    key={c.key}
                    className={`ta-th ta-th--${c.align}${sortKey === c.key ? ' is-sorted' : ''}${c.sortable ? ' is-sortable' : ''}`}
                    onClick={() => c.sortable && handleSort(c.key)}
                  >
                    {c.label}
                    {sortKey === c.key && <span className="ta-arrow">{sortDir === 'asc' ? '↑' : '↓'}</span>}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sorted.map((m) => {
                const initials = initialsOf(m.name);
                const points = Number.isFinite(m.points) ? m.points.toFixed(1) : '0.0';
                const isSel = m.id === selectedID;
                return (
                  <tr
                    key={m.id}
                    className={`ta-row${isSel ? ' is-selected' : ''}${m.active === false ? ' is-inactive' : ''}`}
                    onClick={() => selectMember(m.id)}
                    title="Open profile"
                  >
                    <td className="ta-td ta-td--member">
                      <span className="ta-avatar">{initials}</span>
                      <div>
                        <div className="ta-name">{m.name}</div>
                        {m.github && <div className="ta-meta mono">@{m.github}</div>}
                      </div>
                    </td>
                    <td className="ta-td ta-td--right mono">{m.jira}</td>
                    <td className="ta-td ta-td--right mono">{points}</td>
                    <td className="ta-td ta-td--right mono">{m.prs}</td>
                    <td className="ta-td ta-td--right mono">{m.reviews}</td>
                    <td className="ta-td ta-td--right mono">{formatHours(m.latency)}</td>
                    <td className="ta-td"><Spark values={m.spark} color="var(--accent, #b8443c)" /></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {loading && <div className="ta-loading">Loading…</div>}
        </div>
      )}

      {selectedRow && (
        <MemberProfile row={selectedRow} onClose={closeProfile} onSaved={onProfileSaved} />
      )}

      <footer className="ta-foot">
        <span className="mono">SCOPE · B2</span>
        <span>Member profile + manual GitHub linking · Slice Explorer (B3) is next.</span>
      </footer>
    </div>
  );
}
