import React, { useState } from 'react';
import { Toaster } from 'react-hot-toast';
import KanbanBoard from './components/KanbanBoard';
import MetricsDashboard from './components/MetricsDashboard';
import LoginModal from './components/modals/LoginModal';
import { AuthProvider, useAuth } from './hooks/useAuth';
import { RoadmapProvider } from './hooks/useRoadmap';
import { LogOut, LayoutDashboard, BarChart3 } from 'lucide-react';
import './App.css';

function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();
  const { logout } = useAuth();
  const [currentView, setCurrentView] = useState('roadmap'); // 'roadmap' | 'metrics'

  if (isLoading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>Loading...</p>
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
          <div className="app-header__left">
            <h2>Roadmap Planner</h2>
            <nav className="app-nav">
              <button
                className={`app-nav__item ${currentView === 'roadmap' ? 'app-nav__item--active' : ''}`}
                onClick={() => setCurrentView('roadmap')}
              >
                <LayoutDashboard size={16} />
                <span>Roadmap</span>
              </button>
              <button
                className={`app-nav__item ${currentView === 'metrics' ? 'app-nav__item--active' : ''}`}
                onClick={() => setCurrentView('metrics')}
              >
                <BarChart3 size={16} />
                <span>Metrics</span>
              </button>
            </nav>
          </div>
          <div className="right-top">
            <button onClick={logout} className="btn " title="Logout">
              <LogOut size={16} />
            </button>
          </div>
        </header>
        <main className="app-main">
          {currentView === 'roadmap' ? <KanbanBoard /> : <MetricsDashboard />}
        </main>
      </div>
    </RoadmapProvider>
  );
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#363636',
            color: '#fff',
          },
          success: {
            duration: 3000,
            iconTheme: {
              primary: '#4ade80',
              secondary: '#fff',
            },
          },
          error: {
            duration: 5000,
            iconTheme: {
              primary: '#ef4444',
              secondary: '#fff',
            },
          },
        }}
      />
    </AuthProvider>
  );
}

export default App;
