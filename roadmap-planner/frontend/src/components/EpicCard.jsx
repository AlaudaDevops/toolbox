import React from 'react';
import { ExternalLink, ArrowRightLeft, Pencil, Tag } from 'lucide-react';
import { getJiraIssueUrl } from '../utils/jiraUtils';
import './EpicCard.css';

const STATUS_KIND = {
  done: ['done', 'closed', 'resolved', '已完成', '已取消'],
  progress: ['in progress', 'in-progress', '调研中', '调研完成', '设计完成', '开发完成', '测试完成', '验收完成'],
  todo: ['to do', 'todo', 'open', '待处理'],
};

const PRIORITY_KIND = {
  critical: ['highest', 'l0 - critical'],
  high: ['l1 - high'],
  medium: ['l2 - medium'],
  low: ['l3 - low'],
  lowest: ['lowest'],
};

const matchKind = (value, table) => {
  if (!value) return null;
  const v = value.toLowerCase();
  for (const [kind, list] of Object.entries(table)) {
    if (list.includes(v)) return kind;
  }
  return null;
};

const EpicCard = ({ epic, isDragging, onMoveEpic, onUpdateEpic, currentMilestone }) => {
  const statusKind = matchKind(epic.status, STATUS_KIND) || 'default';
  const priorityKind = matchKind(epic.priority, PRIORITY_KIND) || 'default';
  const jiraUrl = getJiraIssueUrl(epic.key);

  const handleOpenJira = (e) => {
    e.stopPropagation();
    if (jiraUrl) window.open(jiraUrl, '_blank', 'noopener,noreferrer');
  };

  const handleMoveClick = (e) => {
    e.stopPropagation();
    if (onMoveEpic && currentMilestone) onMoveEpic(epic, currentMilestone);
  };

  const handleEditClick = (e) => {
    e.stopPropagation();
    if (onUpdateEpic) onUpdateEpic(epic);
  };

  const versionsLabel = epic.versions && epic.versions.length ? epic.versions.join(', ') : null;

  return (
    <div className={`epic-card-content${isDragging ? ' is-dragging' : ''}`}>
      <div className="epic-row epic-row--top">
        <span className={`epic-key mono epic-key--${statusKind}`}>{epic.key}</span>
        <span className="epic-title" title={epic.name}>{epic.name}</span>
        {jiraUrl && (
          <button
            type="button"
            className="epic-icon-btn"
            onClick={handleOpenJira}
            title="Open in Jira"
            aria-label="Open in Jira"
          >
            <ExternalLink size={11} strokeWidth={1.75} />
          </button>
        )}
      </div>

      <div className="epic-row epic-row--bottom">
        <div className="epic-row__left">
          {epic.priority && (
            <span className={`epic-priority epic-priority--${priorityKind}`} title={`Priority: ${epic.priority}`}>
              <span className="epic-priority__dot" aria-hidden />
              <span className="epic-priority__label">{epic.priority?.replace(/^L\d - /i, '')}</span>
            </span>
          )}
          <span className="epic-versions" title={versionsLabel ? `Versions: ${versionsLabel}` : 'No versions'}>
            <Tag size={10} strokeWidth={1.75} />
            <span className="mono">{versionsLabel || '—'}</span>
          </span>
        </div>
        <div className="epic-row__right">
          {onUpdateEpic && (
            <button
              type="button"
              onClick={handleEditClick}
              className="epic-icon-btn"
              title="Edit epic"
              aria-label="Edit epic"
            >
              <Pencil size={11} strokeWidth={1.75} />
            </button>
          )}
          {onMoveEpic && (
            <button
              type="button"
              onClick={handleMoveClick}
              className="epic-icon-btn"
              title="Move epic to another milestone"
              aria-label="Move epic"
            >
              <ArrowRightLeft size={11} strokeWidth={1.75} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default EpicCard;
