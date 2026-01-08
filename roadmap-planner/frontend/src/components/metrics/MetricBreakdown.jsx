import React, { useEffect, useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { useMetrics } from '../../hooks/useMetrics';
import { METRIC_CONFIG, getStatus } from './MetricCard';
import './MetricBreakdown.css';

// Color palette for charts
const CHART_COLORS = {
  excellent: '#10b981',
  good: '#3b82f6',
  warning: '#f59e0b',
  critical: '#ef4444',
  default: '#64748b',
};

// Get color based on value and metric config
const getBarColor = (value, metricName) => {
  const config = METRIC_CONFIG[metricName];
  if (!config) return CHART_COLORS.default;

  const status = getStatus(value, config);
  return CHART_COLORS[status] || CHART_COLORS.default;
};

// Format breakdown data for table display
const formatBreakdownData = (results, metricName) => {
  if (!results || !Array.isArray(results)) return [];

  return results.map(result => {
    const component = result.labels?.component || 'Unknown';
    const metadata = result.metadata || {};

    // Common fields
    const base = {
      component,
      value: result.value,
    };

    // Add metric-specific fields
    switch (metricName) {
      case 'release_frequency':
        return {
          ...base,
          totalReleases: metadata.total_releases || 0,
          majorReleases: metadata.major_releases || 0,
          minorReleases: metadata.minor_releases || 0,
          patchReleases: metadata.patch_releases || 0,
        };
      case 'patch_ratio':
        return {
          ...base,
          totalReleases: metadata.total_releases || 0,
          patchCount: metadata.patch_count || 0,
        };
      case 'lead_time_to_release':
      case 'cycle_time':
      case 'time_to_patch':
        return {
          ...base,
          min: metadata.min || 0,
          max: metadata.max || 0,
          count: metadata.count || 0,
        };
      default:
        return base;
    }
  });
};

// Table columns per metric type
const TABLE_COLUMNS = {
  release_frequency: [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'Rate/Month', format: (v) => v.toFixed(2) },
    { key: 'totalReleases', label: 'Total' },
    { key: 'majorReleases', label: 'Major' },
    { key: 'minorReleases', label: 'Minor' },
    { key: 'patchReleases', label: 'Patch' },
  ],
  lead_time_to_release: [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'P50 (days)', format: (v) => Math.round(v) },
    { key: 'min', label: 'Min', format: (v) => Math.round(v) },
    { key: 'max', label: 'Max', format: (v) => Math.round(v) },
    { key: 'count', label: 'Epics' },
  ],
  cycle_time: [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'P50 (days)', format: (v) => Math.round(v) },
    { key: 'min', label: 'Min', format: (v) => Math.round(v) },
    { key: 'max', label: 'Max', format: (v) => Math.round(v) },
    { key: 'count', label: 'Epics' },
  ],
  patch_ratio: [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'Ratio', format: (v) => v.toFixed(2) },
    { key: 'totalReleases', label: 'Total' },
    { key: 'patchCount', label: 'Patches' },
  ],
  time_to_patch: [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'P50 (days)', format: (v) => Math.round(v) },
    { key: 'min', label: 'Min', format: (v) => Math.round(v) },
    { key: 'max', label: 'Max', format: (v) => Math.round(v) },
    { key: 'count', label: 'Issues' },
  ],
};

const MetricBreakdown = ({ metricName }) => {
  const { loadMetric } = useMetrics();
  const [data, setData] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      setError(null);

      const result = await loadMetric(metricName);
      if (result.success) {
        // Handle both single result and array of results
        const results = result.data.results || [result.data];
        setData(formatBreakdownData(results, metricName));
      } else {
        setError(result.error);
      }

      setIsLoading(false);
    };

    fetchData();
  }, [metricName, loadMetric]);

  if (isLoading) {
    return (
      <div className="metric-breakdown metric-breakdown--loading">
        <div className="loading-spinner" />
        <span>Loading breakdown data...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="metric-breakdown metric-breakdown--error">
        <p>Failed to load breakdown: {error}</p>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="metric-breakdown metric-breakdown--empty">
        <p>No breakdown data available</p>
      </div>
    );
  }

  const config = METRIC_CONFIG[metricName];
  const columns = TABLE_COLUMNS[metricName] || [
    { key: 'component', label: 'Component' },
    { key: 'value', label: 'Value' },
  ];

  // Prepare chart data - sort by value descending
  const chartData = [...data]
    .sort((a, b) => b.value - a.value)
    .slice(0, 10) // Show top 10
    .map(item => ({
      name: item.component.length > 15
        ? item.component.substring(0, 15) + '...'
        : item.component,
      fullName: item.component,
      value: item.value,
    }));

  return (
    <div className="metric-breakdown">
      <div className="metric-breakdown__header">
        <h4 className="metric-breakdown__title">
          {config?.displayName || metricName} - Breakdown by Component
        </h4>
      </div>

      <div className="metric-breakdown__content">
        {/* Chart */}
        <div className="metric-breakdown__chart">
          <ResponsiveContainer width="100%" height={Math.max(200, chartData.length * 35)}>
            <BarChart
              data={chartData}
              layout="vertical"
              margin={{ top: 5, right: 30, left: 80, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={true} vertical={false} />
              <XAxis type="number" tick={{ fontSize: 12 }} />
              <YAxis
                type="category"
                dataKey="name"
                tick={{ fontSize: 12 }}
                width={75}
              />
              <Tooltip
                formatter={(value) => [
                  config?.unit?.includes('days')
                    ? `${Math.round(value)} days`
                    : value.toFixed(2),
                  config?.displayName || metricName,
                ]}
                labelFormatter={(label, payload) =>
                  payload?.[0]?.payload?.fullName || label
                }
              />
              <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                {chartData.map((entry, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={getBarColor(entry.value, metricName)}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Table */}
        <div className="metric-breakdown__table-container">
          <table className="metric-breakdown__table">
            <thead>
              <tr>
                {columns.map(col => (
                  <th key={col.key}>{col.label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((row, index) => (
                <tr key={index}>
                  {columns.map(col => (
                    <td key={col.key}>
                      {col.format
                        ? col.format(row[col.key])
                        : row[col.key] ?? '-'}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default MetricBreakdown;
