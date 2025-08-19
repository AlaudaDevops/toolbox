import React, { useState, useEffect } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import AssigneeSelect from '../AssigneeSelect';
import './Modal.css';

const UpdateEpicModal = ({ epic, onClose }) => {
  const { updateEpic, getComponentVersions } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [versions, setVersions] = useState([]);
  const [loadingVersions, setLoadingVersions] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      name: epic.name,
      component: epic.component || '',
      version: epic.version || '',
      priority: epic.priority || '',
      assignee: null, // Will be set by AssigneeSelect
    },
  });

  const watchedComponent = watch('component');

  // Load versions when component changes
  useEffect(() => {
    const loadVersions = async () => {
      if (watchedComponent) {
        setLoadingVersions(true);
        try {
          const result = await getComponentVersions(watchedComponent);
          if (result.success) {
            setVersions(result.data.versions || []);
          } else {
            setVersions([]);
          }
        } catch (error) {
          console.error('Failed to load versions:', error);
          setVersions([]);
        } finally {
          setLoadingVersions(false);
        }
      } else {
        setVersions([]);
      }
    };

    loadVersions();
  }, [watchedComponent, getComponentVersions]);

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
            <input
              id="component"
              type="text"
              className={`form-input ${errors.component ? 'error' : ''}`}
              placeholder="Enter component name"
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
            <select
              id="version"
              className={`form-select ${errors.version ? 'error' : ''}`}
              disabled={loadingVersions || versions.length === 0}
              {...register('version')}
            >
              <option value="">Select a version</option>
              {versions.map((version) => (
                <option key={version} value={version}>
                  {version}
                </option>
              ))}
            </select>
            {loadingVersions && (
              <span className="form-help">Loading versions...</span>
            )}
            {!loadingVersions && watchedComponent && versions.length === 0 && (
              <span className="form-help">No versions found for this component</span>
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
            >
              <option value="">Select priority</option>
              <option value="Highest">Highest</option>
              <option value="High">High</option>
              <option value="Medium">Medium</option>
              <option value="Low">Low</option>
              <option value="Lowest">Lowest</option>
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
