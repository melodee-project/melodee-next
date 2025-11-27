import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';

const StagingDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [item, setItem] = useState(null);
  const [metadata, setMetadata] = useState(null);

  useEffect(() => {
    fetchStagingItem();
  }, [id]);

  const fetchStagingItem = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/staging/${id}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to fetch staging item');
      
      const data = await response.json();
      setItem(data.item);
      setMetadata(data.metadata);
    } catch (error) {
      console.error('Error fetching staging item:', error);
      alert('Failed to load staging item');
      navigate('/staging');
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async () => {
    if (!confirm('Approve this album for promotion?')) return;
    
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
      
      alert('Album approved!');
      navigate('/staging');
    } catch (error) {
      console.error('Error approving:', error);
      alert('Failed to approve album');
    }
  };

  const handleReject = async () => {
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
      
      alert('Album rejected');
      navigate('/staging');
    } catch (error) {
      console.error('Error rejecting:', error);
      alert('Failed to reject album');
    }
  };

  const handlePromote = async () => {
    if (!confirm('Promote this album to production? This cannot be undone.')) return;
    
    try {
      const response = await fetch(`/api/v1/staging/${id}/promote`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (!response.ok) throw new Error('Failed to promote');
      
      alert('Album promoted successfully!');
      navigate('/staging');
    } catch (error) {
      console.error('Error promoting:', error);
      alert('Failed to promote album');
    }
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDuration = (ms) => {
    const minutes = Math.floor(ms / 60000);
    const seconds = ((ms % 60000) / 1000).toFixed(0);
    return `${minutes}:${seconds.padStart(2, '0')}`;
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  if (!item || !metadata) {
    return <div className="error">Failed to load staging item</div>;
  }

  return (
    <div className="staging-detail-page">
      <div className="page-header">
        <button onClick={() => navigate('/staging')} className="back-button">
          ← Back to Staging
        </button>
        <h1>{metadata.album.name}</h1>
        <p className="artist-name">{metadata.artist.name}</p>
      </div>

      {/* Album Info */}
      <div className="info-grid">
        <div className="info-card">
          <h3>Album Information</h3>
          <dl>
            <dt>Status:</dt>
            <dd className={`status status-${item.status}`}>
              {item.status.replace('_', ' ').toUpperCase()}
            </dd>

            <dt>Artist:</dt>
            <dd>{metadata.artist.name}</dd>

            <dt>Directory Code:</dt>
            <dd>{metadata.artist.directory_code}</dd>

            <dt>Year:</dt>
            <dd>{metadata.album.year || 'Unknown'}</dd>

            <dt>Album Type:</dt>
            <dd>{metadata.album.album_type}</dd>

            <dt>Compilation:</dt>
            <dd>{metadata.album.is_compilation ? 'Yes' : 'No'}</dd>

            {metadata.album.genres && metadata.album.genres.length > 0 && (
              <>
                <dt>Genres:</dt>
                <dd>{metadata.album.genres.join(', ')}</dd>
              </>
            )}
          </dl>
        </div>

        <div className="info-card">
          <h3>File Information</h3>
          <dl>
            <dt>Track Count:</dt>
            <dd>{item.track_count}</dd>

            <dt>Total Size:</dt>
            <dd>{formatBytes(item.total_size)}</dd>

            <dt>Staging Path:</dt>
            <dd className="path">{item.staging_path}</dd>

            <dt>Metadata File:</dt>
            <dd className="path">{item.metadata_file}</dd>

            <dt>Scan ID:</dt>
            <dd>{item.scan_id}</dd>

            <dt>Processed:</dt>
            <dd>{new Date(item.processed_at).toLocaleString()}</dd>

            <dt>Checksum:</dt>
            <dd className="checksum">{item.checksum.substring(0, 16)}...</dd>
          </dl>
        </div>
      </div>

      {/* Validation */}
      {metadata.validation && (
        <div className={`validation-card ${metadata.validation.is_valid ? 'valid' : 'invalid'}`}>
          <h3>Validation {metadata.validation.is_valid ? '✓' : '✗'}</h3>
          
          {metadata.validation.errors && metadata.validation.errors.length > 0 && (
            <div className="errors">
              <h4>Errors:</h4>
              <ul>
                {metadata.validation.errors.map((error, i) => (
                  <li key={i}>{error}</li>
                ))}
              </ul>
            </div>
          )}

          {metadata.validation.warnings && metadata.validation.warnings.length > 0 && (
            <div className="warnings">
              <h4>Warnings:</h4>
              <ul>
                {metadata.validation.warnings.map((warning, i) => (
                  <li key={i}>{warning}</li>
                ))}
              </ul>
            </div>
          )}

          {metadata.validation.is_valid && 
           metadata.validation.errors.length === 0 && 
           metadata.validation.warnings.length === 0 && (
            <p className="success-message">No issues found</p>
          )}
        </div>
      )}

      {/* Tracks List */}
      <div className="tracks-section">
        <h3>Tracks ({metadata.tracks.length})</h3>
        <div className="tracks-table">
          <table>
            <thead>
              <tr>
                <th>#</th>
                <th>Title</th>
                <th>Duration</th>
                <th>Size</th>
                <th>Bitrate</th>
                <th>Sample Rate</th>
              </tr>
            </thead>
            <tbody>
              {metadata.tracks
                .sort((a, b) => {
                  if (a.disc_number !== b.disc_number) {
                    return a.disc_number - b.disc_number;
                  }
                  return a.track_number - b.track_number;
                })
                .map((track, index) => (
                  <tr key={index}>
                    <td>
                      {track.disc_number > 1 && `${track.disc_number}-`}
                      {track.track_number || index + 1}
                    </td>
                    <td className="track-name">{track.name}</td>
                    <td>{formatDuration(track.duration)}</td>
                    <td>{formatBytes(track.file_size)}</td>
                    <td>{track.bitrate} kbps</td>
                    <td>{(track.sample_rate / 1000).toFixed(1)} kHz</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Actions */}
      <div className="actions-bar">
        {item.status === 'pending_review' && (
          <>
            <button className="btn btn-success" onClick={handleApprove}>
              ✓ Approve
            </button>
            <button className="btn btn-danger" onClick={handleReject}>
              ✗ Reject
            </button>
          </>
        )}
        
        {item.status === 'approved' && (
          <button className="btn btn-primary" onClick={handlePromote}>
            → Promote to Production
          </button>
        )}
      </div>

      <style jsx>{`
        .staging-detail-page {
          padding: 2rem;
          max-width: 1200px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 2rem;
        }

        .back-button {
          background: none;
          border: 1px solid #ddd;
          padding: 0.5rem 1rem;
          border-radius: 4px;
          cursor: pointer;
          margin-bottom: 1rem;
          transition: all 0.2s;
        }

        .back-button:hover {
          background: #f5f5f5;
        }

        .page-header h1 {
          margin: 0.5rem 0;
          font-size: 2rem;
        }

        .artist-name {
          font-size: 1.3rem;
          color: #666;
          margin: 0;
        }

        .info-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
          gap: 1rem;
          margin-bottom: 2rem;
        }

        .info-card, .validation-card, .tracks-section {
          background: white;
          border: 1px solid #ddd;
          border-radius: 8px;
          padding: 1.5rem;
        }

        .info-card h3, .validation-card h3, .tracks-section h3 {
          margin: 0 0 1rem 0;
          font-size: 1.2rem;
          border-bottom: 2px solid #eee;
          padding-bottom: 0.5rem;
        }

        dl {
          margin: 0;
          display: grid;
          grid-template-columns: 140px 1fr;
          gap: 0.75rem;
        }

        dt {
          font-weight: 600;
          color: #666;
        }

        dd {
          margin: 0;
        }

        .path, .checksum {
          font-family: monospace;
          font-size: 0.9rem;
          color: #666;
          word-break: break-all;
        }

        .status {
          display: inline-block;
          padding: 0.25rem 0.75rem;
          border-radius: 4px;
          font-weight: 600;
          font-size: 0.9rem;
        }

        .status-pending_review {
          background: #fff3cd;
          color: #856404;
        }

        .status-approved {
          background: #d4edda;
          color: #155724;
        }

        .status-rejected {
          background: #f8d7da;
          color: #721c24;
        }

        .validation-card {
          margin-bottom: 2rem;
        }

        .validation-card.valid {
          border-left: 4px solid #28a745;
        }

        .validation-card.invalid {
          border-left: 4px solid #dc3545;
        }

        .errors, .warnings {
          margin-top: 1rem;
        }

        .errors h4, .warnings h4 {
          margin: 0 0 0.5rem 0;
          font-size: 1rem;
        }

        .errors ul {
          color: #dc3545;
        }

        .warnings ul {
          color: #ffc107;
        }

        .success-message {
          color: #28a745;
          margin: 1rem 0 0 0;
        }

        .tracks-section {
          margin-bottom: 2rem;
        }

        .tracks-table {
          overflow-x: auto;
        }

        table {
          width: 100%;
          border-collapse: collapse;
        }

        th, td {
          padding: 0.75rem;
          text-align: left;
          border-bottom: 1px solid #eee;
        }

        th {
          font-weight: 600;
          background: #f8f9fa;
          color: #666;
        }

        .track-name {
          font-weight: 500;
        }

        .actions-bar {
          display: flex;
          gap: 1rem;
          justify-content: center;
          padding: 2rem 0;
        }

        .btn {
          padding: 0.75rem 2rem;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 1rem;
          font-weight: 600;
          transition: all 0.2s;
        }

        .btn:hover {
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }

        .btn-primary {
          background: #007bff;
          color: white;
        }

        .btn-success {
          background: #28a745;
          color: white;
        }

        .btn-danger {
          background: #dc3545;
          color: white;
        }

        .loading, .error {
          text-align: center;
          padding: 3rem;
          font-size: 1.2rem;
          color: #666;
        }
      `}</style>
    </div>
  );
};

export default StagingDetailPage;
