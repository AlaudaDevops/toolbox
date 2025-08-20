import React, { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import AssigneeSelect from '../AssigneeSelect';
import FilterableSelect from '../FilterableSelect';
import '../FilterableSelect.css';
import './Modal.css';

const UpdateEpicModal = ({ epic, onClose }) => {
  const { roadmapData, updateEpic } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      name: epic.name,
      component: epic.components? epic.components[0] : '',
      version: epic.versions? epic.versions[0] : '',
      priority: epic.priority || '',
      assignee: epic.assignee || null, // Will be handled separately for AssigneeSelect
    },
  });

  // Get available components and versions from roadmap data
  const availableComponents = roadmapData?.project?.components || [];
  const availableVersions = roadmapData?.project?.versions || [];

  const onSubmit = async (data) => {
    setIsSubmitting(true);

    const epicData = {
      name: data.name,
      component: data.component,
      version: data.version,
      priority: data.priority,
      assignee: data.assignee, // This contains the full user object from AssigneeSelect
    };

    const result = await updateEpic(epic.id, epicData);

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
            <h2>Update Epic</h2>
            <p>Edit epic details for {epic.key}</p>
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
              Epic Name *
            </label>
            <input
              id="name"
              type="text"
              className={`form-input ${errors.name ? 'error' : ''}`}
              placeholder="Enter epic name"
              {...register('name', {
                required: 'Epic name is required',
                minLength: {
                  value: 3,
                  message: 'Epic name must be at least 3 characters',
                },
                maxLength: {
                  value: 200,
                  message: 'Epic name must be less than 200 characters',
                },
              })}
            />
            {errors.name && (
              <span className="form-error">{errors.name.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="component" className="form-label">
              Component
            </label>
            <Controller
              name="component"
              control={control}
              render={({ field }) => (
                <FilterableSelect
                  id="component"
                  options={availableComponents}
                  placeholder="Select Component"
                  getOptionValue={(option) => option.name}
                  getOptionLabel={(option) => option.name}
                  value={field.value || ''}
                  onChange={(selectedOption) => {
                    console.log("component selectedOption:", selectedOption);
                    field.onChange(selectedOption?.value || '');
                  }}
                />
              )}
            />
            {errors.component && (
              <span className="form-error">{errors.component.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="version" className="form-label">
              Version
            </label>
            <Controller
              name="version"
              control={control}
              render={({ field }) => (
                <FilterableSelect
                  id="version"
                  options={availableVersions}
                  placeholder="Select Version"
                  getOptionValue={(option) => option.name}
                  getOptionLabel={(option) => option.name}
                  value={field.value || ''}
                  onChange={(selectedOption) => {
                    console.log("version selectedOption:", selectedOption);
                    field.onChange(selectedOption?.value || '');
                  }}
                />
              )}
            />
            {errors.version && (
              <span className="form-error">{errors.version.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="priority" className="form-label">
              Priority
            </label>
            <select
              id="priority"
              className={`form-select ${errors.priority ? 'error' : ''}`}
              {...register('priority')}
            >
              <option value="">Select priority</option>
              <option value="L0 - Critical">L0 - Critical</option>
              <option value="L1 - High">L1 - High</option>
              <option value="L2 - Medium">L2 - Medium</option>
              <option value="L3 - Low">L3 - Low</option>
            </select>
            {errors.priority && (
              <span className="form-error">{errors.priority.message}</span>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Assignee</label>
            <Controller
              name="assignee"
              control={control}
              render={({ field }) => (
                <AssigneeSelect
                  value={field.value}
                  onChange={field.onChange}
                  issueKey={epic.key}
                />
              )}
            />
            {errors.assignee && (
              <span className="form-error">{errors.assignee.message}</span>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Epic Key</label>
            <div className="epic-info">
              <strong>{epic.key}</strong>
              <span className="epic-status-display">Status: {epic.status}</span>
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
                'Update Epic'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default UpdateEpicModal;
