// apiService.js

import axios from 'axios';

// Create an axios instance with defaults
const apiService = axios.create({
  baseURL: process.env.REACT_APP_API_BASE_URL || '/api', // Use REACT_APP_API_BASE_URL from environment or relative path
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
  getDLQItems: () => apiService.get('/admin/jobs/dlq'),
  requeueDLQItems: (jobIds) => apiService.post('/admin/jobs/requeue', { job_ids: jobIds }),
  purgeDLQItems: (jobIds) => apiService.post('/admin/jobs/purge', { job_ids: jobIds }),
  getJobById: (id) => apiService.get(`/admin/jobs/${id}`),
  getSettings: () => apiService.get('/settings'), // Note: Per INTERNAL_API_ROUTES.md, it's /settings not /admin/settings
  updateSetting: (key, value) => apiService.put('/settings', { key, value }), // Per spec, it updates single key
  getShares: () => apiService.get('/shares'),
  createShare: (data) => apiService.post('/shares', data),
  updateShare: (id, data) => apiService.put(`/shares/${id}`, data),
  deleteShare: (id) => apiService.delete(`/shares/${id}`),
};

// Library-related API endpoints
export const libraryService = {
  getStats: () => apiService.get('/libraries/stats'),
  scanLibrary: () => apiService.post('/libraries/scan'),
  processInbound: () => apiService.post('/libraries/process'),
  moveOkAlbums: () => apiService.post('/libraries/move-ok'),
  getLibraries: () => apiService.get('/libraries'),
};

// Metrics and health-related endpoints
export const metricsService = {
  getMetrics: () => apiService.get('/metrics'),
  getHealth: () => apiService.get('/healthz'),
};

// Media-related API endpoints
export const mediaService = {
  // For OpenSubsonic API access - this would use a different base URL typically
  getMusicFolders: () => apiService.get('/rest/getMusicFolders.view?u=admin&p=enc:xxx&t=xxx&s=xxx'),
  getArtists: () => apiService.get('/rest/getArtists.view?u=admin&p=enc:xxx&t=xxx&s=xxx'),
  getAlbum: (id) => apiService.get(`/rest/getAlbum.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`),
  stream: (id) => `${process.env.REACT_APP_API_BASE_URL || ''}/rest/stream.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`,
  getCoverArt: (id) => `${process.env.REACT_APP_API_BASE_URL || ''}/rest/getCoverArt.view?u=admin&p=enc:xxx&t=xxx&s=xxx&id=${id}`,
};

export default apiService;