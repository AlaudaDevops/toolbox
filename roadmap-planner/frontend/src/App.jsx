import React, { useEffect, useState } from 'react';
import { Toaster } from 'react-hot-toast';
import KanbanBoard from './components/KanbanBoard';
import MetricsDashboard from './components/MetricsDashboard';
import LoginModal from './components/modals/LoginModal';
import { AuthProvider, useAuth } from './hooks/useAuth';
import { RoadmapProvider } from './hooks/useRoadmap';
import { LogOut, LayoutGrid, Activity, Sun, Moon } from 'lucide-react';
import './App.css';

const THEME_KEY = 'roadmap-planner-theme';

function getInitialTheme() {
  if (typeof window === 'undefined') return 'light';
  try {
    const stored = window.localStorage.getItem(THEME_KEY);
    if (stored === 'light' || stored === 'dark') return stored;
  } catch { /* ignore */ }
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function useTheme() {
  const [theme, setTheme] = useState(getInitialTheme);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    try { window.localStorage.setItem(THEME_KEY, theme); } catch { /* ignore */ }
  }, [theme]);

  const toggle = () => setTheme((t) => (t === 'light' ? 'dark' : 'light'));
  return [theme, toggle];
}

function AppContent() {
  const { isAuthenticated, isLoading, logout, user, project } = useAuth();
  const [currentView, setCurrentView] = useState('roadmap');
  const [theme, toggleTheme] = useTheme();

  if (isLoading) {
    return (
      <div className="app-loading">
        <div className="atlas-spinner" />
        <p className="serif app-loading__text">Loading the atlas…</p>
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
            <div className="app-header__mark">
              <span className="app-header__bracket">[</span>
              <span className="app-header__index mono">N° 01</span>
              <span className="app-header__bracket">]</span>
            </div>
            <div className="app-header__title">
              <span className="serif app-header__display">Roadmap</span>
              <span className="app-header__display-stacked mono">PLANNER</span>
            </div>
            <div className="app-header__meta">
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
            <button
              type="button"
              onClick={toggleTheme}
              className="btn btn-icon btn-ghost"
              aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
              title={theme === 'light' ? 'Dark mode' : 'Light mode'}
            >
              {theme === 'light' ? <Moon size={16} strokeWidth={1.75} /> : <Sun size={16} strokeWidth={1.75} />}
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

        <footer className="app-footer">
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
    document.documentElement.setAttribute('data-theme', getInitialTheme());
  }, []);

  return (
    <AuthProvider>
      <AppContent />
      <Toaster
        position="bottom-right"
        toastOptions={{
          duration: 3500,
          style: {
            background: 'var(--ink)',
            color: 'var(--paper)',
            borderRadius: 0,
            border: '1px solid var(--ink-2)',
            fontFamily: 'var(--font-ui)',
            fontSize: '13px',
            padding: '10px 14px',
          },
          success: {
            iconTheme: { primary: '#4ade80', secondary: 'var(--ink)' },
          },
          error: {
            duration: 5000,
            iconTheme: { primary: '#f87171', secondary: 'var(--ink)' },
          },
        }}
      />
    </AuthProvider>
  );
}

export default App;
