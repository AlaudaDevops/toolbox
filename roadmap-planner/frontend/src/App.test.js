import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import App from './App';

vi.mock('./hooks/useAuth', () => {
  const React = require('react');
  return {
    AuthProvider: ({ children }) => React.createElement('div', { 'data-testid': 'auth-provider' }, children),
    useAuth: () => ({
      isAuthenticated: false,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      credentials: null,
      project: 'TEST',
    }),
  };
});

vi.mock('./hooks/useRoadmap', () => {
  const React = require('react');
  return {
    RoadmapProvider: ({ children }) => React.createElement('div', { 'data-testid': 'roadmap-provider' }, children),
    useRoadmap: () => ({
      pillars: [],
      milestones: [],
      epics: [],
      loading: false,
      error: null,
      refreshData: vi.fn(),
      createMilestone: vi.fn(),
      createEpic: vi.fn(),
      updateEpic: vi.fn(),
      getAssignableUsers: vi.fn(() => Promise.resolve({ success: true, data: [] })),
    }),
  };
});

describe('App', () => {
  it('renders the login page when unauthenticated', () => {
    render(<App />);
    expect(screen.getByText(/sign in · jira/i)).toBeInTheDocument();
  });

  it('mounts without crashing', () => {
    const { container } = render(<App />);
    expect(container.firstChild).toBeTruthy();
  });
});
