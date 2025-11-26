import React, { useState, useEffect } from 'react';
import { adminService } from '../services/apiService';

function SharesManagement() {
  const [shares, setShares] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newShare, setNewShare] = useState({
    name: '',
    ids: '',
    expires_at: '',
    max_streaming_minutes: 0,
    allow_download: false
  });
  const [errors, setErrors] = useState({});

  useEffect(() => {
    fetchShares();
  }, []);

  const fetchShares = async () => {
    try {
      const response = await adminService.getShares();
      setShares(response.data.data || response.data || []);
    } catch (error) {
      console.error('Error fetching shares:', error);
      // Handle error, maybe show a message to the user
    } finally {
      setLoading(false);
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!newShare.name.trim()) {
      newErrors.name = 'Name is required';
    }
    
    if (!newShare.ids.trim()) {
      newErrors.ids = 'IDs are required';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleCreateShare = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    try {
      // Parse the IDs field - could be a comma-separated list
      const idsArray = newShare.ids.split(',').map(id => id.trim()).filter(id => id);
      
      const shareData = {
        name: newShare.name,
        ids: idsArray, // Assuming backend expects an array
        expires_at: newShare.expires_at || null,
        max_streaming_minutes: parseInt(newShare.max_streaming_minutes) || 0,
        allow_download: newShare.allow_download
      };

      await adminService.createShare(shareData);
      setNewShare({
        name: '',
        ids: '',
        expires_at: '',
        max_streaming_minutes: 0,
        allow_download: false
      });
      setShowCreateForm(false);
      fetchShares(); // Refresh the list
    } catch (error) {
      console.error('Error creating share:', error);
      alert('Error creating share: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleDeleteShare = async (shareId) => {
    if (window.confirm('Are you sure you want to delete this share?')) {
      try {
        await adminService.deleteShare(shareId);
        fetchShares(); // Refresh the list
      } catch (error) {
        console.error('Error deleting share:', error);
        alert('Error deleting share: ' + (error.response?.data?.error || error.message));
      }
    }
  };

  if (loading) {
    return <div className="p-4 text-gray-900 dark:text-gray-100">Loading shares...</div>;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Shares Management</h1>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-500 dark:bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-600 dark:hover:bg-blue-700"
        >
          {showCreateForm ? 'Cancel' : '+ Add Share'}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreateShare} className="mb-6 p-4 bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
              <input
                type="text"
                value={newShare.name}
                onChange={(e) => setNewShare({...newShare, name: e.target.value})}
                className={`mt-1 block w-full border ${errors.name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100`}
                placeholder="Share name"
              />
              {errors.name && <p className="mt-1 text-sm text-red-600">{errors.name}</p>}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700">Item IDs</label>
              <input
                type="text"
                value={newShare.ids}
                onChange={(e) => setNewShare({...newShare, ids: e.target.value})}
                className={`mt-1 block w-full border ${errors.ids ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100`}
                placeholder="Comma-separated IDs (e.g., 1,2,3)"
              />
              {errors.ids && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.ids}</p>}
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Expiration Date (optional)</label>
              <input
                type="datetime-local"
                value={newShare.expires_at}
                onChange={(e) => setNewShare({...newShare, expires_at: e.target.value})}
                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Max Streaming Minutes (optional)</label>
              <input
                type="number"
                value={newShare.max_streaming_minutes}
                onChange={(e) => setNewShare({...newShare, max_streaming_minutes: e.target.value})}
                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                min="0"
              />
            </div>
            
            <div className="flex items-center">
              <input
                type="checkbox"
                checked={newShare.allow_download}
                onChange={(e) => setNewShare({...newShare, allow_download: e.target.checked})}
                className="mr-2 h-4 w-4 text-blue-600"
              />
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Allow Download</label>
            </div>
          </div>
          <button type="submit" className="mt-4 bg-green-500 dark:bg-green-600 text-white px-4 py-2 rounded hover:bg-green-600 dark:hover:bg-green-700">
            Create Share
          </button>
        </form>
      )}

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">IDs</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Expires</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Max Minutes</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Allow Download</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
            {shares.map((share) => (
              <tr key={share.id || share.ID}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                  {share.name || share.Name || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {Array.isArray(share.ids) ? share.ids.join(', ') : 
                   Array.isArray(share.IDs) ? share.IDs.join(', ') : 
                   share.ids || share.IDs || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {share.expires_at || share.ExpiresAt ? new Date(share.expires_at || share.ExpiresAt).toLocaleString() : 'Never'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {share.max_streaming_minutes || share.MaxStreamingMinutes || 0}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {share.allow_download || share.AllowDownload ? 'Yes' : 'No'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  <button
                    onClick={() => handleDeleteShare(share.id || share.ID)}
                    className="text-red-600 dark:text-red-400 hover:text-red-900 dark:hover:text-red-300"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        
        {shares.length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No shares found. Create one to get started.
          </div>
        )}
      </div>
    </div>
  );
}

export default SharesManagement;