import React, { useState, useEffect, useRef } from 'react';
import { adminService } from '../services/apiService';

function LogViewer() {
  const [logs, setLogs] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [tailMode, setTailMode] = useState(false);
  const [filters, setFilters] = useState({
    level: '',
    module: '',
    search: '',
    start_time: '',
    end_time: '',
    limit: 100,
  });
  const [pagination, setPagination] = useState({ page: 1, pageSize: 100 });
  const logsEndRef = useRef(null);
  const refreshIntervalRef = useRef(null);

  useEffect(() => {
    fetchLogs();
    fetchStats();
  }, [pagination.page, filters]);

  useEffect(() => {
    if (autoRefresh) {
      refreshIntervalRef.current = setInterval(() => {
        fetchLogs();
        fetchStats();
      }, 5000); // Refresh every 5 seconds
    } else if (refreshIntervalRef.current) {
      clearInterval(refreshIntervalRef.current);
    }

    return () => {
      if (refreshIntervalRef.current) {
        clearInterval(refreshIntervalRef.current);
      }
    };
  }, [autoRefresh, pagination.page, filters]);

  useEffect(() => {
    if (tailMode && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, tailMode]);

  const fetchLogs = async () => {
    try {
      const params = new URLSearchParams();
      params.append('page', pagination.page);
      params.append('page_size', pagination.pageSize);
      
      if (filters.level) params.append('level', filters.level);
      if (filters.module) params.append('module', filters.module);
      if (filters.search) params.append('search', filters.search);
      if (filters.start_time) params.append('start_time', filters.start_time);
      if (filters.end_time) params.append('end_time', filters.end_time);

      const response = await adminService.getLogs(params.toString());
      setLogs(response.data.data || []);
      setPagination(prev => ({
        ...prev,
        ...response.data.pagination
      }));
    } catch (error) {
      console.error('Error fetching logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await adminService.getLogStats();
      setStats(response.data);
    } catch (error) {
      console.error('Error fetching log stats:', error);
    }
  };

  const handleFilterChange = (key, value) => {
    setFilters(prev => ({ ...prev, [key]: value }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handleDownload = async () => {
    try {
      const params = new URLSearchParams();
      if (filters.level) params.append('level', filters.level);
      if (filters.module) params.append('module', filters.module);
      if (filters.search) params.append('search', filters.search);
      if (filters.start_time) params.append('start_time', filters.start_time);
      if (filters.end_time) params.append('end_time', filters.end_time);

      const response = await adminService.downloadLogs(params.toString());
      const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `melodee-logs-${new Date().toISOString()}.json`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Error downloading logs:', error);
      alert('Error downloading logs: ' + (error.response?.data?.error || error.message));
    }
  };

  const getLevelColor = (level) => {
    const colors = {
      debug: 'text-gray-500 dark:text-gray-400',
      info: 'text-blue-600 dark:text-blue-400',
      warn: 'text-yellow-600 dark:text-yellow-400',
      error: 'text-red-600 dark:text-red-400',
      fatal: 'text-red-800 dark:text-red-300 font-bold',
    };
    return colors[level?.toLowerCase()] || 'text-gray-600 dark:text-gray-400';
  };

  const getLevelBadgeColor = (level) => {
    const colors = {
      debug: 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300',
      info: 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200',
      warn: 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200',
      error: 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
      fatal: 'bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100',
    };
    return colors[level?.toLowerCase()] || 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300';
  };

  if (loading) {
    return <div className="p-4 text-gray-900 dark:text-gray-100">Loading logs...</div>;
  }

  return (
    <div className="p-4">
      {/* Header with stats */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4 text-gray-900 dark:text-gray-100">System Logs</h1>
        
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-4">
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">Errors (24h)</h3>
              <p className="text-2xl font-bold text-red-600 dark:text-red-400">{stats.error_count}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">Warnings (24h)</h3>
              <p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">{stats.warn_count}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">Total Logs</h3>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{pagination.total || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">Auto Refresh</h3>
              <button
                onClick={() => setAutoRefresh(!autoRefresh)}
                className={`mt-1 px-3 py-1 rounded ${
                  autoRefresh
                    ? 'bg-green-500 text-white'
                    : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'
                }`}
              >
                {autoRefresh ? 'ON' : 'OFF'}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 p-4 rounded shadow mb-4 border border-gray-200 dark:border-gray-700">
        <div className="grid grid-cols-1 md:grid-cols-6 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Level</label>
            <select
              value={filters.level}
              onChange={(e) => handleFilterChange('level', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            >
              <option value="">All</option>
              <option value="debug">Debug</option>
              <option value="info">Info</option>
              <option value="warn">Warning</option>
              <option value="error">Error</option>
              <option value="fatal">Fatal</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Module</label>
            <input
              type="text"
              value={filters.module}
              onChange={(e) => handleFilterChange('module', e.target.value)}
              placeholder="e.g., media, auth"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>

          <div className="md:col-span-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Search</label>
            <input
              type="text"
              value={filters.search}
              onChange={(e) => handleFilterChange('search', e.target.value)}
              placeholder="Search in messages..."
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>

          <div className="md:col-span-2 flex items-end gap-2">
            <button
              onClick={fetchLogs}
              className="px-4 py-2 bg-blue-500 dark:bg-blue-600 text-white rounded hover:bg-blue-600 dark:hover:bg-blue-700"
            >
              Refresh
            </button>
            <button
              onClick={handleDownload}
              className="px-4 py-2 bg-green-500 dark:bg-green-600 text-white rounded hover:bg-green-600 dark:hover:bg-green-700"
            >
              Download
            </button>
            <button
              onClick={() => setTailMode(!tailMode)}
              className={`px-4 py-2 rounded ${
                tailMode
                  ? 'bg-purple-500 text-white hover:bg-purple-600'
                  : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-400'
              }`}
            >
              {tailMode ? 'Tail ON' : 'Tail OFF'}
            </button>
          </div>
        </div>
      </div>

      {/* Logs table */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
        <div className="overflow-x-auto max-h-[600px] overflow-y-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700 sticky top-0">
              <tr>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-40">
                  Timestamp
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-20">
                  Level
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24">
                  Module
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Message
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-32">
                  Details
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {logs.map((log) => (
                <tr key={log.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="px-3 py-2 whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">
                    {new Date(log.timestamp).toLocaleString()}
                  </td>
                  <td className="px-3 py-2 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-medium rounded ${getLevelBadgeColor(log.level)}`}>
                      {log.level?.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-3 py-2 whitespace-nowrap text-xs text-gray-600 dark:text-gray-400">
                    {log.module || '-'}
                  </td>
                  <td className="px-3 py-2 text-sm">
                    <div className={getLevelColor(log.level)}>
                      {log.message}
                    </div>
                    {log.error && (
                      <div className="text-xs text-red-600 dark:text-red-400 mt-1">
                        Error: {log.error}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 text-xs text-gray-500 dark:text-gray-400">
                    {log.library_id && <div>Library: {log.library_id}</div>}
                    {log.user_id && <div>User: {log.user_id}</div>}
                    {log.job_type && <div>Job: {log.job_type}</div>}
                    {log.duration_ms && <div>{log.duration_ms}ms</div>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div ref={logsEndRef} />
        </div>

        {logs.length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No logs found matching the current filters.
          </div>
        )}
      </div>

      {/* Pagination */}
      {pagination.total_pages > 1 && (
        <div className="mt-4 flex justify-between items-center">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Showing {((pagination.page - 1) * pagination.page_size) + 1} to{' '}
            {Math.min(pagination.page * pagination.page_size, pagination.total)} of{' '}
            {pagination.total} logs
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setPagination(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }))}
              disabled={pagination.page === 1}
              className="px-4 py-2 bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded disabled:opacity-50"
            >
              Previous
            </button>
            <span className="px-4 py-2 text-gray-700 dark:text-gray-300">
              Page {pagination.page} of {pagination.total_pages}
            </span>
            <button
              onClick={() => setPagination(prev => ({ ...prev, page: Math.min(pagination.total_pages, prev.page + 1) }))}
              disabled={pagination.page === pagination.total_pages}
              className="px-4 py-2 bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default LogViewer;
