import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useAuth } from '../../hooks/useAuth';
import { Eye, EyeOff, Compass } from 'lucide-react';
import './Modal.css';
import './LoginModal.css';

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
      project: '',
    },
  });

  const onSubmit = async (data) => {
    const result = await login(data);
    if (!result.success) setError('root', { message: result.error });
  };

  return (
    <div className="login-page">
      <div className="login-page__bg" aria-hidden />
      <div className="login-page__plate">
        <aside className="login-page__editorial">
          <div className="login-page__pretitle">
            <span className="serif">An almanac for</span>
            <span className="mono">DEVOPS · 2026</span>
          </div>
          <h1 className="login-page__title">
            <span className="serif">The </span>
            Roadmap
            <span className="serif">,</span>
            <br />
            redrawn
            <span className="serif"> quarterly.</span>
          </h1>
          <p className="login-page__lede">
            Pillars in rows, quarters in columns. Move epics
            with a hand on the gripper, watch the chart
            redraw itself before the meeting starts.
          </p>
          <div className="login-page__signature">
            <span className="login-page__rule" />
            <Compass size={14} strokeWidth={1.5} />
            <span className="serif">Atlas Edition · Vol. II</span>
          </div>
        </aside>

        <section className="login-page__form-wrap">
          <header className="login-page__form-head">
            <span className="login-page__form-eyebrow mono">Sign in · Jira</span>
            <h2 className="login-page__form-title">
              <span className="serif">Open the </span>atlas
            </h2>
            <p className="login-page__form-sub">
              Use your Jira credentials, or an API token. Nothing leaves your browser
              beyond the API request.
            </p>
          </header>

          <form onSubmit={handleSubmit(onSubmit)} className="login-form">
            <div className="form-group">
              <label htmlFor="base_url" className="form-label">Jira Base URL</label>
              <input
                id="base_url"
                type="url"
                className={`form-input ${errors.base_url ? 'error' : ''}`}
                placeholder="https://jira.example.com"
                autoComplete="url"
                {...register('base_url', {
                  required: 'Jira Base URL is required',
                  pattern: { value: /^https?:\/\/.+/, message: 'Please enter a valid URL' },
                })}
              />
              {errors.base_url && <span className="form-error">{errors.base_url.message}</span>}
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="username" className="form-label">Username / Email</label>
                <input
                  id="username"
                  type="text"
                  className={`form-input ${errors.username ? 'error' : ''}`}
                  placeholder="you@company.com"
                  autoComplete="username"
                  {...register('username', { required: 'Username is required' })}
                />
                {errors.username && <span className="form-error">{errors.username.message}</span>}
              </div>

              <div className="form-group">
                <label htmlFor="project" className="form-label">Project</label>
                <input
                  id="project"
                  type="text"
                  className={`form-input mono ${errors.project ? 'error' : ''}`}
                  placeholder="DEVOPS"
                  {...register('project', {
                    required: 'Project is required',
                    pattern: { value: /^[A-Z][A-Z0-9]+/, message: 'Project must be uppercase letters/digits' },
                  })}
                />
                {errors.project && <span className="form-error">{errors.project.message}</span>}
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="password" className="form-label">Password / API Token</label>
              <div className="password-input-container">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  className={`form-input ${errors.password ? 'error' : ''}`}
                  placeholder="••••••••"
                  autoComplete="current-password"
                  {...register('password', { required: 'Password is required' })}
                />
                <button
                  type="button"
                  className="password-toggle"
                  onClick={() => setShowPassword(!showPassword)}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? <EyeOff size={14} strokeWidth={1.75} /> : <Eye size={14} strokeWidth={1.75} />}
                </button>
              </div>
              {errors.password && <span className="form-error">{errors.password.message}</span>}
            </div>

            {errors.root && (
              <div className="form-banner form-banner--error">{errors.root.message}</div>
            )}

            <button
              type="submit"
              className="btn btn-primary btn-lg w-full login-form__submit"
              disabled={isLoading}
            >
              {isLoading ? (
                <>
                  <div className="atlas-spinner sm" />
                  Connecting…
                </>
              ) : (
                <>
                  <span>Sign in</span>
                  <span className="login-form__submit-arrow serif">→</span>
                </>
              )}
            </button>
          </form>

          <ul className="login-form__tips">
            <li><span className="serif">·</span> Use an API token rather than your password when possible.</li>
            <li><span className="serif">·</span> Make sure the Jira URL is reachable from your network.</li>
            <li><span className="serif">·</span> You need read+edit access to the project to make changes.</li>
          </ul>
        </section>
      </div>
    </div>
  );
};

export default LoginModal;
