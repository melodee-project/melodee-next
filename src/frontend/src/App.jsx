import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useNavigate, Navigate } from 'react-router-dom';
import axios from 'axios';
import apiService from './services/apiService';
import LoginPage from './pages/LoginPage';
import AdminDashboard from './components/AdminDashboard';
import DLQManagement from './components/DLQManagement';
import UserManagement from './components/UserManagement';
import SettingsManagement from './components/SettingsManagement';

// Check if user is authenticated
function isAuthenticated() {
  return localStorage.getItem('accessToken') !== null;
}

// ProtectedRoute component to restrict access to authenticated users
function ProtectedRoute({ children }) {
  return isAuthenticated() ? children : <Navigate to="/login" />;
}

// AdminRoute component to restrict access to admin users
function AdminRoute({ children }) {
  const userIsAdmin = localStorage.getItem('userIsAdmin') === 'true';
  return isAuthenticated() && userIsAdmin ? children : <Navigate to="/login" />;
}

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-100">
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/admin" element={
            <ProtectedRoute>
              <AdminDashboard />
            </ProtectedRoute>
          } />
          <Route path="/admin/dlq" element={
            <ProtectedRoute>
              <DLQManagement />
            </ProtectedRoute>
          } />
          <Route path="/admin/users" element={
            <ProtectedRoute>
              <UserManagement />
            </ProtectedRoute>
          } />
          <Route path="/admin/settings" element={
            <ProtectedRoute>
              <SettingsManagement />
            </ProtectedRoute>
          } />
          <Route path="/" element={
            isAuthenticated() ? <Navigate to="/admin" /> : <Navigate to="/login" />
          } />
        </Routes>
      </div>
    </Router>
  );
}

export default App;

// Configure axios defaults
axios.defaults.baseURL = API_BASE_URL;
axios.defaults.withCredentials = true;

// Admin Dashboard Component
function AdminDashboard() {
  const [stats, setStats] = useState({});
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      // Fetch library stats
      const statsResponse = await axios.get('/stats');
      setStats(statsResponse.data);
      
      // Fetch recent jobs
      const jobsResponse = await axios.get('/admin/jobs/recent');
      setJobs(jobsResponse.data);
      
      setLoading(false);
    } catch (error) {
      console.error('Error fetching dashboard data:', error);
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="p-4">Loading dashboard...</div>;
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Admin Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Artists</h3>
          <p className="text-2xl">{stats.totalArtists || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Albums</h3>
          <p className="text-2xl">{stats.totalAlbums || 0}</p>
        </div>
        <div className="bg-white p-4 rounded shadow">
          <h3 className="font-semibold">Total Songs</h3>
          <p className="text-2xl">{stats.totalSongs || 0}</p>
        </div>
      </div>
      
      <div className="bg-white p-4 rounded shadow">
        <h2 className="text-xl font-semibold mb-2">Recent Jobs</h2>
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Job</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Started</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {jobs.map((job, index) => (
              <tr key={index}>
                <td className="px-6 py-4 whitespace-nowrap">{job.type}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                    ${job.status === 'completed' ? 'bg-green-100 text-green-800' : 
                      job.status === 'failed' ? 'bg-red-100 text-red-800' : 
                      'bg-yellow-100 text-yellow-800'}`}>
                    {job.status}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">{job.duration}s</td>
                <td className="px-6 py-4 whitespace-nowrap">{new Date(job.startedAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// DLQ Management Component
function DLQManagement() {
  const [dlqItems, setDlqItems] = useState([]);
  const [selectedItems, setSelectedItems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDLQItems();
  }, []);

  const fetchDLQItems = async () => {
    try {
      const response = await axios.get('/admin/jobs/dlq');
      setDlqItems(response.data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching DLQ items:', error);
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
    try {
      await axios.post('/admin/jobs/dlq/requeue', { job_ids: selectedItems });
      // Refresh the list
      fetchDLQItems();
      setSelectedItems([]);
    } catch (error) {
      console.error('Error requeuing items:', error);
    }
  };

  const handlePurgeSelected = async () => {
    try {
      await axios.post('/admin/jobs/dlq/purge', { job_ids: selectedItems });
      // Refresh the list
      fetchDLQItems();
      setSelectedItems([]);
    } catch (error) {
      console.error('Error purging items:', error);
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
            Requeue Selected
          </button>
          <button 
            onClick={handlePurgeSelected}
            disabled={selectedItems.length === 0}
            className="bg-red-500 text-white px-4 py-2 rounded disabled:opacity-50"
          >
            Purge Selected
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
                      setSelectedItems(dlqItems.map(item => item.id));
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
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Error</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {dlqItems.map((item) => (
              <tr key={item.id}>
                <td className="px-6 py-4 whitespace-nowrap">
                  <input
                    type="checkbox"
                    checked={selectedItems.includes(item.id)}
                    onChange={() => handleItemSelect(item.id)}
                  />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{item.queue}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{item.type}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{item.reason}</td>
                <td className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">{item.error_message}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// User Management Component
function UserManagement() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newUser, setNewUser] = useState({ username: '', email: '', password: '', is_admin: false });

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const response = await axios.get('/users');
      setUsers(response.data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching users:', error);
      setLoading(false);
    }
  };

  const handleCreateUser = async (e) => {
    e.preventDefault();
    
    try {
      await axios.post('/users', newUser);
      setNewUser({ username: '', email: '', password: '', is_admin: false });
      setShowCreateForm(false);
      fetchUsers(); // Refresh the list
    } catch (error) {
      console.error('Error creating user:', error);
    }
  };

  const handleToggleAdmin = async (userId, currentAdminStatus) => {
    try {
      const user = users.find(u => u.id === userId);
      await axios.put(`/users/${userId}`, { 
        is_admin: !currentAdminStatus,
        username: user.username,
        email: user.email
      });
      fetchUsers(); // Refresh the list
    } catch (error) {
      console.error('Error updating user:', error);
    }
  };

  const handleDeleteUser = async (userId) => {
    if (window.confirm('Are you sure you want to delete this user?')) {
      try {
        await axios.delete(`/users/${userId}`);
        fetchUsers(); // Refresh the list
      } catch (error) {
        console.error('Error deleting user:', error);
      }
    }
  };

  if (loading) {
    return <div className="p-4">Loading users...</div>;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">User Management</h1>
        <button 
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-500 text-white px-4 py-2 rounded"
        >
          {showCreateForm ? 'Cancel' : '+ Add User'}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreateUser} className="mb-6 p-4 bg-white rounded shadow">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Username</label>
              <input
                type="text"
                value={newUser.username}
                onChange={(e) => setNewUser({...newUser, username: e.target.value})}
                className="mt-1 block w-full border border-gray-300 rounded-md p-2"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Email</label>
              <input
                type="email"
                value={newUser.email}
                onChange={(e) => setNewUser({...newUser, email: e.target.value})}
                className="mt-1 block w-full border border-gray-300 rounded-md p-2"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Password</label>
              <input
                type="password"
                value={newUser.password}
                onChange={(e) => setNewUser({...newUser, password: e.target.value})}
                className="mt-1 block w-full border border-gray-300 rounded-md p-2"
                required
              />
            </div>
            <div className="flex items-center">
              <input
                type="checkbox"
                checked={newUser.is_admin}
                onChange={(e) => setNewUser({...newUser, is_admin: e.target.checked})}
                className="mr-2"
              />
              <label className="block text-sm font-medium text-gray-700">Admin User</label>
            </div>
          </div>
          <button type="submit" className="mt-4 bg-green-500 text-white px-4 py-2 rounded">
            Create User
          </button>
        </form>
      )}

      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Username</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Admin</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {users.map((user) => (
              <tr key={user.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{user.username}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{user.email}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {user.is_admin ? 'Yes' : 'No'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  <button
                    onClick={() => handleToggleAdmin(user.id, user.is_admin)}
                    className={`mr-2 ${user.is_admin ? 'text-red-600' : 'text-green-600'}`}
                  >
                    {user.is_admin ? 'Remove Admin' : 'Make Admin'}
                  </button>
                  <button
                    onClick={() => handleDeleteUser(user.id)}
                    className="text-red-600"
                  >
                    Delete
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

// Settings Management Component
function SettingsManagement() {
  const [settings, setSettings] = useState([]);
  const [editingSetting, setEditingSetting] = useState(null);
  const [newValue, setNewValue] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const response = await axios.get('/admin/settings');
      setSettings(response.data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching settings:', error);
      setLoading(false);
    }
  };

  const handleEditClick = (setting) => {
    setEditingSetting(setting.key);
    setNewValue(setting.value);
  };

  const handleSave = async (key) => {
    try {
      await axios.put(`/admin/settings/${key}`, { value: newValue });
      setEditingSetting(null);
      fetchSettings(); // Refresh the list
    } catch (error) {
      console.error('Error updating setting:', error);
    }
  };

  if (loading) {
    return <div className="p-4">Loading settings...</div>;
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Settings Management</h1>
      
      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Key</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Value</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Category</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {settings.map((setting) => (
              <tr key={setting.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{setting.key}</td>
                <td className="px-6 py-4 text-sm text-gray-500">
                  {editingSetting === setting.key ? (
                    <input
                      type="text"
                      value={newValue}
                      onChange={(e) => setNewValue(e.target.value)}
                      className="border border-gray-300 rounded p-1"
                    />
                  ) : (
                    setting.value
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{setting.category}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  {editingSetting === setting.key ? (
                    <>
                      <button
                        onClick={() => handleSave(setting.key)}
                        className="text-green-600 mr-2"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingSetting(null)}
                        className="text-gray-600"
                      >
                        Cancel
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => handleEditClick(setting)}
                      className="text-blue-600"
                    >
                      Edit
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// Main App Component
function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-100">
        {/* Navigation */}
        <nav className="bg-blue-600 text-white p-4">
          <div className="container mx-auto">
            <div className="flex justify-between items-center">
              <h1 className="text-xl font-bold">Melodee Admin</h1>
              <ul className="flex space-x-4">
                <li><Link to="/admin" className="hover:underline">Dashboard</Link></li>
                <li><Link to="/admin/dlq" className="hover:underline">DLQ</Link></li>
                <li><Link to="/admin/users" className="hover:underline">Users</Link></li>
                <li><Link to="/admin/settings" className="hover:underline">Settings</Link></li>
                <li><Link to="/admin/shares" className="hover:underline">Shares</Link></li>
              </ul>
            </div>
          </div>
        </nav>

        {/* Main Content */}
        <main className="container mx-auto mt-4">
          <Routes>
            <Route path="/admin" element={<AdminDashboard />} />
            <Route path="/admin/dlq" element={<DLQManagement />} />
            <Route path="/admin/users" element={<UserManagement />} />
            <Route path="/admin/settings" element={<SettingsManagement />} />
            <Route 
              path="/admin/shares" 
              element={
                <div className="p-4">
                  <h1 className="text-2xl font-bold mb-4">Shares Management</h1>
                  <p>Shares management interface coming soon.</p>
                </div>
              } 
            />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;