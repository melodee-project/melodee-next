import React, { useEffect, useMemo, useState } from 'react';
import { adminService } from '../services/apiService';

function computeOpenSubsonicBase() {
  const envBase = import.meta.env.VITE_API_BASE_URL;
  try {
    if (envBase) {
      // Absolute URL provided
      if (/^https?:\/\//i.test(envBase)) {
        const u = new URL(envBase);
        return `${u.origin}/rest`;
      }
      // Relative path (e.g., "/api") → assume same origin
      if (envBase.startsWith('/')) {
        return `${window.location.origin}/rest`;
      }
    }
  } catch (_) {
    // ignore and fallback
  }
  // Fallback to same origin
  return `${window.location.origin}/rest`;
}

const OpenSubsonicInfo = () => {
  const [status, setStatus] = useState('checking');
  const [detail, setDetail] = useState('');
  const [base, setBase] = useState(computeOpenSubsonicBase());
  const pingUrl = `${base}/ping.view`;

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    (async () => {
      try {
        // Try to get canonical base from backend
        try {
          const resp = await adminService.getOpenSubsonicInfo();
          let baseToUse = base;
          if (resp?.data?.base) {
            baseToUse = resp.data.base;
            if (!cancelled) setBase(resp.data.base);
          }
          const res = await fetch(`${baseToUse}/ping.view`, { method: 'GET', signal: controller.signal });
          if (cancelled) return;
          if (res.ok) {
            setStatus('online');
            setDetail('Ping successful');
          } else {
            setStatus('error');
            setDetail(`HTTP ${res.status}`);
          }
        } catch (_) {
          // Fallback: use computed base if backend call fails
          const res = await fetch(`${base}/ping.view`, { method: 'GET', signal: controller.signal });
          if (cancelled) return;
          if (res.ok) {
            setStatus('online');
            setDetail('Ping successful');
          } else {
            setStatus('error');
            setDetail(`HTTP ${res.status}`);
          }
        }
      } catch (err) {
        if (cancelled) return;
        setStatus('error');
        setDetail(err?.message || 'Network error');
      }
    })();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [pingUrl]);

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
        Use this endpoint in your OpenSubsonic-compatible client.
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
        Computed from {import.meta.env.VITE_API_BASE_URL ? 'VITE_API_BASE_URL' : 'window.location.origin'}
        {' '}→ <code>/rest</code>
      </div>
    </div>
  );
};

export default OpenSubsonicInfo;
