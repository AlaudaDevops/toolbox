/**
 * Utility functions for managing quarter selection persistence
 */

import { sortQuarters } from './sortingUtils';

const QUARTER_STORAGE_KEY = 'roadmap-selected-quarters';

/**
 * Save selected quarters to localStorage
 * @param {string[]} quarters - Array of selected quarter strings
 */
export const saveSelectedQuarters = (quarters) => {
  try {
    const data = {
      quarters: quarters,
      timestamp: Date.now(),
      version: '1.0' // For future compatibility
    };
    localStorage.setItem(QUARTER_STORAGE_KEY, JSON.stringify(data));
  } catch (error) {
    console.warn('Failed to save selected quarters to localStorage:', error);
  }
};

/**
 * Load selected quarters from localStorage
 * @param {string[]} availableQuarters - Array of all available quarters
 * @returns {string[]|null} - Array of selected quarters or null if not found/invalid
 */
export const loadSelectedQuarters = (availableQuarters) => {
  try {
    const stored = localStorage.getItem(QUARTER_STORAGE_KEY);
    if (!stored) return null;

    const data = JSON.parse(stored);

    // Validate data structure
    if (!data.quarters || !Array.isArray(data.quarters)) {
      return null;
    }

    // Filter out quarters that are no longer available
    const validQuarters = data.quarters.filter(quarter =>
      availableQuarters.includes(quarter)
    );

    // Return valid quarters if we have any, otherwise null
    return validQuarters.length > 0 ? validQuarters : null;
  } catch (error) {
    console.warn('Failed to load selected quarters from localStorage:', error);
    return null;
  }
};

/**
 * Clear stored quarter selection
 */
export const clearSelectedQuarters = () => {
  try {
    localStorage.removeItem(QUARTER_STORAGE_KEY);
  } catch (error) {
    console.warn('Failed to clear selected quarters from localStorage:', error);
  }
};

/**
 * Get default quarters (first 3 available quarters, sorted)
 * @param {string[]} availableQuarters - Array of all available quarters
 * @returns {string[]} - Array of default quarters (max 3, sorted)
 */
export const getDefaultQuarters = (availableQuarters) => {
  if (!availableQuarters || availableQuarters.length === 0) {
    return [];
  }
  const sortedQuarters = sortQuarters(availableQuarters);
  return sortedQuarters.slice(0, 3);
};

/**
 * Validate and normalize quarter selection
 * @param {string[]} selectedQuarters - Currently selected quarters
 * @param {string[]} availableQuarters - All available quarters
 * @returns {string[]} - Validated and normalized quarter selection (sorted)
 */
export const validateQuarterSelection = (selectedQuarters, availableQuarters) => {
  if (!selectedQuarters || !Array.isArray(selectedQuarters)) {
    return getDefaultQuarters(availableQuarters);
  }

  // Filter out invalid quarters
  const validQuarters = selectedQuarters.filter(quarter =>
    availableQuarters.includes(quarter)
  );

  // If no valid quarters, return defaults
  if (validQuarters.length === 0) {
    return getDefaultQuarters(availableQuarters);
  }

  // Sort and limit to maximum 3 quarters
  const sortedQuarters = sortQuarters(validQuarters);
  return sortedQuarters.slice(0, 3);
};
