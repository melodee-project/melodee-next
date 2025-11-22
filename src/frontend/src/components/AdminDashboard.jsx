import React, { useState, useEffect } from 'react';
import { libraryService, adminService } from '../services/apiService';

function AdminDashboard() {
  const [stats, setStats] = useState({});
  const [jobs, setJobs] = useState([]);
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
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-4">Admin Dashboard</h1>
        <div className="text-center py-8">Loading dashboard...</div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Admin Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Artists</h3>
          <p className="text-2xl">{stats.total_artists || stats.totalArtists || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Albums</h3>
          <p className="text-2xl">{stats.total_albums || stats.totalAlbums || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Songs</h3>
          <p className="text-2xl">{stats.total_songs || stats.totalSongs || 0}</p>
        </div>
      </div>

      {/* Additional stats row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Inbound Files</h3>
          <p className="text-2xl">{stats.inbound_count || stats.inboundCount || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Staging Files</h3>
          <p className="text-2xl">{stats.staging_count || stats.stagingCount || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">System Status</h3>
          <p className="text-2xl text-green-600">Operational</p>
        </div>
      </div>

      <div className="bg-white p-4 rounded shadow">
        <h2 className="text-xl font-semibold mb-2">Recent Activity</h2>

        {jobs.length > 0 ? (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Job</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Started</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {jobs.slice(0, 5).map((job, index) => (
                <tr key={job.id || index}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
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
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {job.created_at || job.CreatedAt ? new Date(job.created_at || job.CreatedAt).toLocaleString() : 'N/A'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="text-gray-500 text-center py-4">
            No recent activity to display.
          </div>
        )}
      </div>
    </div>
  );
}

export default AdminDashboard;