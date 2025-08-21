import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import './Modal.css';

const UpdateMilestoneModal = ({ milestone, quarters, onClose }) => {
  const { updateMilestone } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      name: milestone.name,
      quarter: milestone.quarter,
    },
  });

  const onSubmit = async (data) => {
    setIsSubmitting(true);

    const milestoneData = {
      name: data.name,
      quarter: data.quarter,
    };

    const result = await updateMilestone(milestone.id, milestoneData);

    if (result.success) {
      onClose();
    } else {
      setError('root', { message: result.error });
    }

    setIsSubmitting(false);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h2>Update Milestone</h2>
            <p>Edit milestone details for {milestone.key}</p>
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
            <label htmlFor="name" className="form-label">
              Milestone Name *
            </label>
            <input
              id="name"
              type="text"
              className={`form-input ${errors.name ? 'error' : ''}`}
              placeholder="Enter milestone name"
              {...register('name', {
                required: 'Milestone name is required',
                minLength: {
                  value: 3,
                  message: 'Milestone name must be at least 3 characters',
                },
                maxLength: {
                  value: 100,
                  message: 'Milestone name must be less than 100 characters',
                },
              })}
            />
            {errors.name && (
              <span className="form-error">{errors.name.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="quarter" className="form-label">
              Quarter *
            </label>
            <select
              id="quarter"
              className={`form-select ${errors.quarter ? 'error' : ''}`}
              {...register('quarter', {
                required: 'Quarter is required',
              })}
            >
              <option value="">Select a quarter</option>
              {quarters.map((quarter) => (
                <option key={quarter} value={quarter}>
                  {quarter}
                </option>
              ))}
            </select>
            {errors.quarter && (
              <span className="form-error">{errors.quarter.message}</span>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Milestone Key</label>
            <div className="milestone-info">
              <strong>{milestone.key}</strong>
              <span className="milestone-status-display">Status: {milestone.status}</span>
            </div>
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
                  Updating...
                </>
              ) : (
                'Update Milestone'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default UpdateMilestoneModal;
