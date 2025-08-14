/**
 * Utility functions for sorting roadmap data
 */

/**
 * Parse a quarter string (e.g., "2025Q1") to a comparable number
 * @param {string} quarter - Quarter string in format YYYYQX
 * @returns {number} - Comparable number for sorting
 */
export const parseQuarter = (quarter) => {
  if (!quarter || quarter.length < 6) {
    return 0;
  }
  
  const yearStr = quarter.slice(0, 4);
  const quarterStr = quarter.slice(5);
  
  const year = parseInt(yearStr, 10);
  const q = parseInt(quarterStr, 10);
  
  if (isNaN(year) || isNaN(q)) {
    return 0;
  }
  
  // Convert to comparable integer: year * 10 + quarter
  // e.g., 2025Q1 -> 20251, 2025Q2 -> 20252
  return year * 10 + q;
};

/**
 * Sort quarters chronologically (older first)
 * @param {string[]} quarters - Array of quarter strings
 * @returns {string[]} - Sorted array of quarters
 */
export const sortQuarters = (quarters) => {
  if (!quarters || !Array.isArray(quarters)) {
    return [];
  }
  
  return [...quarters].sort((a, b) => parseQuarter(a) - parseQuarter(b));
};

/**
 * Sort pillars by sequence, then by name
 * @param {Object[]} pillars - Array of pillar objects
 * @returns {Object[]} - Sorted array of pillars
 */
export const sortPillars = (pillars) => {
  if (!pillars || !Array.isArray(pillars)) {
    return [];
  }
  
  return [...pillars].sort((a, b) => {
    // Sort by sequence first
    if (a.sequence !== b.sequence) {
      return (a.sequence || 0) - (b.sequence || 0);
    }
    // Then by name
    return (a.name || '').localeCompare(b.name || '');
  });
};

/**
 * Sort milestones by sequence, then by name
 * @param {Object[]} milestones - Array of milestone objects
 * @returns {Object[]} - Sorted array of milestones
 */
export const sortMilestones = (milestones) => {
  if (!milestones || !Array.isArray(milestones)) {
    return [];
  }
  
  return [...milestones].sort((a, b) => {
    // Sort by sequence first
    if (a.sequence !== b.sequence) {
      return (a.sequence || 0) - (b.sequence || 0);
    }
    // Then by name
    return (a.name || '').localeCompare(b.name || '');
  });
};

/**
 * Sort epics by fix version (blanks first), then by name
 * @param {Object[]} epics - Array of epic objects
 * @returns {Object[]} - Sorted array of epics
 */
export const sortEpics = (epics) => {
  if (!epics || !Array.isArray(epics)) {
    return [];
  }
  
  return [...epics].sort((a, b) => {
    const versionA = a.version || '';
    const versionB = b.version || '';
    
    // Blanks (empty versions) should come first
    if (versionA === '' && versionB !== '') {
      return -1;
    }
    if (versionA !== '' && versionB === '') {
      return 1;
    }
    
    // If both have versions or both are blank, sort by version then name
    if (versionA !== versionB) {
      return versionA.localeCompare(versionB);
    }
    return (a.name || '').localeCompare(b.name || '');
  });
};

/**
 * Sort all roadmap data recursively
 * @param {Object} roadmapData - Roadmap data object
 * @returns {Object} - Sorted roadmap data
 */
export const sortRoadmapData = (roadmapData) => {
  if (!roadmapData) {
    return roadmapData;
  }
  
  const sortedData = { ...roadmapData };
  
  // Sort quarters
  if (sortedData.quarters) {
    sortedData.quarters = sortQuarters(sortedData.quarters);
  }
  
  // Sort pillars and their nested data
  if (sortedData.pillars) {
    sortedData.pillars = sortPillars(sortedData.pillars).map(pillar => ({
      ...pillar,
      milestones: sortMilestones(pillar.milestones || []).map(milestone => ({
        ...milestone,
        epics: sortEpics(milestone.epics || [])
      }))
    }));
  }
  
  return sortedData;
};

/**
 * Validate and sort selected quarters
 * @param {string[]} selectedQuarters - Currently selected quarters
 * @param {string[]} availableQuarters - All available quarters
 * @returns {string[]} - Validated and sorted quarters
 */
export const validateAndSortSelectedQuarters = (selectedQuarters, availableQuarters) => {
  if (!selectedQuarters || !Array.isArray(selectedQuarters)) {
    return [];
  }
  
  // Filter out invalid quarters and sort
  const validQuarters = selectedQuarters.filter(quarter => 
    availableQuarters && availableQuarters.includes(quarter)
  );
  
  return sortQuarters(validQuarters);
};
