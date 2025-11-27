import React, { useState, useEffect } from 'react';
import { adminService } from '../services/apiService';

function JobMonitor() {
  const [activeTab, setActiveTab] = useState('overview');
  const [jobs, setJobs] = useState([]);
  const [dlqItems, setDlqItems] = useState([]);
  const [stats, setStats] = useState([]);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [selectedJobs, setSelectedJobs] = useState([]);
  const [message, setMessage] = useState('');
  const [initialLoad, setInitialLoad] = useState(true);

  useEffect(() => {
    fetchData();
    
    if (autoRefresh) {
      const interval = setInterval(() => fetchData(false), 5000); // Don't show loading on auto-refresh
      return () => clearInterval(interval);
    }
  }, [activeTab, autoRefresh]);

  const fetchData = async (showLoading = true) => {
    try {
      if (showLoading) {
        setLoading(true);
      }
      
      if (activeTab === 'dlq') {
        const response = await adminService.getDLQItems();
        setDlqItems(response.data.data || response.data || []);
      } else if (activeTab === 'overview') {
        // Fetch all data for overview
        const [activeResp, pendingResp, scheduledResp, dlqResp, statsResp] = await Promise.all([
          adminService.getActiveJobs().catch((err) => { console.error('Failed to fetch active jobs:', err); return { data: { data: [] } }; }),
          adminService.getPendingJobs().catch((err) => { console.error('Failed to fetch pending jobs:', err); return { data: { data: [] } }; }),
          adminService.getScheduledJobs().catch((err) => { console.error('Failed to fetch scheduled jobs:', err); return { data: { data: [] } }; }),
          adminService.getDLQItems().catch((err) => { console.error('Failed to fetch DLQ items:', err); return { data: { data: [] } }; }),
          adminService.getJobStats().catch((err) => { console.error('Failed to fetch job stats:', err); return { data: { data: [] } }; })
        ]);
        
        console.log('Job stats response:', statsResp);
        setStats(statsResp.data.data || statsResp.data || []);
      } else {
        // Fetch active, pending, or scheduled jobs
        const endpoint = activeTab === 'active' ? 'getActiveJobs' : 
                        activeTab === 'pending' ? 'getPendingJobs' : 
                        'getScheduledJobs';
        const response = await adminService[endpoint]();
        setJobs(response.data.data || []);
      }
      
      // Always fetch stats for tab counts
      if (activeTab !== 'overview') {
        const statsResp = await adminService.getJobStats();
        console.log('Job stats response:', statsResp);
        setStats(statsResp.data.data || statsResp.data || []);
      }
      
      if (initialLoad) {
        setInitialLoad(false);
      }
    } catch (error) {
      console.error('Error fetching job data:', error);
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  };

  const handleCancelJob = async (jobId) => {
    if (!confirm('Are you sure you want to cancel this job?')) return;
    
    try {
      await adminService.cancelJob(jobId);
      fetchData();
    } catch (error) {
      alert('Error canceling job: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleRequeueDLQ = async () => {
    if (selectedJobs.length === 0) return;
    
    try {
      await adminService.requeueDLQItems(selectedJobs);
      setSelectedJobs([]);
      fetchData();
    } catch (error) {
      alert('Error requeuing items: ' + (error.response?.data?.error || error.message));
    }
  };

  const handlePurgeDLQ = async () => {
    if (selectedJobs.length === 0) return;
    if (!confirm(`Are you sure you want to permanently delete ${selectedJobs.length} failed job(s)?`)) return;
    
    try {
      await adminService.purgeDLQItems(selectedJobs);
      setSelectedJobs([]);
      fetchData();
    } catch (error) {
      alert('Error purging items: ' + (error.response?.data?.error || error.message));
    }
  };

  const toggleJobSelection = (jobId) => {
    setSelectedJobs(prev =>
      prev.includes(jobId) ? prev.filter(id => id !== jobId) : [...prev, jobId]
    );
  };

  const getStateColor = (state) => {
    const colors = {
      active: 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200',
      pending: 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200',
      scheduled: 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200',
      failed: 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
      completed: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
    };
    return colors[state] || 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300';
  };

  const formatDate = (dateString) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString();
  };

  // Calculate counts for tabs
  const getTotalCount = (type) => {
    if (!stats || stats.length === 0) return 0;
    return stats.reduce((sum, stat) => sum + (stat[type] || 0), 0);
  };

  const activeCount = getTotalCount('active');
  const pendingCount = getTotalCount('pending');
  const scheduledCount = getTotalCount('scheduled');
  const dlqCount = getTotalCount('archived');

  const renderOverview = () => (
    <>
      <div className="mb-6 p-4 bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Job Queue Controls</h2>
          <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="rounded"
            />
            Auto-refresh (5s)
          </label>
        </div>
        
        {message && (
          <div className="mb-4 p-3 bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 rounded border border-blue-200 dark:border-blue-800">
            {message}
          </div>
        )}
        
        {stats.length === 0 && !loading && (
          <div className="mb-4 p-3 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded border border-yellow-200 dark:border-yellow-800">
            <strong>No job queues found.</strong> Job queues will appear once jobs are enqueued. 
            The worker is connected, but no jobs have been submitted yet.
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300">Active Jobs</h3>
          <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">{activeCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Currently processing</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300">Pending Jobs</h3>
          <p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">{pendingCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Waiting in queue</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300">Scheduled Jobs</h3>
          <p className="text-2xl font-bold text-purple-600 dark:text-purple-400">{scheduledCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Scheduled for later</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-700 dark:text-gray-300">Failed Jobs</h3>
          <p className="text-2xl font-bold text-red-600 dark:text-red-400">{dlqCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">In dead letter queue</p>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-gray-100">Queue Statistics</h2>
        {renderStats()}
      </div>
    </>
  );

  const renderStats = () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {stats.map((stat, idx) => (
        <div key={idx} className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
            {stat.queue} Queue
            {stat.paused && <span className="ml-2 text-xs text-red-600 dark:text-red-400">(Paused)</span>}
          </h3>
          <div className="space-y-2">
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Active:</span>
              <span className="font-semibold text-blue-600 dark:text-blue-400">{stat.active}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Pending:</span>
              <span className="font-semibold text-yellow-600 dark:text-yellow-400">{stat.pending}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Scheduled:</span>
              <span className="font-semibold text-purple-600 dark:text-purple-400">{stat.scheduled}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Retry:</span>
              <span className="font-semibold text-orange-600 dark:text-orange-400">{stat.retry}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Failed:</span>
              <span className="font-semibold text-red-600 dark:text-red-400">{stat.archived}</span>
            </div>
            <div className="flex justify-between border-t border-gray-200 dark:border-gray-700 pt-2 mt-2">
              <span className="text-gray-600 dark:text-gray-400">Total Size:</span>
              <span className="font-bold text-gray-900 dark:text-gray-100">{stat.size}</span>
            </div>
          </div>
        </div>
      ))}
      {stats.length === 0 && (
        <div className="col-span-3 text-center py-8 text-gray-500 dark:text-gray-400">
          No queue statistics available
        </div>
      )}
    </div>
  );

  const renderJobsTable = () => (
    <>
      {activeTab === 'dlq' && dlqItems.length > 0 && (
        <div className="mb-4 flex gap-2">
          <button
            onClick={handleRequeueDLQ}
            disabled={selectedJobs.length === 0}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Requeue Selected ({selectedJobs.length})
          </button>
          <button
            onClick={handlePurgeDLQ}
            disabled={selectedJobs.length === 0}
            className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Purge Selected ({selectedJobs.length})
          </button>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                {activeTab === 'dlq' && (
                  <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase w-12">
                    <input
                      type="checkbox"
                      checked={selectedJobs.length === (activeTab === 'dlq' ? dlqItems : jobs).length && (activeTab === 'dlq' ? dlqItems : jobs).length > 0}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedJobs((activeTab === 'dlq' ? dlqItems : jobs).map(j => j.id || j.ID));
                        } else {
                          setSelectedJobs([]);
                        }
                      }}
                      className="rounded"
                    />
                  </th>
                )}
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  ID
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  Queue
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  Type
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  State
                </th>
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  Retry
                </th>
                {activeTab === 'dlq' && (
                  <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                    Reason
                  </th>
                )}
                {activeTab !== 'dlq' && activeTab !== 'active' && (
                  <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                    Next Run
                  </th>
                )}
                <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {(activeTab === 'dlq' ? dlqItems : jobs).map((job) => (
                <tr key={job.id || job.ID} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  {activeTab === 'dlq' && (
                    <td className="px-3 py-2">
                      <input
                        type="checkbox"
                        checked={selectedJobs.includes(job.id || job.ID)}
                        onChange={() => toggleJobSelection(job.id || job.ID)}
                        className="rounded"
                      />
                    </td>
                  )}
                  <td className="px-3 py-2 text-xs font-mono text-gray-600 dark:text-gray-400">
                    {(job.id || job.ID)?.substring(0, 8)}...
                  </td>
                  <td className="px-3 py-2 text-sm text-gray-900 dark:text-gray-100">
                    {job.queue || job.Queue}
                  </td>
                  <td className="px-3 py-2 text-sm text-gray-900 dark:text-gray-100">
                    {job.type || job.Type}
                  </td>
                  <td className="px-3 py-2">
                    <span className={`px-2 py-1 text-xs font-medium rounded ${getStateColor(job.state || 'pending')}`}>
                      {(job.state || 'pending').toUpperCase()}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
                    {job.retried || job.RetryCount || 0} / {job.max_retry || job.MaxRetry || 3}
                  </td>
                  {activeTab === 'dlq' && (
                    <td className="px-3 py-2 text-sm text-red-600 dark:text-red-400">
                      {job.reason || job.Reason || job.last_error || '-'}
                    </td>
                  )}
                  {activeTab !== 'dlq' && activeTab !== 'active' && (
                    <td className="px-3 py-2 text-xs text-gray-600 dark:text-gray-400">
                      {formatDate(job.next_process_at)}
                    </td>
                  )}
                  <td className="px-3 py-2">
                    {activeTab === 'active' && (
                      <button
                        onClick={() => handleCancelJob(job.id)}
                        className="text-xs text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300"
                      >
                        Cancel
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {(activeTab === 'dlq' ? dlqItems : jobs).length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No {activeTab} jobs found
          </div>
        )}
      </div>
    </>
  );

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4 text-gray-900 dark:text-gray-100">Job Monitor</h1>

      {/* Tab Navigation */}
      <div className="mb-6">
        <nav className="flex flex-wrap gap-2">
          {[
            { id: 'overview', label: 'Overview' },
            { id: 'active', label: 'Active', count: activeCount },
            { id: 'pending', label: 'Pending', count: pendingCount },
            { id: 'scheduled', label: 'Scheduled', count: scheduledCount },
            { id: 'dlq', label: 'DLQ (Failed)', count: dlqCount }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => {
                setActiveTab(tab.id);
                setSelectedJobs([]);
              }}
              className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
                activeTab === tab.id
                  ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
                  : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
              }`}
            >
              {tab.label} {tab.count !== undefined && tab.count > 0 && <span className={`ml-2 px-2 py-0.5 rounded-full text-xs font-semibold ${
                activeTab === tab.id 
                  ? 'bg-blue-500 dark:bg-blue-600 text-white' 
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
              }`}>{tab.count}</span>}
            </button>
          ))}
        </nav>
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">Loading...</div>
      ) : activeTab === 'overview' ? (
        renderOverview()
      ) : (
        renderJobsTable()
      )}
    </div>
  );
}

export default JobMonitor;
