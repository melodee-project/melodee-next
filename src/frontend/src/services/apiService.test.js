// src/frontend/src/services/apiService.test.js

import axios from 'axios';
import { 
  authService, 
  userService, 
  adminService, 
  libraryService, 
  mediaService 
} from './apiService';

// Mock axios
jest.mock('axios');

describe('API Service - Admin Flows', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
    
    // Mock successful token in localStorage
    localStorage.setItem('accessToken', 'mock-access-token');
    localStorage.setItem('refreshToken', 'mock-refresh-token');
  });

  describe('Auth Service', () => {
    test('should login successfully', async () => {
      const mockResponse = {
        data: {
          access_token: 'new-access-token',
          refresh_token: 'new-refresh-token',
          user: { id: 1, username: 'admin', is_admin: true }
        }
      };
      
      axios.post.mockResolvedValue(mockResponse);
      
      const result = await authService.login('admin', 'password');
      
      expect(axios.post).toHaveBeenCalledWith('/api/auth/login', {
        username: 'admin',
        password: 'password'
      });
      expect(result).toEqual(mockResponse);
    });

    test('should handle login failure', async () => {
      const mockError = new Error('Invalid credentials');
      axios.post.mockRejectedValue(mockError);

      await expect(authService.login('admin', 'invalid')).rejects.toThrow();
    });
  });

  describe('User Service', () => {
    test('should get users with pagination', async () => {
      const mockUsers = { data: [], pagination: { page: 1, size: 50, total: 0 } };
      axios.get.mockResolvedValue({ data: mockUsers });

      const result = await userService.getUsers(1, 50);

      expect(axios.get).toHaveBeenCalledWith('/users');
      expect(result.data).toEqual(mockUsers);
    });

    test('should create a new user', async () => {
      const userData = { username: 'testuser', email: 'test@example.com', password: 'password' };
      const mockResponse = { data: { id: 2, ...userData } };
      axios.post.mockResolvedValue(mockResponse);

      const result = await userService.createUser(userData);

      expect(axios.post).toHaveBeenCalledWith('/users', userData);
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Admin Service', () => {
    test('should get DLQ items', async () => {
      const mockDLQItems = { data: [], pagination: { page: 1, size: 50, total: 0 } };
      axios.get.mockResolvedValue({ data: mockDLQItems });

      const result = await adminService.getDLQItems();

      expect(axios.get).toHaveBeenCalledWith('/admin/jobs/dlq');
      expect(result.data).toEqual(mockDLQItems);
    });

    test('should requeue DLQ items', async () => {
      const jobIds = ['job-1', 'job-2'];
      const mockResponse = { status: 'ok', requeued: 2, failed_ids: [] };
      axios.post.mockResolvedValue({ data: mockResponse });

      const result = await adminService.requeueDLQItems(jobIds);

      expect(axios.post).toHaveBeenCalledWith('/admin/jobs/requeue', { job_ids: jobIds });
      expect(result.data).toEqual(mockResponse);
    });

    test('should get settings', async () => {
      const mockSettings = { data: [{ key: 'smtp.host', value: 'smtp.example.com' }] };
      axios.get.mockResolvedValue({ data: mockSettings });

      const result = await adminService.getSettings();

      expect(axios.get).toHaveBeenCalledWith('/settings');
      expect(result.data).toEqual(mockSettings);
    });

    test('should update a setting', async () => {
      const key = 'smtp.host';
      const value = 'smtp2.example.com';
      const mockResponse = { status: 'ok', setting: { key, value } };
      axios.put.mockResolvedValue({ data: mockResponse });

      const result = await adminService.updateSetting(key, value);

      expect(axios.put).toHaveBeenCalledWith('/settings', { key, value });
      expect(result.data).toEqual(mockResponse);
    });

    test('should get shares', async () => {
      const mockShares = { data: [], pagination: { page: 1, size: 50, total: 0 } };
      axios.get.mockResolvedValue({ data: mockShares });

      const result = await adminService.getShares();

      expect(axios.get).toHaveBeenCalledWith('/shares');
      expect(result.data).toEqual(mockShares);
    });
  });

  describe('Library Service', () => {
    test('should get library stats', async () => {
      const mockStats = {
        total_libraries: 1,
        total_artists: 2345,
        total_albums: 6789,
        total_tracks: 123456,
        total_size_bytes: 987654321,
        last_full_scan_at: '2025-11-22T12:00:00Z'
      };
      axios.get.mockResolvedValue({ data: mockStats });

      const result = await libraryService.getStats();

      expect(axios.get).toHaveBeenCalledWith('/libraries/stats');
      expect(result.data).toEqual(mockStats);
    });

    test('should trigger library scan', async () => {
      const mockResponse = { status: 'queued', job_id: 'scan-job-1' };
      axios.post.mockResolvedValue({ data: mockResponse });

      const result = await libraryService.scanLibrary();

      expect(axios.post).toHaveBeenCalledWith('/libraries/scan');
      expect(result.data).toEqual(mockResponse);
    });
  });

  describe('Media Service (OpenSubsonic Compatibility)', () => {
    beforeEach(() => {
      process.env.REACT_APP_OPEN_SUBSONIC_ENABLED = 'true';
    });

    test('should get music folders when enabled', async () => {
      const mockResponse = { data: '<xml>...</xml>' };
      axios.get.mockResolvedValue(mockResponse);

      const result = await mediaService.getMusicFolders();

      expect(axios.get).toHaveBeenCalledWith('/rest/getMusicFolders.view?u=admin&p=enc:xxx&t=xxx&s=xxx');
      expect(result).toEqual(mockResponse);
    });

    test('should check if OpenSubsonic is enabled', () => {
      expect(mediaService.isOpenSubsonicEnabled()).toBe(true);
      
      process.env.REACT_APP_OPEN_SUBSONIC_ENABLED = 'false';
      expect(mediaService.isOpenSubsonicEnabled()).toBe(false);
    });

    test('should reject when OpenSubsonic is disabled', () => {
      process.env.REACT_APP_OPEN_SUBSONIC_ENABLED = 'false';
      
      expect(() => mediaService.getMusicFolders()).rejects.toThrow('OpenSubsonic features are disabled');
    });
  });

  describe('Authorization Interceptor', () => {
    test('should include auth token in requests', async () => {
      const mockResponse = { data: { success: true } };
      axios.get.mockResolvedValue(mockResponse);

      // This would test the axios instance behavior, which is already set up in the service
      const result = await userService.getUsers();

      // Check that the default headers include the Authorization header
      expect(axios.defaults.headers.common['Authorization']).toBe('Bearer mock-access-token');
    });
  });
});