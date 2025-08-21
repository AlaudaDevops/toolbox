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

  // Custom styles matching your existing design
  const customStyles = {
    control: (provided, state) => ({
      ...provided,
      borderColor: error ? '#ef4444' : state.isFocused ? '#3b82f6' : '#d1d5db',
      boxShadow: error
        ? '0 0 0 3px rgba(239, 68, 68, 0.1)'
        : state.isFocused
          ? '0 0 0 3px rgba(59, 130, 246, 0.1)'
          : 'none',
      '&:hover': {
        borderColor: error ? '#ef4444' : '#9ca3af',
      },
      minHeight: '38px',
      fontSize: '14px',
      borderRadius: '6px',
      backgroundColor: disabled ? '#f9fafb' : 'white',
      cursor: disabled ? 'not-allowed' : 'pointer',
    }),
    placeholder: (provided) => ({
      ...provided,
      color: '#9ca3af',
    }),
    input: (provided) => ({
      ...provided,
      fontSize: '14px',
    }),
    singleValue: (provided) => ({
      ...provided,
      fontSize: '14px',
      color: '#374151',
    }),
    option: (provided, state) => ({
      ...provided,
      backgroundColor: state.isSelected
        ? '#3b82f6'
        : state.isFocused
          ? '#f3f4f6'
          : 'white',
      color: state.isSelected ? 'white' : '#374151',
      fontSize: '14px',
      padding: '8px 12px',
      cursor: 'pointer',
      '&:hover': {
        backgroundColor: state.isSelected ? '#3b82f6' : '#f3f4f6',
      },
    }),
    menu: (provided) => ({
      ...provided,
      borderRadius: '6px',
      boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
      border: '1px solid #d1d5db',
      marginTop: '4px',
      zIndex: 1000,
    }),
    menuList: (provided) => ({
      ...provided,
      maxHeight: '200px',
      padding: 0,
    }),
    menuPortal: (provided) => ({
      ...provided,
      zIndex: 1000,
    }),
    loadingMessage: (provided) => ({
      ...provided,
      fontSize: '14px',
      color: '#6b7280',
      fontStyle: 'italic',
    }),
    noOptionsMessage: (provided) => ({
      ...provided,
      fontSize: '14px',
      color: '#6b7280',
      fontStyle: 'italic',
      padding: '12px',
    }),
    clearIndicator: (provided) => ({
      ...provided,
      color: '#6b7280',
      cursor: 'pointer',
      '&:hover': {
        color: '#374151',
      },
    }),
    dropdownIndicator: (provided, state) => ({
      ...provided,
      color: '#6b7280',
      cursor: 'pointer',
      transform: state.selectProps.menuIsOpen ? 'rotate(180deg)' : 'rotate(0deg)',
      transition: 'transform 0.2s',
      '&:hover': {
        color: '#374151',
      },
    }),
    indicatorSeparator: (provided) => ({
      ...provided,
      backgroundColor: '#e5e7eb',
    }),
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
