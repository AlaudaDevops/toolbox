import React from 'react';
import Select from 'react-select';
import './FilterableSelect.css';

const FilterableSelect = ({
  id,
  value,
  onChange,
  options = [],
  placeholder = "Search and select...",
  className = "",
  error = false,
  disabled = false,
  getOptionLabel = (option) => option.label || option.name || option.toString(),
  getOptionValue = (option) => option.value || option.id || option,
  filterFunction = null, // Custom filter function
  emptyMessage = "No options found",
  isClearable = true,
  isSearchable = true,
  menuPortalTarget = null,
  menuPosition = "absolute",
}) => {
  // Transform options to react-select format

//   console.log(id, "Options: ", options);
//   console.log(id, "Value: ", value);
  const selectOptions = options.map(option => ({
    value: getOptionValue(option),
    label: getOptionLabel(option),
    data: option, // Keep original option data
  }));

  // Find selected option
  const selectedOption = selectOptions.find(option => option.value === value) || null;

  // Custom styles — Atlas theme via CSS variables
  const customStyles = {
    control: (provided, state) => ({
      ...provided,
      borderRadius: 0,
      borderColor: error ? 'var(--crimson)' : state.isFocused ? 'var(--fg)' : 'var(--border)',
      boxShadow: state.isFocused ? `inset 0 0 0 1px ${error ? 'var(--crimson)' : 'var(--fg)'}` : 'none',
      backgroundColor: disabled ? 'var(--bg-sunken)' : 'var(--bg-elevated)',
      '&:hover': { borderColor: error ? 'var(--crimson)' : 'var(--fg-faint)' },
      minHeight: '40px',
      fontSize: '14px',
      fontFamily: 'var(--font-ui)',
      cursor: disabled ? 'not-allowed' : 'pointer',
    }),
    valueContainer: (provided) => ({ ...provided, padding: '0 0.625rem' }),
    placeholder: (provided) => ({ ...provided, color: 'var(--fg-faint)', fontSize: '14px' }),
    input: (provided) => ({ ...provided, color: 'var(--fg)', fontSize: '14px' }),
    singleValue: (provided) => ({ ...provided, color: 'var(--fg)', fontSize: '14px' }),
    option: (provided, state) => ({
      ...provided,
      backgroundColor: state.isSelected ? 'var(--ink)' : state.isFocused ? 'var(--bg-sunken)' : 'var(--bg-elevated)',
      color: state.isSelected ? 'var(--paper)' : 'var(--fg)',
      fontSize: '13px',
      padding: '0.5rem 0.75rem',
      cursor: 'pointer',
      borderBottom: '1px solid var(--border)',
    }),
    menu: (provided) => ({
      ...provided,
      borderRadius: 0,
      border: '1px solid var(--fg-faint)',
      boxShadow: '-3px 3px 0 var(--ink), -3px 3px 0 1px var(--fg)',
      backgroundColor: 'var(--bg-elevated)',
      marginTop: 4,
      zIndex: 1000,
    }),
    menuList: (provided) => ({ ...provided, maxHeight: '220px', padding: 0 }),
    menuPortal: (provided) => ({ ...provided, zIndex: 1000 }),
    loadingMessage: (provided) => ({ ...provided, fontSize: '13px', color: 'var(--fg-muted)', fontStyle: 'italic' }),
    noOptionsMessage: (provided) => ({ ...provided, fontSize: '13px', color: 'var(--fg-muted)', fontStyle: 'italic', padding: '0.625rem 0.75rem' }),
    clearIndicator: (provided) => ({ ...provided, color: 'var(--fg-faint)', '&:hover': { color: 'var(--fg)' } }),
    dropdownIndicator: (provided, state) => ({
      ...provided,
      color: 'var(--fg-faint)',
      transform: state.selectProps.menuIsOpen ? 'rotate(180deg)' : 'rotate(0deg)',
      transition: 'transform 200ms cubic-bezier(0.32, 0.72, 0, 1)',
      '&:hover': { color: 'var(--fg)' },
    }),
    indicatorSeparator: (provided) => ({ ...provided, backgroundColor: 'var(--border)' }),
  };

  // Custom filter function if provided
  const filterOption = filterFunction
    ? (option, inputValue) => {
        return filterFunction(option.data, inputValue);
      }
    : undefined;

  // Handle selection change
  const handleChange = (selectedOption) => {
    console.log(id, "change?", selectedOption, "onChange?", onChange);
    onChange(selectedOption);
  };

  return (
    <div className={`filterable-select ${className}`}>
      <Select
        inputId={id}
        value={selectedOption}
        onChange={handleChange}
        options={selectOptions}
        isDisabled={disabled}
        isSearchable={isSearchable}
        isClearable={isClearable}
        placeholder={placeholder}
        noOptionsMessage={({ inputValue }) =>
          inputValue ? `No options found matching "${inputValue}"` : emptyMessage
        }
        styles={customStyles}
        menuPortalTarget={menuPortalTarget}
        menuPosition={menuPosition}
        filterOption={filterOption}
        className={`react-select-container ${error ? 'error' : ''}`}
        classNamePrefix="react-select"
      />
    </div>
  );
};

export default FilterableSelect;
