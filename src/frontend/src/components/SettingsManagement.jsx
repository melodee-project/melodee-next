import React, { useState, useEffect } from 'react';
import { adminService } from '../services/apiService';

// Settings documentation
const SETTINGS_DOCS = {
  'processing.scan_workers': {
    description: 'Number of concurrent workers for library directory scanning',
    defaultValue: '8',
    validRange: '1-32',
    notes: 'Increase to 16-32 for NVMe SSDs. Decrease to 4-6 for HDDs or network storage.',
  },
  'processing.scan_buffer_size': {
    description: 'Buffer size for scan file channel',
    defaultValue: '1000',
    validRange: '100-10000',
    notes: 'Increase for very large directories to reduce synchronization overhead. Decrease if memory is limited.',
  },
  'processing.scan_max_files': {
    description: 'Maximum number of files to scan (0 = no limit)',
    defaultValue: '0',
    validRange: '0 or any positive number',
    notes: 'Useful for troubleshooting. Set to small values like 10-100 to test scanning quickly.',
  },
};

function SettingsManagement() {
  const [settings, setSettings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editingSetting, setEditingSetting] = useState(null);
  const [newValue, setNewValue] = useState('');
  const [newSetting, setNewSetting] = useState({ key: '', value: '' });
  const [showDocs, setShowDocs] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

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

  // Filter settings based on search term
  const filteredSettings = settings.filter((setting) => {
    if (!searchTerm) return true;
    
    const key = (setting.key || setting.Key || '').toLowerCase();
    const value = (setting.value || setting.Value || setting.setting_value || '').toLowerCase();
    const comment = (setting.comment || setting.Comment || '').toLowerCase();
    const search = searchTerm.toLowerCase();
    
    // Also check documentation if available
    const settingKey = setting.key || setting.Key;
    const doc = SETTINGS_DOCS[settingKey];
    const docText = doc ? `${doc.description} ${doc.notes}`.toLowerCase() : '';
    
    return key.includes(search) || 
           value.includes(search) || 
           comment.includes(search) ||
           docText.includes(search);
  });

  // Filter documentation based on search term
  const filteredDocs = Object.entries(SETTINGS_DOCS).filter(([key, doc]) => {
    if (!searchTerm) return true;
    
    const search = searchTerm.toLowerCase();
    return key.toLowerCase().includes(search) ||
           doc.description.toLowerCase().includes(search) ||
           doc.notes.toLowerCase().includes(search) ||
           doc.defaultValue.toLowerCase().includes(search) ||
           doc.validRange.toLowerCase().includes(search);
  });

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Settings Management</h1>
        <button
          onClick={() => setShowDocs(!showDocs)}
          className="bg-blue-500 dark:bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-600 dark:hover:bg-blue-700 flex items-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {showDocs ? 'Hide' : 'Show'} Documentation
        </button>
      </div>

      {/* Search Bar */}
      <div className="mb-6">
        <div className="relative">
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search settings by key, value, or description..."
            className="w-full px-4 py-3 pl-10 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <svg
            className="absolute left-3 top-3.5 w-5 h-5 text-gray-400 dark:text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          {searchTerm && (
            <button
              onClick={() => setSearchTerm('')}
              className="absolute right-3 top-3 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
        {searchTerm && (
          <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Found {filteredSettings.length} setting(s) and {filteredDocs.length} documentation entr{filteredDocs.length === 1 ? 'y' : 'ies'}
          </div>
        )}
      </div>

      {/* Documentation Panel */}
      {showDocs && (
        <div className="mb-6 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-4 text-blue-900 dark:text-blue-300">Configuration Settings Documentation</h2>
          {filteredDocs.length > 0 ? (
            <div className="space-y-4">
              {filteredDocs.map(([key, doc]) => (
              <div key={key} className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-blue-100 dark:border-gray-700">
                <div className="flex items-start gap-3">
                  <div className="flex-shrink-0 w-2 h-2 bg-blue-500 rounded-full mt-2"></div>
                  <div className="flex-1">
                    <h3 className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">{key}</h3>
                    <p className="text-gray-700 dark:text-gray-300 mb-2">{doc.description}</p>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
                      <div>
                        <span className="font-semibold text-gray-600 dark:text-gray-400">Default:</span>
                        <span className="ml-2 font-mono bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100 px-2 py-1 rounded">{doc.defaultValue}</span>
                      </div>
                      <div>
                        <span className="font-semibold text-gray-600 dark:text-gray-400">Valid Range:</span>
                        <span className="ml-2 font-mono bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100 px-2 py-1 rounded">{doc.validRange}</span>
                      </div>
                    </div>
                    {doc.notes && (
                      <div className="mt-2 text-sm text-gray-600 dark:text-gray-300 italic bg-yellow-50 dark:bg-yellow-900/20 p-2 rounded border-l-4 border-yellow-400 dark:border-yellow-600">
                        <span className="font-semibold not-italic">💡 Tip:</span> {doc.notes}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No documentation entries match "{searchTerm}"
          </div>
        )}
          <div className="mt-4 p-3 bg-blue-100 dark:bg-blue-900/30 rounded text-sm text-blue-900 dark:text-blue-200">
            <span className="font-semibold">Note:</span> Changes to these settings take effect on the next library scan without requiring a service restart.
          </div>
        </div>
      )}

      <div className="mb-6 p-4 bg-white dark:bg-gray-800 rounded shadow border border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-semibold mb-3 text-gray-900 dark:text-gray-100">Create New Setting</h2>
        <form onSubmit={handleCreateSetting} className="flex flex-col sm:flex-row gap-2">
          <div className="flex-1">
            <input
              type="text"
              value={newSetting.key}
              onChange={(e) => setNewSetting({...newSetting, key: e.target.value})}
              placeholder="Setting key"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              required
            />
          </div>
          <div className="flex-1">
            <input
              type="text"
              value={newSetting.value}
              onChange={(e) => setNewSetting({...newSetting, value: e.target.value})}
              placeholder="Setting value"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              required
            />
          </div>
          <button
            type="submit"
            className="px-3 py-2 rounded-md text-sm font-semibold bg-green-600 hover:bg-green-700 text-white transition-colors"
          >
            Add Setting
          </button>
        </form>
      </div>

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Key</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Value</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Category</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
            {filteredSettings.map((setting) => (
              <tr key={setting.id || setting.ID || setting.key || setting.Key}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                  {setting.key || setting.Key || setting.id || setting.ID || 'N/A'}
                </td>
                <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                  {editingSetting === (setting.key || setting.Key || setting.id || setting.ID) ? (
                    <input
                      type="text"
                      value={newValue}
                      onChange={(e) => setNewValue(e.target.value)}
                      className="border border-gray-300 dark:border-gray-600 rounded p-1 w-full bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                      autoFocus
                    />
                  ) : (
                    <div>
                      <div className="font-mono text-gray-900 dark:text-gray-100">{setting.value || setting.Value || setting.setting_value || 'N/A'}</div>
                      {SETTINGS_DOCS[setting.key || setting.Key] && (
                        <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          {SETTINGS_DOCS[setting.key || setting.Key].description}
                        </div>
                      )}
                    </div>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {setting.category || setting.Category || setting.category_id || 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  {editingSetting === (setting.key || setting.Key || setting.id || setting.ID) ? (
                    <>
                      <button
                        onClick={() => handleSave(setting.key || setting.Key || setting.id || setting.ID)}
                        className="mr-2 px-3 py-1.5 rounded-md text-sm font-semibold bg-green-600 hover:bg-green-700 text-white transition-colors"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingSetting(null)}
                        className="px-3 py-1.5 rounded-md text-sm font-semibold bg-gray-600 hover:bg-gray-700 text-white transition-colors"
                      >
                        Cancel
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => handleEditClick(setting)}
                      className="px-3 py-1.5 rounded-md text-sm font-semibold bg-blue-600 hover:bg-blue-700 text-white transition-colors"
                    >
                      Edit
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {filteredSettings.length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            {searchTerm ? `No settings match "${searchTerm}"` : 'No settings found.'}
          </div>
        )}
      </div>
    </div>
  );
}

export default SettingsManagement;