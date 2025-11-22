import React, { useState, useEffect } from 'react';

function DLQManagement() {
  const [dlqItems, setDlqItems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulate loading DLQ items
    setTimeout(() => {
      setDlqItems([
        { id: 1, jobType: 'process-file', queue: 'default', reason: 'checksum_mismatch', createdAt: '2023-10-05T10:30:00Z' },
        { id: 2, jobType: 'transcode-file', queue: 'transcode', reason: 'ffmpeg_failure', createdAt: '2023-10-05T11:15:00Z' }
      ]);
      setLoading(false);
    }, 500);
  }, []);

  const handleRequeue = (id) => {
    // Simulate requeue action
    console.log(`Requeuing item ${id}`);
    setDlqItems(prev => prev.filter(item => item.id !== id));
  };

  const handlePurge = (id) => {
    // Simulate purge action
    console.log(`Purging item ${id}`);
    setDlqItems(prev => prev.filter(item => item.id !== id));
  };

  if (loading) {
    return <div className="p-6">Loading dead letter queue items...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">DLQ Management</h1>
      <div className="bg-white rounded shadow overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Job Type</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Queue</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created At</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {dlqItems.map((item) => (
              <tr key={item.id}>
                <td className="px-6 py-4 whitespace-nowrap">{item.id}</td>
                <td className="px-6 py-4 whitespace-nowrap">{item.jobType}</td>
                <td className="px-6 py-4 whitespace-nowrap">{item.queue}</td>
                <td className="px-6 py-4 whitespace-nowrap">{item.reason}</td>
                <td className="px-6 py-4 whitespace-nowrap">{new Date(item.createdAt).toLocaleString()}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <button
                    onClick={() => handleRequeue(item.id)}
                    className="text-blue-600 hover:text-blue-900 mr-4"
                  >
                    Requeue
                  </button>
                  <button
                    onClick={() => handlePurge(item.id)}
                    className="text-red-600 hover:text-red-900"
                  >
                    Purge
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default DLQManagement;