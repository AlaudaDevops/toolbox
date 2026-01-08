import React, { useState } from 'react';
import Select from 'react-select';
import { RefreshCw, X, AlertCircle } from 'lucide-react';
import { useMetrics, MetricsProvider } from '../hooks/useMetrics';
import { useRoadmap } from '../hooks/useRoadmap';
import MetricCard, { METRIC_CONFIG } from './metrics/MetricCard';
import MetricBreakdown from './metrics/MetricBreakdown';
import './MetricsDashboard.css';

// Format relative time
const formatRelativeTime = (timestamp) => {
  if (!timestamp) return 'Never';

  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now - date;
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'Just now';
  if (diffMins === 1) return '1 minute ago';
  if (diffMins < 60) return `${diffMins} minutes ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours === 1) return '1 hour ago';
  if (diffHours < 24) return `${diffHours} hours ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays === 1) return 'Yesterday';
  return `${diffDays} days ago`;
};

// Status badge component
const StatusBadge = ({ status }) => {
  const statusConfig = {
    healthy: { label: 'Healthy', className: 'status-badge--healthy' },
    initializing: { label: 'Initializing', className: 'status-badge--initializing' },
    stale: { label: 'Stale', className: 'status-badge--stale' },
    disabled: { label: 'Disabled', className: 'status-badge--disabled' },
  };

  const config = statusConfig[status] || statusConfig.disabled;

  return (
    <span className={`status-badge ${config.className}`}>
      {config.label}
    </span>
  );
};

// Custom select styles
const selectStyles = {
  control: (provided, state) => ({
    ...provided,
    borderColor: state.isFocused ? '#667eea' : '#d1d5db',
    boxShadow: state.isFocused ? '0 0 0 3px rgba(102, 126, 234, 0.1)' : 'none',
    '&:hover': { borderColor: '#9ca3af' },
    minHeight: '38px',
    fontSize: '0.875rem',
  }),
  multiValue: (provided) => ({
    ...provided,
    backgroundColor: '#e0e7ff',
    borderRadius: '4px',
  }),
  multiValueLabel: (provided) => ({
    ...provided,
    color: '#4338ca',
    fontSize: '0.75rem',
  }),
  multiValueRemove: (provided) => ({
    ...provided,
    color: '#6366f1',
    '&:hover': {
      backgroundColor: '#c7d2fe',
      color: '#4338ca',
    },
  }),
  placeholder: (provided) => ({
    ...provided,
    color: '#9ca3af',
    fontSize: '0.875rem',
  }),
  menu: (provided) => ({
    ...provided,
    zIndex: 100,
  }),
};

// Metric order for display
const METRIC_ORDER = [
  'release_frequency',
  'lead_time_to_release',
  'cycle_time',
  'patch_ratio',
  'time_to_patch',
];

const MetricsDashboardContent = () => {
  const {
    metrics,
    status,
    isLoading,
    error,
    filters,
    updateFilters,
    clearFilters,
    refresh,
  } = useMetrics();

  const { roadmapData } = useRoadmap();

  const [expandedMetric, setExpandedMetric] = useState(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Get available components from roadmap data
  const components = roadmapData?.components || [];

  // Transform options for react-select
  const componentOptions = components.map(c => ({
    value: c.name || c,
    label: c.name || c,
  }));

  // Handle filter changes
  const handleComponentChange = (selected) => {
    updateFilters({
      components: selected ? selected.map(s => s.value) : [],
    });
  };

  // Handle clear filters
  const handleClearFilters = () => {
    clearFilters();
    setExpandedMetric(null);
  };

  // Handle refresh
  const handleRefresh = async () => {
    setIsRefreshing(true);
    await refresh();
    setIsRefreshing(false);
  };

  // Toggle metric expansion
  const toggleExpand = (metricName) => {
    setExpandedMetric(prev => prev === metricName ? null : metricName);
  };

  // Check if filters are active
  const hasActiveFilters = filters.components.length > 0;

  // Get selected filter values for react-select
  const selectedComponents = componentOptions.filter(opt =>
    filters.components.includes(opt.value)
  );

  // Transform metrics data for display
  const metricsData = metrics?.metrics || {};

  return (
    <div className="metrics-dashboard">
      {/* Header */}
      <header className="metrics-dashboard__header">
        <div className="metrics-dashboard__title-section">
          <h1 className="metrics-dashboard__title">DORA Metrics Dashboard</h1>
          {status && (
            <div className="metrics-dashboard__status">
              <StatusBadge status={status.status} />
              <span className="metrics-dashboard__last-updated">
                Last updated: {formatRelativeTime(status.last_collected)}
              </span>
              {status.releases_count !== undefined && (
                <span className="metrics-dashboard__counts">
                  {status.releases_count} releases, {status.epics_count} epics, {status.issues_count || 0} issues
                </span>
              )}
            </div>
          )}
        </div>
        <button
          className="btn btn-secondary metrics-dashboard__refresh"
          onClick={handleRefresh}
          disabled={isRefreshing || isLoading}
        >
          <RefreshCw size={16} className={isRefreshing ? 'spinning' : ''} />
          <span>Refresh</span>
        </button>
      </header>

      {/* Filters */}
      <div className="metrics-dashboard__filters">
        <div className="metrics-dashboard__filter-group">
          <label className="metrics-dashboard__filter-label">Component</label>
          <Select
            isMulti
            options={componentOptions}
            value={selectedComponents}
            onChange={handleComponentChange}
            placeholder="Filter by component..."
            styles={selectStyles}
            isClearable
            className="metrics-dashboard__select"
          />
        </div>

        {hasActiveFilters && (
          <button
            className="btn btn-secondary metrics-dashboard__clear-filters"
            onClick={handleClearFilters}
          >
            <X size={14} />
            <span>Clear Filters</span>
          </button>
        )}
      </div>

      {/* Error State */}
      {error && (
        <div className="metrics-dashboard__error">
          <AlertCircle size={20} />
          <span>{error}</span>
        </div>
      )}

      {/* Loading State */}
      {isLoading && !metrics && (
        <div className="metrics-dashboard__loading">
          <div className="loading-spinner" />
          <span>Loading metrics...</span>
        </div>
      )}

      {/* Metrics Grid */}
      {!isLoading && metrics && (
        <>
          <div className="metrics-dashboard__grid">
            {METRIC_ORDER.map(metricName => {
              const metricData = metricsData[metricName];
              if (!METRIC_CONFIG[metricName]) return null;

              return (
                <MetricCard
                  key={metricName}
                  metricName={metricName}
                  data={metricData}
                  expanded={expandedMetric === metricName}
                  onToggle={() => toggleExpand(metricName)}
                />
              );
            })}
          </div>

          {/* Expanded Breakdown */}
          {expandedMetric && (
            <MetricBreakdown metricName={expandedMetric} />
          )}
        </>
      )}

      {/* Empty State */}
      {!isLoading && !error && (!metrics || Object.keys(metricsData).length === 0) && (
        <div className="metrics-dashboard__empty">
          <AlertCircle size={40} />
          <h3>No Metrics Available</h3>
          <p>
            Metrics data is not yet available. This could be because:
          </p>
          <ul>
            <li>The metrics collector is still initializing</li>
            <li>No releases or epics match the current filters</li>
            <li>Metrics collection is disabled in the configuration</li>
          </ul>
        </div>
      )}
    </div>
  );
};

// Wrapper component with MetricsProvider
const MetricsDashboard = () => {
  return (
    <MetricsProvider>
      <MetricsDashboardContent />
    </MetricsProvider>
  );
};

export default MetricsDashboard;
