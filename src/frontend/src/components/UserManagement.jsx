import React, { useState, useEffect } from 'react';
import { userService } from '../services/apiService';

function UserManagement() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newUser, setNewUser] = useState({
    username: '',
    email: '',
    password: '',
    is_admin: false
  });
  const [errors, setErrors] = useState({});

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const response = await userService.getUsers();
      // Handle different response formats that might come from backend
      const userList = response.data.data || response.data || [];
      setUsers(userList);
    } catch (error) {
      console.error('Error fetching users:', error);
      alert('Error fetching users: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const validateForm = () => {
    const newErrors = {};

    if (!newUser.username.trim()) {
      newErrors.username = 'Username is required';
    }

    if (!newUser.password.trim()) {
      newErrors.password = 'Password is required';
    } else if (newUser.password.length < 12) {
      newErrors.password = 'Password must be at least 12 characters';
    }

    if (newUser.email && !/\S+@\S+\.\S+/.test(newUser.email)) {
      newErrors.email = 'Email is invalid';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleCreateUser = async (e) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      await userService.createUser(newUser);
      setNewUser({ username: '', email: '', password: '', is_admin: false });
      setShowCreateForm(false);
      fetchUsers(); // Refresh the list
    } catch (error) {
      console.error('Error creating user:', error);
      alert('Error creating user: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleToggleAdmin = async (userId, currentAdminStatus) => {
    try {
      // First get the current user data to preserve other fields
      const currentUser = users.find(u => u.id === userId);

      await userService.updateUser(userId, {
        username: currentUser.username,
        email: currentUser.email || '',
        is_admin: !currentAdminStatus
      });
      fetchUsers(); // Refresh the list
    } catch (error) {
      console.error('Error updating user:', error);
      alert('Error updating user: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleDeleteUser = async (userId) => {
    if (window.confirm('Are you sure you want to delete this user?')) {
      try {
        await userService.deleteUser(userId);
        fetchUsers(); // Refresh the list
      } catch (error) {
        console.error('Error deleting user:', error);
        alert('Error deleting user: ' + (error.response?.data?.error || error.message));
      }
    }
  };

  if (loading) {
    return <div className="p-4 text-gray-900 dark:text-gray-100">Loading users...</div>;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">User Management</h1>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-500 dark:bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-600 dark:hover:bg-blue-700"
        >
          {showCreateForm ? 'Cancel' : '+ Add User'}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreateUser} className="mb-6 p-4 bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Username</label>
              <input
                type="text"
                value={newUser.username}
                onChange={(e) => setNewUser({...newUser, username: e.target.value})}
                className={`mt-1 block w-full border ${errors.username ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100`}
                required
              />
              {errors.username && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.username}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Email</label>
              <input
                type="email"
                value={newUser.email}
                onChange={(e) => setNewUser({...newUser, email: e.target.value})}
                className={`mt-1 block w-full border ${errors.email ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100`}
              />
              {errors.email && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.email}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Password</label>
              <input
                type="password"
                value={newUser.password}
                onChange={(e) => setNewUser({...newUser, password: e.target.value})}
                className={`mt-1 block w-full border ${errors.password ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md p-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100`}
                required
              />
              {errors.password && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.password}</p>}
            </div>
            <div className="flex items-center">
              <input
                type="checkbox"
                checked={newUser.is_admin}
                onChange={(e) => setNewUser({...newUser, is_admin: e.target.checked})}
                className="mr-2 h-4 w-4 text-blue-600"
              />
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Admin User</label>
            </div>
          </div>
          <button type="submit" className="mt-4 bg-green-500 dark:bg-green-600 text-white px-4 py-2 rounded hover:bg-green-600 dark:hover:bg-green-700">
            Create User
          </button>
        </form>
      )}

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Username</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Admin</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Last Login</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
            {users.map((user) => (
              <tr key={user.id || user.ID}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                  {user.username || user.Username}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {user.email || user.Email || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {(user.is_admin || user.isAdmin || user.IsAdmin) ? 'Yes' : 'No'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {user.last_login_at || user.lastLoginAt || user.LastLoginAt ?
                    new Date(user.last_login_at || user.lastLoginAt || user.LastLoginAt).toLocaleString() :
                    'Never'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  <button
                    onClick={() => handleToggleAdmin(user.id || user.ID, user.is_admin || user.isAdmin || user.IsAdmin)}
                    className={`mr-2 hover:underline ${user.is_admin || user.isAdmin || user.IsAdmin ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}
                  >
                    {(user.is_admin || user.isAdmin || user.IsAdmin) ? 'Remove Admin' : 'Make Admin'}
                  </button>
                  <button
                    onClick={() => handleDeleteUser(user.id || user.ID)}
                    className="text-red-600 dark:text-red-400 hover:underline"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {users.length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No users found.
          </div>
        )}
      </div>
    </div>
  );
}

export default UserManagement;