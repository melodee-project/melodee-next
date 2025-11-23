// e2e/library-management.spec.js - End-to-end tests for library management

import { test, expect } from '@playwright/test';

test.describe('Library Management E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the admin panel and log in before each test
    await page.goto('/admin');
    
    // Wait for and fill login form
    await page.waitForSelector('[data-testid="login-username"]');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    // Wait for successful login and dashboard
    await page.waitForURL('**/admin');
  });

  test('should display library statistics correctly', async ({ page }) => {
    // Navigate to library management
    await page.click('text=Libraries');
    
    // Wait for library page to load
    await expect(page.locator('h1:has-text("Library Management")')).toBeVisible();
    
    // Verify library stats are displayed
    await expect(page.locator('.bg-white p:has-text(/\d+/)')).toHaveCount(4); // Total artists/albums/songs/duration
    
    // Check if scan button is present
    await expect(page.locator('button:has-text("Scan Libraries")')).toBeVisible();
  });

  test('should trigger library scan and show feedback', async ({ page }) => {
    await page.click('text=Libraries');
    
    // Click the scan button
    const scanButton = page.locator('button:has-text("Scan Libraries")');
    await scanButton.click();
    
    // Check for feedback message
    await expect(page.locator('[data-testid="status-message"]')).toContainText('Scanning');
  });

  test('should list libraries in table format', async ({ page }) => {
    await page.click('text=Libraries');
    
    // Check if libraries table is present
    const table = page.locator('table');
    await expect(table).toBeVisible();
    
    // Verify table headers
    await expect(page.locator('th:has-text("Name")')).toBeVisible();
    await expect(page.locator('th:has-text("Type")')).toBeVisible();
    await expect(page.locator('th:has-text("Path")')).toBeVisible();
  });

  test('should display quarantine items with error details', async ({ page }) => {
    // Navigate to quarantine section
    await page.click('text=Quarantine');
    
    // Wait for quarantine page to load
    await expect(page.locator('h1:has-text("Quarantine Management")')).toBeVisible();
    
    // Check for quarantine items table
    const quarantineTable = page.locator('table');
    await expect(quarantineTable).toBeVisible();
    
    // Verify quarantine columns exist
    await expect(page.locator('th:has-text("File Path")')).toBeVisible();
    await expect(page.locator('th:has-text("Reason")')).toBeVisible();
    await expect(page.locator('th:has-text("Library")')).toBeVisible();
  });

  test('should allow resolving quarantine items', async ({ page }) => {
    await page.click('text=Quarantine');
    
    // Check if resolve buttons are present
    const resolveButtons = page.locator('button:has-text("Resolve")');
    const count = await resolveButtons.count();
    
    if (count > 0) {
      // Click first resolve button
      await resolveButtons.first().click();
      
      // Should show confirmation or update status
      await expect(page.locator('[data-testid="status-message"]')).toContainText('resolved');
    }
  });

  test('should show system health status', async ({ page }) => {
    await page.click('text=Dashboard');
    
    // Check for health status indicators
    await expect(page.locator('text=Operational').or(page.locator('text=Healthy'))).toBeVisible();
    
    // Check for health metrics
    const healthCards = page.locator('.bg-white:has-text("Health")');
    await expect(healthCards).toHaveCount(2); // Database and Redis/Cache health
  });
});

test.describe('Search Functionality E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
    
    // Login
    await page.waitForSelector('[data-testid="login-username"]');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    await page.waitForURL('**/admin');
  });

  test('should search for artists and display results', async ({ page }) => {
    // Navigate to search
    await page.click('text=Search');
    
    // Fill search box
    await page.fill('[data-testid="search-input"]', 'beatles');
    await page.press('[data-testid="search-input"]', 'Enter');
    
    // Verify search results appear
    await expect(page.locator('[data-testid="search-results"]')).toBeVisible();
    await expect(page.locator('[data-testid="artist-result"]')).toHaveCount({ '>=': 1 });
  });

  test('should search for albums and display results', async ({ page }) => {
    await page.click('text=Search');
    
    // Search for albums
    await page.fill('[data-testid="search-input"]', 'album');
    await page.click('[data-testid="search-type-album"]');
    await page.click('[data-testid="search-button"]');
    
    await expect(page.locator('[data-testid="album-result"]')).toHaveCount({ '>=': 1 });
  });
});

test.describe('Playlist Management E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
    
    // Login
    await page.waitForSelector('[data-testid="login-username"]');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    await page.waitForURL('**/admin');
  });

  test('should create a new playlist', async ({ page }) => {
    await page.click('text=Playlists');
    
    // Click create playlist button
    await page.click('button:has-text("Create Playlist")');
    
    // Fill playlist form
    await page.fill('[data-testid="playlist-name"]', 'Test Playlist');
    await page.fill('[data-testid="playlist-comment"]', 'Test description');
    
    // Submit form
    await page.click('button:has-text("Save")');
    
    // Verify playlist was created
    await expect(page.locator('text=Test Playlist')).toBeVisible();
  });

  test('should list existing playlists', async ({ page }) => {
    await page.click('text=Playlists');
    
    // Verify playlists are listed
    const playlistItems = page.locator('[data-testid="playlist-item"]');
    await expect(playlistItems).toHaveCount({ '>=': 1 });
  });

  test('should allow editing playlist details', async ({ page }) => {
    await page.click('text=Playlists');
    
    // Click edit on first playlist
    const editButtons = page.locator('button:has-text("Edit")');
    if (await editButtons.count() > 0) {
      await editButtons.first().click();
      
      // Modify playlist name
      await page.fill('[data-testid="playlist-name-edit"]', 'Updated Playlist Name');
      
      // Save changes
      await page.click('button:has-text("Update")');
      
      // Verify change was saved
      await expect(page.locator('text=Updated Playlist Name')).toBeVisible();
    }
  });
});

test.describe('User Management E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
    
    // Login as admin
    await page.waitForSelector('[data-testid="login-username"]');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    await page.waitForURL('**/admin');
  });

  test('should display user list', async ({ page }) => {
    await page.click('text=Users');
    
    // Verify user management page loads
    await expect(page.locator('h1:has-text("User Management")')).toBeVisible();
    
    // Check for users table
    await expect(page.locator('table')).toBeVisible();
    
    // Verify table has users
    const userRows = page.locator('tr:has-text("@"))'); // Look for email-like pattern
    await expect(userRows).toHaveCount({ '>=': 1 });
  });

  test('should create a new user', async ({ page }) => {
    await page.click('text=Users');
    
    // Click create user button
    await page.click('button:has-text("Create User")');
    
    // Fill user form
    await page.fill('[data-testid="username"]', 'testuser');
    await page.fill('[data-testid="email"]', 'test@example.com');
    await page.fill('[data-testid="password"]', 'SecurePass123!');
    await page.fill('[data-testid="confirm-password"]', 'SecurePass123!');
    
    // Toggle admin rights if needed
    await page.check('[data-testid="admin-checkbox"]');
    
    // Submit form
    await page.click('button:has-text("Create User")');
    
    // Verify user was created
    await expect(page.locator('text=testuser')).toBeVisible();
  });

  test('should handle user authentication', async ({ page }) => {
    // Log out first
    await page.click('text=Logout');
    await page.waitForURL('**/login');
    
    // Try to log in with new user (if created above)
    await page.fill('[data-testid="login-username"]', 'testuser');
    await page.fill('[data-testid="login-password"]', 'SecurePass123!');
    await page.click('[data-testid="login-submit"]');
    
    // Verify login works
    await expect(page).toHaveURL(/.*\/admin/);
  });
});

// Additional E2E tests for advanced functionality
test.describe('Advanced Features E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
    
    await page.waitForSelector('[data-testid="login-username"]');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    await page.waitForURL('**/admin');
  });

  test('should display capacity monitoring information', async ({ page }) => {
    await page.click('text=Dashboard');
    
    // Look for capacity monitoring elements
    await expect(page.locator('text=Capacity').or(page.locator('text=Storage'))).toBeVisible();
    
    // Check for storage percentage indicators
    const capacityElements = page.locator(':has-text("%")');
    await expect(capacityElements).toHaveCount({ '>=': 1 });
  });

  test('should handle invalid credentials gracefully', async ({ page }) => {
    // Navigate to admin
    await page.goto('/admin');
    
    // Enter invalid credentials
    await page.fill('[data-testid="login-username"]', 'invaliduser');
    await page.fill('[data-testid="login-password"]', 'wrongpassword');
    await page.click('[data-testid="login-submit"]');
    
    // Verify error message is displayed
    await expect(page.locator('[data-testid="error-message"]')).toContainText(/invalid|error|failed/i);
  });

  test('should maintain user session', async ({ page }) => {
    // Login
    await page.goto('/admin');
    await page.fill('[data-testid="login-username"]', 'admin');
    await page.fill('[data-testid="login-password"]', 'password');
    await page.click('[data-testid="login-submit"]');
    
    // Navigate to different sections to verify session persists
    await page.click('text=Libraries');
    await page.waitForSelector('h1:has-text("Library Management")');
    
    await page.click('text=Playlists');
    await page.waitForSelector('h1:has-text("Playlists")');
    
    await page.click('text=Users');
    await page.waitForSelector('h1:has-text("User Management")');
    
    // Session should remain active throughout navigation
    expect(page.url()).toContain('/admin');
  });
});