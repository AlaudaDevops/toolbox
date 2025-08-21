import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { AssigneeSelect } from './AssigneeSelect';

// Mock data for testing
const mockUsers = [
  { accountId: '1', displayName: 'John Doe', emailAddress: 'john@example.com' },
  { accountId: '2', displayName: 'Jane Smith', emailAddress: 'jane@example.com' },
  { accountId: '3', displayName: 'Bob Wilson', emailAddress: 'bob@example.com' }
];

describe('AssigneeSelect', () => {
  const defaultProps = {
    users: mockUsers,
    value: null,
    onChange: jest.fn(),
    isLoading: false
  };

  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();
  });

  test('renders without crashing', () => {
    render(<AssigneeSelect {...defaultProps} />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  test('displays loading state when isLoading is true', () => {
    render(<AssigneeSelect {...defaultProps} isLoading={true} />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  test('displays users in the dropdown', () => {
    render(<AssigneeSelect {...defaultProps} />);

    // Open the dropdown
    const select = screen.getByRole('combobox');
    fireEvent.mouseDown(select);

    // Check if users are displayed
    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('Jane Smith')).toBeInTheDocument();
    expect(screen.getByText('Bob Wilson')).toBeInTheDocument();
  });

  test('calls onChange when a user is selected', () => {
    const mockOnChange = jest.fn();
    render(<AssigneeSelect {...defaultProps} onChange={mockOnChange} />);

    // Open the dropdown
    const select = screen.getByRole('combobox');
    fireEvent.mouseDown(select);

    // Select a user
    const johnOption = screen.getByText('John Doe');
    fireEvent.click(johnOption);

    // Verify onChange was called
    expect(mockOnChange).toHaveBeenCalledWith(mockUsers[0]);
  });

  test('displays selected user correctly', () => {
    render(<AssigneeSelect {...defaultProps} value={mockUsers[0]} />);

    // Check if the selected user is displayed
    expect(screen.getByDisplayValue('John Doe')).toBeInTheDocument();
  });

  test('handles empty users list', () => {
    render(<AssigneeSelect {...defaultProps} users={[]} />);

    const select = screen.getByRole('combobox');
    fireEvent.mouseDown(select);

    // Should show "No options" or similar message
    expect(screen.getByText(/no.*options?/i)).toBeInTheDocument();
  });

  test('is accessible', () => {
    render(<AssigneeSelect {...defaultProps} />);

    const select = screen.getByRole('combobox');
    expect(select).toHaveAttribute('aria-expanded');
    expect(select).toHaveAttribute('aria-haspopup');
  });
});
