import React, { useState, useEffect } from 'react';
import { Toaster } from 'react-hot-toast';
import KanbanBoard from './components/KanbanBoard';
import LoginModal from './components/modals/LoginModal';
import { AuthProvider, useAuth } from './hooks/useAuth';
import { RoadmapProvider } from './hooks/useRoadmap';
import { LogOut } from 'lucide-react';
import './App.css';

function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();
  const { logout, user } = useAuth();

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
          <h2>Roadmap Planner</h2>
          <button onClick={logout} className="btn " title="Logout">
            <LogOut size={16} />
          </button>
        </header>
        <main className="app-main">
          <KanbanBoard />
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
