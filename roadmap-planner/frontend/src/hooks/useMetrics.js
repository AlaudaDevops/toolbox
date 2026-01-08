import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { metricsAPI, handleAPIError } from '../services/api';
import toast from 'react-hot-toast';

const MetricsContext = createContext();

export const useMetrics = () => {
  const context = useContext(MetricsContext);
  if (!context) {
    throw new Error('useMetrics must be used within a MetricsProvider');
  }
  return context;
};

export const MetricsProvider = ({ children }) => {
  const [metrics, setMetrics] = useState(null);
  const [status, setStatus] = useState(null);
  const [availableMetrics, setAvailableMetrics] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({ components: [] });

  // Load metrics summary with current filters
  const loadMetrics = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      const [summaryData, statusData] = await Promise.all([
        metricsAPI.getSummary(filters),
        metricsAPI.getStatus(),
      ]);

      setMetrics(summaryData);
      setStatus(statusData);
    } catch (err) {
      const errorInfo = handleAPIError(err);
      console.error('Failed to load metrics:', errorInfo);
      setError(errorInfo.message);
      toast.error(`Failed to load metrics: ${errorInfo.message}`);
    } finally {
      setIsLoading(false);
    }
  }, [filters]);

  // Load available metrics list
  const loadAvailableMetrics = useCallback(async () => {
    try {
      const data = await metricsAPI.listMetrics();
      setAvailableMetrics(data.metrics || []);
    } catch (err) {
      console.error('Failed to load available metrics:', err);
    }
  }, []);

  // Load a specific metric with filters
  const loadMetric = useCallback(async (name) => {
    try {
      const data = await metricsAPI.getMetric(name, filters);
      return { success: true, data };
    } catch (err) {
      const errorInfo = handleAPIError(err);
      return { success: false, error: errorInfo.message };
    }
  }, [filters]);

  // Update filters
  const updateFilters = useCallback((newFilters) => {
    setFilters(prev => ({
      ...prev,
      ...newFilters,
    }));
  }, []);

  // Clear all filters
  const clearFilters = useCallback(() => {
    setFilters({ components: [] });
  }, []);

  // Refresh metrics data
  const refresh = useCallback(async () => {
    await loadMetrics();
    toast.success('Metrics refreshed');
  }, [loadMetrics]);

  // Load metrics on mount and when filters change
  useEffect(() => {
    loadMetrics();
  }, [loadMetrics]);

  // Load available metrics list on mount
  useEffect(() => {
    loadAvailableMetrics();
  }, [loadAvailableMetrics]);

  const value = {
    metrics,
    status,
    availableMetrics,
    isLoading,
    error,
    filters,
    updateFilters,
    clearFilters,
    refresh,
    loadMetric,
  };

  return (
    <MetricsContext.Provider value={value}>
      {children}
    </MetricsContext.Provider>
  );
};
