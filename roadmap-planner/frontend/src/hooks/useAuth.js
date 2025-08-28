import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { authAPI, getStoredAuth, setStoredAuth, clearStoredAuth, handleAPIError } from '../services/api';
import toast from 'react-hot-toast';

const AuthContext = createContext();

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [project, setProject] = useState('');
  const [user, setUser] = useState(null);
  const [projectCallbacks, setProjectCallbacks] = useState(new Set());

  // Check authentication status on mount
  useEffect(() => {
    checkAuthStatus();
  }, []);

  const checkAuthStatus = async () => {
    try {
      const storedAuth = getStoredAuth();
      if (!storedAuth) {
        setIsLoading(false);
        return;
      }

      // Verify with server
      const response = await authAPI.status();
      if (response.authenticated) {
        setIsAuthenticated(true);
        setUser(response.user);
      } else {
        clearStoredAuth();
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      clearStoredAuth();
    } finally {
      setIsLoading(false);
    }
  };

  // Add callback registration for project changes
  const onProjectChange = useCallback((callback) => {
    setProjectCallbacks(prev => new Set(prev).add(callback));

    // Return unsubscribe function
    return () => {
      setProjectCallbacks(prev => {
        const newSet = new Set(prev);
        newSet.delete(callback);
        return newSet;
      });
    };
  }, []);


  // Enhanced setProject function that triggers callbacks
  const updateProject = useCallback((newProject) => {
    setProject(prevProject => {
      if (prevProject !== newProject) {
        // Update auth data
        setStoredAuth(prevAuth => ({
          ...prevAuth,
          project: newProject,
        }));

        // Trigger all registered callbacks
        projectCallbacks.forEach(callback => {
          try {
            callback(newProject, prevProject);
          } catch (error) {
            console.error('Project change callback error:', error);
          }
        });
      }
      return newProject;
    });
  }, [projectCallbacks]);

  const login = async (credentials) => {
    try {
      setIsLoading(true);

      // Validate credentials with server
      const response = await authAPI.login(credentials);

      // Store credentials for API requests
      const authData = {
        username: credentials.username,
        password: credentials.password,
        baseURL: credentials.base_url,
        project: credentials.project,
      };

      setProject(credentials.project);
      setStoredAuth(authData);
      setIsAuthenticated(true);
      setUser(response.user);

      toast.success('Successfully logged in!');
      return { success: true };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(errorInfo.message);
      return { success: false, error: errorInfo.message };
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    try {
      await authAPI.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      clearStoredAuth();
      setIsAuthenticated(false);
      setUser(null);
      setProject('');
      toast.success('Successfully logged out!');
    }
  };

  const value = {
    isAuthenticated,
    isLoading,
    user,
    project,
    login,
    logout,
    checkAuthStatus,
    onProjectChange,
    updateProject,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};
