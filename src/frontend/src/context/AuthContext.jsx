import React, { createContext, useContext, useEffect, useState } from 'react';
import { authService } from '../services/apiService';

const AuthContext = createContext();

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  // Set up axios interceptor to include auth token (this is handled in apiService)
  useEffect(() => {
    const token = localStorage.getItem('accessToken');
    if (token) {
      setIsAuthenticated(true);
      // Try to get user info
      fetchUserInfo(token);
    } else {
      setLoading(false);
    }
  }, []);

  const fetchUserInfo = async (token) => {
    try {
      // In a real app, you'd have an endpoint to get user info
      // For now, we'll simulate by storing user info when login happens
      const userInfo = JSON.parse(localStorage.getItem('userInfo'));
      if (userInfo) {
        setUser(userInfo);
      }
    } catch (error) {
      console.error('Error fetching user info:', error);
      logout();
    } finally {
      setLoading(false);
    }
  };

  const login = async (username, password) => {
    try {
      const response = await authService.login(username, password);

      const { access_token, refresh_token, user: userData } = response.data;

      // Store tokens
      localStorage.setItem('accessToken', access_token);
      localStorage.setItem('refreshToken', refresh_token);
      localStorage.setItem('userInfo', JSON.stringify(userData));

      setUser(userData);
      setIsAuthenticated(true);

      return { success: true };
    } catch (error) {
      // Extract more detailed error information from the response
      let errorMessage = 'Login failed';
      let errorDetails = '';

      if (error.response) {
        // Server responded with error status
        if (error.response.data && typeof error.response.data === 'object') {
          if (error.response.data.error) {
            errorMessage = error.response.data.error;
          } else if (error.response.data.message) {
            errorMessage = error.response.data.message;
          } else if (error.response.data.Error) {
            // Handle OpenSubsonic error format
            errorMessage = error.response.data.Error?.Message || 'Login failed';
          }
        } else {
          errorMessage = error.response.statusText || 'Server error occurred';
        }
      } else if (error.request) {
        // Request was made but no response received
        errorMessage = 'Network error - could not connect to server';
      } else {
        // Something else happened in setting up the request
        errorMessage = error.message || 'Login failed';
      }

      return {
        success: false,
        error: errorMessage,
        errorDetails: errorDetails
      };
    }
  };

  const logout = () => {
    // Clear tokens and user data
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('userInfo');

    // The axios interceptors handle removing the auth header automatically
    setUser(null);
    setIsAuthenticated(false);
  };

  const requestPasswordReset = async (email) => {
    try {
      await authService.requestPasswordReset(email);
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Password reset request failed'
      };
    }
  };

  const resetPassword = async (resetToken, newPassword) => {
    try {
      await authService.resetPassword(resetToken, newPassword);
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Password reset failed'
      };
    }
  };

  const value = {
    user,
    isAuthenticated,
    loading,
    login,
    logout,
    requestPasswordReset,
    resetPassword,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};