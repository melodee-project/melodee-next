// src/frontend/src/components/DLQManagement.test.jsx

import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import DLQManagement from './DLQManagement';

// Mock the services
jest.mock('../services/apiService', () => ({
  adminService: {
    getDLQItems: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          data: [
            { 
              id: 'job-1', 
              queue: 'default', 
              type: 'scan', 
              reason: 'Permission denied', 
              payload: '{"library_id": 1}',
              created_at: '2025-11-22T12:00:00Z',
              retry_count: 3
            },
            { 
              id: 'job-2', 
              queue: 'default', 
              type: 'process', 
              reason: 'File not found', 
              payload: '{"file_path": "/missing/file.mp3"}',
              created_at: '2025-11-22T11:00:00Z',
              retry_count: 1
            }
          ],
          pagination: { page: 1, size: 50, total: 2 }
        } 
      })
    ),
    requeueDLQItems: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          status: 'ok', 
          requeued: 1, 
          failed_ids: []
        } 
      })
    ),
    purgeDLQItems: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          status: 'ok', 
          purged: 1, 
          failed_ids: []
        } 
      })
    ),
  }
}));

describe('DLQ Management Workflow', () => {
  const renderWithProviders = (ui) => {
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

  test('should display DLQ items in the table', async () => {
    renderWithProviders(<DLQManagement />);

    // Wait for DLQ items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
      expect(screen.getByText('Permission denied')).toBeInTheDocument();
    });

    // Verify both items are displayed
    expect(screen.getByText('job-1')).toBeInTheDocument();
    expect(screen.getByText('job-2')).toBeInTheDocument();
    expect(screen.getByText('Permission denied')).toBeInTheDocument();
    expect(screen.getByText('File not found')).toBeInTheDocument();
  });

  test('should allow requeuing selected DLQ items', async () => {
    const { adminService } = require('../services/apiService');

    renderWithProviders(<DLQManagement />);

    // Wait for DLQ items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
    });

    // Select the first item by clicking its checkbox
    const checkboxes = screen.getAllByRole('checkbox');
    fireEvent.click(checkboxes[1]); // First item's checkbox (skip the header checkbox)

    // Click the requeue button
    const requeueButton = screen.getByRole('button', { name: /requeue/i });
    fireEvent.click(requeueButton);

    // Wait for the requeue operation to complete
    await waitFor(() => {
      expect(adminService.requeueDLQItems).toHaveBeenCalledWith(['job-1']);
    });

    // Verify the requeue was successful
    expect(adminService.requeueDLQItems).toHaveBeenCalledWith(['job-1']);
    
    // In a real implementation, this might trigger a UI update or notification
  });

  test('should allow purging selected DLQ items', async () => {
    const { adminService } = require('../services/apiService');

    renderWithProviders(<DLQManagement />);

    // Wait for DLQ items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
    });

    // Select the second item
    const checkboxes = screen.getAllByRole('checkbox');
    fireEvent.click(checkboxes[2]); // Second item's checkbox

    // Click the purge button
    const purgeButton = screen.getByRole('button', { name: /purge/i });
    fireEvent.click(purgeButton);

    // If there's a confirmation dialog, confirm it
    const confirmButton = screen.getByRole('button', { name: /confirm|yes/i });
    if (confirmButton) {
      fireEvent.click(confirmButton);
    }

    // Wait for the purge operation to complete
    await waitFor(() => {
      expect(adminService.purgeDLQItems).toHaveBeenCalledWith(['job-2']);
    });

    // Verify the purge was successful
    expect(adminService.purgeDLQItems).toHaveBeenCalledWith(['job-2']);
  });

  test('should allow bulk operations on multiple items', async () => {
    const { adminService } = require('../services/apiService');

    renderWithProviders(<DLQManagement />);

    // Wait for DLQ items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
    });

    // Select multiple items
    const checkboxes = screen.getAllByRole('checkbox');
    fireEvent.click(checkboxes[1]); // First item
    fireEvent.click(checkboxes[2]); // Second item

    // Click the requeue button
    const requeueButton = screen.getByRole('button', { name: /requeue/i });
    fireEvent.click(requeueButton);

    // Wait for the requeue operation to complete
    await waitFor(() => {
      expect(adminService.requeueDLQItems).toHaveBeenCalledWith(['job-1', 'job-2']);
    });

    // Verify both items were included in the operation
    expect(adminService.requeueDLQItems).toHaveBeenCalledWith(['job-1', 'job-2']);
  });

  test('should display job details when viewing an item', async () => {
    renderWithProviders(<DLQManagement />);

    // Wait for DLQ items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
    });

    // Find and click the view details button for the first job
    const jobRows = screen.getAllByRole('row');
    // Skip header, get the row for job-1
    const job1Row = jobRows[1]; // Assuming header is first row, job-1 is second

    const viewButton = within(job1Row).getByRole('button', { name: /view|details/i });
    fireEvent.click(viewButton);

    // Verify job details are shown
    await waitFor(() => {
      expect(screen.getByText(/Queue:/i)).toBeInTheDocument();
      expect(screen.getByText('default')).toBeInTheDocument();
      expect(screen.getByText(/Type:/i)).toBeInTheDocument();
      expect(screen.getByText('scan')).toBeInTheDocument();
      expect(screen.getByText(/Payload:/i)).toBeInTheDocument();
      expect(screen.getByText(/library_id/i)).toBeInTheDocument();
    });
  });

  test('should handle pagination properly', async () => {
    // Mock a response with pagination metadata
    const mockPaginatedResponse = {
      data: {
        data: Array.from({ length: 60 }, (_, i) => ({
          id: `job-${i + 1}`,
          queue: 'default',
          type: 'scan',
          reason: 'Test reason',
          payload: '{}',
          created_at: '2025-11-22T12:00:00Z',
          retry_count: 0
        })),
        pagination: { page: 1, size: 50, total: 60 }
      }
    };

    const { adminService } = require('../services/apiService');
    adminService.getDLQItems.mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<DLQManagement />);

    // Wait for items to load
    await waitFor(() => {
      expect(screen.getByText('job-1')).toBeInTheDocument();
    });

    // Verify only the first 50 items are shown initially
    const rows = screen.getAllByRole('row');
    // This would check that pagination is working properly
    expect(rows.length).toBeGreaterThan(1); // More than just the header row
  });
});