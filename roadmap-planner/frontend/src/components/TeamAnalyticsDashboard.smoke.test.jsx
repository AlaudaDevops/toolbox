/* Smoke-mount TeamAnalyticsDashboard with realistic input.
 * If it throws, the render error message goes to stderr. */
import React from 'react';
import { render, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

const MOCK_TEAM_RESPONSE = { members: [] };
vi.mock('../services/api', () => ({
  contributionsAPI: {
    team: vi.fn(() => Promise.resolve(MOCK_TEAM_RESPONSE)),
  },
}));
import DashboardView from './TeamAnalyticsDashboard';

const mkMember = (id, pillar, weekTotals) => ({
  id, name: id, pillar, pillars: [pillar], github: id, gitlab: id,
  active: true, week_totals: weekTotals,
});
const wk = (w, prsMerged, prsOpened, reviews, jiraDone, points) => ({
  week_start: `${w}T00:00:00Z`,
  prs_merged: prsMerged, prs_opened: prsOpened,
  reviews, jira_done: jiraDone, points,
});

describe('DashboardView smoke', () => {
  it('mounts with empty rows', () => {
    render(<DashboardView rows={[]} orderedPillarNames={['CI/CD', 'Essentials']} onMemberClick={() => {}} />);
  });
  it('mounts with one member, one bucket', () => {
    const rows = [mkMember('alice', 'CI/CD', [wk('2026-05-04', 3, 2, 1, 0, 0)])];
    render(<DashboardView rows={rows} orderedPillarNames={['CI/CD']} onMemberClick={() => {}} />);
  });
  it('mounts with member missing week_totals', () => {
    const rows = [mkMember('alice', 'CI/CD', undefined)];
    render(<DashboardView rows={rows} orderedPillarNames={['CI/CD']} onMemberClick={() => {}} />);
  });
  it('mounts and resolves API team payload that matches dev response', async () => {
    const allWeeks = Array.from({ length: 27 }, (_, i) => {
      const d = new Date('2025-11-03');
      d.setUTCDate(d.getUTCDate() + i * 7);
      return d.toISOString().slice(0, 10);
    });
    const pillars = ['CI/CD', 'Essentials', 'AI-Augmented SDLC', 'Tool Deployment'];
    // Mimic backend response shape (member_id, pillars, week_totals with full ISO timestamps)
    MOCK_TEAM_RESPONSE.members = Array.from({ length: 20 }, (_, mi) => ({
      member_id: `m${mi}`,
      pillars: [pillars[mi % pillars.length]],
      week_totals: allWeeks.map((w, wi) => ({
        week_start: `${w}T00:00:00Z`,
        prs_merged: ((mi + wi) % 5),
        prs_opened: ((mi + wi + 1) % 6),
        reviews:    ((mi * 3 + wi) % 4),
        jira_done:  ((mi + wi * 2) % 7),
        points:     ((mi + wi) % 4) + 0.5,
      })),
    }));
    // Initial rows can be empty — the dashboard will replace from the API.
    let result;
    await act(async () => {
      result = render(<DashboardView rows={[]} orderedPillarNames={pillars} onMemberClick={() => {}} />);
    });
    expect(result.container.querySelector('.ta-kpis')).not.toBeNull();
  });

  it('mounts when a member has a pillar NOT in orderedPillarNames (regression: dashboard crash 2026-05-10)', () => {
    // Reproduces the prod crash: pillarKeys built from an empty
    // orderedPillarNames + "Unassigned", but the team API returns members
    // with concrete pillars like "AI-Augmented SDLC". Pre-fix this threw
    // `Cannot read properties of undefined (reading '0')` inside the
    // pillarStack useMemo.
    const rows = [
      mkMember('alice', 'AI-Augmented SDLC', [wk('2026-05-04', 3, 2, 1, 0, 0)]),
      mkMember('bo',    'Tool Integration',  [wk('2026-05-04', 1, 0, 0, 0, 0)]),
    ];
    render(<DashboardView rows={rows} orderedPillarNames={[]} onMemberClick={() => {}} />);
  });

  it('mounts with realistic 20-member × 27-week dataset', () => {
    const allWeeks = Array.from({ length: 27 }, (_, i) => {
      const d = new Date('2025-11-03');
      d.setUTCDate(d.getUTCDate() + i * 7);
      return d.toISOString().slice(0, 10);
    });
    const pillars = ['CI/CD', 'Essentials', 'AI-Augmented SDLC', 'Tool Deployment'];
    const rows = Array.from({ length: 20 }, (_, mi) => mkMember(
      `m${mi}`,
      pillars[mi % pillars.length],
      allWeeks.map((w, wi) => wk(w,
        ((mi + wi) % 5),
        ((mi + wi + 1) % 6),
        ((mi * 3 + wi) % 4),
        ((mi + wi * 2) % 7),
        ((mi + wi) % 4) + 0.5,
      )),
    ));
    render(<DashboardView rows={rows} orderedPillarNames={pillars} onMemberClick={() => {}} />);
  });
});
