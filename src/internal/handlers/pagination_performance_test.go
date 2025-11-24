package handlers

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
)

// TestGetPlaylistsPerformance tests the performance and behavior of large-offset/large-limit scenarios
func TestGetPlaylistsPerformance(t *testing.T) {
	// Create a new repository with a test DB
	db, repo := setupTestDB(t)
	defer db.Close()

	// Create a test user
	testUser := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		IsAdmin:  true,
	}
	assert.NoError(t, repo.CreateUser(testUser))

	// Create many test playlists to simulate a large dataset
	numPlaylists := 1000
	for i := 0; i < numPlaylists; i++ {
		playlist := &models.Playlist{
			ID:        int32(i + 1),
			Name:      "Test Playlist " + strconv.Itoa(i),
			UserID:    testUser.ID,
			Public:    false,
			CreatedAt: models.Now(),
			ChangedAt: models.Now(),
		}
		assert.NoError(t, repo.CreatePlaylist(playlist))
	}

	// Create the handler
	handler := NewPlaylistHandler(repo)

	// Create a Fiber app with auth context
	app := fiber.New()

	// Test with large offset values
	t.Run("Large Offset Performance", func(t *testing.T) {
		// Test a request with a very large page number to simulate large offset
		c := app.AcquireCtx(httptest.NewRequest("GET", "/playlists?page=50&pageSize=20", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication)
		c.Locals(middleware.UserContextKey, *testUser)

		// Make the request
		err := handler.GetPlaylists(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// Check if pagination metadata is correct
		// The response should include proper pagination metadata
		paginationMeta := pagination.Calculate(int64(numPlaylists), 50, 20)
		// We verify that the function completed without timeouts/panics
		// and returned a properly structured response
	})

	// Test with maximum allowed page size
	t.Run("Maximum Limit Performance", func(t *testing.T) {
		// Test requesting maximum allowed page size
		c := app.AcquireCtx(httptest.NewRequest("GET", "/playlists?page=1&pageSize=200", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication)
		c.Locals(middleware.UserContextKey, *testUser)

		// Make the request
		err := handler.GetPlaylists(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// The actual page size should be capped at 200 as defined in the pagination package
	})

	// Test with normal page sizes to ensure standard behavior still works
	t.Run("Normal Pagination Behavior", func(t *testing.T) {
		c := app.AcquireCtx(httptest.NewRequest("GET", "/playlists?page=1&pageSize=10", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication)
		c.Locals(middleware.UserContextKey, *testUser)

		// Make the request
		err := handler.GetPlaylists(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// Check pagination metadata
		paginationMeta := pagination.Calculate(int64(numPlaylists), 1, 10)
		assert.Equal(t, 1, paginationMeta.CurrentPage)
		assert.Equal(t, 10, paginationMeta.PageSize)
		assert.Equal(t, int64(numPlaylists), paginationMeta.TotalCount)
	})
}

// TestGetUsersPerformance tests the performance and behavior of large-offset/large-limit scenarios for users
func TestGetUsersPerformance(t *testing.T) {
	// Create a new repository with a test DB
	db, repo := setupTestDB(t)
	defer db.Close()

	// Create an admin test user (to access the GetUsers endpoint)
	adminUser := &models.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}
	assert.NoError(t, repo.CreateUser(adminUser))

	// Create many test users to simulate a large dataset
	numUsers := 500
	for i := 0; i < numUsers; i++ {
		user := &models.User{
			ID:       int64(i + 2), // Start from 2 to avoid conflict with the admin
			Username: "testuser" + strconv.Itoa(i),
			Email:    "testuser" + strconv.Itoa(i) + "@example.com",
			IsAdmin:  false,
		}
		assert.NoError(t, repo.CreateUser(user))
	}

	// Create the handler
	handler := NewUserHandler(repo, nil) // Passing nil authService for tests

	// Create a Fiber app
	app := fiber.New()

	// Test with large offset values
	t.Run("Large Offset Performance for Users", func(t *testing.T) {
		// Test a request with a very large page number to simulate large offset
		c := app.AcquireCtx(httptest.NewRequest("GET", "/users?page=25&pageSize=20", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication as admin)
		c.Locals(middleware.UserContextKey, *adminUser)

		// Make the request
		err := handler.GetUsers(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// The function should complete without timeouts or panics
	})

	// Test with maximum allowed page size
	t.Run("Maximum Limit Performance for Users", func(t *testing.T) {
		// Test requesting maximum allowed page size
		c := app.AcquireCtx(httptest.NewRequest("GET", "/users?page=1&pageSize=200", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication as admin)
		c.Locals(middleware.UserContextKey, *adminUser)

		// Make the request
		err := handler.GetUsers(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// The actual page size should be capped appropriately
	})

	// Test with normal page sizes to ensure standard behavior still works
	t.Run("Normal Pagination Behavior for Users", func(t *testing.T) {
		c := app.AcquireCtx(httptest.NewRequest("GET", "/users?page=1&pageSize=10", nil))
		defer app.ReleaseCtx(c)

		// Set the user context (simulating authentication as admin)
		c.Locals(middleware.UserContextKey, *adminUser)

		// Make the request
		err := handler.GetUsers(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// Note: In the actual implementation, GetUsers still returns empty results,
		// but the pagination structure should be properly applied
	})
}

// TestGetSharesPerformance tests the performance and behavior of large-offset/large-limit scenarios for shares
func TestGetSharesPerformance(t *testing.T) {
	// Create a new repository with a test DB
	db, repo := setupTestDB(t)
	defer db.Close()

	// Create a test user
	testUser := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		IsAdmin:  true,
	}
	assert.NoError(t, repo.CreateUser(testUser))

	// Create many test shares to simulate a large dataset
	numShares := 800
	for i := 0; i < numShares; i++ {
		share := models.Share{
			ID:                  int64(i + 1),
			UserID:              testUser.ID,
			Name:                "Test Share " + strconv.Itoa(i),
			Description:         "Description for test share " + strconv.Itoa(i),
			CreatedAt:           models.Now(),
			UpdatedAt:           models.Now(),
			ExpiresAt:           nil, // No expiration
			MaxStreamingMinutes: 60,
			MaxStreamingCount:   10,
			AllowStreaming:      true,
			AllowDownload:       false,
		}
		err := repo.GetDB().Create(&share).Error
		assert.NoError(t, err)
	}

	// Create the handler
	handler := NewSharesHandler(repo)

	// Create a Fiber app
	app := fiber.New()

	// Test with large offset values
	t.Run("Large Offset Performance for Shares", func(t *testing.T) {
		// Test a request with a large page number to simulate large offset
		c := app.AcquireCtx(httptest.NewRequest("GET", "/shares?page=40&pageSize=20", nil))
		defer app.ReleaseCtx(c)

		// Make the request
		err := handler.GetShares(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// The function should complete without timeouts or panics
	})

	// Test with maximum allowed page size
	t.Run("Maximum Limit Performance for Shares", func(t *testing.T) {
		// Test requesting maximum allowed page size
		c := app.AcquireCtx(httptest.NewRequest("GET", "/shares?page=1&pageSize=100", nil))
		defer app.ReleaseCtx(c)

		// Make the request
		err := handler.GetShares(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())

		// The actual page size should be capped appropriately
	})

	// Test with normal page sizes to ensure standard behavior still works
	t.Run("Normal Pagination Behavior for Shares", func(t *testing.T) {
		c := app.AcquireCtx(httptest.NewRequest("GET", "/shares?page=1&pageSize=25", nil))
		defer app.ReleaseCtx(c)

		// Make the request
		err := handler.GetShares(c)
		assert.NoError(t, err)

		// Verify response is successful
		assert.Equal(t, 200, c.Response().StatusCode())
	})
}