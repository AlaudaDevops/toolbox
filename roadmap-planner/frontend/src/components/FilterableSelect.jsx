import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, X } from 'lucide-react';

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
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const dropdownRef = useRef(null);
  const inputRef = useRef(null);
  const listRef = useRef(null);

  // Filter options based on search term
  const filteredOptions = searchTerm
    ? options.filter(option => {
        if (filterFunction) {
          return filterFunction(option, searchTerm);
        }
        const label = getOptionLabel(option).toLowerCase();
        const search = searchTerm.toLowerCase();
        return label.includes(search);
      })
    : options;

  // Find selected option
  const selectedOption = options.find(option => getOptionValue(option) === value);
  const selectedLabel = selectedOption ? getOptionLabel(selectedOption) : '';

  // Handle outside clicks
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
        setSearchTerm('');
        setHighlightedIndex(-1);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (event) => {
      if (!isOpen) return;

      switch (event.key) {
        case 'ArrowDown':
          event.preventDefault();
          setHighlightedIndex(prev =>
            prev < filteredOptions.length - 1 ? prev + 1 : 0
          );
          break;
        case 'ArrowUp':
          event.preventDefault();
          setHighlightedIndex(prev =>
            prev > 0 ? prev - 1 : filteredOptions.length - 1
          );
          break;
        case 'Enter':
          event.preventDefault();
          if (highlightedIndex >= 0 && filteredOptions[highlightedIndex]) {
            handleSelectOption(filteredOptions[highlightedIndex]);
          }
          break;
        case 'Escape':
          setIsOpen(false);
          setSearchTerm('');
          setHighlightedIndex(-1);
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, highlightedIndex, filteredOptions]);

  // Scroll highlighted option into view
  useEffect(() => {
    if (highlightedIndex >= 0 && listRef.current) {
      const highlightedElement = listRef.current.children[highlightedIndex];
      if (highlightedElement) {
        highlightedElement.scrollIntoView({
          block: 'nearest',
          behavior: 'smooth',
        });
      }
    }
  }, [highlightedIndex]);

  const handleSelectOption = (option) => {
    const optionValue = getOptionValue(option);
    onChange(optionValue);
    setIsOpen(false);
    setSearchTerm('');
    setHighlightedIndex(-1);
  };

  const handleInputClick = () => {
    if (!disabled) {
      setIsOpen(!isOpen);
      if (!isOpen) {
        setTimeout(() => inputRef.current?.focus(), 0);
      }
    }
  };

  const handleClear = (e) => {
    e.stopPropagation();
    onChange('');
    setSearchTerm('');
    setIsOpen(false);
  };

  const handleInputChange = (e) => {
    setSearchTerm(e.target.value);
    setHighlightedIndex(-1);
    if (!isOpen) {
      setIsOpen(true);
    }
  };

  return (
    <div className={`filterable-select ${className}`} ref={dropdownRef}>
      <div
        className={`filterable-select-control ${error ? 'error' : ''} ${disabled ? 'disabled' : ''} ${isOpen ? 'open' : ''}`}
        onClick={handleInputClick}
      >
        <input
          ref={inputRef}
          id={id}
          type="text"
          value={isOpen ? searchTerm : selectedLabel}
          onChange={handleInputChange}
          placeholder={selectedLabel || placeholder}
          className="filterable-select-input"
          disabled={disabled}
          autoComplete="off"
          readOnly={!isOpen}
        />

        <div className="filterable-select-indicators">
          {value && !disabled && (
            <button
              type="button"
              className="filterable-select-clear"
              onClick={handleClear}
              tabIndex={-1}
            >
              <X size={14} />
            </button>
          )}
          <div className="filterable-select-separator" />
          <div className={`filterable-select-arrow ${isOpen ? 'open' : ''}`}>
            <ChevronDown size={16} />
          </div>
        </div>
      </div>

      {isOpen && (
        <div className="filterable-select-menu">
          <div className="filterable-select-list" ref={listRef}>
            {filteredOptions.length > 0 ? (
              filteredOptions.map((option, index) => {
                const optionValue = getOptionValue(option);
                const optionLabel = getOptionLabel(option);
                return (
                  <div
                    key={optionValue}
                    className={`filterable-select-option ${
                      index === highlightedIndex ? 'highlighted' : ''
                    } ${optionValue === value ? 'selected' : ''}`}
                    onClick={() => handleSelectOption(option)}
                    onMouseEnter={() => setHighlightedIndex(index)}
                  >
                    {optionLabel}
                  </div>
                );
              })
            ) : (
              <div className="filterable-select-empty">
                {emptyMessage}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default FilterableSelect;
