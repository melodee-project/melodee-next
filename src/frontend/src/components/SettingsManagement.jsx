import React, { useState, useEffect } from 'react';
import { adminService } from '../services/apiService';

function SettingsManagement() {
  const [settings, setSettings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editingSetting, setEditingSetting] = useState(null);
  const [newValue, setNewValue] = useState('');
  const [newSetting, setNewSetting] = useState({ key: '', value: '' });

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const response = await adminService.getSettings();
      // Handle different response formats that might come from backend
      const settingsList = response.data.data || response.data || [];
      setSettings(settingsList);
    } catch (error) {
      console.error('Error fetching settings:', error);
      alert('Error fetching settings: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const handleEditClick = (setting) => {
    setEditingSetting(setting.key || setting.Key || setting.id || setting.ID);
    setNewValue(setting.value || setting.Value || setting.setting_value || '');
  };

  const handleSave = async (key) => {
    try {
      await adminService.updateSetting(key, newValue);
      setEditingSetting(null);
      fetchSettings(); // Refresh the list
    } catch (error) {
      console.error('Error updating setting:', error);
      alert('Error updating setting: ' + (error.response?.data?.error || error.message));
    }
  };

  const handleCreateSetting = async (e) => {
    e.preventDefault();

    try {
      // The API expects the key and value in the request body
      await adminService.updateSetting(newSetting.key, newSetting.value);
      setNewSetting({ key: '', value: '' });
      fetchSettings(); // Refresh the list
    } catch (error) {
      console.error('Error creating setting:', error);
      alert('Error creating setting: ' + (error.response?.data?.error || error.message));
    }
  };

  if (loading) {
    return <div className="p-4">Loading settings...</div>;
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">Settings Management</h1>

      <div className="mb-6 p-4 bg-white rounded shadow">
        <h2 className="text-lg font-semibold mb-3">Create New Setting</h2>
        <form onSubmit={handleCreateSetting} className="flex flex-col sm:flex-row gap-2">
          <div className="flex-1">
            <input
              type="text"
              value={newSetting.key}
              onChange={(e) => setNewSetting({...newSetting, key: e.target.value})}
              placeholder="Setting key"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div className="flex-1">
            <input
              type="text"
              value={newSetting.value}
              onChange={(e) => setNewSetting({...newSetting, value: e.target.value})}
              placeholder="Setting value"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <button
            type="submit"
            className="bg-green-500 text-white px-4 py-2 rounded hover:bg-green-600"
          >
            Add Setting
          </button>
        </form>
      </div>

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
              <tr key={setting.id || setting.ID || setting.key || setting.Key}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {setting.key || setting.Key || setting.id || setting.ID || 'N/A'}
                </td>
                <td className="px-6 py-4 text-sm text-gray-500">
                  {editingSetting === (setting.key || setting.Key || setting.id || setting.ID) ? (
                    <input
                      type="text"
                      value={newValue}
                      onChange={(e) => setNewValue(e.target.value)}
                      className="border border-gray-300 rounded p-1 w-full"
                      autoFocus
                    />
                  ) : (
                    setting.value || setting.Value || setting.setting_value || 'N/A'
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {setting.category || setting.Category || setting.category_id || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  {editingSetting === (setting.key || setting.Key || setting.id || setting.ID) ? (
                    <>
                      <button
                        onClick={() => handleSave(setting.key || setting.Key || setting.id || setting.ID)}
                        className="text-green-600 mr-2 hover:underline"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingSetting(null)}
                        className="text-gray-600 hover:underline"
                      >
                        Cancel
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => handleEditClick(setting)}
                      className="text-blue-600 hover:underline"
                    >
                      Edit
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {settings.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No settings found.
          </div>
        )}
      </div>
    </div>
  );
}

export default SettingsManagement;