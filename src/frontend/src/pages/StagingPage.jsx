import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

const StagingPage = () => {
  const [stagingItems, setStagingItems] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('pending_review');
  const navigate = useNavigate();

  useEffect(() => {
    fetchStagingItems();
    fetchStats();
  }, [filter]);

  const fetchStagingItems = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/staging?status=${filter}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to fetch staging items');
      
      const data = await response.json();
      setStagingItems(data);
    } catch (error) {
      console.error('Error fetching staging items:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await fetch('/api/v1/staging/stats', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to fetch stats');
      
      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error('Error fetching stats:', error);
    }
  };

  const handleApprove = async (id) => {
    if (!confirm('Approve this album for promotion to production?')) return;
    
    try {
      const response = await fetch(`/api/v1/staging/${id}/approve`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ notes: '' })
      });
      
      if (!response.ok) throw new Error('Failed to approve');
      
      fetchStagingItems();
      fetchStats();
    } catch (error) {
      console.error('Error approving:', error);
      alert('Failed to approve album');
    }
  };

  const handleReject = async (id) => {
    const reason = prompt('Enter rejection reason:');
    if (!reason) return;
    
    try {
      const response = await fetch(`/api/v1/staging/${id}/reject`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ notes: reason })
      });
      
      if (!response.ok) throw new Error('Failed to reject');
      
      fetchStagingItems();
      fetchStats();
    } catch (error) {
      console.error('Error rejecting:', error);
      alert('Failed to reject album');
    }
  };

  const handlePromote = async (id) => {
    if (!confirm('Promote this album to production? This will move files and create database records.')) return;
    
    try {
      const response = await fetch(`/api/v1/staging/${id}/promote`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to promote');
      
      alert('Album promoted successfully!');
      fetchStagingItems();
      fetchStats();
    } catch (error) {
      console.error('Error promoting:', error);
      alert('Failed to promote album');
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this rejected album and its files?')) return;
    
    try {
      const response = await fetch(`/api/v1/staging/${id}?delete_files=true`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to delete');
      
      fetchStagingItems();
      fetchStats();
    } catch (error) {
      console.error('Error deleting:', error);
      alert('Failed to delete album');
    }
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="p-4 max-w-6xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-1 text-gray-900 dark:text-gray-100">Staging Area</h1>
        <p className="text-sm text-gray-700 dark:text-gray-300">Review and approve albums before promoting to production</p>
      </div>

      {/* Statistics */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Albums</h3>
            <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.total}</p>
          </div>
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Pending Review</h3>
            <p className="text-2xl text-amber-600 dark:text-amber-400">{stats.pending_review}</p>
          </div>
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Approved</h3>
            <p className="text-2xl text-emerald-600 dark:text-emerald-400">{stats.approved}</p>
          </div>
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Rejected</h3>
            <p className="text-2xl text-rose-600 dark:text-rose-400">{stats.rejected}</p>
          </div>
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Tracks</h3>
            <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.total_tracks}</p>
          </div>
          <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Size</h3>
            <p className="text-2xl text-gray-900 dark:text-gray-100">{formatBytes(stats.total_size_bytes)}</p>
          </div>
        </div>
      )}

      {/* Filter Tabs */}
      <div className="mb-6">
        <nav className="flex flex-wrap gap-2">
        <button
          className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
            filter === 'pending_review'
              ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
              : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
          }`}
          onClick={() => setFilter('pending_review')}
        >
          Pending Review ({stats?.pending_review || 0})
        </button>
        <button
          className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
            filter === 'approved'
              ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
              : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
          }`}
          onClick={() => setFilter('approved')}
        >
          Approved ({stats?.approved || 0})
        </button>
        <button
          className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
            filter === 'rejected'
              ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
              : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
          }`}
          onClick={() => setFilter('rejected')}
        >
          Rejected ({stats?.rejected || 0})
        </button>
        <button
          className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
            filter === ''
              ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
              : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
          }`}
          onClick={() => setFilter('')}
        >
          All ({stats?.total || 0})
        </button>
        </nav>
      </div>

      {/* Albums List */}
      {loading ? (
        <div className="bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700 text-center py-12 text-gray-700 dark:text-gray-200">Loading...</div>
      ) : stagingItems.length === 0 ? (
        <div className="bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700 text-center py-12 text-gray-700 dark:text-gray-200">No albums in this category</div>
      ) : (
        <div className="space-y-4">
          {stagingItems.map(item => (
            <div
              key={item.id}
              className={`bg-white dark:bg-gray-800 rounded shadow border p-4 flex flex-col md:flex-row md:items-start md:justify-between gap-4 ${
                item.status === 'pending_review'
                  ? 'border-amber-300 dark:border-amber-500'
                  : item.status === 'approved'
                  ? 'border-emerald-300 dark:border-emerald-500'
                  : item.status === 'rejected'
                  ? 'border-rose-300 dark:border-rose-500'
                  : 'border-gray-200 dark:border-gray-700'
              }`}
            >
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">{item.album_name}</h3>
                <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">{item.artist_name}</p>
                <div className="flex flex-wrap gap-3 text-xs text-gray-600 dark:text-gray-300 mb-2">
                  <span>{item.track_count} tracks</span>
                  <span>{formatBytes(item.total_size)}</span>
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                  <span className="mr-4">Scan: {item.scan_id}</span>
                  <span>{formatDate(item.processed_at)}</span>
                </div>
                {item.notes && (
                  <div className="mt-2 text-sm text-gray-800 dark:text-gray-100 bg-gray-50 dark:bg-gray-900/40 border border-gray-200 dark:border-gray-700 rounded p-2">
                    <span className="font-semibold">Notes:</span> {item.notes}
                  </div>
                )}
              </div>

              <div className="flex flex-wrap gap-2 md:justify-end">
                {item.status === 'pending_review' && (
                  <>
                    <button 
                      className="px-3 py-1.5 rounded-md text-sm font-semibold bg-green-600 hover:bg-green-700 text-white shadow-sm"
                      onClick={() => handleApprove(item.id)}
                    >
                      ✓ Approve
                    </button>
                    <button 
                      className="px-3 py-1.5 rounded-md text-sm font-semibold bg-red-600 hover:bg-red-700 text-white shadow-sm"
                      onClick={() => handleReject(item.id)}
                    >
                      ✗ Reject
                    </button>
                  </>
                )}
                
                {item.status === 'approved' && (
                  <button 
                    className="px-3 py-1.5 rounded-md text-sm font-semibold bg-blue-600 hover:bg-blue-700 text-white shadow-sm"
                    onClick={() => handlePromote(item.id)}
                  >
                    → Promote to Production
                  </button>
                )}
                
                {item.status === 'rejected' && (
                  <button 
                    className="px-3 py-1.5 rounded-md text-sm font-semibold bg-red-600 hover:bg-red-700 text-white shadow-sm"
                    onClick={() => handleDelete(item.id)}
                  >
                    🗑 Delete
                  </button>
                )}
                
                <button 
                  className="px-3 py-1.5 rounded-md text-sm font-semibold bg-gray-600 hover:bg-gray-700 text-white shadow-sm"
                  onClick={() => navigate(`/staging/${item.id}`)}
                >
                  View Details
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

    </div>
  );
};

export default StagingPage;
