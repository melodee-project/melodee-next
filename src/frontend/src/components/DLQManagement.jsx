import React, { useState, useEffect } from 'react';
import { adminService } from '../services/apiService';

function DLQManagement() {
  const [dlqItems, setDlqItems] = useState([]);
  const [selectedItems, setSelectedItems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDLQItems();
  }, []);

  const fetchDLQItems = async () => {
    try {
      const response = await adminService.getDLQItems();
      // Handle different response formats that might come from backend
      const items = response.data.data || response.data || [];
      setDlqItems(items);
    } catch (error) {
      console.error('Error fetching DLQ items:', error);
      // Show an error message to the user
      alert('Error fetching DLQ items: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const handleItemSelect = (itemId) => {
    if (selectedItems.includes(itemId)) {
      setSelectedItems(selectedItems.filter(id => id !== itemId));
    } else {
      setSelectedItems([...selectedItems, itemId]);
    }
  };

  const handleRequeueSelected = async () => {
    if (selectedItems.length === 0) return;

    try {
      await adminService.requeueDLQItems(selectedItems);
      // Refresh the list
      fetchDLQItems();
      setSelectedItems([]);
    } catch (error) {
      console.error('Error requeuing items:', error);
      alert('Error requeuing items: ' + (error.response?.data?.error || error.message));
    }
  };

  const handlePurgeSelected = async () => {
    if (selectedItems.length === 0) return;

    if (window.confirm(`Are you sure you want to purge ${selectedItems.length} selected items?`)) {
      try {
        await adminService.purgeDLQItems(selectedItems);
        // Refresh the list
        fetchDLQItems();
        setSelectedItems([]);
      } catch (error) {
        console.error('Error purging items:', error);
        alert('Error purging items: ' + (error.response?.data?.error || error.message));
      }
    }
  };

  if (loading) {
    return <div className="p-4">Loading DLQ items...</div>;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">DLQ Management</h1>
        <div className="space-x-2">
          <button
            onClick={handleRequeueSelected}
            disabled={selectedItems.length === 0}
            className="bg-blue-500 text-white px-4 py-2 rounded disabled:opacity-50"
          >
            Requeue Selected ({selectedItems.length})
          </button>
          <button
            onClick={handlePurgeSelected}
            disabled={selectedItems.length === 0}
            className="bg-red-500 text-white px-4 py-2 rounded disabled:opacity-50"
          >
            Purge Selected ({selectedItems.length})
          </button>
        </div>
      </div>

      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                <input
                  type="checkbox"
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedItems(dlqItems.map(item => item.id || item.ID));
                    } else {
                      setSelectedItems([]);
                    }
                  }}
                />
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Job ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Queue</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created At</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {dlqItems.map((item) => (
              <tr key={item.id || item.ID}>
                <td className="px-6 py-4 whitespace-nowrap">
                  <input
                    type="checkbox"
                    checked={selectedItems.includes(item.id || item.ID)}
                    onChange={() => handleItemSelect(item.id || item.ID)}
                  />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {item.id || item.ID}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.queue || item.Queue || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.type || item.Type || item.jobType || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.reason || item.Reason || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.created_at || item.CreatedAt ? new Date(item.created_at || item.CreatedAt).toLocaleString() : 'N/A'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {dlqItems.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No DLQ items found.
          </div>
        )}
      </div>
    </div>
  );
}

export default DLQManagement;