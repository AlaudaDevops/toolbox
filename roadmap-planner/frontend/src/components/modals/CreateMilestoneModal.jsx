import React, { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import AssigneeSelect from '../AssigneeSelect';
import './Modal.css';

const CreateMilestoneModal = ({ pillar, quarters, preselectedQuarter, onClose }) => {
  const { createMilestone } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    trigger,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      quarter: preselectedQuarter || '',
    },
  });

  const onSubmit = async (data) => {
    setIsSubmitting(true);

    const milestoneData = {
      name: data.name,
      quarter: data.quarter,
      pillar_id: pillar.id,
      assignee: data.assignee_id, // This now contains the full user object
    };

    const result = await createMilestone(milestoneData);

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
            <h2>Create Milestone</h2>
            <p>Add a new milestone to {pillar.name}</p>
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
            <label htmlFor="assignee_id" className="form-label">
              Assignee *
            </label>
            <Controller
              name="assignee_id"
              control={control}
              rules={{
                required: 'Assignee is required',
                validate: (value) => value && value.account_id ? true : 'Please select a valid assignee'
              }}
              render={({ field, fieldState }) => (
                <AssigneeSelect
                  issueKey={pillar.key}
                  value={field.value || null}
                  onChange={(value) => {
                    field.onChange(value);
                    // Trigger validation immediately when value changes
                    if (value) {
                      trigger('assignee_id');
                    }
                  }}
                  error={fieldState.error}
                  placeholder="Search and select an assignee..."
                  isRequired={true}
                />
              )}
            />
            {errors.assignee_id && (
              <span className="form-error">{errors.assignee_id.message}</span>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Pillar</label>
            <div className="pillar-info">
              <strong>{pillar.name}</strong>
              <span className="pillar-key-display">{pillar.key}</span>
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
                  Creating...
                </>
              ) : (
                'Create Milestone'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateMilestoneModal;
