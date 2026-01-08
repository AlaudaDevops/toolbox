import axios from 'axios';

// Create axios instance with default configuration
const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || (process.env.NODE_ENV === 'production' ? '' : 'http://localhost:8080'),
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add authentication headers
api.interceptors.request.use(
  (config) => {
    const auth = getStoredAuth();
    if (auth) {
      config.headers['X-Jira-Username'] = auth.username;
      config.headers['X-Jira-Password'] = auth.password;
      config.headers['X-Jira-BaseURL'] = auth.baseURL;
      config.headers['X-Jira-Project'] = auth.project || 'DEVOPS';
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear stored auth on unauthorized
      clearStoredAuth();
      window.location.reload();
    }
    return Promise.reject(error);
  }
);

// Auth storage helpers
const AUTH_STORAGE_KEY = 'roadmap_planner_auth';

export const getStoredAuth = () => {
  try {
    const stored = localStorage.getItem(AUTH_STORAGE_KEY);
    return stored ? JSON.parse(stored) : null;
  } catch {
    return null;
  }
};

export const setStoredAuth = (auth) => {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(auth));
};

export const clearStoredAuth = () => {
  localStorage.removeItem(AUTH_STORAGE_KEY);
};

// API methods
export const authAPI = {
  login: async (credentials) => {
    const response = await api.post('/api/auth/login', credentials);
    return response.data;
  },

  logout: async () => {
    const response = await api.post('/api/auth/logout');
    clearStoredAuth();
    return response.data;
  },

  status: async () => {
    const response = await api.get('/api/auth/status');
    return response.data;
  },
};

export const roadmapAPI = {
  getBasicData: async () => {
    const response = await api.get('/api/basic');
    return response.data;
  },

  // Filtering APIs for batch operations
  getMilestones: async (filters = {}) => {
    const params = new URLSearchParams();
    if (filters.pillarIds) {
      filters.pillarIds.forEach(id => params.append('pillar_id', id));
    }
    if (filters.quarters) {
      filters.quarters.forEach(quarter => params.append('quarter', quarter));
    }
    const response = await api.get(`/api/milestones?${params.toString()}`);
    return response.data;
  },

  getEpics: async (filters = {}) => {
    const params = new URLSearchParams();
    if (filters.milestoneIds) {
      filters.milestoneIds.forEach(id => params.append('milestone_id', id));
    }
    if (filters.pillarIds) {
      filters.pillarIds.forEach(id => params.append('pillar_id', id));
    }
    if (filters.components) {
      filters.components.forEach(component => params.append('component', component));
    }
    if (filters.versions) {
      filters.versions.forEach(version => params.append('version', version));
    }
    const response = await api.get(`/api/epics?${params.toString()}`);
    return response.data;
  },

  createMilestone: async (milestoneData) => {
    const response = await api.post('/api/milestones', milestoneData);
    return response.data;
  },

  updateMilestone: async (milestoneId, milestoneData) => {
    const response = await api.put(`/api/milestones/${milestoneId}`, milestoneData);
    return response.data;
  },

  createEpic: async (epicData) => {
    const response = await api.post('/api/epics', epicData);
    return response.data;
  },

  updateEpic: async (epicId, epicData) => {
    const response = await api.put(`/api/epics/${epicId}`, epicData);
    return response.data;
  },

  updateEpicMilestone: async (epicId, milestoneId) => {
    const response = await api.put(`/api/epics/${epicId}/milestone`, {
      milestone_id: milestoneId,
    });
    return response.data;
  },

  getComponentVersions: async (componentName) => {
    const response = await api.get(`/api/components/${componentName}/versions`);
    return response.data;
  },

  getAssignableUsers: async (issueKey, query = '') => {
    const params = new URLSearchParams();
    if (issueKey) {
      params.append('issueKey', issueKey);
    }
    if (query) {
      params.append('query', query);
    }

    const response = await api.get(`/api/users/assignable?${params.toString()}`);
    return response.data;
  },
};

export const metricsAPI = {
  // Get all available metrics
  listMetrics: async () => {
    const response = await api.get('/api/metrics');
    return response.data;
  },

  // Get specific metric with filters
  getMetric: async (name, filters = {}) => {
    const params = new URLSearchParams();
    if (filters.components) {
      filters.components.forEach(c => params.append('component', c));
    }
    const response = await api.get(`/api/metrics/${name}?${params.toString()}`);
    return response.data;
  },

  // Get all metrics summary
  getSummary: async (filters = {}) => {
    const params = new URLSearchParams();
    if (filters.components) {
      filters.components.forEach(c => params.append('component', c));
    }
    const response = await api.get(`/api/metrics/summary?${params.toString()}`);
    return response.data;
  },

  // Get collector status
  getStatus: async () => {
    const response = await api.get('/api/metrics/status');
    return response.data;
  },
};

// Error handling helper
export const handleAPIError = (error) => {
  if (error.response) {
    // Server responded with error status
    const message = error.response.data?.error || error.response.data?.message || 'Server error';
    return {
      message,
      status: error.response.status,
      data: error.response.data,
    };
  } else if (error.request) {
    // Request was made but no response received
    return {
      message: 'Network error - please check your connection',
      status: 0,
      data: null,
    };
  } else {
    // Something else happened
    return {
      message: error.message || 'An unexpected error occurred',
      status: 0,
      data: null,
    };
  }
};

export default api;
