import React from 'react';

const ProfilePage = () => {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Profile</h1>
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">User Profile</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-gray-700 font-medium mb-1">Username</label>
            <p className="text-gray-900">Loading...</p>
          </div>
          <div>
            <label className="block text-gray-700 font-medium mb-1">Email</label>
            <p className="text-gray-900">Loading...</p>
          </div>
          <div>
            <label className="block text-gray-700 font-medium mb-1">Admin</label>
            <p className="text-gray-900">Loading...</p>
          </div>
          <div className="mt-6">
            <button className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
              Update Profile
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProfilePage;