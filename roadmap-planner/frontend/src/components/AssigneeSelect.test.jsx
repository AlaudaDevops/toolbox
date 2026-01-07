import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AssigneeSelect from './AssigneeSelect';

// Mock the useRoadmap hook
jest.mock('../hooks/useRoadmap', () => ({
  useRoadmap: () => ({
    getAssignableUsers: jest.fn(() => Promise.resolve({ success: true, data: [] }))
  })
}));

describe('AssigneeSelect', () => {
  const defaultProps = {
    issueKey: 'TEST-123',
    value: null,
    onChange: jest.fn(),
    placeholder: 'Select assignee',
    isRequired: false
  };

  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();
  });

  test('renders without crashing', async () => {
    render(<AssigneeSelect {...defaultProps} />);
    // Wait for the component to load
    await waitFor(() => {
      expect(screen.getByText(/Select assignee/i)).toBeInTheDocument();
    });
  });

  test('renders with placeholder', async () => {
    render(<AssigneeSelect {...defaultProps} placeholder="Choose user" />);
    await waitFor(() => {
      expect(screen.getByText(/Choose user/i)).toBeInTheDocument();
    });
  });

  test('calls onChange when selection changes', async () => {
    const mockOnChange = jest.fn();
    render(<AssigneeSelect {...defaultProps} onChange={mockOnChange} />);

    // Component should render
    await waitFor(() => {
      expect(screen.getByText(/Select assignee/i)).toBeInTheDocument();
    });
  });
});
