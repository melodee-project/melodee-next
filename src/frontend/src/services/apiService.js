// apiService.js

import axios from 'axios';

// Create an axios instance with defaults
const apiService = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api', // Use Vite's import.meta.env instead of process.env
  timeout: 10000, // 10 seconds timeout
  withCredentials: true, // Include cookies in cross-origin requests if needed
});

// Request interceptor to include auth token
apiService.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('accessToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle token refresh
apiService.interceptors.response.use(
  (response) => {
    return response;
  },
  async (error) => {
    const originalRequest = error.config;

    // If unauthorized and not retrying
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        const refreshToken = localStorage.getItem('refreshToken');
        if (refreshToken) {
          const response = await axios.post('/api/auth/refresh', {
            refresh_token: refreshToken,
          });

          const { access_token } = response.data;
          localStorage.setItem('accessToken', access_token);

          // Retry the original request with new token
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
          return apiService(originalRequest);
        }
      } catch (refreshError) {
        // If refresh token is also invalid, redirect to login
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

// Auth-related API endpoints
export const authService = {
  login: (username, password) => apiService.post('/auth/login', { username, password }),
  refresh: (refreshToken) => apiService.post('/auth/refresh', { refresh_token: refreshToken }),
  logout: () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
  },
  requestPasswordReset: (email) => apiService.post('/auth/request-reset', { email }),
  resetPassword: (resetToken, newPassword) => apiService.post('/auth/reset', { token: resetToken, password: newPassword }),
};

// User-related API endpoints
export const userService = {
  getUsers: (page = 1, size = 50) => apiService.get('/users'), // Pagination handled by backend
  getUserById: (id) => apiService.get(`/users/${id}`),
  createUser: (userData) => apiService.post('/users', userData),
  updateUser: (id, userData) => apiService.put(`/users/${id}`, userData),
  deleteUser: (id) => apiService.delete(`/users/${id}`),
};

// Playlist-related API endpoints
export const playlistService = {
  getPlaylists: () => apiService.get('/playlists'), // Pagination handled by backend
  getPlaylistById: (id) => apiService.get(`/playlists/${id}`),
  createPlaylist: (data) => apiService.post('/playlists', data),
  updatePlaylist: (id, data) => apiService.put(`/playlists/${id}`, data),
  deletePlaylist: (id) => apiService.delete(`/playlists/${id}`),
};

// Admin-related API endpoints
export const adminService = {
  // Job monitoring
  getActiveJobs: () => apiService.get('/admin/jobs/active'),
  getPendingJobs: () => apiService.get('/admin/jobs/pending'),
  getScheduledJobs: () => apiService.get('/admin/jobs/scheduled'),
  getJobStats: () => apiService.get('/admin/jobs/stats'),
  cancelJob: (id) => apiService.post(`/admin/jobs/cancel/${id}`),
  runTask: (taskType, queue, payload) => apiService.post('/admin/jobs/run', { task_type: taskType, queue, payload }),
  
  // DLQ (Dead Letter Queue)
  getDLQItems: () => apiService.get('/admin/jobs/dlq'),
  requeueDLQItems: (jobIds) => apiService.post('/admin/jobs/dlq/requeue', { job_ids: jobIds }),
  purgeDLQItems: (jobIds) => apiService.post('/admin/jobs/dlq/purge', { job_ids: jobIds }),
  getJobById: (id) => apiService.get(`/admin/jobs/dlq/${id}`),
  getSettings: () => apiService.get('/settings'), // Note: Per INTERNAL_API_ROUTES.md, it's /settings not /admin/settings
  updateSetting: (key, value) => apiService.put('/settings', { key, value }), // Per spec, it updates single key
  getShares: () => apiService.get('/shares'),
  createShare: (data) => apiService.post('/shares', data),
  updateShare: (id, data) => apiService.put(`/shares/${id}`, data),
  deleteShare: (id) => apiService.delete(`/shares/${id}`),
  getLogs: (params) => apiService.get(`/admin/logs?${params}`),
  getLogStats: () => apiService.get('/admin/logs/stats'),
  downloadLogs: (params) => apiService.get(`/admin/logs/download?${params}`),
  cleanupLogs: (olderThanDays) => apiService.post(`/admin/logs/cleanup?older_than_days=${olderThanDays}`),
};

// Library-related API endpoints
export const libraryService = {
  getStats: () => apiService.get('/library-stats'),
  scanLibrary: (libraryId) => apiService.get(`/libraries/${libraryId}/scan`),
  processInbound: (libraryId) => apiService.get(`/libraries/${libraryId}/process`),
  moveOkAlbums: (libraryId) => apiService.get(`/libraries/${libraryId}/move-ok`),
  getLibraries: () => apiService.get('/libraries'),
  updateLibrary: (id, data) => apiService.put(`/libraries/${id}`, data),
  getQuarantineItems: (params = {}) => apiService.get('/libraries/quarantine', { params }),
  resolveQuarantineItem: (id) => apiService.post(`/libraries/quarantine/${id}/resolve`),
  requeueQuarantineItem: (id) => apiService.post(`/libraries/quarantine/${id}/requeue`),
};

// System health and capacity monitoring endpoints
export const healthService = {
  getHealth: () => axios.get('/healthz'), // Direct call without /api prefix
  getMetrics: () => axios.get('/metrics'), // Direct call without /api prefix
  getCapacity: () => apiService.get('/admin/capacity'),
  getCapacityForLibrary: (libraryId) => apiService.get(`/admin/capacity/${libraryId}`),
  probeCapacityNow: () => apiService.post('/admin/capacity/probe-now'),
};

// Quarantine management endpoints
export const quarantineService = {
  getQuarantineItems: (params = {}) => apiService.get('/libraries/quarantine', { params }),
  resolveQuarantineItem: (id) => apiService.post(`/libraries/quarantine/${id}/resolve`),
  requeueQuarantineItem: (id) => apiService.post(`/libraries/quarantine/${id}/requeue`),
  deleteQuarantineItem: (id) => apiService.delete(`/libraries/quarantine/${id}`)
};

// Metrics and health-related endpoints
export const metricsService = {
  getMetrics: () => apiService.get('/metrics'),
  getHealth: () => apiService.get('/healthz'),
};

// Media-related API endpoints (OpenSubsonic compatibility helpers)
// These are flagged as compatibility features for third-party client support
export const mediaService = {
  // Check if OpenSubsonic compatibility features are enabled in config
  isOpenSubsonicEnabled: () => import.meta.env.VITE_OPEN_SUBSONIC_ENABLED === 'true',

  // For OpenSubsonic API access - only use for compatibility features
  // These endpoints support existing Subsonic/OpenSubsonic clients and should be
  // used exclusively for compatibility purposes, not for core admin functionality
  getMusicFolders: () => {
    if (!import.meta.env.VITE_OPEN_SUBSONIC_ENABLED) {
      return Promise.reject(new Error('OpenSubsonic features are disabled'));
    }
    return apiService.get('/rest/getMusicFolders.view?u=admin&p=enc:xxx&t=xxx&s=xxx');
  },
  getArtists: () => {
    if (!import.meta.env.VITE_OPEN_SUBSONIC_ENABLED) {
      return Promise.reject(new Error('OpenSubsonic features are disabled'));
    }
    return apiService.get('/rest/getArtists.view?u=admin&p=enc:xxx&t=xxx&s=xxx');
  },
  getAlbum: (id) => {
    if (!import.meta.env.VITE_OPEN_SUBSONIC_ENABLED) {
      return Promise.reject(new Error('OpenSubsonic features are disabled'));
    }
    return apiService.get(`/rest/getAlbum.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`);
  },
  stream: (id) => {
    if (!import.meta.env.VITE_OPEN_SUBSONIC_ENABLED) {
      throw new Error('OpenSubsonic features are disabled');
    }
    return `${import.meta.env.VITE_API_BASE_URL || ''}/rest/stream.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`;
  },
  getCoverArt: (id) => {
    if (!import.meta.env.VITE_OPEN_SUBSONIC_ENABLED) {
      throw new Error('OpenSubsonic features are disabled');
    }
    return `${import.meta.env.VITE_API_BASE_URL || ''}/rest/getCoverArt.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`;
  },
};

export default apiService;