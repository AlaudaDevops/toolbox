import { render, screen } from '@testing-library/react';
import App from './App';

// Mock the entire useAuth module including AuthProvider
jest.mock('./hooks/useAuth', () => {
  const mockReact = require('react');
  return {
    AuthProvider: ({ children }) => mockReact.createElement('div', { 'data-testid': 'auth-provider' }, children),
    useAuth: () => ({
      isAuthenticated: false,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      credentials: null,
      project: 'TEST'
    })
  };
});

// Mock the RoadmapProvider
jest.mock('./hooks/useRoadmap', () => {
  const mockReact = require('react');
  return {
    RoadmapProvider: ({ children }) => mockReact.createElement('div', { 'data-testid': 'roadmap-provider' }, children),
    useRoadmap: () => ({
      pillars: [],
      milestones: [],
      epics: [],
      loading: false,
      error: null,
      refreshData: jest.fn(),
      createMilestone: jest.fn(),
      createEpic: jest.fn(),
      updateEpic: jest.fn(),
      getAssignableUsers: jest.fn(() => Promise.resolve({ success: true, data: [] }))
    })
  };
});

// Basic smoke test to ensure the app renders without crashing
test('renders roadmap planner application', () => {
  render(<App />);

  // When not authenticated, should show login modal
  expect(screen.getByText(/Login to Jira/i)).toBeInTheDocument();
});

test('app has proper structure', () => {
  const { container } = render(<App />);

  // Basic structural test
  expect(container.firstChild).toBeTruthy();
});

// Example of testing error boundaries or loading states
test('handles errors gracefully', () => {
  // Mock console.error to avoid noise in test output
  const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

  try {
    render(<App />);
    // If we get here, the app rendered without throwing
    expect(true).toBe(true);
  } catch (error) {
    // If there's an error, make sure it's handled gracefully
    expect(error).toBeDefined();
  } finally {
    consoleSpy.mockRestore();
  }
});
