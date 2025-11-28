import React, { useEffect, useState } from 'react';
import { adminService } from '../services/apiService';

const OpenSubsonicInfo = () => {
  const [status, setStatus] = useState('checking');
  const [detail, setDetail] = useState('');
  const [base, setBase] = useState('');
  const [source, setSource] = useState('loading...');

  useEffect(() => {
    let cancelled = false;
    
    adminService.getOpenSubsonicInfo()
      .then(resp => {
        if (cancelled) return;
        const url = resp?.data?.base_for_client || window.location.origin;
        setBase(url);
        setSource(resp?.data?.source || 'fallback to local origin');
        
        // Test the endpoint
        return fetch(`${url}/rest/ping.view`);
      })
      .then(res => {
        if (cancelled) return;
        setStatus(res?.ok ? 'online' : 'error');
        setDetail(res?.ok ? 'Ping successful' : `HTTP ${res?.status}`);
      })
      .catch(err => {
        if (cancelled) return;
        setBase(window.location.origin);
        setSource('error fetching from server');
        setStatus('error');
        setDetail(err?.message || 'Network error');
      });

    return () => { cancelled = true; };
  }, []);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(base);
    } catch {
      // no-op
    }
  };

  const badge = {
    checking: { label: 'Checking…', cls: 'bg-gray-100 text-gray-700' },
    online: { label: 'Online', cls: 'bg-green-100 text-green-800' },
    error: { label: 'Unavailable', cls: 'bg-red-100 text-red-800' },
  }[status] || { label: 'Unknown', cls: 'bg-gray-100 text-gray-700' };

  return (
    <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 mb-6">
      <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-3">OpenSubsonic API Endpoint</h2>
      <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
        Enter this URL in your OpenSubsonic client (the client will add <code>/rest</code> automatically).
      </p>
      <div className="flex items-start gap-3 mb-3">
        <code className="px-3 py-2 rounded bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-gray-100 break-all flex-1">
          {base}
        </code>
        <button
          onClick={copyToClipboard}
          className="px-3 py-2 text-sm rounded bg-blue-600 text-white hover:bg-blue-700"
          aria-label="Copy OpenSubsonic endpoint"
        >
          Copy
        </button>
      </div>
      <div className="flex items-center gap-2 text-sm">
        <span className={`px-2 py-1 rounded ${badge.cls}`}>{badge.label}</span>
        <span className="text-gray-500 dark:text-gray-400">{detail}</span>
      </div>
      <div className="mt-3 text-xs text-gray-500 dark:text-gray-400">
        {source}
      </div>
      {base && base.startsWith('http://') && (
        <div className="mt-3 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded text-sm">
          <strong className="text-yellow-800 dark:text-yellow-200">⚠️ HTTP Warning:</strong>
          <span className="text-yellow-700 dark:text-yellow-300 ml-1">
            HTTPS clients (like hosted Feishin) cannot connect to HTTP servers due to browser security.
            Use a desktop client or set up HTTPS.
          </span>
        </div>
      )}
    </div>
  );
};

export default OpenSubsonicInfo;
