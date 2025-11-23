import React, { useState, useEffect } from 'react';
import { libraryService } from '../services/apiService';

function QuarantineManagement() {
  const [quarantineItems, setQuarantineItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedItems, setSelectedItems] = useState([]);
  const [message, setMessage] = useState('');
  const [filters, setFilters] = useState({
    reason: '',
    page: 1,
    limit: 50
  });

  useEffect(() => {
    fetchQuarantineItems();
  }, [filters]);

  const fetchQuarantineItems = async () => {
    try {
      setLoading(true);
      const params = {
        reason: filters.reason || undefined,
        page: filters.page,
        limit: filters.limit
      };

      const response = await libraryService.getQuarantineItems(params);
      const items = Array.isArray(response.data.data) ? response.data.data : 
                   Array.isArray(response.data.items) ? response.data.items : 
                   Array.isArray(response.data) ? response.data : [];
      setQuarantineItems(items);
    } catch (error) {
      console.error('Error fetching quarantine items:', error);
      setMessage('Error fetching quarantine items: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const handleItemSelect = (itemId) => {
    setSelectedItems(prev =>
      prev.includes(itemId)
        ? prev.filter(id => id !== itemId)
        : [...prev, itemId]
    );
  };

  const handleSelectAll = () => {
    if (selectedItems.length === quarantineItems.length) {
      setSelectedItems([]);
    } else {
      setSelectedItems(quarantineItems.map(item => item.ID || item.id));
    }
  };

  const handleResolveSelected = async () => {
    if (selectedItems.length === 0) return;

    try {
      setMessage(`Resolving ${selectedItems.length} items...`);

      // Process items individually to handle partial failures
      const promises = selectedItems.map(id => libraryService.resolveQuarantineItem(id));
      await Promise.all(promises);

      setMessage(`${selectedItems.length} items resolved successfully.`);
      setSelectedItems([]);
      fetchQuarantineItems(); // Refresh the list
    } catch (error) {
      console.error('Error resolving items:', error);
      setMessage('Error resolving items: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleRequeueSelected = async () => {
    if (selectedItems.length === 0) return;

    try {
      setMessage(`Requeuing ${selectedItems.length} items...`);

      // Process items individually to handle partial failures
      const promises = selectedItems.map(id => libraryService.requeueQuarantineItem(id));
      await Promise.all(promises);

      setMessage(`${selectedItems.length} items requeued successfully.`);
      setSelectedItems([]);
      fetchQuarantineItems(); // Refresh the list
    } catch (error) {
      console.error('Error requeuing items:', error);
      setMessage('Error requeuing items: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleFilterChange = (field, value) => {
    setFilters(prev => ({
      ...prev,
      [field]: value,
      page: 1 // Reset to first page when filters change
    }));
  };

  const getReasonColor = (reason) => {
    const reasonColors = {
      'checksum_mismatch': 'bg-red-100 text-red-800',
      'tag_parse_error': 'bg-yellow-100 text-yellow-800',
      'unsupported_container': 'bg-orange-100 text-orange-800',
      'ffmpeg_failure': 'bg-red-100 text-red-800',
      'path_safety': 'bg-red-100 text-red-800',
      'validation_bounds': 'bg-orange-100 text-orange-800',
      'metadata_conflict': 'bg-purple-100 text-purple-800',
      'disk_full': 'bg-red-100 text-red-800',
      'cue_missing_audio': 'bg-red-100 text-purple-800'
    };
    return reasonColors[reason] || 'bg-gray-100 text-gray-800';
  };

  if (loading && quarantineItems.length === 0) {
    return <div className="p-4">Loading quarantine items...</div>;
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Quarantine Management</h1>

      {message && (
        <div className={`mb-4 p-3 rounded ${
          message.toLowerCase().includes('error') ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
        }`}>
          {message}
        </div>
      )}

      {/* Filters and Controls */}
      <div className="mb-6 p-4 bg-white rounded shadow">
        <h2 className="text-lg font-semibold mb-3">Filters</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Reason</label>
            <select
              value={filters.reason}
              onChange={(e) => handleFilterChange('reason', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="">All Reasons</option>
              <option value="checksum_mismatch">Checksum Mismatch</option>
              <option value="tag_parse_error">Tag Parse Error</option>
              <option value="unsupported_container">Unsupported Container</option>
              <option value="ffmpeg_failure">FFmpeg Failure</option>
              <option value="path_safety">Path Safety</option>
              <option value="validation_bounds">Validation Bounds</option>
              <option value="metadata_conflict">Metadata Conflict</option>
              <option value="disk_full">Disk Full</option>
              <option value="cue_missing_audio">CUE Missing Audio</option>
            </select>
          </div>
        </div>

        <div className="flex flex-wrap gap-4">
          <button
            onClick={handleResolveSelected}
            disabled={selectedItems.length === 0}
            className="bg-green-500 text-white px-4 py-2 rounded hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Resolve Selected ({selectedItems.length})
          </button>
          <button
            onClick={handleRequeueSelected}
            disabled={selectedItems.length === 0}
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Requeue Selected ({selectedItems.length})
          </button>
        </div>
      </div>

      {/* Quarantine Items Table */}
      <div className="bg-white shadow rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left">
                  <input
                    type="checkbox"
                    checked={quarantineItems.length > 0 && selectedItems.length === quarantineItems.length}
                    onChange={handleSelectAll}
                    className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                  />
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File Path</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Library</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created At</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {quarantineItems.map((item) => (
                <tr key={item.ID || item.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <input
                      type="checkbox"
                      checked={selectedItems.includes(item.ID || item.id)}
                      onChange={() => handleItemSelect(item.ID || item.id)}
                      className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                    />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {item.ID || item.id || 'N/A'}
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-sm text-gray-900 break-all max-w-xs">{item.FilePath || item.filePath || 'N/A'}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getReasonColor(item.Reason || item.reason)}`}>
                      {(item.Reason || item.reason || 'Unknown').split('_').map(word => 
                        word.charAt(0).toUpperCase() + word.slice(1)
                      ).join(' ')}
                    </span>
                    <div className="text-xs text-gray-500 mt-1">
                      {item.Message || item.message || ''}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {item.LibraryID || item.libraryId || 'N/A'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {item.CreatedAt ? new Date(item.CreatedAt).toLocaleString() : 'N/A'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <div className="flex space-x-2">
                      <button
                        onClick={async () => {
                          try {
                            await libraryService.resolveQuarantineItem(item.ID || item.id);
                            setMessage('Item resolved successfully');
                            fetchQuarantineItems();
                          } catch (error) {
                            setMessage('Error resolving item: ' + (error.response?.data?.error || error.message));
                          }
                        }}
                        className="text-green-600 hover:text-green-900"
                        title="Resolve Item"
                      >
                        Resolve
                      </button>
                      <button
                        onClick={async () => {
                          try {
                            await libraryService.requeueQuarantineItem(item.ID || item.id);
                            setMessage('Item requeued successfully');
                            fetchQuarantineItems();
                          } catch (error) {
                            setMessage('Error requeuing item: ' + (error.response?.data?.error || error.message));
                          }
                        }}
                        className="text-blue-600 hover:text-blue-900"
                        title="Requeue Item"
                      >
                        Requeue
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {quarantineItems.length === 0 && !loading && (
          <div className="text-center py-8 text-gray-500">
            No quarantine items found.
          </div>
        )}

        {loading && quarantineItems.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            Loading quarantine items...
          </div>
        )}
      </div>
    </div>
  );
}

export default QuarantineManagement;