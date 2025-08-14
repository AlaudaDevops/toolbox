import React, { useState, useEffect } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import AssigneeSelect from '../AssigneeSelect';
import './Modal.css';

const CreateEpicModal = ({ milestone, onClose }) => {
  const { createEpic, getComponentVersions } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [versions, setVersions] = useState([]);
  const [loadingVersions, setLoadingVersions] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    trigger,
    formState: { errors },
    setError,
    watch,
  } = useForm();

  const watchedComponent = watch('component');

  // Load versions when component changes
  useEffect(() => {
    if (watchedComponent && watchedComponent.trim()) {
      loadVersions(watchedComponent.trim());
    } else {
      setVersions([]);
    }
  }, [watchedComponent]);

  const loadVersions = async (component) => {
    setLoadingVersions(true);
    const result = await getComponentVersions(component);
    if (result.success) {
      setVersions(result.data || []);
    } else {
      setVersions([]);
    }
    setLoadingVersions(false);
  };

  const onSubmit = async (data) => {
    setIsSubmitting(true);

    const epicData = {
      name: data.name,
      component: data.component || '',
      version: data.version || '',
      priority: data.priority || 'Medium',
      milestone_id: milestone.id,
      assignee: data.assignee_id, // This now contains the full user object
    };

    const result = await createEpic(epicData);

    if (result.success) {
      onClose();
    } else {
      setError('root', { message: result.error });
    }

    setIsSubmitting(false);
  };

  const priorityOptions = [
    { value: 'Highest', label: 'Highest' },
    { value: 'High', label: 'High' },
    { value: 'Medium', label: 'Medium' },
    { value: 'Low', label: 'Low' },
    { value: 'Lowest', label: 'Lowest' },
  ];

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h2>Create Epic</h2>
            <p>Add a new epic to {milestone.name}</p>
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
                  value: 100,
                  message: 'Epic name must be less than 100 characters',
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
            <input
              id="component"
              type="text"
              className={`form-input ${errors.component ? 'error' : ''}`}
              placeholder="e.g., connectors-operator"
              {...register('component')}
            />
            {errors.component && (
              <span className="form-error">{errors.component.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="version" className="form-label">
              Version
            </label>
            {loadingVersions ? (
              <div className="loading-versions">
                <div className="loading-spinner-sm"></div>
                Loading versions...
              </div>
            ) : versions.length > 0 ? (
              <select
                id="version"
                className={`form-select ${errors.version ? 'error' : ''}`}
                {...register('version')}
              >
                <option value="">Select a version</option>
                {versions.map((version) => (
                  <option key={version} value={version}>
                    {version}
                  </option>
                ))}
              </select>
            ) : (
              <input
                id="version"
                type="text"
                className={`form-input ${errors.version ? 'error' : ''}`}
                placeholder="Enter version"
                {...register('version')}
              />
            )}
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
              defaultValue="Medium"
            >
              {priorityOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            {errors.priority && (
              <span className="form-error">{errors.priority.message}</span>
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
                  issueKey={milestone.key}
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
            <label className="form-label">Milestone</label>
            <div className="milestone-info">
              <strong>{milestone.name}</strong>
              <span className="milestone-key-display">{milestone.key}</span>
              <span className="milestone-quarter-display">{milestone.quarter}</span>
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
                'Create Epic'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateEpicModal;
