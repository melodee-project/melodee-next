// src/frontend/src/components/AdminDashboard.test.jsx

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import AdminDashboard from './AdminDashboard';

// Mock the services
jest.mock('../services/apiService', () => ({
  healthService: {
    getHealth: jest.fn(() => Promise.resolve({ data: { status: 'ok', db: { status: 'ok' }, redis: { status: 'ok' } } })),
  },
  libraryService: {
    getStats: jest.fn(() => Promise.resolve({ data: { total_tracks: 123456, total_artists: 2345 } })),
  },
  adminService: {
    getDLQItems: jest.fn(() => Promise.resolve({ data: { data: [], pagination: { total: 0 } } })),
  },
  userService: {
    getUsers: jest.fn(() => Promise.resolve({ data: { data: [], pagination: { total: 1 } } })),
  }
}));

describe('Admin Dashboard Component', () => {
  const renderWithProviders = (ui, { initialState = {} } = {}) => {
    return render(
      <MemoryRouter>
        <AuthProvider value={{ user: { id: 1, username: 'admin', is_admin: true }, isAuthenticated: true }}>
          {ui}
        </AuthProvider>
      </MemoryRouter>
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render dashboard header and metrics', async () => {
    renderWithProviders(<AdminDashboard />);

    // Check for dashboard header
    expect(screen.getByText(/Admin Dashboard/i)).toBeInTheDocument();

    // Wait for async data to load
    await waitFor(() => {
      // Check for metrics that should appear after data is loaded
      expect(screen.getByText(/Metrics Overview/i)).toBeInTheDocument();
    });
  });

  test('should display system health status', async () => {
    const { healthService } = require('../services/apiService');

    renderWithProviders(<AdminDashboard />);

    await waitFor(() => {
      expect(healthService.getHealth).toHaveBeenCalled();
    });

    // Should show health status
    expect(screen.getByText(/Status: ok/i)).toBeInTheDocument();
  });

  test('should display library statistics', async () => {
    const { libraryService } = require('../services/apiService');

    renderWithProviders(<AdminDashboard />);

    await waitFor(() => {
      expect(libraryService.getStats).toHaveBeenCalled();
    });

    // Should show library stats
    expect(screen.getByText(/123,456/)).toBeInTheDocument(); // total tracks
    expect(screen.getByText(/2,345/)).toBeInTheDocument(); // total artists
  });

  test('should show DLQ status', async () => {
    const { adminService } = require('../services/apiService');

    renderWithProviders(<AdminDashboard />);

    await waitFor(() => {
      expect(adminService.getDLQItems).toHaveBeenCalled();
    });

    // Should indicate DLQ status
    expect(screen.getByText(/DLQ Items:/i)).toBeInTheDocument();
  });
});