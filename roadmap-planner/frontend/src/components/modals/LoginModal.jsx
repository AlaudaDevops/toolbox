import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useAuth } from '../../hooks/useAuth';
import './Modal.css';

const LoginModal = () => {
  const { login, isLoading } = useAuth();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm({
    defaultValues: {
      base_url: 'https://jira.alauda.cn',
      username: '',
      password: '',
    },
  });

  const onSubmit = async (data) => {
    const result = await login(data);
    if (!result.success) {
      setError('root', { message: result.error });
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2>Login to Jira</h2>
          <p>Enter your Jira credentials to access the roadmap planner</p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="modal-body">
          <div className="form-group">
            <label htmlFor="base_url" className="form-label">
              Jira Base URL
            </label>
            <input
              id="base_url"
              type="url"
              className={`form-input ${errors.base_url ? 'error' : ''}`}
              placeholder="https://jira.alauda.cn"
              {...register('base_url', {
                required: 'Jira Base URL is required',
                pattern: {
                  value: /^https?:\/\/.+/,
                  message: 'Please enter a valid URL',
                },
              })}
            />
            {errors.base_url && (
              <span className="form-error">{errors.base_url.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="username" className="form-label">
              Username/Email
            </label>
            <input
              id="username"
              type="text"
              className={`form-input ${errors.username ? 'error' : ''}`}
              placeholder="your.email@company.com"
              {...register('username', {
                required: 'Username is required',
              })}
            />
            {errors.username && (
              <span className="form-error">{errors.username.message}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="password" className="form-label">
              Password/API Token
            </label>
            <div className="password-input-container">
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                className={`form-input ${errors.password ? 'error' : ''}`}
                placeholder="Your password or API token"
                {...register('password', {
                  required: 'Password is required',
                })}
              />
              <button
                type="button"
                className="password-toggle"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? '👁️' : '👁️‍🗨️'}
              </button>
            </div>
            {errors.password && (
              <span className="form-error">{errors.password.message}</span>
            )}
          </div>

          {errors.root && (
            <div className="form-error mb-4">
              {errors.root.message}
            </div>
          )}

          <div className="modal-footer">
            <button
              type="submit"
              className="btn btn-primary btn-lg w-full"
              disabled={isLoading}
            >
              {isLoading ? (
                <>
                  <div className="loading-spinner-sm"></div>
                  Connecting...
                </>
              ) : (
                'Login'
              )}
            </button>
          </div>
        </form>

        <div className="modal-help">
          <h4>Need help?</h4>
          <ul>
            <li>Use your Jira username and password, or an API token for better security</li>
            <li>Make sure your Jira instance URL is correct (e.g., https://company.atlassian.net)</li>
            <li>You need access to the DEVOPS project to use this tool</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default LoginModal;
