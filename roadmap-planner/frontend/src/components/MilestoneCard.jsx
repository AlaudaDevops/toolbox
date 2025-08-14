import React from 'react';
import { ExternalLink, Edit } from 'lucide-react';
import { getJiraIssueUrl } from '../utils/jiraUtils';
import './MilestoneCard.css';

const MilestoneCard = ({ milestone, onUpdateMilestone }) => {

  const jiraUrl = getJiraIssueUrl(milestone.key);

  const handleTitleClick = (e) => {
    e.stopPropagation();
    if (jiraUrl) {
      window.open(jiraUrl, '_blank', 'noopener,noreferrer');
    }
  };

  const handleEditClick = (e) => {
    e.stopPropagation();
    if (onUpdateMilestone) {
      onUpdateMilestone(milestone);
    }
  };

  return (
    <div className="milestone-card">
      <div className="milestone-header">
        <div className="milestone-title-container" onClick={handleTitleClick}>
          <h4 className="milestone-title">{milestone.name}</h4>
          {jiraUrl && <ExternalLink size={12} className="milestone-link-icon" />}
        </div>
        <div className="milestone-actions">
          <button
            onClick={handleEditClick}
            className="btn btn-sm btn-secondary"
            title="Edit milestone"
          >
            <Edit size={12} />
          </button>
        </div>
      </div>
    </div>
  );
};

export default MilestoneCard;
