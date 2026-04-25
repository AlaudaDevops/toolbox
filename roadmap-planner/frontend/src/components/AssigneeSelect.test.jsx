import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import AssigneeSelect from './AssigneeSelect';

vi.mock('../hooks/useRoadmap', () => ({
  useRoadmap: () => ({
    getAssignableUsers: vi.fn(() => Promise.resolve({ success: true, data: [] })),
  }),
}));

describe('AssigneeSelect', () => {
  const defaultProps = {
    issueKey: 'TEST-123',
    value: null,
    onChange: vi.fn(),
    placeholder: 'Select assignee',
    isRequired: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders with placeholder', async () => {
    render(<AssigneeSelect {...defaultProps} />);
    await waitFor(() => {
      expect(screen.getByText(/select assignee/i)).toBeInTheDocument();
    });
  });

  it('renders with custom placeholder', async () => {
    render(<AssigneeSelect {...defaultProps} placeholder="Choose user" />);
    await waitFor(() => {
      expect(screen.getByText(/choose user/i)).toBeInTheDocument();
    });
  });
});
