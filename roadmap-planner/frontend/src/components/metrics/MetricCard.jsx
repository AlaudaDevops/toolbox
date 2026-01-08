import React from 'react';
import { ChevronDown, ChevronUp, TrendingUp, Clock, Repeat, AlertTriangle, Zap } from 'lucide-react';
import './MetricCard.css';

// Metric configuration with thresholds and icons
const METRIC_CONFIG = {
  release_frequency: {
    displayName: 'Release Frequency',
    icon: TrendingUp,
    unit: 'releases/month',
    thresholds: { excellent: 2, good: 1, warning: 0.5 },
    higherIsBetter: true,
    description: 'How often releases are published per month',
  },
  lead_time_to_release: {
    displayName: 'Lead Time to Release',
    icon: Clock,
    unit: 'days (P50)',
    thresholds: { excellent: 30, good: 60, warning: 90 },
    higherIsBetter: false,
    description: 'Time from epic creation to release',
  },
  cycle_time: {
    displayName: 'Cycle Time',
    icon: Repeat,
    unit: 'days (P50)',
    thresholds: { excellent: 14, good: 30, warning: 45 },
    higherIsBetter: false,
    description: 'Time from work started to completed',
  },
  patch_ratio: {
    displayName: 'Patch Ratio',
    icon: AlertTriangle,
    unit: '',
    thresholds: { excellent: 0.2, good: 0.4, warning: 0.6 },
    higherIsBetter: false,
    description: 'Ratio of patch releases to total releases',
    statusLabels: {
      excellent: 'Excellent',
      good: 'Good',
      warning: 'Concerning',
      critical: 'Critical',
    },
  },
  time_to_patch: {
    displayName: 'Time to Patch',
    icon: Zap,
    unit: 'days (P50)',
    thresholds: { excellent: 7, good: 14, warning: 21 },
    higherIsBetter: false,
    description: 'Time to release bug/security fixes',
  },
};

// Determine status based on value and thresholds
const getStatus = (value, config) => {
  if (value === null || value === undefined || isNaN(value)) return 'unknown';

  const { thresholds, higherIsBetter } = config;

  if (higherIsBetter) {
    if (value >= thresholds.excellent) return 'excellent';
    if (value >= thresholds.good) return 'good';
    if (value >= thresholds.warning) return 'warning';
    return 'critical';
  } else {
    if (value <= thresholds.excellent) return 'excellent';
    if (value <= thresholds.good) return 'good';
    if (value <= thresholds.warning) return 'warning';
    return 'critical';
  }
};

// Format value for display
const formatValue = (value, metricName) => {
  if (value === null || value === undefined) return 'N/A';
  if (isNaN(value)) return 'N/A';

  if (metricName === 'patch_ratio') {
    return value.toFixed(2);
  }

  // For days-based metrics, show whole numbers
  if (metricName.includes('time') || metricName.includes('lead') || metricName.includes('cycle')) {
    return Math.round(value);
  }

  // For release frequency, show one decimal
  return value.toFixed(1);
};

const MetricCard = ({ metricName, data, expanded, onToggle }) => {
  const config = METRIC_CONFIG[metricName] || {
    displayName: metricName,
    icon: TrendingUp,
    unit: '',
    thresholds: { excellent: 0, good: 0, warning: 0 },
    higherIsBetter: true,
    description: '',
  };

  const Icon = config.icon;
  const value = data?.value ?? null;
  const status = getStatus(value, config);
  const formattedValue = formatValue(value, metricName);

  // Get status label for patch ratio
  const statusLabel = config.statusLabels?.[status] || '';

  return (
    <div className={`metric-card metric-card--${status}`}>
      <div className="metric-card__header">
        <div className="metric-card__icon">
          <Icon size={20} />
        </div>
        <h3 className="metric-card__title">{config.displayName}</h3>
      </div>

      <div className="metric-card__body">
        <div className="metric-card__value-container">
          <span className="metric-card__value">{formattedValue}</span>
          {config.unit && <span className="metric-card__unit">{config.unit}</span>}
        </div>

        {statusLabel && (
          <span className={`metric-card__status-label metric-card__status-label--${status}`}>
            {statusLabel}
          </span>
        )}

        <p className="metric-card__description">{config.description}</p>
      </div>

      <button
        className="metric-card__toggle"
        onClick={onToggle}
        aria-expanded={expanded}
        aria-label={expanded ? 'Collapse details' : 'Expand details'}
      >
        {expanded ? (
          <>
            <span>Hide Details</span>
            <ChevronUp size={16} />
          </>
        ) : (
          <>
            <span>Show Details</span>
            <ChevronDown size={16} />
          </>
        )}
      </button>
    </div>
  );
};

export default MetricCard;
export { METRIC_CONFIG, getStatus, formatValue };
