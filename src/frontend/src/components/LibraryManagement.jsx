import React, { useState, useEffect } from 'react';
import { libraryService } from '../services/apiService';

function LibraryManagement() {
  const [stats, setStats] = useState({});
  const [libraries, setLibraries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview'); // 'overview', 'inbound', 'staging', 'production', 'quarantine'
  const [message, setMessage] = useState('');

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

  const handleScan = async () => {
    try {
      setMessage('Scanning initiated...');
      await libraryService.scanLibrary();
      setMessage('Scan completed. Refreshing stats...');
      setTimeout(() => {
        fetchLibraryStats();
        setMessage('Scan completed and stats refreshed.');
      }, 2000); // Give some time for the scan to process
    } catch (error) {
      console.error('Error initiating scan:', error);
      setMessage('Error initiating scan: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleProcess = async () => {
    try {
      setMessage('Processing inbound files...');
      await libraryService.processInbound();
      setMessage('Inbound processing completed.');
      setTimeout(() => {
        fetchLibraryStats();
        setMessage('Inbound processing completed and stats refreshed.');
      }, 2000);
    } catch (error) {
      console.error('Error processing inbound:', error);
      setMessage('Error processing inbound: ' + (error.response?.data?.error || error.message));
    }
  };

  const handlePromote = async () => {
    try {
      setMessage('Promoting OK albums...');
      await libraryService.moveOkAlbums();
      setMessage('Album promotion completed.');
      setTimeout(() => {
        fetchLibraryStats();
        setMessage('Album promotion completed and stats refreshed.');
      }, 2000);
    } catch (error) {
      console.error('Error promoting albums:', error);
      setMessage('Error promoting albums: ' + (error.response?.data?.error || error.message));
    }
  };

  if (loading) {
    return <div className="p-4">Loading library statistics...</div>;
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
      if (activeTab === 'quarantine') return lib.quarantine_count > 0 || lib.quarantineCount > 0;
      return true;
    });
  };

  const filteredLibraries = getFilteredLibraries();

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Library Management</h1>

      {/* Tab Navigation */}
      <div className="mb-4 border-b border-gray-200">
        <nav className="flex space-x-8">
          {[
            { id: 'overview', label: 'Overview', count: libraries.length },
            { id: 'inbound', label: 'Inbound', count: libraries.filter(lib => lib.inbound_count > 0 || lib.inboundCount > 0).length },
            { id: 'staging', label: 'Staging', count: libraries.filter(lib => lib.staging_count > 0 || lib.stagingCount > 0).length },
            { id: 'production', label: 'Production', count: libraries.filter(lib => lib.production_count > 0 || lib.productionCount > 0).length },
            { id: 'quarantine', label: 'Quarantine', count: libraries.filter(lib => lib.quarantine_count > 0 || lib.quarantineCount > 0).length }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-indigo-500 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label} {tab.count > 0 && <span className="bg-gray-100 text-gray-800 text-xs font-medium px-2 py-0.5 rounded-full ml-1">{tab.count}</span>}
            </button>
          ))}
        </nav>
      </div>

      {activeTab === 'overview' && (
        <>
          <div className="mb-6 p-4 bg-white rounded shadow">
            <h2 className="text-xl font-semibold mb-4">Library Controls</h2>
            <div className="flex flex-wrap gap-4">
              <button
                onClick={handleScan}
                className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
              >
                Scan Libraries
              </button>
              <button
                onClick={handleProcess}
                className="bg-purple-500 text-white px-4 py-2 rounded hover:bg-purple-600"
              >
                Process Inbound → Staging
              </button>
              <button
                onClick={handlePromote}
                className="bg-green-500 text-white px-4 py-2 rounded hover:bg-green-600"
              >
                Promote OK Albums to Production
              </button>
            </div>

            {message && (
              <div className="mt-4 p-3 bg-blue-100 text-blue-800 rounded">
                {message}
              </div>
            )}
          </div>

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

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-white p-4 rounded shadow">
              <h3 className="font-semibold">Inbound Files</h3>
              <p className="text-2xl">{stats.inbound_count || stats.inboundCount || 0}</p>
            </div>
            <div className="bg-white p-4 rounded shadow">
              <h3 className="font-semibold">Staging Files</h3>
              <p className="text-2xl">{stats.staging_count || stats.stagingCount || 0}</p>
            </div>
            <div className="bg-white p-4 rounded shadow">
              <h3 className="font-semibold">Production Files</h3>
              <p className="text-2xl">{stats.production_count || stats.productionCount || 0}</p>
            </div>
            <div className="bg-white p-4 rounded shadow">
              <h3 className="font-semibold">Total Duration</h3>
              <p className="text-2xl">{formatDuration(stats.total_duration || stats.totalDuration || 0)}</p>
            </div>
          </div>

          <div className="mt-6 bg-white p-4 rounded shadow">
            <h2 className="text-xl font-semibold mb-2">Library Status</h2>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span>Inbound Path:</span>
                <span>{stats.inbound_path || stats.inboundPath || '/melodee/inbound'}</span>
              </div>
              <div className="flex justify-between">
                <span>Staging Path:</span>
                <span>{stats.staging_path || stats.stagingPath || '/melodee/staging'}</span>
              </div>
              <div className="flex justify-between">
                <span>Production Path:</span>
                <span>{stats.production_path || stats.productionPath || '/melodee/storage'}</span>
              </div>
            </div>
          </div>
        </>
      )}

      {(activeTab !== 'overview') && (
        <div className="bg-white rounded shadow overflow-hidden">
          <div className="p-4 border-b border-gray-200">
            <h2 className="text-xl font-semibold capitalize">{activeTab} Libraries</h2>
          </div>

          {filteredLibraries.length > 0 ? (
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Library</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Inbound</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Staging</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Production</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Quarantine</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredLibraries.map((lib) => (
                  <tr key={lib.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">{lib.name}</div>
                      <div className="text-sm text-gray-500">{lib.path}</div>
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
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {lib.inbound_count || lib.inboundCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {lib.staging_count || lib.stagingCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {lib.production_count || lib.productionCount || 0}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {lib.quarantine_count || lib.quarantineCount || 0}
                      {lib.quarantine_count > 0 && lib.quarantineCount > 0 && (
                        <button
                          onClick={() => setActiveTab('quarantine')}
                          className="ml-2 text-blue-600 hover:text-blue-900 underline text-xs"
                        >
                          View Quarantine Items
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="text-center py-8 text-gray-500">
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