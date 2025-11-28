import React, { useState, useEffect } from 'react';
import { libraryService } from '../services/apiService';

function LibraryManagement() {
  const [stats, setStats] = useState({});
  const [libraries, setLibraries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview'); // 'overview', 'inbound', 'staging', 'production'
  const [message, setMessage] = useState('');
  const [editingLibrary, setEditingLibrary] = useState(null);
  const [editForm, setEditForm] = useState({ name: '', path: '' });
  const [showWorkflowHelp, setShowWorkflowHelp] = useState(false);

  useEffect(() => {
    fetchLibraryStats();
    fetchLibraries();
  }, []);

  const fetchLibraryStats = async () => {
    try {
      const response = await libraryService.getStats();
      setStats(response.data.data || response.data || {});
    } catch (error) {
      console.error('Error fetching library stats:', error);
      setStats({});
    }
  };

  const fetchLibraries = async () => {
    try {
      const response = await libraryService.getLibraries();
      setLibraries(response.data.data || response.data || []);
    } catch (error) {
      console.error('Error fetching libraries:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleEditLibrary = (library) => {
    setEditingLibrary(library);
    setEditForm({
      name: library.name,
      path: library.path
    });
  };

  const handleSaveLibrary = async () => {
    try {
      await libraryService.updateLibrary(editingLibrary.id, editForm);
      setMessage('Library updated successfully');
      setEditingLibrary(null);
      fetchLibraries();
    } catch (error) {
      console.error('Error updating library:', error);
      setMessage('Error updating library: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleCancelEdit = () => {
    setEditingLibrary(null);
    setEditForm({ name: '', path: '' });
  };

  if (loading) {
    return <div className="p-4 text-gray-900 dark:text-gray-100">Loading library statistics...</div>;
  }

  // Filter libraries based on active tab
  const getFilteredLibraries = () => {
    if (activeTab === 'overview') {
      return libraries;
    }

    return libraries.filter(lib => {
      if (activeTab === 'inbound') return lib.inbound_count > 0 || lib.inboundCount > 0;
      if (activeTab === 'staging') return lib.staging_count > 0 || lib.stagingCount > 0;
      if (activeTab === 'production') return lib.production_count > 0 || lib.productionCount > 0;
      return true;
    });
  };

  const filteredLibraries = getFilteredLibraries();

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Library Management</h1>
        <button
          onClick={() => setShowWorkflowHelp(!showWorkflowHelp)}
          className="flex items-center gap-2 px-4 py-2 text-sm bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded hover:bg-blue-200 dark:hover:bg-blue-900/50 border border-blue-300 dark:border-blue-700"
        >
          <span>📋</span>
          <span>{showWorkflowHelp ? 'Hide' : 'Show'} Workflow Help</span>
        </button>
      </div>

      {/* Edit Library Modal */}
      {editingLibrary && (
        <div className="fixed inset-0 bg-black bg-opacity-50 dark:bg-opacity-70 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4 border border-gray-200 dark:border-gray-700">
            <h2 className="text-xl font-bold mb-4 text-gray-900 dark:text-gray-100">Edit Library: {editingLibrary.name}</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Name</label>
                <input
                  type="text"
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Path</label>
                <input
                  type="text"
                  value={editForm.path}
                  onChange={(e) => setEditForm({ ...editForm, path: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., /mnt/music or C:\Music or /nfs/media"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={handleCancelEdit}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-gray-200 dark:bg-gray-700 rounded hover:bg-gray-300 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveLibrary}
                className="px-4 py-2 text-white bg-blue-500 dark:bg-blue-600 rounded hover:bg-blue-600 dark:hover:bg-blue-700"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Tab Navigation */}
      <div className="mb-6">
        <nav className="flex flex-wrap gap-2">
          {[
            { id: 'overview', label: 'Overview', count: libraries.length },
            { id: 'inbound', label: 'Inbound', count: libraries.filter(lib => lib.inbound_count > 0 || lib.inboundCount > 0).length },
            { id: 'staging', label: 'Staging', count: libraries.filter(lib => lib.staging_count > 0 || lib.stagingCount > 0).length },
            { id: 'production', label: 'Production', count: libraries.filter(lib => lib.production_count > 0 || lib.productionCount > 0).length }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2 rounded-lg font-medium text-sm transition-all ${
                activeTab === tab.id
                  ? 'bg-blue-600 dark:bg-blue-700 text-white shadow-md'
                  : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
              }`}
            >
              {tab.label} {tab.count > 0 && <span className={`ml-2 px-2 py-0.5 rounded-full text-xs font-semibold ${
                activeTab === tab.id 
                  ? 'bg-blue-500 dark:bg-blue-600 text-white' 
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
              }`}>{tab.count}</span>}
            </button>
          ))}
        </nav>
      </div>

      {activeTab === 'overview' && (
        <>
          {showWorkflowHelp && (
            <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded shadow border border-blue-200 dark:border-blue-800">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-semibold text-blue-900 dark:text-blue-100">📋 Media Processing Workflow</h2>
                <button
                  onClick={() => setShowWorkflowHelp(false)}
                  className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200"
                  aria-label="Close workflow help"
                >
                  ✕
                </button>
              </div>
              <p className="text-blue-800 dark:text-blue-200 mb-3">
                Follow these steps to add new media to your library:
              </p>
              <ol className="list-decimal list-inside space-y-2 text-blue-800 dark:text-blue-200 mb-4">
                <li><strong>Scan inbound files:</strong> Run <code className="bg-blue-100 dark:bg-blue-900 px-2 py-1 rounded">./scan-inbound -path /inbound -output /scans</code></li>
                <li><strong>Process to staging:</strong> Run <code className="bg-blue-100 dark:bg-blue-900 px-2 py-1 rounded">./process-scan -scan scan.db -staging /staging</code></li>
                <li><strong>Review & approve:</strong> Navigate to <a href="/staging" className="underline font-semibold hover:text-blue-600 dark:hover:text-blue-400">Staging page</a></li>
                <li><strong>Promote to production:</strong> Click "Promote" button on approved albums</li>
              </ol>
              <div className="flex gap-3">
                <a 
                  href="/staging" 
                  className="inline-block bg-blue-600 dark:bg-blue-700 text-white px-4 py-2 rounded hover:bg-blue-700 dark:hover:bg-blue-800 font-medium"
                >
                  Go to Staging →
                </a>
                <a 
                  href="/docs/QUICKSTART.md" 
                  target="_blank"
                  className="inline-block bg-gray-600 dark:bg-gray-700 text-white px-4 py-2 rounded hover:bg-gray-700 dark:hover:bg-gray-800 font-medium"
                >
                  View Documentation
                </a>
              </div>
            </div>
          )}

          {message && (
            <div className="mb-4 p-3 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded border border-yellow-200 dark:border-yellow-800">
                {message}
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Artists</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.total_artists || stats.totalArtists || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Albums</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.total_albums || stats.totalAlbums || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Songs</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.total_songs || stats.totalSongs || 0}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Inbound Files</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.inbound_count || stats.inboundCount || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Staging Files</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.staging_count || stats.stagingCount || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Production Files</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{stats.production_count || stats.productionCount || 0}</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
              <h3 className="font-semibold text-gray-700 dark:text-gray-300">Total Duration</h3>
              <p className="text-2xl text-gray-900 dark:text-gray-100">{formatDuration(stats.total_duration || stats.totalDuration || 0)}</p>
            </div>
          </div>

          <div className="mt-6 bg-white dark:bg-gray-800 p-4 rounded shadow border border-gray-200 dark:border-gray-700">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Configured Libraries</h2>
            </div>
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Type</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Path</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {libraries.map((lib) => (
                  <tr key={lib.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">{lib.name}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full
                        ${lib.type === 'inbound' ? 'bg-blue-100 text-blue-800' :
                          lib.type === 'staging' ? 'bg-purple-100 text-purple-800' :
                          lib.type === 'production' ? 'bg-green-100 text-green-800' :
                          'bg-gray-100 text-gray-800'}`}>
                        {lib.type}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">{lib.path}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <button
                        onClick={() => handleEditLibrary(lib)}
                        className="px-3 py-1 bg-blue-600 dark:bg-blue-600 text-white rounded hover:bg-blue-700 dark:hover:bg-blue-500 font-medium"
                      >
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {(activeTab !== 'overview') && (
        <div className="bg-white dark:bg-gray-800 rounded shadow overflow-hidden border border-gray-200 dark:border-gray-700">
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-xl font-semibold capitalize text-gray-900 dark:text-gray-100">{activeTab} Libraries</h2>
          </div>

          {filteredLibraries.length > 0 ? (
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Library</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Type</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Inbound</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Staging</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Production</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Rejected</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {filteredLibraries.map((lib) => (
                  <tr key={lib.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900 dark:text-gray-100">{lib.name}</div>
                      <div className="text-sm text-gray-500 dark:text-gray-400">{lib.path}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full
                        ${lib.type === 'inbound' ? 'bg-blue-100 text-blue-800' :
                          lib.type === 'staging' ? 'bg-purple-100 text-purple-800' :
                          lib.type === 'production' ? 'bg-green-100 text-green-800' :
                          'bg-gray-100 text-gray-800'}`}>
                        {lib.type}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {lib.inbound_count || lib.inboundCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {lib.staging_count || lib.stagingCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {lib.production_count || lib.productionCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {lib.rejected_count || lib.rejectedCount || 0}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              No {activeTab} libraries found.
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Helper function to format duration in seconds to HH:MM:SS
function formatDuration(durationInSeconds) {
  if (!durationInSeconds) return '00:00:00';
  
  const hours = Math.floor(durationInSeconds / 3600);
  const minutes = Math.floor((durationInSeconds % 3600) / 60);
  const seconds = durationInSeconds % 60;

  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
}

export default LibraryManagement;