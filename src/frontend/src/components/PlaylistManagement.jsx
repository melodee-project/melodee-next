import React, { useState, useEffect } from 'react';
import { playlistService, searchService } from '../services/apiService';

function PlaylistManagement() {
  const [playlists, setPlaylists] = useState([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState(null);
  const [playlistSongs, setPlaylistSongs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [newPlaylist, setNewPlaylist] = useState({ name: '', comment: '', public: false });
  const [editPlaylist, setEditPlaylist] = useState({ id: null, name: '', comment: '', public: false });
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);

      // Load playlists
      const playlistsResponse = await playlistService.getPlaylists();
      setPlaylists(playlistsResponse.data.data || playlistsResponse.data.playlists || []);

    } catch (error) {
      console.error('Error fetching data:', error);
      setMessage('Error fetching playlists: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePlaylist = async () => {
    try {
      await playlistService.createPlaylist(newPlaylist);
      setMessage('Playlist created successfully!');
      setShowCreateModal(false);
      setNewPlaylist({ name: '', comment: '', public: false });
      fetchData(); // Refresh data
    } catch (error) {
      console.error('Error creating playlist:', error);
      setMessage('Error creating playlist: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleUpdatePlaylist = async () => {
    try {
      await playlistService.updatePlaylist(editPlaylist.id, {
        name: editPlaylist.name,
        comment: editPlaylist.comment,
        public: editPlaylist.public
      });
      setMessage('Playlist updated successfully!');
      setShowEditModal(false);
      fetchData(); // Refresh data
    } catch (error) {
      console.error('Error updating playlist:', error);
      setMessage('Error updating playlist: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleDeletePlaylist = async (id) => {
    if (window.confirm('Are you sure you want to delete this playlist?')) {
      try {
        await playlistService.deletePlaylist(id);
        setMessage('Playlist deleted successfully!');
        fetchData(); // Refresh data
      } catch (error) {
        console.error('Error deleting playlist:', error);
        setMessage('Error deleting playlist: ' + (error.response?.data?.error || error.message));
      }
    }
  };

  const handleViewPlaylist = async (id) => {
    try {
      const response = await playlistService.getPlaylistById(id);
      const playlist = response.data.data || response.data.playlist || {};
      setSelectedPlaylist(playlist);

      // Load the songs in the playlist
      setPlaylistSongs(playlist.entries || playlist.Entries || []);
    } catch (error) {
      console.error('Error fetching playlist:', error);
      setMessage('Error loading playlist: ' + (error.response?.data?.error || error.message));
    }
  };

  const filteredPlaylists = playlists.filter(playlist => 
    (playlist.name || playlist.Name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (playlist.comment || playlist.Comment || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (playlist.owner || playlist.Owner || '').toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading && playlists.length === 0) {
    return <div className="p-4">Loading playlists...</div>;
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Playlist Management</h1>

      {message && (
        <div className={`mb-4 p-3 rounded ${
          message.toLowerCase().includes('error') ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
        }`}>
          {message}
        </div>
      )}

      {/* Controls and Search */}
      <div className="mb-6 p-4 bg-white rounded shadow">
        <h2 className="text-lg font-semibold mb-3">Playlist Management</h2>
        <div className="flex flex-col md:flex-row md:items-center gap-4">
          <div className="flex-1">
            <input
              type="text"
              placeholder="Search playlists..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
          >
            Create Playlist
          </button>
        </div>
      </div>

      {/* Playlists Grid/List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredPlaylists.length > 0 ? (
          filteredPlaylists.map((playlist) => (
            <div key={playlist.id || playlist.ID} className="bg-white rounded shadow overflow-hidden">
              <div className="p-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-semibold text-lg truncate">{playlist.name || playlist.Name || 'Unnamed Playlist'}</h3>
                    <p className="text-sm text-gray-600 truncate">{playlist.comment || playlist.Comment}</p>
                    <p className="text-xs text-gray-500 mt-1">
                      Owner: {playlist.owner || playlist.Owner || 'Unknown'} • 
                      Songs: {playlist.song_count || playlist.SongCount || 0} • 
                      {playlist.public || playlist.Public ? ' Public' : ' Private'}
                    </p>
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => {
                        setEditPlaylist({
                          id: playlist.id || playlist.ID,
                          name: playlist.name || playlist.Name || '',
                          comment: playlist.comment || playlist.Comment || '',
                          public: playlist.public || playlist.Public || false
                        });
                        setShowEditModal(true);
                      }}
                      className="text-blue-600 hover:text-blue-900"
                      title="Edit Playlist"
                    >
                      ✏️
                    </button>
                    <button
                      onClick={() => handleDeletePlaylist(playlist.id || playlist.ID)}
                      className="text-red-600 hover:text-red-900"
                      title="Delete Playlist"
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                <div className="mt-3 flex justify-between">
                  <button
                    onClick={() => handleViewPlaylist(playlist.id || playlist.ID)}
                    className="text-indigo-600 hover:text-indigo-900 text-sm"
                  >
                    View Details
                  </button>
                  <span className="text-xs text-gray-500">
                    {playlist.created_at || playlist.CreatedAt ? new Date(playlist.created_at || playlist.CreatedAt).toLocaleDateString() : 'N/A'}
                  </span>
                </div>
              </div>
            </div>
          ))
        ) : (
          <div className="col-span-full text-center py-8 text-gray-500">
            {loading ? 'Loading playlists...' : 'No playlists found.'}
          </div>
        )}
      </div>

      {/* Selected Playlist Detail View */}
      {selectedPlaylist && (
        <div className="mt-6 bg-white rounded shadow p-4">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold">
              {selectedPlaylist.name || selectedPlaylist.Name} ({playlistSongs.length} songs)
            </h2>
            <button
              onClick={() => setSelectedPlaylist(null)}
              className="text-gray-500 hover:text-gray-700"
            >
              Close
            </button>
          </div>

          <div className="mb-4">
            <p className="text-gray-600 mb-2">{selectedPlaylist.comment || selectedPlaylist.Comment}</p>
            <div className="text-sm text-gray-500">
              <div>Owner: {selectedPlaylist.owner || selectedPlaylist.Owner}</div>
              <div>Created: {selectedPlaylist.created_at || selectedPlaylist.CreatedAt ? new Date(selectedPlaylist.created_at || selectedPlaylist.CreatedAt).toLocaleString() : 'N/A'}</div>
              <div>Changed: {selectedPlaylist.changed_at || selectedPlaylist.ChangedAt ? new Date(selectedPlaylist.changed_at || selectedPlaylist.ChangedAt).toLocaleString() : 'N/A'}</div>
              <div>Status: {selectedPlaylist.public || selectedPlaylist.Public ? 'Public' : 'Private'}</div>
            </div>
          </div>

          {/* Playlist Songs */}
          <div>
            <h3 className="text-lg font-semibold mb-2">Playlist Contents</h3>
            {playlistSongs.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">#</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Title</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Artist</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Album</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {playlistSongs.map((song, index) => (
                      <tr key={song.id || song.ID || index}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{index + 1}</td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{song.title || song.Title || 'Unknown'}</td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{song.artist || song.Artist || 'Unknown'}</td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{song.album || song.Album || 'Unknown'}</td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {song.duration ? `${Math.floor(song.duration / 60)}:${(song.duration % 60).toString().padStart(2, '0')}` : 'N/A'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="text-gray-500 text-center py-4">
                No songs in this playlist.
              </div>
            )}
          </div>
        </div>
      )}

      {/* Create Playlist Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h2 className="text-xl font-semibold mb-4">Create New Playlist</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
                <input
                  type="text"
                  value={newPlaylist.name}
                  onChange={(e) => setNewPlaylist({...newPlaylist, name: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder="Enter playlist name"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Comment</label>
                <textarea
                  value={newPlaylist.comment}
                  onChange={(e) => setNewPlaylist({...newPlaylist, comment: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder="Enter playlist description"
                  rows="3"
                />
              </div>
              
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="publicCheckbox"
                  checked={newPlaylist.public}
                  onChange={(e) => setNewPlaylist({...newPlaylist, public: e.target.checked})}
                  className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                />
                <label htmlFor="publicCheckbox" className="ml-2 block text-sm text-gray-900">
                  Make public
                </label>
              </div>
            </div>
            
            <div className="mt-6 flex justify-end space-x-3">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreatePlaylist}
                className="px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Playlist Modal */}
      {showEditModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h2 className="text-xl font-semibold mb-4">Edit Playlist</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
                <input
                  type="text"
                  value={editPlaylist.name}
                  onChange={(e) => setEditPlaylist({...editPlaylist, name: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Comment</label>
                <textarea
                  value={editPlaylist.comment}
                  onChange={(e) => setEditPlaylist({...editPlaylist, comment: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                  rows="3"
                />
              </div>
              
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="editPublicCheckbox"
                  checked={editPlaylist.public}
                  onChange={(e) => setEditPlaylist({...editPlaylist, public: e.target.checked})}
                  className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                />
                <label htmlFor="editPublicCheckbox" className="ml-2 block text-sm text-gray-900">
                  Make public
                </label>
              </div>
            </div>
            
            <div className="mt-6 flex justify-end space-x-3">
              <button
                onClick={() => setShowEditModal(false)}
                className="px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleUpdatePlaylist}
                className="px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
              >
                Update
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default PlaylistManagement;