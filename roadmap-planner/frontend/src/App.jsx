import React, { useEffect, useState } from 'react';
import { Toaster } from 'react-hot-toast';
import KanbanBoard from './components/KanbanBoard';
import MetricsDashboard from './components/MetricsDashboard';
import LoginModal from './components/modals/LoginModal';
import { AuthProvider, useAuth } from './hooks/useAuth';
import { RoadmapProvider } from './hooks/useRoadmap';
import { LogOut, LayoutGrid, Activity, Sun, Moon, BookOpen, Layers } from 'lucide-react';
import './App.css';

const THEME_KEY = 'roadmap-planner-theme';   // 'platform' | 'atlas'
const MODE_KEY = 'roadmap-planner-mode';     // 'light' | 'dark'

function readStored(key, allowed, fallback) {
  if (typeof window === 'undefined') return fallback;
  try {
    const v = window.localStorage.getItem(key);
    if (allowed.includes(v)) return v;
  } catch { /* ignore */ }
  return fallback;
}

// In v1 of this app, THEME_KEY stored 'light' | 'dark' (single-axis dark mode).
// In v2, the same key stores 'platform' | 'atlas'. Migrate during the read so
// the migration result becomes the actual initial state — otherwise the
// useEffect that persists state would immediately overwrite it.
function getInitialTheme() {
  if (typeof window === 'undefined') return 'platform';
  let stored;
  try { stored = window.localStorage.getItem(THEME_KEY); } catch { return 'platform'; }
  if (stored === 'platform' || stored === 'atlas') return stored;
  if (stored === 'light' || stored === 'dark') {
    try {
      // Preserve the v1 user's mode preference under MODE_KEY,
      // and treat them as Atlas users (they only had Atlas).
      window.localStorage.setItem(MODE_KEY, stored);
      window.localStorage.setItem(THEME_KEY, 'atlas');
    } catch { /* ignore */ }
    return 'atlas';
  }
  return 'platform';
}

function getInitialMode() {
  const stored = readStored(MODE_KEY, ['light', 'dark'], null);
  if (stored) return stored;
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyAttributes(theme, mode) {
  const html = document.documentElement;
  html.setAttribute('data-theme', theme);
  html.setAttribute('data-mode', mode);
}

function useThemeAndMode() {
  const [theme, setTheme] = useState(getInitialTheme);
  const [mode, setMode] = useState(getInitialMode);

  useEffect(() => {
    applyAttributes(theme, mode);
    try {
      window.localStorage.setItem(THEME_KEY, theme);
      window.localStorage.setItem(MODE_KEY, mode);
    } catch { /* ignore */ }
  }, [theme, mode]);

  return {
    theme,
    mode,
    setTheme,
    toggleMode: () => setMode((m) => (m === 'light' ? 'dark' : 'light')),
  };
}

function ThemePicker({ theme, setTheme }) {
  return (
    <div className="theme-picker" role="radiogroup" aria-label="Theme">
      <button
        type="button"
        role="radio"
        aria-checked={theme === 'platform'}
        className={`theme-picker__opt${theme === 'platform' ? ' is-active' : ''}`}
        onClick={() => setTheme('platform')}
        title="Platform — matches the Alauda console"
      >
        <Layers size={13} strokeWidth={1.75} />
        <span>Platform</span>
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={theme === 'atlas'}
        className={`theme-picker__opt${theme === 'atlas' ? ' is-active' : ''}`}
        onClick={() => setTheme('atlas')}
        title="Atlas — editorial alternate"
      >
        <BookOpen size={13} strokeWidth={1.75} />
        <span>Atlas</span>
      </button>
    </div>
  );
}

function AppContent() {
  const { isAuthenticated, isLoading, logout, user, project } = useAuth();
  const [currentView, setCurrentView] = useState('roadmap');
  const { theme, mode, setTheme, toggleMode } = useThemeAndMode();

  if (isLoading) {
    return (
      <div className="app-loading">
        <div className="atlas-spinner" />
        <p className="serif app-loading__text">Loading…</p>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginModal />;
  }

  return (
    <RoadmapProvider>
      <div className="app">
        <header className="app-header">
          <div className="app-header__brand">
            <div className="app-header__mark" data-editorial>
              <span className="app-header__bracket">[</span>
              <span className="app-header__index mono">N° 01</span>
              <span className="app-header__bracket">]</span>
            </div>
            <div className="app-header__title">
              <span className="serif app-header__display">Roadmap</span>
              <span className="app-header__display-stacked mono">PLANNER</span>
            </div>
            <div className="app-header__meta" data-editorial>
              <span className="app-header__rule" aria-hidden />
              <span className="serif app-header__edition">Atlas Edition · Vol. II</span>
            </div>
          </div>

          <nav className="app-nav" aria-label="Primary">
            <button
              type="button"
              className={`app-nav__item ${currentView === 'roadmap' ? 'is-active' : ''}`}
              onClick={() => setCurrentView('roadmap')}
            >
              <LayoutGrid size={14} strokeWidth={1.75} />
              <span>Roadmap</span>
            </button>
            <button
              type="button"
              className={`app-nav__item ${currentView === 'metrics' ? 'is-active' : ''}`}
              onClick={() => setCurrentView('metrics')}
            >
              <Activity size={14} strokeWidth={1.75} />
              <span>Metrics</span>
            </button>
          </nav>

          <div className="app-header__right">
            {project && (
              <span className="app-header__project mono" title="Active Jira project">
                {project}
              </span>
            )}
            {user?.display_name && (
              <span className="app-header__user" title={user.email_address}>
                <span className="serif">by</span>&nbsp;{user.display_name}
              </span>
            )}
            <ThemePicker theme={theme} setTheme={setTheme} />
            <button
              type="button"
              onClick={toggleMode}
              className="btn btn-icon btn-ghost"
              aria-label={mode === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
              title={mode === 'light' ? 'Dark mode' : 'Light mode'}
            >
              {mode === 'light' ? <Moon size={16} strokeWidth={1.75} /> : <Sun size={16} strokeWidth={1.75} />}
            </button>
            <button
              type="button"
              onClick={logout}
              className="btn btn-icon btn-ghost"
              aria-label="Log out"
              title="Log out"
            >
              <LogOut size={16} strokeWidth={1.75} />
            </button>
          </div>
        </header>

        <main className="app-main">
          {currentView === 'roadmap' ? <KanbanBoard /> : <MetricsDashboard />}
        </main>

        <footer className="app-footer" data-editorial>
          <span className="serif">Issue 02 · </span>
          <span className="mono">EST. 2024</span>
          <span className="app-footer__dot" aria-hidden>·</span>
          <span>Alauda DevOps</span>
        </footer>
      </div>
    </RoadmapProvider>
  );
}

function App() {
  useEffect(() => {
    applyAttributes(getInitialTheme(), getInitialMode());
  }, []);

  return (
    <AuthProvider>
      <AppContent />
      <Toaster
        position="bottom-right"
        toastOptions={{
          duration: 3500,
          style: {
            background: 'var(--fg)',
            color: 'var(--bg-elevated)',
            borderRadius: 'var(--radius-default)',
            border: '1px solid var(--border)',
            fontFamily: 'var(--font-ui)',
            fontSize: '13px',
            padding: '10px 14px',
          },
          success: {
            iconTheme: { primary: '#22c55e', secondary: 'var(--fg)' },
          },
          error: {
            duration: 5000,
            iconTheme: { primary: '#ef4444', secondary: 'var(--fg)' },
          },
        }}
      />
    </AuthProvider>
  );
}

export default App;
