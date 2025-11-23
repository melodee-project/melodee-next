import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import LoginPage from './pages/LoginPage';
import AdminDashboard from './components/AdminDashboard';
import DLQManagement from './components/DLQManagement';
import UserManagement from './components/UserManagement';
import SettingsManagement from './components/SettingsManagement';
import SharesManagement from './components/SharesManagement';
import LibraryManagement from './components/LibraryManagement';
import QuarantineManagement from './components/QuarantineManagement';
import PlaylistManagement from './components/PlaylistManagement';

// ProtectedRoute component to restrict access to authenticated users
function ProtectedRoute({ children }) {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return <div className="p-4">Loading...</div>;
  }

  return isAuthenticated ? children : <Navigate to="/login" />;
}

// AdminRoute component to restrict access to admin users
function AdminRoute({ children }) {
  const { user, isAuthenticated, loading } = useAuth();

  if (loading) {
    return <div className="p-4">Loading...</div>;
  }

  const isAdmin = user?.is_admin || user?.isAdmin || false;
  return isAuthenticated && isAdmin ? children : <Navigate to="/login" />;
}


// Main Layout Component with Navigation
function Layout({ children }) {
  const { user, isAuthenticated, logout } = useAuth();

  const handleLogout = () => {
    logout();
  };

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Navigation */}
      <nav className="bg-blue-600 text-white p-4">
        <div className="container mx-auto">
          <div className="flex justify-between items-center">
            <Link to="/admin" className="text-xl font-bold hover:underline">Melodee Admin</Link>
            <ul className="flex space-x-6">
              <li><Link to="/admin" className="hover:underline">Dashboard</Link></li>
              <li><Link to="/admin/dlq" className="hover:underline">DLQ</Link></li>
              <li><Link to="/admin/users" className="hover:underline">Users</Link></li>
              <li><Link to="/admin/settings" className="hover:underline">Settings</Link></li>
              <li><Link to="/admin/shares" className="hover:underline">Shares</Link></li>
              <li><Link to="/admin/libraries" className="hover:underline">Libraries</Link></li>
              <li><Link to="/admin/quarantine" className="hover:underline">Quarantine</Link></li>
              <li><Link to="/admin/playlists" className="hover:underline">Playlists</Link></li>
            </ul>
            <div className="flex items-center space-x-4">
              <span>Welcome, {user?.username || user?.Username || 'User'}!</span>
              <button
                onClick={handleLogout}
                className="bg-red-500 hover:bg-red-600 text-white px-3 py-1 rounded"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="container mx-auto mt-4 pb-6">
        {children}
      </main>
    </div>
  );
}

// Combined app with AuthProvider and Layout
function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={
            <ProtectedRoute>
              <Layout>
                <AdminDashboard />
              </Layout>
            </ProtectedRoute>
          } />
          <Route path="/admin" element={
            <ProtectedRoute>
              <Layout>
                <AdminDashboard />
              </Layout>
            </ProtectedRoute>
          } />
          <Route path="/admin/dlq" element={
            <AdminRoute>
              <Layout>
                <DLQManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/users" element={
            <AdminRoute>
              <Layout>
                <UserManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/settings" element={
            <AdminRoute>
              <Layout>
                <SettingsManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/shares" element={
            <AdminRoute>
              <Layout>
                <SharesManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/libraries" element={
            <AdminRoute>
              <Layout>
                <LibraryManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/quarantine" element={
            <AdminRoute>
              <Layout>
                <QuarantineManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/playlists" element={
            <AdminRoute>
              <Layout>
                <PlaylistManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="*" element={
            <ProtectedRoute>
              <Layout>
                <AdminDashboard />
              </Layout>
            </ProtectedRoute>
          } />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;