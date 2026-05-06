/*
 * Team Analytics — initial production view, B1 scope.
 *
 * What this renders:
 *   - the team-overview table fed by /api/contributions/team
 *   - a per-source last-sync footer (Jira / GitHub) from /api/contributions/status
 *
 * What is intentionally not here yet (see PROPOSAL.md §5):
 *   - full Member Profile drill-in (B2)
 *   - Slice Explorer (B3)
 *   - sparkline rendering (replaced with weekly bar microchart for now)
 *
 * The visual / interaction language is shared with the rest of the app.
 * The companion docs/team-analytics/prototype.html is the higher-fidelity
 * mock for B2/B3 — keep it in sync as those slices land.
 */
import React, { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { contributionsAPI, handleAPIError } from '../services/api';
import './TeamAnalytics.css';

const formatHours = (h) => (h == null ? '—' : `${h.toFixed(1)}h`);
const formatDate = (s) => {
  if (!s) return '—';
  try { return new Date(s).toISOString().slice(0, 16).replace('T', ' '); }
  catch { return s; }
};

// Tiny inline sparkline. We deliberately don't pull Recharts for this —
// 30 lines of SVG renders in 1ms and looks identical at this size.
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

export default function TeamAnalytics() {
  const [members, setMembers] = useState([]);
  const [team, setTeam] = useState([]);
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [sortKey, setSortKey] = useState('name');
  const [sortDir, setSortDir] = useState('asc');

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
  }, []);

  // Join members directory with rollups so the table has names and
  // pillars even when some members have zero activity.
  const rows = useMemo(() => {
    const byID = new Map((team || []).map((t) => [t.member_id, t]));
    const merged = (members || []).map((m) => {
      const t = byID.get(m.id) || {};
      const sparkValues = (t.week_totals || []).map((w) => w.prs_merged || 0);
      return {
        id: m.id,
        name: m.display_name || m.id,
        github: m.github_login,
        pillar: m.pillar_id,
        jira: t.jira_issues_done || 0,
        points: t.jira_points_done || 0,
        prs: t.prs_merged || 0,
        reviews: t.prs_reviewed || 0,
        latency: t.review_latency_p50_hours,
        spark: sparkValues,
      };
    });
    // include orphan rollups (member rows with no directory entry)
    (team || []).forEach((t) => {
      if (!members.find((m) => m.id === t.member_id)) {
        merged.push({
          id: t.member_id, name: t.member_id, github: '', pillar: '',
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
    if (k === sortKey) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(k);
      setSortDir(k === 'name' || k === 'pillar' ? 'asc' : 'desc');
    }
  };

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
              {sorted.map((m) => (
                <tr key={m.id}>
                  <td className="ta-td ta-td--member">
                    <span className="ta-avatar">
                      {m.name.split(' ').filter(Boolean).map((s) => s[0]).slice(0, 2).join('').toUpperCase()}
                    </span>
                    <div>
                      <div className="ta-name">{m.name}</div>
                      {m.github && <div className="ta-meta mono">@{m.github}</div>}
                    </div>
                  </td>
                  <td className="ta-td ta-td--right mono">{m.jira}</td>
                  <td className="ta-td ta-td--right mono">{m.points.toFixed(1)}</td>
                  <td className="ta-td ta-td--right mono">{m.prs}</td>
                  <td className="ta-td ta-td--right mono">{m.reviews}</td>
                  <td className="ta-td ta-td--right mono">{formatHours(m.latency)}</td>
                  <td className="ta-td"><Spark values={m.spark} color="var(--accent, #b8443c)" /></td>
                </tr>
              ))}
            </tbody>
          </table>
          {loading && <div className="ta-loading">Loading…</div>}
        </div>
      )}

      <footer className="ta-foot">
        <span className="mono">SCOPE · B1</span>
        <span>Member Profile and Slice Explorer follow in B2 / B3 — see PROPOSAL.md.</span>
      </footer>
    </div>
  );
}
