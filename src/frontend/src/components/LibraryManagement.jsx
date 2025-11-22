import React, { useState, useEffect } from 'react';
import { libraryService } from '../services/apiService';

function LibraryManagement() {
  const [stats, setStats] = useState({});
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetchLibraryStats();
  }, []);

  const fetchLibraryStats = async () => {
    try {
      const response = await libraryService.getStats();
      setStats(response.data.data || response.data || {});
    } catch (error) {
      console.error('Error fetching library stats:', error);
      setStats({});
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

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Library Management</h1>

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