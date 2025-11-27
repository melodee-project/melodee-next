import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import ThemeSelector from './components/ThemeSelector';
import LoginPage from './pages/LoginPage';
import AdminDashboard from './components/AdminDashboard';
import DLQManagement from './components/DLQManagement';
import LogViewer from './components/LogViewer';
import UserManagement from './components/UserManagement';
import SettingsManagement from './components/SettingsManagement';
import SharesManagement from './components/SharesManagement';
import LibraryManagement from './components/LibraryManagement';
import QuarantineManagement from './components/QuarantineManagement';
import PlaylistManagement from './components/PlaylistManagement';
import { 
  LayoutDashboard, 
  AlertTriangle, 
  FileText, 
  Users, 
  Settings, 
  Share2, 
  Library, 
  AlertCircle, 
  Music,
  LogOut
} from 'lucide-react';

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
  const { currentTheme } = useTheme();

  const handleLogout = () => {
    logout();
  };

  // Get theme-specific classes
  const navbarClass = currentTheme?.colors?.navbar || 'bg-blue-700 dark:bg-gray-800';
  const navbarTextClass = currentTheme?.colors?.navbarText || 'text-white';
  const navbarHoverClass = currentTheme?.colors?.navbarHover || 'hover:text-blue-200 dark:hover:text-blue-300';
  const backgroundClass = currentTheme?.colors?.background || 'bg-gray-100 dark:bg-gray-900';

  return (
    <div className={`min-h-screen ${backgroundClass} flex flex-col transition-colors`}>
      {/* Navigation */}
      <nav className={`${navbarClass} ${navbarTextClass} shadow-md`}>
        <div className="w-full px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <Link to="/admin" className={`text-xl font-bold ${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-2`}>
              <Music className="w-6 h-6" />
              Melodee Admin
            </Link>
            <ul className="hidden md:flex space-x-4 lg:space-x-6">
              <li><Link to="/admin" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><LayoutDashboard className="w-4 h-4" />Dashboard</Link></li>
              <li><Link to="/admin/dlq" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><AlertTriangle className="w-4 h-4" />DLQ</Link></li>
              <li><Link to="/admin/logs" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><FileText className="w-4 h-4" />Logs</Link></li>
              <li><Link to="/admin/users" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><Users className="w-4 h-4" />Users</Link></li>
              <li><Link to="/admin/settings" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><Settings className="w-4 h-4" />Settings</Link></li>
              <li><Link to="/admin/shares" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><Share2 className="w-4 h-4" />Shares</Link></li>
              <li><Link to="/admin/libraries" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><Library className="w-4 h-4" />Libraries</Link></li>
              <li><Link to="/admin/quarantine" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><AlertCircle className="w-4 h-4" />Quarantine</Link></li>
              <li><Link to="/admin/playlists" className={`${navbarTextClass} ${navbarHoverClass} transition-colors flex items-center gap-1.5`}><Music className="w-4 h-4" />Playlists</Link></li>
            </ul>
            <div className="flex items-center space-x-4">
              <ThemeSelector />
              <span className="text-sm md:text-base text-white font-medium">Welcome, <span className="font-bold">{user?.username || user?.Username || 'User'}</span>!</span>
              <button
                onClick={handleLogout}
                className="bg-red-600 hover:bg-red-700 dark:bg-red-700 dark:hover:bg-red-800 text-white px-4 py-2 rounded transition-colors font-medium flex items-center gap-2"
              >
                <LogOut className="w-4 h-4" />
                Logout
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="flex-1 w-full px-4 sm:px-6 lg:px-8 py-6">
        {children}
      </main>
    </div>
  );
}

// Combined app with AuthProvider and Layout
function App() {
  return (
    <ThemeProvider>
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
          <Route path="/admin/logs" element={
            <AdminRoute>
              <Layout>
                <LogViewer />
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
    </ThemeProvider>
  );
}

export default App;