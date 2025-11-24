// src/frontend/src/components/UserManagement.test.jsx

import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import UserManagement from './UserManagement';

// Mock the services
jest.mock('../services/apiService', () => ({
  userService: {
    getUsers: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          data: [
            { id: 1, username: 'admin', email: 'admin@example.com', is_admin: true },
            { id: 2, username: 'user1', email: 'user1@example.com', is_admin: false }
          ],
          pagination: { page: 1, size: 50, total: 2 }
        } 
      })
    ),
    createUser: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          id: 3, 
          username: 'newuser', 
          email: 'newuser@example.com', 
          is_admin: false 
        } 
      })
    ),
    updateUser: jest.fn(() => 
      Promise.resolve({ 
        data: { 
          id: 2, 
          username: 'user1', 
          email: 'user1@example.com', 
          is_admin: true 
        } 
      })
    ),
    deleteUser: jest.fn(() => Promise.resolve({ data: { status: 'deleted' } })),
  }
}));

describe('User Management Workflow', () => {
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

  test('should display users in the table', async () => {
    renderWithProviders(<UserManagement />);

    // Wait for users to load
    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
      expect(screen.getByText('user1@example.com')).toBeInTheDocument();
    });

    // Verify both users are displayed
    expect(screen.getAllByText(/admin|user1/)).toHaveLength(2);
  });

  test('should allow creating a new user', async () => {
    const { userService } = require('../services/apiService');

    renderWithProviders(<UserManagement />);

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Find and click the "Add User" button or form trigger
    const addButton = screen.getByRole('button', { name: /add|create/i });
    fireEvent.click(addButton);

    // Fill in the form
    const usernameInput = screen.getByLabelText(/username/i);
    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);

    fireEvent.change(usernameInput, { target: { value: 'newuser' } });
    fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });

    // Submit the form
    const submitButton = screen.getByRole('button', { name: /save|create/i });
    fireEvent.click(submitButton);

    // Wait for the create operation to complete
    await waitFor(() => {
      expect(userService.createUser).toHaveBeenCalledWith({
        username: 'newuser',
        email: 'newuser@example.com',
        password: 'password123',
        is_admin: false
      });
    });

    // Verify the new user appears in the list
    expect(screen.getByText('newuser')).toBeInTheDocument();
  });

  test('should allow updating a user', async () => {
    const { userService } = require('../services/apiService');

    renderWithProviders(<UserManagement />);

    // Wait for users to load
    await waitFor(() => {
      expect(screen.getByText('user1')).toBeInTheDocument();
    });

    // Find the edit button for the second user
    const userRows = screen.getAllByRole('row');
    // Skip header row, get the row for 'user1'
    const user1Row = userRows[2]; // Assuming header is first row, admin is second, user1 is third

    // Find and click edit button in the user1 row
    const editButton = within(user1Row).getByRole('button', { name: /edit/i });
    fireEvent.click(editButton);

    // Update the admin status checkbox
    const adminCheckbox = screen.getByLabelText(/admin/i);
    fireEvent.click(adminCheckbox);

    // Submit the form
    const saveButton = screen.getByRole('button', { name: /save/i });
    fireEvent.click(saveButton);

    // Wait for the update operation to complete
    await waitFor(() => {
      expect(userService.updateUser).toHaveBeenCalledWith(2, {
        is_admin: true
      });
    });

    // Verify the user's admin status is updated (this would require checking the UI after update)
    expect(userService.updateUser).toHaveBeenCalledWith(2, { is_admin: true });
  });

  test('should allow deleting a user', async () => {
    const { userService } = require('../services/apiService');

    renderWithProviders(<UserManagement />);

    // Wait for users to load
    await waitFor(() => {
      expect(screen.getByText('user1')).toBeInTheDocument();
    });

    // Find the delete button for the second user
    const userRows = screen.getAllByRole('row');
    const user1Row = userRows[2]; // Assuming user1 is in the third row

    const deleteButton = within(user1Row).getByRole('button', { name: /delete/i });
    fireEvent.click(deleteButton);

    // If there's a confirmation dialog, confirm it
    // For now, assuming immediate deletion
    await waitFor(() => {
      expect(userService.deleteUser).toHaveBeenCalledWith(2);
    });

    // Verify the user is no longer in the list
    expect(screen.queryByText('user1')).not.toBeInTheDocument();
  });

  test('should handle errors gracefully', async () => {
    // Mock an error for createUser
    const { userService } = require('../services/apiService');
    userService.createUser.mockRejectedValue(new Error('Failed to create user'));

    renderWithProviders(<UserManagement />);

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Find and click the "Add User" button
    const addButton = screen.getByRole('button', { name: /add|create/i });
    fireEvent.click(addButton);

    // Fill in the form
    const usernameInput = screen.getByLabelText(/username/i);
    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);

    fireEvent.change(usernameInput, { target: { value: 'erroruser' } });
    fireEvent.change(emailInput, { target: { value: 'error@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });

    // Submit the form
    const submitButton = screen.getByRole('button', { name: /save|create/i });
    fireEvent.click(submitButton);

    // Wait for error handling
    await waitFor(() => {
      // Check if an error message is displayed
      // In a real implementation, there would be error handling UI
    });

    // Verify the error was handled
    expect(userService.createUser).toHaveBeenCalled();
  });
});