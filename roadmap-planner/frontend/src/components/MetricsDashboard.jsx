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

// Custom select styles — Atlas theme
const selectStyles = {
  control: (provided, state) => ({
    ...provided,
    borderRadius: 0,
    borderColor: state.isFocused ? 'var(--fg)' : 'var(--border)',
    boxShadow: state.isFocused ? 'inset 0 0 0 1px var(--fg)' : 'none',
    backgroundColor: 'var(--bg-elevated)',
    '&:hover': { borderColor: 'var(--fg-faint)' },
    minHeight: '40px',
    fontSize: '14px',
    fontFamily: 'var(--font-ui)',
  }),
  valueContainer: (provided) => ({ ...provided, padding: '0 0.5rem' }),
  multiValue: (provided) => ({
    ...provided,
    backgroundColor: 'var(--ink)',
    color: 'var(--paper)',
    borderRadius: 0,
  }),
  multiValueLabel: (provided) => ({
    ...provided,
    color: 'var(--paper)',
    fontSize: '11px',
    fontFamily: 'var(--font-mono)',
    letterSpacing: '0.04em',
    padding: '2px 6px',
  }),
  multiValueRemove: (provided) => ({
    ...provided,
    color: 'var(--paper)',
    '&:hover': { backgroundColor: 'var(--accent)', color: 'white' },
  }),
  input: (provided) => ({ ...provided, color: 'var(--fg)' }),
  placeholder: (provided) => ({ ...provided, color: 'var(--fg-faint)', fontSize: '13px' }),
  option: (provided, state) => ({
    ...provided,
    backgroundColor: state.isSelected ? 'var(--ink)' : state.isFocused ? 'var(--bg-sunken)' : 'var(--bg-elevated)',
    color: state.isSelected ? 'var(--paper)' : 'var(--fg)',
    fontSize: '13px',
    padding: '0.5rem 0.75rem',
    borderBottom: '1px solid var(--border)',
  }),
  menu: (provided) => ({
    ...provided,
    borderRadius: 0,
    border: '1px solid var(--fg-faint)',
    boxShadow: '-3px 3px 0 var(--ink)',
    backgroundColor: 'var(--bg-elevated)',
    zIndex: 100,
  }),
  indicatorSeparator: (provided) => ({ ...provided, backgroundColor: 'var(--border)' }),
  dropdownIndicator: (provided) => ({ ...provided, color: 'var(--fg-faint)' }),
  clearIndicator: (provided) => ({ ...provided, color: 'var(--fg-faint)' }),
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
    <div className="metrics-dashboard fade-in">
      <div className="metrics-dashboard__masthead">
        <div className="metrics-dashboard__title-section">
          <span className="metrics-dashboard__chapter mono">CHAPTER 02</span>
          <h1 className="metrics-dashboard__title">
            <span className="serif">A study in</span> DORA <span className="serif">— five gauges, told in numbers.</span>
          </h1>
          {status && (
            <div className="metrics-dashboard__status">
              <StatusBadge status={status.status} />
              <span className="metrics-dashboard__last-updated">
                <span className="serif">Updated</span> {formatRelativeTime(status.last_collected)}
              </span>
              {status.releases_count !== undefined && (
                <span className="metrics-dashboard__counts mono">
                  {status.releases_count} releases · {status.epics_count} epics · {status.issues_count || 0} issues
                </span>
              )}
            </div>
          )}
        </div>
        <button
          className="btn btn-sm metrics-dashboard__refresh"
          onClick={handleRefresh}
          disabled={isRefreshing || isLoading}
        >
          <RefreshCw size={13} strokeWidth={1.75} className={isRefreshing ? 'spinning' : ''} />
          <span>Refresh</span>
        </button>
      </div>

      {/* Filters */}
      <div className="metrics-dashboard__filters">
        <div className="metrics-dashboard__filter-group">
          <label className="metrics-dashboard__filter-label">Component</label>
          <Select
            isMulti
            options={componentOptions}
            value={selectedComponents}
            onChange={handleComponentChange}
            placeholder="Filter by component…"
            styles={selectStyles}
            isClearable
            className="metrics-dashboard__select"
          />
        </div>

        {hasActiveFilters && (
          <button
            className="btn btn-sm btn-ghost metrics-dashboard__clear-filters"
            onClick={handleClearFilters}
          >
            <X size={14} strokeWidth={1.75} />
            <span>Clear filters</span>
          </button>
        )}
      </div>

      {error && (
        <div className="metrics-dashboard__error">
          <AlertCircle size={18} strokeWidth={1.75} />
          <span>{error}</span>
        </div>
      )}

      {isLoading && !metrics && (
        <div className="metrics-dashboard__loading">
          <div className="atlas-spinner" />
          <span className="serif">Reading the dials…</span>
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
          <AlertCircle size={32} strokeWidth={1.5} />
          <h3 className="serif">No metrics yet</h3>
          <p>
            The collector hasn't reported anything yet. Likely culprits:
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
