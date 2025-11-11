import React, { useState, useEffect, useCallback } from 'react';
import Select from 'react-select';
import { useRoadmap } from '../hooks/useRoadmap';
import './AssigneeSelect.css';

const AssigneeSelect = ({
  issueKey,
  value,
  onChange,
  error,
  placeholder = "Search and select an assignee...",
  isRequired = true
}) => {
  const { getAssignableUsers } = useRoadmap();
  const [users, setUsers] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Debounce search to avoid too many API calls
  const debounceTimeout = React.useRef(null);

  const loadUsers = useCallback(async (query = '') => {
    if (!issueKey) return;

    setIsLoading(true);
    try {

      const selectedUser = {
        value: value ? value.name : "",
        label: value? `${value.display_name} (${value.email_address})`: "",
        user: value || null
      }
      const result = await getAssignableUsers(issueKey, query);
      if (result.success) {
        const userOptions = (result.data || []).map(user => ({
          value: user.name,
          label: `${user.display_name} (${user.email_address})`,
          user: user
        }));
        if (selectedUser.value && !userOptions.find(user => user.value === selectedUser.value)) {
          userOptions.unshift(selectedUser);
        }
        setUsers(userOptions);
      } else {
        if (selectedUser.value) {
          setUsers([selectedUser]);
        } else {
          setUsers([]);
        }
      }
    } catch (error) {
      console.error('Failed to load users:', error);
      setUsers([]);
    } finally {
      setIsLoading(false);
    }
  }, [issueKey, getAssignableUsers, value]);

  // Load initial users on mount
  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  // Handle search input with debouncing
  const handleInputChange = (inputValue, { action }) => {
    console.debug("setSearchQuery", searchQuery);
    if (action === 'input-change') {
      setSearchQuery(inputValue);

      // Clear previous timeout
      if (debounceTimeout.current) {
        clearTimeout(debounceTimeout.current);
      }

      // Set new timeout for API call
      debounceTimeout.current = setTimeout(() => {
        loadUsers(inputValue);
      }, 300); // 300ms debounce
    }
  };

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (debounceTimeout.current) {
        clearTimeout(debounceTimeout.current);
      }
    };
  }, []);

  // Find selected option
  const selectedOption = value ? users.find(user => user.value === value.name) : null;

  // Custom styles for react-select
  const customStyles = {
    control: (provided, state) => ({
      ...provided,
      borderColor: error ? '#ef4444' : state.isFocused ? '#667eea' : '#d1d5db',
      boxShadow: error
        ? '0 0 0 3px rgba(239, 68, 68, 0.1)'
        : state.isFocused
          ? '0 0 0 3px rgba(102, 126, 234, 0.1)'
          : 'none',
      '&:hover': {
        borderColor: error ? '#ef4444' : '#667eea',
      },
      minHeight: '42px',
      fontSize: '0.875rem',
    }),
    placeholder: (provided) => ({
      ...provided,
      color: '#9ca3af',
    }),
    option: (provided, state) => ({
      ...provided,
      backgroundColor: state.isSelected
        ? '#667eea'
        : state.isFocused
          ? '#f3f4f6'
          : 'white',
      color: state.isSelected ? 'white' : '#374151',
      fontSize: '0.875rem',
      '&:hover': {
        backgroundColor: state.isSelected ? '#667eea' : '#f3f4f6',
      },
    }),
    menu: (provided) => ({
      ...provided,
      zIndex: 9999,
    }),
    menuPortal: (provided) => ({
      ...provided,
      zIndex: 9999,
    }),
    loadingMessage: (provided) => ({
      ...provided,
      fontSize: '0.875rem',
      color: '#6b7280',
    }),
    noOptionsMessage: (provided) => ({
      ...provided,
      fontSize: '0.875rem',
      color: '#6b7280',
    }),
  };

  // Handle selection change
  const handleChange = (option) => {
    const newValue = option ? option.user : null;
    onChange(newValue);
  };

  return (
    <div className="assignee-select-container">
      <Select
        value={selectedOption}
        onChange={handleChange}
        onInputChange={handleInputChange}
        options={users}
        isLoading={isLoading}
        isSearchable={true}
        isClearable={!isRequired}
        placeholder={placeholder}
        noOptionsMessage={({ inputValue }) =>
          inputValue ? `No users found matching "${inputValue}"` : 'No users available'
        }
        loadingMessage={() => 'Searching users...'}
        styles={customStyles}
        menuPortalTarget={document.body}
        menuPosition="fixed"
        className={`assignee-select ${error ? 'error' : ''}`}
        classNamePrefix="assignee-select"
      />
    </div>
  );
};

export default AssigneeSelect;
