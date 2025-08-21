import { getStoredAuth } from '../services/api';

/**
 * Generates a Jira issue URL from the issue key
 * @param {string} issueKey - The Jira issue key (e.g., "DEVOPS-123")
 * @returns {string|null} - The full URL to the Jira issue, or null if no auth data
 */
export const getJiraIssueUrl = (issueKey) => {
  const auth = getStoredAuth();
  if (!auth || !auth.baseURL || !issueKey) {
    return null;
  }

  // Remove trailing slash from baseURL if present
  const baseURL = auth.baseURL.replace(/\/$/, '');
  
  // Construct the Jira issue URL
  return `${baseURL}/browse/${issueKey}`;
};

/**
 * Opens a Jira issue in a new tab
 * @param {string} issueKey - The Jira issue key
 */
export const openJiraIssue = (issueKey) => {
  const url = getJiraIssueUrl(issueKey);
  if (url) {
    window.open(url, '_blank', 'noopener,noreferrer');
  }
};
