import React, { useState } from 'react';

function SettingsManagement() {
  const [settings, setSettings] = useState({
    transcoding: {
      enabled: true,
      maxBitRate: 320,
      profiles: {
        high: '-c:a libmp3lame -b:a 320k -ar 44100 -ac 2',
        mid: '-c:a libmp3lame -b:a 192k -ar 44100 -ac 2',
        opus_mobile: '-c:a libopus -b:a 96k -application audio'
      }
    },
    libraries: {
      scanInterval: 3600, // in seconds
      capacityThresholds: {
        warn: 80.0,
        alert: 90.0
      }
    },
    capacity: {
      probeInterval: 600 // in seconds
    }
  });

  const [saving, setSaving] = useState(false);

  const handleTranscodingEnabledChange = (enabled) => {
    setSettings(prev => ({
      ...prev,
      transcoding: {
        ...prev.transcoding,
        enabled: enabled
      }
    }));
  };

  const handleMaxBitrateChange = (maxBitRate) => {
    setSettings(prev => ({
      ...prev,
      transcoding: {
        ...prev.transcoding,
        maxBitRate: parseInt(maxBitRate) || 0
      }
    }));
  };

  const handleProfileChange = (profile, value) => {
    setSettings(prev => ({
      ...prev,
      transcoding: {
        ...prev.transcoding,
        profiles: {
          ...prev.transcoding.profiles,
          [profile]: value
        }
      }
    }));
  };

  const handleLibraryIntervalChange = (scanInterval) => {
    setSettings(prev => ({
      ...prev,
      libraries: {
        ...prev.libraries,
        scanInterval: parseInt(scanInterval) || 0
      }
    }));
  };

  const handleLibraryThresholdChange = (threshold, value) => {
    setSettings(prev => ({
      ...prev,
      libraries: {
        ...prev.libraries,
        capacityThresholds: {
          ...prev.libraries.capacityThresholds,
          [threshold]: parseFloat(value) || 0
        }
      }
    }));
  };

  const handleCapacityIntervalChange = (probeInterval) => {
    setSettings(prev => ({
      ...prev,
      capacity: {
        ...prev.capacity,
        probeInterval: parseInt(probeInterval) || 0
      }
    }));
  };

  const handleSave = async () => {
    setSaving(true);
    // Simulate saving settings
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSaving(false);
    alert('Settings saved successfully!');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Settings Management</h1>
      
      <div className="space-y-6">
        {/* Transcoding Settings */}
        <div className="bg-white p-4 rounded shadow">
          <h2 className="text-xl font-semibold mb-4">Transcoding Settings</h2>
          
          <div className="flex items-center mb-4">
            <input
              type="checkbox"
              id="transcoding-enabled"
              checked={settings.transcoding.enabled}
              onChange={(e) => handleTranscodingEnabledChange(e.target.checked)}
              className="mr-2"
            />
            <label htmlFor="transcoding-enabled" className="text-gray-700">Enable Transcoding</label>
          </div>
          
          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2">Max Bitrate (kbps)</label>
            <input
              type="number"
              value={settings.transcoding.maxBitRate}
              onChange={(e) => handleMaxBitrateChange(e.target.value)}
              className="shadow appearance-none border rounded w-32 py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            />
          </div>
          
          <div className="space-y-4">
            <h3 className="font-medium">FFmpeg Profiles:</h3>
            
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-1">High Quality</label>
              <input
                type="text"
                value={settings.transcoding.profiles.high}
                onChange={(e) => handleProfileChange('high', e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
            
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-1">Mid Quality</label>
              <input
                type="text"
                value={settings.transcoding.profiles.mid}
                onChange={(e) => handleProfileChange('mid', e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
            
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-1">Opus Mobile</label>
              <input
                type="text"
                value={settings.transcoding.profiles.opus_mobile}
                onChange={(e) => handleProfileChange('opus_mobile', e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
          </div>
        </div>
        
        {/* Library Settings */}
        <div className="bg-white p-4 rounded shadow">
          <h2 className="text-xl font-semibold mb-4">Library Settings</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-2">Scan Interval (seconds)</label>
              <input
                type="number"
                value={settings.libraries.scanInterval}
                onChange={(e) => handleLibraryIntervalChange(e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
            
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-2">Capacity Warning Threshold (%)</label>
              <input
                type="number"
                step="0.1"
                value={settings.libraries.capacityThresholds.warn}
                onChange={(e) => handleLibraryThresholdChange('warn', e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
            
            <div>
              <label className="block text-gray-700 text-sm font-bold mb-2">Capacity Alert Threshold (%)</label>
              <input
                type="number"
                step="0.1"
                value={settings.libraries.capacityThresholds.alert}
                onChange={(e) => handleLibraryThresholdChange('alert', e.target.value)}
                className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              />
            </div>
          </div>
        </div>
        
        {/* Capacity Settings */}
        <div className="bg-white p-4 rounded shadow">
          <h2 className="text-xl font-semibold mb-4">Capacity Probe Settings</h2>
          
          <div>
            <label className="block text-gray-700 text-sm font-bold mb-2">Probe Interval (seconds)</label>
            <input
              type="number"
              value={settings.capacity.probeInterval}
              onChange={(e) => handleCapacityIntervalChange(e.target.value)}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            />
          </div>
        </div>
        
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </div>
    </div>
  );
}

export default SettingsManagement;