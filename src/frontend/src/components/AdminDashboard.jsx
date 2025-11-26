import React, { useState, useEffect } from 'react';
import { libraryService, adminService, healthService } from '../services/apiService';

function AdminDashboard() {
  const [stats, setStats] = useState({});
  const [jobs, setJobs] = useState([]);
  const [health, setHealth] = useState({});
  const [capacity, setCapacity] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      // Fetch library stats
      const statsResponse = await libraryService.getStats();
      setStats(statsResponse.data.data || statsResponse.data || {});

      // Fetch recent jobs - using DLQ as proxy for recent jobs since actual endpoint might not exist yet
      const jobsResponse = await adminService.getDLQItems();
      // For now, we'll just use DLQ items as the jobs list, but in real implementation
      // there would be a dedicated endpoint for recent jobs
      setJobs(jobsResponse.data.data || jobsResponse.data || []);

      // Fetch system health
      try {
        const healthResponse = await healthService.getHealth();
        setHealth(healthResponse.data.data || healthResponse.data || {});
      } catch (healthError) {
        console.error('Error fetching health status:', healthError);
        setHealth({});
      }

      // Fetch capacity metrics
      try {
        const capacityResponse = await healthService.getCapacity();
        setCapacity(capacityResponse.data.data || capacityResponse.data || []);
      } catch (capacityError) {
        console.error('Error fetching capacity status:', capacityError);
        setCapacity([]);
      }
    } catch (error) {
      console.error('Error fetching dashboard data:', error);
      // Set default values in case of error
      setStats({});
      setJobs([]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Admin Dashboard</h1>
        <div className="text-center py-12 text-gray-600 dark:text-gray-400">Loading dashboard...</div>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Admin Dashboard</h1>

      {/* Health Status Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Overall Health</h3>
          <p className={`text-3xl font-bold ${
            health.status === 'ok' ? 'text-green-600' :
            health.status === 'degraded' ? 'text-yellow-600' : 
            health.status ? 'text-red-600' : 'text-gray-400'
          }`}>
            {health.status ? health.status.charAt(0).toUpperCase() + health.status.slice(1) : 'No Data'}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Database Status</h3>
          <p className={`text-3xl font-bold ${
            health.db?.status === 'ok' ? 'text-green-600' :
            health.db?.status === 'degraded' ? 'text-yellow-600' :
            health.db?.status ? 'text-red-600' : 'text-gray-400'
          }`}>
            {health.db?.status ? health.db.status.charAt(0).toUpperCase() + health.db.status.slice(1) : 'No Data'}
          </p>
          {health.db?.latency_ms !== undefined && (
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{health.db.latency_ms}ms latency</p>
          )}
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Redis Status</h3>
          <p className={`text-3xl font-bold ${
            health.redis?.status === 'ok' ? 'text-green-600' :
            health.redis?.status === 'degraded' ? 'text-yellow-600' :
            health.redis?.status ? 'text-red-600' : 'text-gray-400'
          }`}>
            {health.redis?.status ? health.redis.status.charAt(0).toUpperCase() + health.redis.status.slice(1) : 'No Data'}
          </p>
          {health.redis?.latency_ms !== undefined && (
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{health.redis.latency_ms}ms latency</p>
          )}
        </div>
      </div>

      {/* Library Stats Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Total Artists</h3>
          <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{stats.total_artists || stats.totalArtists || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Total Albums</h3>
          <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{stats.total_albums || stats.totalAlbums || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Total Tracks</h3>
          <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{stats.total_tracks || stats.totalTracks || stats.total_songs || stats.totalSongs || 0}</p>
        </div>
      </div>

      {/* Capacity Status Row */}
      {capacity.length > 0 && (
        <div className="mb-6">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">Storage Capacity</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {capacity.map((item, index) => (
              <div key={index} className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
                <h3 className="font-semibold text-gray-900 dark:text-gray-100 truncate mb-3">{item.Path || item.path || `Library ${index + 1}`}</h3>
                <div className="mt-2">
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
                    <div
                      className={`h-2.5 rounded-full ${
                        (item.UsedPercent || item.usedPercent || 0) > 90 ? 'bg-red-600' :
                        (item.UsedPercent || item.usedPercent || 0) > 75 ? 'bg-yellow-500' : 'bg-green-500'
                      }`}
                      style={{ width: `${Math.min(100, item.UsedPercent || item.usedPercent || 0)}%` }}
                    ></div>
                  </div>
                  <p className="text-sm mt-1 text-gray-700 dark:text-gray-300">
                    {item.AvailableSpace !== undefined && item.TotalSpace !== undefined ?
                      `${Math.round(((item.TotalSpace - item.AvailableSpace) / item.TotalSpace) * 100)}% Used` :
                      `${(item.UsedPercent || item.usedPercent || 0).toFixed(1)}%`}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Pipeline stats row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Inbound Files</h3>
          <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{stats.inbound_count || stats.inboundCount || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">Staging Files</h3>
          <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{stats.staging_count || stats.stagingCount || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300 text-sm uppercase tracking-wide mb-2">System Status</h3>
          <p className="text-3xl font-bold text-green-600">Operational</p>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">Recent Activity</h2>

        {jobs.length > 0 ? (
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Job</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Started</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {jobs.slice(0, 5).map((job, index) => (
                <tr key={job.id || index}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                    {job.type || job.Type || 'Unknown Job'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full
                      ${job.status === 'completed' || job.Status === 'completed' ? 'bg-green-100 text-green-800' :
                        job.status === 'failed' || job.Status === 'failed' ? 'bg-red-100 text-red-800' :
                        'bg-yellow-100 text-yellow-800'}`}>
                      {job.status || job.Status || 'Unknown'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                    {job.created_at || job.CreatedAt ? new Date(job.created_at || job.CreatedAt).toLocaleString() : 'N/A'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="text-gray-500 dark:text-gray-400 text-center py-4">
            No recent activity to display.
          </div>
        )}
      </div>
    </div>
  );
}

export default AdminDashboard;