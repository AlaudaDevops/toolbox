import React from 'react';
import { AlertCircle, CheckCircle, Clock, ExternalLink, ArrowRightLeft } from 'lucide-react';
import { getJiraIssueUrl } from '../utils/jiraUtils';
import './EpicCard.css';

const EpicCard = ({ epic, isDragging, onMoveEpic, currentMilestone }) => {
  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'done':
      case 'closed':
      case 'resolved':
        return <CheckCircle size={14} />;
      case 'in progress':
      case 'in-progress':
        return <Clock size={14} />;
      case 'to do':
      case 'todo':
      case 'open':
        return <AlertCircle size={14} />;
      default:
        return <AlertCircle size={14} />;
    }
  };

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'done':
      case 'closed':
      case 'resolved':
        return 'status-done';
      case 'in progress':
      case 'in-progress':
        return 'status-progress';
      case 'to do':
      case 'todo':
      case 'open':
        return 'status-todo';
      default:
        return 'status-default';
    }
  };

  const getPriorityColor = (priority) => {
    switch (priority?.toLowerCase()) {
      case 'highest':
      case 'l0 - critical':
        return 'priority-critical';
      case 'l1 - high':
        return 'priority-high';
      case 'l2 - medium':
        return 'priority-medium';
      case 'l3 - low':
        return 'priority-low';
      case 'lowest':
        return 'priority-lowest';
      default:
        return 'priority-default';
    }
  };

  const jiraUrl = getJiraIssueUrl(epic.key);

  const handleCardClick = (e) => {
    // Only prevent default if we're actually opening a link
    if (jiraUrl) {
      e.stopPropagation(); // Prevent event bubbling
      window.open(jiraUrl, '_blank', 'noopener,noreferrer');
    }
  };

  const handleMoveClick = (e) => {
    e.stopPropagation(); // Prevent card click and drag
    if (onMoveEpic && currentMilestone) {
      onMoveEpic(epic, currentMilestone);
    }
  };

  return (
    <div className={`epic-card-content ${isDragging ? 'dragging' : ''}`} onClick={handleCardClick}>
      {/* First Row: Name and Priority */}
      <div className="epic-row-1">
        <div className="epic-name-container">
          <span className="epic-title">{epic.name}</span>
          {jiraUrl && <ExternalLink size={12} className="epic-link-icon" onClick={handleCardClick} />}
        </div>
        {epic.priority && (
          <div className={`epic-priority ${getPriorityColor(epic.priority)}`}>
            {epic.priority}
          </div>
        )}
      </div>

      {/* Second Row: ID, Status, and Actions */}
      <div className="epic-row-2">
        <span className="epic-key">{epic.key}</span>
        <div className="epic-row-2-right">
          {epic.status && (
            <div className={`epic-status ${getStatusColor(epic.status)}`}>
              {getStatusIcon(epic.status)}
              <span>{epic.status}</span>
            </div>
          )}
          {onMoveEpic && (
            <button
              onClick={handleMoveClick}
              className="epic-move-btn"
              title="Move epic to another milestone"
            >
              <ArrowRightLeft size={12} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default EpicCard;
