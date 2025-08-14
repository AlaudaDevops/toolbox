import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X, ArrowRight } from 'lucide-react';
import './Modal.css';

const EpicMoveModal = ({ epic, currentMilestone, availableMilestones, onClose }) => {
  const { moveEpic } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      milestone_id: epic.milestone_id,
    },
  });

  const onSubmit = async (data) => {
    // Don't submit if the milestone hasn't changed
    if (data.milestone_id === epic.milestone_id) {
      onClose();
      return;
    }

    setIsSubmitting(true);

    const result = await moveEpic(epic.id, data.milestone_id);

    if (result.success) {
      onClose();
    } else {
      setError('root', { message: result.error });
    }

    setIsSubmitting(false);
  };

  // Find the selected milestone for display
  // const selectedMilestone = availableMilestones.find(m => m.id === epic.milestone_id);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h2>Move Epic</h2>
            <p>Move epic to a different milestone</p>
          </div>
          <button
            onClick={onClose}
            className="btn btn-secondary"
            aria-label="Close modal"
          >
            <X size={16} />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="modal-body">
          <div className="form-group">
            <label className="form-label">Epic Details</label>
            <div className="epic-info">
              <div className="epic-name">
                <strong>{epic.name}</strong>
              </div>
              <div className="epic-meta">
                <span className="epic-key">{epic.key}</span>
                {epic.component && (
                  <span className="epic-component">Component: {epic.component}</span>
                )}
                {epic.version && (
                  <span className="epic-version">Version: {epic.version}</span>
                )}
              </div>
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">Current Milestone</label>
            <div className="milestone-display">
              <div className="milestone-info">
                <strong>{currentMilestone.name}</strong>
                <span className="milestone-quarter">({currentMilestone.quarter})</span>
              </div>
              <span className="milestone-key">{currentMilestone.key}</span>
            </div>
          </div>

          <div className="move-arrow">
            <ArrowRight size={20} />
          </div>

          <div className="form-group">
            <label htmlFor="milestone_id" className="form-label">
              New Milestone *
            </label>
            <select
              id="milestone_id"
              className={`form-select ${errors.milestone_id ? 'error' : ''}`}
              {...register('milestone_id', {
                required: 'Please select a milestone',
              })}
            >
              <option value="">Select a milestone</option>
              {availableMilestones.map((milestone) => (
                <option key={milestone.id} value={milestone.id}>
                  {milestone.name} ({milestone.quarter}) - {milestone.key}
                </option>
              ))}
            </select>
            {errors.milestone_id && (
              <span className="form-error">{errors.milestone_id.message}</span>
            )}
          </div>

          {errors.root && (
            <div className="form-error mb-4">
              {errors.root.message}
            </div>
          )}

          <div className="modal-footer">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-secondary"
              disabled={isSubmitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <div className="loading-spinner-sm"></div>
                  Moving...
                </>
              ) : (
                'Move Epic'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default EpicMoveModal;
