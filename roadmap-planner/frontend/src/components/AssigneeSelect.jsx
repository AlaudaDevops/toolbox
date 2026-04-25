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

  // Custom styles for react-select — Atlas theme via CSS variables
  const customStyles = {
    control: (provided, state) => ({
      ...provided,
      borderRadius: 0,
      borderColor: error ? 'var(--crimson)' : state.isFocused ? 'var(--fg)' : 'var(--border)',
      boxShadow: state.isFocused ? `inset 0 0 0 1px ${error ? 'var(--crimson)' : 'var(--fg)'}` : 'none',
      backgroundColor: 'var(--bg-elevated)',
      '&:hover': { borderColor: error ? 'var(--crimson)' : 'var(--fg-faint)' },
      minHeight: '40px',
      fontSize: '14px',
      fontFamily: 'var(--font-ui)',
      cursor: 'pointer',
    }),
    valueContainer: (provided) => ({ ...provided, padding: '0 0.625rem' }),
    placeholder: (provided) => ({ ...provided, color: 'var(--fg-faint)' }),
    input: (provided) => ({ ...provided, color: 'var(--fg)', fontSize: '14px' }),
    singleValue: (provided) => ({ ...provided, color: 'var(--fg)', fontSize: '14px' }),
    option: (provided, state) => ({
      ...provided,
      backgroundColor: state.isSelected ? 'var(--ink)' : state.isFocused ? 'var(--bg-sunken)' : 'var(--bg-elevated)',
      color: state.isSelected ? 'var(--paper)' : 'var(--fg)',
      fontSize: '13px',
      cursor: 'pointer',
      padding: '0.5rem 0.75rem',
      borderBottom: '1px solid var(--border)',
    }),
    menu: (provided) => ({
      ...provided,
      borderRadius: 0,
      border: '1px solid var(--fg-faint)',
      boxShadow: '-3px 3px 0 var(--ink), -3px 3px 0 1px var(--fg)',
      backgroundColor: 'var(--bg-elevated)',
      zIndex: 9999,
      marginTop: 4,
    }),
    menuPortal: (provided) => ({ ...provided, zIndex: 9999 }),
    loadingMessage: (provided) => ({ ...provided, fontSize: '13px', color: 'var(--fg-muted)' }),
    noOptionsMessage: (provided) => ({ ...provided, fontSize: '13px', color: 'var(--fg-muted)', fontStyle: 'italic' }),
    indicatorSeparator: (provided) => ({ ...provided, backgroundColor: 'var(--border)' }),
    dropdownIndicator: (provided, state) => ({
      ...provided,
      color: 'var(--fg-faint)',
      transform: state.selectProps.menuIsOpen ? 'rotate(180deg)' : 'rotate(0deg)',
      transition: 'transform 200ms cubic-bezier(0.32, 0.72, 0, 1)',
      '&:hover': { color: 'var(--fg)' },
    }),
    clearIndicator: (provided) => ({ ...provided, color: 'var(--fg-faint)', '&:hover': { color: 'var(--fg)' } }),
    loadingIndicator: (provided) => ({ ...provided, color: 'var(--accent)' }),
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
