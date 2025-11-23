package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestAuthHandler_Login(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	authHandler := NewAuthHandler(authService)

	// Create test user
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/api/auth/login", authHandler.Login)

	// Test successful login
	t.Run("Successful login", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser",
			"password": "ValidPass123!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")
		assert.Contains(t, response, "expires_in")
		assert.Contains(t, response, "user")
		assert.Equal(t, float64(900), response["expires_in"])
	})

	// Test invalid credentials
	t.Run("Invalid credentials", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser",
			"password": "wrongpassword",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test missing credentials
	t.Run("Missing credentials", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "",
			"password": "password",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAuthHandler_RequestReset(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	authHandler := NewAuthHandler(authService)

	// Create test user
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		APIKey:   uuid.New(),
	}

	err := db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/api/auth/request-reset", authHandler.RequestReset)

	// Test successful password reset request
	t.Run("Successful password reset request", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "test@example.com",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/request-reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "message")
	})

	// Test with non-existent email (should not return error to prevent enumeration)
	t.Run("Request reset for non-existent email", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "nonexistent@example.com",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/request-reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should still return 202 to prevent user enumeration
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	// Test missing email
	t.Run("Missing email", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/request-reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	authHandler := NewAuthHandler(authService)

	// Create test user with reset token
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	resetToken := "test-reset-token"
	hashedResetToken, err := bcrypt.GenerateFromPassword([]byte(resetToken), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedResetTokenStr := string(hashedResetToken)

	// Set the password reset expiry to a future time
	futureTime := time.Now().Add(1 * time.Hour)

	user := &models.User{
		Username:            "testuser",
		Email:               "test@example.com",
		PasswordHash:        string(hashedPassword),
		PasswordResetToken:  &hashedResetTokenStr,
		PasswordResetExpiry: &futureTime,
		APIKey:              uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/api/auth/reset", authHandler.ResetPassword)

	// Test successful password reset
	t.Run("Successful password reset", func(t *testing.T) {
		reqBody := map[string]string{
			"token":    resetToken,
			"password": "NewValidPass456!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "status")
		assert.Equal(t, "ok", response["status"])
	})

	// Test password that doesn't meet requirements
	t.Run("Password does not meet requirements", func(t *testing.T) {
		reqBody := map[string]string{
			"token":    resetToken,
			"password": "short", // Too short
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test with expired token
	t.Run("Expired reset token", func(t *testing.T) {
		// Create user with past expiry time
		pastTime := time.Now().Add(-1 * time.Hour)
		userWithExpired := &models.User{
			Username:            "expireduser",
			Email:               "expired@example.com",
			PasswordHash:        string(hashedPassword),
			PasswordResetToken:  &hashedResetTokenStr,
			PasswordResetExpiry: &pastTime,
			APIKey:              uuid.New(),
		}

		err = db.Create(userWithExpired).Error
		assert.NoError(t, err)

		reqBody := map[string]string{
			"token":    resetToken,
			"password": "NewValidPass456!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test invalid token
	t.Run("Invalid reset token", func(t *testing.T) {
		reqBody := map[string]string{
			"token":    "wrong-token",
			"password": "NewValidPass456!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test missing fields
	t.Run("Missing fields", func(t *testing.T) {
		reqBody := map[string]string{
			"token":    "", // empty token
			"password": "NewValidPass456!",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAuthHandler_AccountLockout(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := services.NewAuthService(db, "test-secret-key-change-in-production")
	authHandler := NewAuthHandler(authService)

	// Create test user
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/api/auth/login", authHandler.Login)

	// Simulate 5 failed login attempts to trigger account lockout
	for i := 0; i < 5; i++ {
		reqBody := map[string]string{
			"username": "testuser",
			"password": "wrongpassword", // Wrong password to trigger failed attempt
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// Verify user is locked after 5 failed attempts
	var lockedUser models.User
	err = db.First(&lockedUser, user.ID).Error
	assert.NoError(t, err)
	assert.Greater(t, lockedUser.FailedLoginAttempts, 0)
	assert.NotNil(t, lockedUser.LockedUntil)
	assert.True(t, lockedUser.LockedUntil.After(time.Now()))

	// Now try to log in with correct credentials - should fail due to lockout
	reqBody := map[string]string{
		"username": "testuser",
		"password": "ValidPass123!", // Correct password
	}

	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode) // Should still fail due to lockout

	// Verify user is still locked
	err = db.First(&lockedUser, user.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, lockedUser.LockedUntil)
}