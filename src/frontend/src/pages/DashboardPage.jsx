import React from 'react';

const DashboardPage = () => {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-xl font-semibold mb-2">Library Stats</h2>
          <p>Total Songs: Loading...</p>
          <p>Total Artists: Loading...</p>
          <p>Total Albums: Loading...</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-xl font-semibold mb-2">Recent Activity</h2>
          <p>No recent activity to display</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-xl font-semibold mb-2">Server Status</h2>
          <p>Health: Loading...</p>
        </div>
      </div>
    </div>
  );
};

export default DashboardPage;