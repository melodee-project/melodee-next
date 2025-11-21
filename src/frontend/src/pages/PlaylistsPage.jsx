import React from 'react';

const PlaylistsPage = () => {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Playlists</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="p-4">
          <h2 className="text-xl font-semibold mb-4">Your Playlists</h2>
          <p>Your playlists will be displayed here.</p>
        </div>
      </div>
    </div>
  );
};

export default PlaylistsPage;