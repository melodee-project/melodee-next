import React, { useState, useEffect, useRef } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import ThemeSelector from './components/ThemeSelector';
import LoginPage from './pages/LoginPage';
import StagingPage from './pages/StagingPage';
import StagingDetailPage from './pages/StagingDetailPage';
import AdminDashboard from './components/AdminDashboard';
import JobMonitor from './components/JobMonitor';
import LogViewer from './components/LogViewer';
import UserManagement from './components/UserManagement';
import SettingsManagement from './components/SettingsManagement';
import SharesManagement from './components/SharesManagement';
import LibraryManagement from './components/LibraryManagement';
import PlaylistManagement from './components/PlaylistManagement';
import {
  LayoutDashboard,
  Briefcase,
  FileText,
  Users,
  Settings,
  Share2,
  Library,
  AlertCircle,
  Music,
  FolderCheck,
  LogOut,
  ChevronDown,
  Database
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
  const location = useLocation();
  const [systemDropdownOpen, setSystemDropdownOpen] = useState(false);
  const [dataDropdownOpen, setDataDropdownOpen] = useState(false);
  const systemDropdownRef = useRef(null);
  const dataDropdownRef = useRef(null);

  const handleLogout = () => {
    logout();
  };

  // Close dropdowns when clicking outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (systemDropdownRef.current && !systemDropdownRef.current.contains(event.target)) {
        setSystemDropdownOpen(false);
      }
      if (dataDropdownRef.current && !dataDropdownRef.current.contains(event.target)) {
        setDataDropdownOpen(false);
      }
    }

    if (systemDropdownOpen || dataDropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [systemDropdownOpen, dataDropdownOpen]);

  // Get theme-specific classes
  const navbarClass = currentTheme?.colors?.navbar || 'bg-blue-700 dark:bg-gray-800';
  const navbarTextClass = currentTheme?.colors?.navbarText || 'text-white';
  const navbarHoverClass = currentTheme?.colors?.navbarHover || 'hover:text-blue-200 dark:hover:text-blue-300';
  const backgroundClass = currentTheme?.colors?.background || 'bg-gray-100 dark:bg-gray-900';
  const dropdownBgClass = currentTheme?.colors?.navbarDropdownBg || 'bg-blue-700/80 dark:bg-gray-900/80';
  const dropdownActiveClass = currentTheme?.colors?.navbarDropdownActiveBg || 'bg-white/10 shadow-sm';
  const menuBgClass = currentTheme?.colors?.menuBg || 'bg-white dark:bg-slate-700';
  const menuBorderClass = currentTheme?.colors?.menuBorder || 'border border-gray-200 dark:border-slate-600';
  const menuItemHoverClass = currentTheme?.colors?.menuHoverBg || 'hover:bg-gray-100 dark:hover:bg-slate-600';

  const isActive = (pathPrefix) => location.pathname.startsWith(pathPrefix);

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
              <li>
                <Link
                  to="/admin"
                  className={`transition-colors flex items-center gap-1.5 px-2 py-1 rounded-md ${
                    isActive('/admin') && location.pathname === '/admin'
                      ? 'bg-white/10 shadow-sm'
                      : ''
                  } ${navbarTextClass} ${navbarHoverClass}`}
                >
                  <LayoutDashboard className="w-4 h-4" />Dashboard
                </Link>
              </li>
              <li>
                <Link
                  to="/admin/logs"
                  className={`transition-colors flex items-center gap-1.5 px-2 py-1 rounded-md ${
                    isActive('/admin/logs') ? 'bg-white/10 shadow-sm' : ''
                  } ${navbarTextClass} ${navbarHoverClass}`}
                >
                  <FileText className="w-4 h-4" />Logs
                </Link>
              </li>
              <li>
                <Link
                  to="/staging"
                  className={`transition-colors flex items-center gap-1.5 px-2 py-1 rounded-md ${
                    isActive('/staging') ? 'bg-white/10 shadow-sm' : ''
                  } ${navbarTextClass} ${navbarHoverClass}`}
                >
                  <FolderCheck className="w-4 h-4" />Staging
                </Link>
              </li>
              {/* Data Dropdown */}
              <li ref={dataDropdownRef} className="relative">
                <button
                  onClick={() => setDataDropdownOpen(!dataDropdownOpen)}
                  className={`flex items-center gap-1.5 px-2 py-1 rounded-md transition-colors ${
                    (isActive('/admin/users') || isActive('/admin/shares') || isActive('/admin/playlists'))
                      ? dropdownActiveClass
                      : dropdownBgClass
                  } ${navbarTextClass} ${navbarHoverClass}`}
                >
                  <Database className="w-4 h-4" />Data <ChevronDown className="w-3 h-3 ml-1" />
                </button>

                {dataDropdownOpen && (
                  <div className={`absolute left-0 mt-2 w-48 rounded-lg shadow-xl py-1 z-50 ${menuBgClass} ${menuBorderClass}`}>
                    <Link
                      to="/admin/users"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/users') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setDataDropdownOpen(false)}
                    >
                      <Users className="w-4 h-4" />
                      <span>Users</span>
                    </Link>
                    <Link
                      to="/admin/shares"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/shares') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setDataDropdownOpen(false)}
                    >
                      <Share2 className="w-4 h-4" />
                      <span>Shares</span>
                    </Link>
                    <Link
                      to="/admin/playlists"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/playlists') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setDataDropdownOpen(false)}
                    >
                      <Music className="w-4 h-4" />
                      <span>Playlists</span>
                    </Link>
                  </div>
                )}
              </li>

              {/* System Dropdown */}
              <li ref={systemDropdownRef} className="relative">
                <button
                  onClick={() => setSystemDropdownOpen(!systemDropdownOpen)}
                  className={`flex items-center gap-1.5 px-2 py-1 rounded-md transition-colors ${
                    (isActive('/admin/jobs') || isActive('/admin/libraries') || isActive('/admin/settings'))
                      ? dropdownActiveClass
                      : dropdownBgClass
                  } ${navbarTextClass} ${navbarHoverClass}`}
                >
                  <Settings className="w-4 h-4" />System <ChevronDown className="w-3 h-3 ml-1" />
                </button>

                {systemDropdownOpen && (
                  <div className={`absolute left-0 mt-2 w-48 rounded-lg shadow-xl py-1 z-50 ${menuBgClass} ${menuBorderClass}`}>
                    <Link
                      to="/admin/jobs"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/jobs') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setSystemDropdownOpen(false)}
                    >
                      <Briefcase className="w-4 h-4" />
                      <span>Jobs</span>
                    </Link>
                    <Link
                      to="/admin/libraries"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/libraries') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setSystemDropdownOpen(false)}
                    >
                      <Library className="w-4 h-4" />
                      <span>Libraries</span>
                    </Link>
                    <Link
                      to="/admin/settings"
                      className={`block px-4 py-2 text-sm transition-colors flex items-center gap-2 rounded-md text-gray-900 dark:text-white ${
                        isActive('/admin/settings') ? 'bg-blue-100 dark:bg-slate-600' : menuItemHoverClass
                      }`}
                      onClick={() => setSystemDropdownOpen(false)}
                    >
                      <Settings className="w-4 h-4" />
                      <span>Settings</span>
                    </Link>
                  </div>
                )}
              </li>
            </ul>
            <div className="flex items-center space-x-4">
              <ThemeSelector />
              <span className="text-sm md:text-base font-medium flex items-center gap-1">
                <span className={`${navbarTextClass}`}>Welcome,</span>
                <span className="px-2 py-1 rounded-full bg-black/40 text-white font-semibold">
                  {user?.username || user?.Username || 'User'}
                </span>
                <span className={`${navbarTextClass}`}>!</span>
              </span>
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
        <Router future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
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
          <Route path="/admin/jobs" element={
            <AdminRoute>
              <Layout>
                <JobMonitor />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/admin/dlq" element={<Navigate to="/admin/jobs" replace />} />
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
          <Route path="/admin/playlists" element={
            <AdminRoute>
              <Layout>
                <PlaylistManagement />
              </Layout>
            </AdminRoute>
          } />
          <Route path="/staging" element={
            <ProtectedRoute>
              <Layout>
                <StagingPage />
              </Layout>
            </ProtectedRoute>
          } />
          <Route path="/staging/:id" element={
            <ProtectedRoute>
              <Layout>
                <StagingDetailPage />
              </Layout>
            </ProtectedRoute>
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