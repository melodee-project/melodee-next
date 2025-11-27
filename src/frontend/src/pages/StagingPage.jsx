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
    <div className="staging-page">
      <div className="page-header">
        <h1>Staging Area</h1>
        <p>Review and approve albums before promoting to production</p>
      </div>

      {/* Statistics */}
      {stats && (
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-value">{stats.total}</div>
            <div className="stat-label">Total Albums</div>
          </div>
          <div className="stat-card pending">
            <div className="stat-value">{stats.pending_review}</div>
            <div className="stat-label">Pending Review</div>
          </div>
          <div className="stat-card approved">
            <div className="stat-value">{stats.approved}</div>
            <div className="stat-label">Approved</div>
          </div>
          <div className="stat-card rejected">
            <div className="stat-value">{stats.rejected}</div>
            <div className="stat-label">Rejected</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.total_tracks}</div>
            <div className="stat-label">Total Tracks</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{formatBytes(stats.total_size_bytes)}</div>
            <div className="stat-label">Total Size</div>
          </div>
        </div>
      )}

      {/* Filter Tabs */}
      <div className="filter-tabs">
        <button 
          className={filter === 'pending_review' ? 'active' : ''}
          onClick={() => setFilter('pending_review')}
        >
          Pending Review ({stats?.pending_review || 0})
        </button>
        <button 
          className={filter === 'approved' ? 'active' : ''}
          onClick={() => setFilter('approved')}
        >
          Approved ({stats?.approved || 0})
        </button>
        <button 
          className={filter === 'rejected' ? 'active' : ''}
          onClick={() => setFilter('rejected')}
        >
          Rejected ({stats?.rejected || 0})
        </button>
        <button 
          className={filter === '' ? 'active' : ''}
          onClick={() => setFilter('')}
        >
          All ({stats?.total || 0})
        </button>
      </div>

      {/* Albums List */}
      {loading ? (
        <div className="loading">Loading...</div>
      ) : stagingItems.length === 0 ? (
        <div className="empty-state">
          <p>No albums in this category</p>
        </div>
      ) : (
        <div className="albums-grid">
          {stagingItems.map(item => (
            <div key={item.id} className={`album-card status-${item.status}`}>
              <div className="album-info">
                <h3>{item.album_name}</h3>
                <p className="artist">{item.artist_name}</p>
                <div className="meta">
                  <span>{item.track_count} tracks</span>
                  <span>{formatBytes(item.total_size)}</span>
                </div>
                <div className="scan-info">
                  <span className="scan-id">Scan: {item.scan_id}</span>
                  <span className="processed-at">{formatDate(item.processed_at)}</span>
                </div>
                {item.notes && (
                  <div className="notes">
                    <strong>Notes:</strong> {item.notes}
                  </div>
                )}
              </div>

              <div className="album-actions">
                {item.status === 'pending_review' && (
                  <>
                    <button 
                      className="btn btn-success"
                      onClick={() => handleApprove(item.id)}
                    >
                      ✓ Approve
                    </button>
                    <button 
                      className="btn btn-danger"
                      onClick={() => handleReject(item.id)}
                    >
                      ✗ Reject
                    </button>
                  </>
                )}
                
                {item.status === 'approved' && (
                  <button 
                    className="btn btn-primary"
                    onClick={() => handlePromote(item.id)}
                  >
                    → Promote to Production
                  </button>
                )}
                
                {item.status === 'rejected' && (
                  <button 
                    className="btn btn-danger"
                    onClick={() => handleDelete(item.id)}
                  >
                    🗑 Delete
                  </button>
                )}
                
                <button 
                  className="btn btn-secondary"
                  onClick={() => navigate(`/staging/${item.id}`)}
                >
                  View Details
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <style jsx>{`
        .staging-page {
          padding: 2rem;
          max-width: 1400px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 2rem;
        }

        .page-header h1 {
          margin: 0 0 0.5rem 0;
          font-size: 2rem;
        }

        .page-header p {
          margin: 0;
          color: #666;
        }

        .stats-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
          gap: 1rem;
          margin-bottom: 2rem;
        }

        .stat-card {
          background: white;
          border: 1px solid #ddd;
          border-radius: 8px;
          padding: 1.5rem;
          text-align: center;
        }

        .stat-card.pending {
          border-left: 4px solid #ffc107;
        }

        .stat-card.approved {
          border-left: 4px solid #28a745;
        }

        .stat-card.rejected {
          border-left: 4px solid #dc3545;
        }

        .stat-value {
          font-size: 2rem;
          font-weight: bold;
          margin-bottom: 0.5rem;
        }

        .stat-label {
          font-size: 0.9rem;
          color: #666;
        }

        .filter-tabs {
          display: flex;
          gap: 0.5rem;
          margin-bottom: 2rem;
          border-bottom: 2px solid #ddd;
        }

        .filter-tabs button {
          background: none;
          border: none;
          padding: 1rem 1.5rem;
          cursor: pointer;
          font-size: 1rem;
          color: #666;
          border-bottom: 3px solid transparent;
          transition: all 0.2s;
        }

        .filter-tabs button:hover {
          color: #333;
          background: #f5f5f5;
        }

        .filter-tabs button.active {
          color: #007bff;
          border-bottom-color: #007bff;
          font-weight: 600;
        }

        .loading, .empty-state {
          text-align: center;
          padding: 3rem;
          color: #666;
        }

        .albums-grid {
          display: grid;
          gap: 1rem;
        }

        .album-card {
          background: white;
          border: 1px solid #ddd;
          border-radius: 8px;
          padding: 1.5rem;
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          transition: box-shadow 0.2s;
        }

        .album-card:hover {
          box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        }

        .album-card.status-pending_review {
          border-left: 4px solid #ffc107;
        }

        .album-card.status-approved {
          border-left: 4px solid #28a745;
        }

        .album-card.status-rejected {
          border-left: 4px solid #dc3545;
        }

        .album-info {
          flex: 1;
        }

        .album-info h3 {
          margin: 0 0 0.5rem 0;
          font-size: 1.3rem;
        }

        .album-info .artist {
          margin: 0 0 1rem 0;
          color: #666;
          font-size: 1.1rem;
        }

        .meta {
          display: flex;
          gap: 1rem;
          margin-bottom: 0.5rem;
        }

        .meta span {
          font-size: 0.9rem;
          color: #666;
        }

        .scan-info {
          font-size: 0.85rem;
          color: #999;
        }

        .scan-info span {
          margin-right: 1rem;
        }

        .notes {
          margin-top: 1rem;
          padding: 0.75rem;
          background: #f8f9fa;
          border-radius: 4px;
          font-size: 0.9rem;
        }

        .album-actions {
          display: flex;
          gap: 0.5rem;
          flex-wrap: wrap;
          align-items: flex-start;
        }

        .btn {
          padding: 0.5rem 1rem;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 0.9rem;
          transition: all 0.2s;
          white-space: nowrap;
        }

        .btn:hover {
          transform: translateY(-1px);
          box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        }

        .btn-primary {
          background: #007bff;
          color: white;
        }

        .btn-primary:hover {
          background: #0056b3;
        }

        .btn-success {
          background: #28a745;
          color: white;
        }

        .btn-success:hover {
          background: #218838;
        }

        .btn-danger {
          background: #dc3545;
          color: white;
        }

        .btn-danger:hover {
          background: #c82333;
        }

        .btn-secondary {
          background: #6c757d;
          color: white;
        }

        .btn-secondary:hover {
          background: #5a6268;
        }
      `}</style>
    </div>
  );
};

export default StagingPage;
