import React from 'react';
import { ExternalLink, Pencil } from 'lucide-react';
import { getJiraIssueUrl } from '../utils/jiraUtils';
import './MilestoneCard.css';

const MilestoneCard = ({ milestone, onUpdateMilestone }) => {
  const jiraUrl = getJiraIssueUrl(milestone.key);
  const epicCount = milestone.epics?.length ?? 0;

  const handleTitleClick = (e) => {
    e.stopPropagation();
    if (jiraUrl) window.open(jiraUrl, '_blank', 'noopener,noreferrer');
  };

  const handleEditClick = (e) => {
    e.stopPropagation();
    if (onUpdateMilestone) onUpdateMilestone(milestone);
  };

  return (
    <div className="milestone-card">
      <div className="milestone-header">
        <button
          type="button"
          className="milestone-title-btn"
          onClick={handleTitleClick}
          title={jiraUrl ? `Open ${milestone.key} in Jira` : milestone.key}
          disabled={!jiraUrl}
        >
          <span className="serif milestone-title">{milestone.name}</span>
          {jiraUrl && <ExternalLink size={11} strokeWidth={1.75} className="milestone-link-icon" />}
        </button>
        <div className="milestone-meta">
          <span className="milestone-count mono tnum" title={`${epicCount} epics`}>
            {String(epicCount).padStart(2, '0')}
          </span>
          <button
            type="button"
            onClick={handleEditClick}
            className="milestone-edit-btn"
            title="Edit milestone"
            aria-label={`Edit ${milestone.name}`}
          >
            <Pencil size={11} strokeWidth={1.75} />
          </button>
        </div>
      </div>
      <span className="milestone-key mono">{milestone.key}</span>
    </div>
  );
};

export default MilestoneCard;
