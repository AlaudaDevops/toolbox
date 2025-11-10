import React, { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRoadmap } from '../../hooks/useRoadmap';
import { X } from 'lucide-react';
import AssigneeSelect from '../AssigneeSelect';
import FilterableSelect from '../FilterableSelect';
import '../FilterableSelect.css';
import './Modal.css';

const CreateEpicModal = ({ milestone, onClose }) => {
  const { roadmapData, createEpic } = useRoadmap();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    trigger,
    formState: { errors },
    setError,
  } = useForm();

  // Get available components and versions from roadmap data
  const availableComponents = roadmapData?.components || [];
  const availableVersions = roadmapData?.versions || [];

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
    { value: 'L0 - Critical', label: 'L0 - Critical' },
    { value: 'L1 - High', label: 'L1 - High', default: true},
    { value: 'L2 - Medium', label: 'L2 - Medium' },
    { value: 'L3 - Low', label: 'L3 - Low' },
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
            <Controller
              name="component"
              control={control}
              render={({ field }) => (
                <FilterableSelect
                  id="component"
                  value={field.value || ''}
                  onChange={(selectedOption) => {
                    console.debug("component changed?", selectedOption);
                    field.onChange(selectedOption?.value || '');
                  }}
                  options={availableComponents}
                  placeholder="Search and select a component..."
                  className={errors.component ? 'error' : ''}
                  error={!!errors.component}
                  disabled={isSubmitting}
                  getOptionLabel={(component) => component.name}
                  getOptionValue={(component) => component.name}
                  filterFunction={(component, searchTerm) => {
                    const search = searchTerm.toLowerCase();
                    return (
                      component.data.name.toLowerCase().includes(search) ||
                      (component.data.description && component.data.description.toLowerCase().includes(search))
                    );
                  }}
                  emptyMessage="No components found"
                  isClearable={true}
                  isSearchable={true}
                  menuPortalTarget={document.body}
                  menuPosition="fixed"
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
                  value={field.value || ''}
                  onChange={(selectedOption) => {
                    console.debug("version changed?", selectedOption);
                    field.onChange(selectedOption?.value || '');
                  }}
                  options={availableVersions}
                  placeholder="Search and select a version..."
                  className={errors.version ? 'error' : ''}
                  error={!!errors.version}
                  disabled={isSubmitting}
                  getOptionLabel={(version) => version.name}
                  getOptionValue={(version) => version.name}
                  filterFunction={(version, searchTerm) => {
                    const search = searchTerm.toLowerCase();
                    return version.data.name.toLowerCase().includes(search);
                  }}
                  emptyMessage="No versions found"
                  isClearable={true}
                  isSearchable={true}
                  menuPortalTarget={document.body}
                  menuPosition="fixed"
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
              defaultValue="L1 - High"
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
