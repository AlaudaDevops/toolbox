import React from 'react';
import { AlertCircle, PlayCircle, CheckCircle, Clock, ExternalLink, ArrowRightLeft, Edit, Tag } from 'lucide-react';
import { getJiraIssueUrl } from '../utils/jiraUtils';
import './EpicCard.css';

const EpicCard = ({ epic, isDragging, onMoveEpic, onUpdateEpic, currentMilestone }) => {
  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case '已完成':
      case '已取消':
      case 'done':
      case 'closed':
      case 'resolved':
        return <CheckCircle title={status} size={14} />;
      case '调研中':
      case '调研完成':
      case '设计完成':
      case '开发完成':
      case '测试完成':
      case '验收完成':
      case 'in progress':
      case 'in-progress':
        return <PlayCircle title={status} size={14} />;
      case '待处理':
      case 'to do':
      case 'todo':
      case 'open':
        return <Clock title={status} size={14} />;
      default:
        return <AlertCircle title={status} size={14} />;
    }
  };

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case '已完成':
      case '已取消':
      case 'done':
      case 'closed':
      case 'resolved':
        return 'status-done';
      case '调研中':
      case '调研完成':
      case '设计完成':
      case '开发完成':
      case '测试完成':
      case '验收完成':
      case 'in progress':
      case 'in-progress':
        return 'status-progress';
      case '待处理':
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

  const handleEditClick = (e) => {
    e.stopPropagation(); // Prevent card click and drag
    if (onUpdateEpic) {
      onUpdateEpic(epic);
    }
  };

  return (
    <div className={`epic-card-content ${isDragging ? 'dragging' : ''}`} >
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
        <div className="epic-row-2-left">
        <span className={`epic-key epic-status ${getStatusColor(epic.status)}`}>{epic.key}</span>
        {epic.status && (
            <div className={`epic-status ${getStatusColor(epic.status)}`}>
              {/* <span>{epic.status}</span> */}
            </div>
          )}
        <Tag size={13}/>
        {epic.versions && (
          <span className="epic-versions">{epic.versions.join(', ')}</span>
        ) || (
          <span className="epic-versions">-</span>
        )}
        </div>
        <div className="epic-row-2-right">
          {/* {epic.status && (
            <div className={`epic-status ${getStatusColor(epic.status)}`}>
              {getStatusIcon(epic.status)}
              <span>{epic.status}</span>
            </div>
          )} */}
          {onUpdateEpic && (
            <button
              onClick={handleEditClick}
              className="epic-edit-btn"
              title="Edit epic details"
            >
              <Edit size={12} />
            </button>
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
