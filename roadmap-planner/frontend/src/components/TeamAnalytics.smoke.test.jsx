/* Mount TeamAnalytics end-to-end with realistic API responses to catch the
 * dashboard crash the user reported on dev. */
import React from 'react';
import { render, act, fireEvent, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

const allWeeks = Array.from({ length: 27 }, (_, i) => {
  const d = new Date('2025-11-03');
  d.setUTCDate(d.getUTCDate() + i * 7);
  return `${d.toISOString().slice(0, 10)}T00:00:00Z`;
});
const PILLARS = ['CI/CD', 'Essentials', 'AI-Augmented SDLC', 'Tool Deployment'];

const teamMembers = Array.from({ length: 20 }, (_, mi) => ({
  member_id: `m${mi}`,
  pillars: [PILLARS[mi % PILLARS.length]],
  week_totals: allWeeks.map((w, wi) => ({
    week_start: w,
    prs_merged: ((mi + wi) % 5),
    prs_opened: ((mi + wi + 1) % 6),
    reviews:    ((mi * 3 + wi) % 4),
    jira_done:  ((mi + wi * 2) % 7),
    points:     ((mi + wi) % 4) + 0.5,
  })),
}));
const memberDir = teamMembers.map((t) => ({
  id: t.member_id, display_name: t.member_id, email: `${t.member_id}@e`, github_login: t.member_id, gitlab_username: t.member_id,
  pillar_id: '', active: true,
}));

vi.mock('../services/api', () => ({
  contributionsAPI: {
    listMembers: vi.fn(() => Promise.resolve({ members: memberDir })),
    team:        vi.fn(() => Promise.resolve({ members: teamMembers })),
    status:      vi.fn(() => Promise.resolve({ jira: {}, github: {}, gitlab: {} })),
    network:     vi.fn(() => Promise.resolve({})),
    pillars:     vi.fn(() => Promise.resolve({ buckets: [], order: PILLARS })),
    member:      vi.fn(() => Promise.resolve({ info: {}, week_totals: teamMembers[0].week_totals, components_touched: [], sprint: null })),
    updateMember: vi.fn(),
  },
  roadmapAPI: { getBasicData: vi.fn(() => Promise.resolve({ pillars: PILLARS.map((n, i) => ({ name: n, sequence: i })) })) },
  handleAPIError: (e) => ({ message: String(e) }),
}));
vi.mock('react-hot-toast', () => ({
  default: { error: vi.fn(), success: vi.fn() },
  Toaster: () => null,
}));
import TeamAnalytics from './TeamAnalytics';

describe('TeamAnalytics dashboard tab', () => {
  it('renders the Dashboard tab as the default with realistic data', async () => {
    let utils;
    await act(async () => {
      utils = render(<TeamAnalytics />);
    });
    expect(utils.container.querySelector('.ta-kpis')).not.toBeNull();
  });

  it('survives clicking each KPI tile (changes active metric)', async () => {
    let utils;
    await act(async () => {
      utils = render(<TeamAnalytics />);
    });
    const kpis = utils.container.querySelectorAll('.ta-kpi--btn');
    expect(kpis.length).toBe(5);
    for (const k of kpis) {
      await act(async () => { fireEvent.click(k); });
    }
  });

  it('survives clicking each window seg + each pillar pill', async () => {
    let utils;
    await act(async () => {
      utils = render(<TeamAnalytics />);
    });
    const segs = utils.container.querySelectorAll('.ta-seg button');
    for (const s of segs) {
      await act(async () => { fireEvent.click(s); });
    }
    const pills = utils.container.querySelectorAll('.ta-pill');
    for (const p of pills) {
      await act(async () => { fireEvent.click(p); });
    }
  });

  it('survives clicking a ranking row to drill into a member', async () => {
    let utils;
    await act(async () => {
      utils = render(<TeamAnalytics />);
    });
    const rankBtns = utils.container.querySelectorAll('.ta-bar-row--btn');
    if (rankBtns.length > 0) {
      await act(async () => { fireEvent.click(rankBtns[0]); });
      // Should have switched to member view; member kpis should render.
      expect(utils.container.querySelector('.ta-prof-name')).not.toBeNull();
    }
  });
});
