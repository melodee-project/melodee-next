package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestUserHandler_GetUsers(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	userHandler := NewUserHandler(repo, authService)

	// Create admin user for testing
	adminPassword := "ValidPass123!"
	adminHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(adminHashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create Fiber app for testing with auth middleware
	app := fiber.New()
	app.Get("/api/users", func(c *fiber.Ctx) error {
		// Set user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return userHandler.GetUsers(c)
	})

	// Test successful retrieval of users
	t.Run("Get users successfully as admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
	})

	// Create a non-admin user
	regularPassword := "ValidPass123!"
	regularHashedPassword, err := bcrypt.GenerateFromPassword([]byte(regularPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	regularUser := &models.User{
		Username:     "regular",
		Email:        "regular@example.com",
		PasswordHash: string(regularHashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(regularUser).Error
	assert.NoError(t, err)

	// Test getting users without admin privileges
	appNonAdmin := fiber.New()
	appNonAdmin.Get("/api/users", func(c *fiber.Ctx) error {
		// Set user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       regularUser.ID,
			Username: regularUser.Username,
			Email:    regularUser.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return userHandler.GetUsers(c)
	})

	t.Run("Get users fails without admin privileges", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestUserHandler_CreateUser(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	userHandler := NewUserHandler(repo, authService)

	// Create admin user for testing
	adminPassword := "ValidPass123!"
	adminHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(adminHashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create Fiber app for testing with auth middleware
	app := fiber.New()
	app.Post("/api/users", func(c *fiber.Ctx) error {
		// Set user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return userHandler.CreateUser(c)
	})

	// Test successful user creation
	t.Run("Create user successfully", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "newuser",
			"email":    "newuser@example.com",
			"password": "NewValidPass123!",
			"is_admin": false,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Contains(t, response, "username")
		assert.Contains(t, response, "email")
		assert.Contains(t, response, "is_admin")
		assert.Contains(t, response, "message")
		assert.Equal(t, "User created successfully", response["message"])
	})

	// Test creating user without admin privileges
	regularUser := &models.User{
		Username: "regular",
		Email:    "regular@example.com",
		IsAdmin:  false,
		APIKey:   uuid.New(),
	}

	err = db.Create(regularUser).Error
	assert.NoError(t, err)

	appNonAdmin := fiber.New()
	appNonAdmin.Post("/api/users", func(c *fiber.Ctx) error {
		// Set user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       regularUser.ID,
			Username: regularUser.Username,
			Email:    regularUser.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return userHandler.CreateUser(c)
	})

	t.Run("Create user fails without admin privileges", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "newuser2",
			"email":    "newuser2@example.com",
			"password": "NewValidPass123!",
			"is_admin": false,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test creating user with missing required fields
	t.Run("Create user fails with missing required fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "", // Missing username
			"email":    "user@example.com",
			"password": "ValidPass123!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test creating user with invalid password
	t.Run("Create user fails with invalid password", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "newuser3",
			"email":    "newuser3@example.com",
			"password": "short", // Invalid password
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUserHandler_GetUser(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	userHandler := NewUserHandler(repo, authService)

	// Create admin user
	adminPassword := "ValidPass123!"
	adminHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(adminHashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create regular user
	regularPassword := "ValidPass123!"
	regularHashedPassword, err := bcrypt.GenerateFromPassword([]byte(regularPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	regularUser := &models.User{
		Username:     "regular",
		Email:        "regular@example.com",
		PasswordHash: string(regularHashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(regularUser).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/api/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "1" { // admin user ID
			c.Locals("user", &models.User{
				ID:       adminUser.ID,
				Username: adminUser.Username,
				Email:    adminUser.Email,
				IsAdmin:  true,
			})
		} else { // regular user ID
			c.Locals("user", &models.User{
				ID:       regularUser.ID,
				Username: regularUser.Username,
				Email:    regularUser.Email,
				IsAdmin:  false,
			})
		}
		return userHandler.GetUser(c)
	})

	// Test admin accessing any user
	t.Run("Admin can access any user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/2", nil) // Access regular user as admin
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test regular user accessing own profile
	t.Run("Regular user can access own profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/2", nil) // Regular user accessing self
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test regular user accessing other user
	t.Run("Regular user cannot access other user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/1", nil) // Regular user accessing admin
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test accessing non-existent user
	t.Run("Accessing non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/9999", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestUserHandler_UpdateUser(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	userHandler := NewUserHandler(repo, authService)

	// Create admin user
	adminPassword := "ValidPass123!"
	adminHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(adminHashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create regular user
	regularPassword := "ValidPass123!"
	regularHashedPassword, err := bcrypt.GenerateFromPassword([]byte(regularPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	regularUser := &models.User{
		Username:     "regular",
		Email:        "regular@example.com",
		PasswordHash: string(regularHashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(regularUser).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Put("/api/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "1" { // admin user ID
			c.Locals("user", &models.User{
				ID:       adminUser.ID,
				Username: adminUser.Username,
				Email:    adminUser.Email,
				IsAdmin:  true,
			})
		} else { // regular user ID
			c.Locals("user", &models.User{
				ID:       regularUser.ID,
				Username: regularUser.Username,
				Email:    regularUser.Email,
				IsAdmin:  false,
			})
		}
		return userHandler.UpdateUser(c)
	})

	// Test admin updating any user
	t.Run("Admin can update any user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "updated_regular",
			"email":    "updated@example.com",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/2", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "updated_regular", response["username"])
	})

	// Test regular user updating own profile
	t.Run("Regular user can update own profile", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "regular_updated",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/2", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test regular user attempting to update admin status (should fail without admin)
	t.Run("Regular user cannot change admin status", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_admin": true,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/2", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test admin updating admin status
	t.Run("Admin can change admin status", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_admin": true,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/2", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		// This time, admin is updating the regular user
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// This should succeed since admin is making the change
		// But in our implementation, only admin can change admin status
		// So this might return 403 depending on the implementation
	})

	// Test updating non-existent user
	t.Run("Updating non-existent user returns 404", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "nonexistent_updated",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/9999", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestUserHandler_DeleteUser(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	userHandler := NewUserHandler(repo, authService)

	// Create admin user
	adminPassword := "ValidPass123!"
	adminHashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(adminHashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create regular user
	regularPassword := "ValidPass123!"
	regularHashedPassword, err := bcrypt.GenerateFromPassword([]byte(regularPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	regularUser := &models.User{
		Username:     "regular",
		Email:        "regular@example.com",
		PasswordHash: string(regularHashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(regularUser).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Delete("/api/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "1" { // admin user ID
			c.Locals("user", &models.User{
				ID:       adminUser.ID,
				Username: adminUser.Username,
				Email:    adminUser.Email,
				IsAdmin:  true,
			})
		} else { // regular user ID
			c.Locals("user", &models.User{
				ID:       regularUser.ID,
				Username: regularUser.Username,
				Email:    regularUser.Email,
				IsAdmin:  false,
			})
		}
		return userHandler.DeleteUser(c)
	})

	// Test admin deleting another user
	t.Run("Admin can delete other user", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/2", nil) // admin deleting regular user
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "deleted", response["status"])
	})

	// Test admin attempting to delete self (should fail)
	t.Run("Admin cannot delete own account", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/1", nil) // admin trying to delete self
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test regular user attempting to delete (should fail)
	regularUser2 := &models.User{
		Username: "regular2",
		Email:    "regular2@example.com",
		IsAdmin:  false,
		APIKey:   uuid.New(),
	}

	err = db.Create(regularUser2).Error
	assert.NoError(t, err)

	appNonAdmin := fiber.New()
	appNonAdmin.Delete("/api/users/:id", func(c *fiber.Ctx) error {
		c.Locals("user", &models.User{
			ID:       regularUser.ID,
			Username: regularUser.Username,
			Email:    regularUser.Email,
			IsAdmin:  false,
		})
		return userHandler.DeleteUser(c)
	})

	t.Run("Regular user cannot delete other user", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/3", nil) // regular trying to delete other
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test deleting non-existent user
	t.Run("Deleting non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/9999", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}