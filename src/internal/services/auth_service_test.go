package services

import (
	"testing"
	"time"

	"melodee/internal/models"
	"melodee/internal/test"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

	// Create a test user
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

	// Test successful login
	authToken, loggedInUser, err := authService.Login("testuser", password)
	assert.NoError(t, err)
	assert.NotNil(t, authToken)
	assert.Equal(t, "testuser", loggedInUser.Username)

	// Verify the token is valid
	token, err := jwt.ParseWithClaims(authToken.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key-change-in-production"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*Claims)
	assert.True(t, ok)
	assert.Equal(t, int64(user.ID), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.False(t, claims.IsAdmin)

	// Test invalid credentials
	_, _, err = authService.Login("testuser", "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")

	// Test non-existent user
	_, _, err = authService.Login("nonexistent", "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestAuthService_ValidatePassword(t *testing.T) {
	authService := &AuthService{}

	// Test valid password
	err := authService.ValidatePassword("ValidPass123!")
	assert.NoError(t, err)

	// Test short password
	err = authService.ValidatePassword("Short1!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 12 characters")

	// Test password without uppercase
	err = authService.ValidatePassword("lowercasepass123!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contain at least one uppercase letter")

	// Test password without lowercase
	err = authService.ValidatePassword("UPPERCASEPASS123!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contain at least one lowercase letter")

	// Test password without number
	err = authService.ValidatePassword("UppercasePass!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contain at least one number")

	// Test password without symbol
	err = authService.ValidatePassword("UppercasePass123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contain at least one special symbol")

	// Test valid password with all requirements
	err = authService.ValidatePassword("ValidPass123!")
	assert.NoError(t, err)
}

func TestAuthService_HashPassword(t *testing.T) {
	authService := &AuthService{}

	password := "ValidPass123!"
	hash, err := authService.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the hash matches the original password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)

	// Verify it doesn't match a different password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("DifferentPass456!"))
	assert.Error(t, err)
}

func TestAuthService_RefreshTokens(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

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

	// Create a refresh token
	refreshClaims := &Claims{
		UserID:   user.ID,
		Username: "testuser",
		IsAdmin:  false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(14 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "melodee",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte("test-secret-key-change-in-production"))
	assert.NoError(t, err)

	authToken, refreshedUser, err := authService.RefreshTokens(refreshTokenString)
	assert.NoError(t, err)
	assert.NotNil(t, authToken)
	assert.Equal(t, "testuser", refreshedUser.Username)

	// Verify the new access token is valid
	newToken, err := jwt.ParseWithClaims(authToken.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key-change-in-production"), nil
	})
	assert.NoError(t, err)
	assert.True(t, newToken.Valid)

	newClaims, ok := newToken.Claims.(*Claims)
	assert.True(t, ok)
	assert.Equal(t, user.ID, newClaims.UserID)
	assert.Equal(t, "testuser", newClaims.Username)
	assert.False(t, newClaims.IsAdmin)

	// Test with invalid token
	_, _, err = authService.RefreshTokens("invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

	// Create test user
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		APIKey:   uuid.New(),
	}

	err := db.Create(user).Error
	assert.NoError(t, err)

	// Test the request reset functionality
	err = authService.RequestPasswordReset("test@example.com")
	assert.NoError(t, err)

	// Verify user was updated with a reset token
	var updatedUser models.User
	err = db.First(&updatedUser, user.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, updatedUser.PasswordResetToken)
	assert.NotNil(t, updatedUser.PasswordResetExpiry)
	assert.True(t, updatedUser.PasswordResetExpiry.After(time.Now()))

	// Test with non-existent email (should not return error to prevent enumeration)
	err = authService.RequestPasswordReset("nonexistent@example.com")
	assert.NoError(t, err) // Should not return an error to prevent email enumeration
}

func TestAuthService_ResetPassword(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

	// Create test user with reset token
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	resetToken := "test-reset-token"
	hashedResetTokenBytes, err := bcrypt.GenerateFromPassword([]byte(resetToken), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedResetToken := string(hashedResetTokenBytes)

	// Set the password reset expiry to a future time
	futureTime := time.Now().Add(1 * time.Hour)

	user := &models.User{
		Username:            "testuser",
		Email:               "test@example.com",
		PasswordHash:        string(hashedPassword),
		PasswordResetToken:  &hashedResetToken,
		PasswordResetExpiry: &futureTime,
		APIKey:              uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Test valid password reset
	newPassword := "NewValidPass456!"
	err = authService.ResetPassword(resetToken, newPassword)
	assert.NoError(t, err)

	// Verify password was updated and reset tokens cleared
	var updatedUser models.User
	err = db.First(&updatedUser, user.ID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, user.PasswordHash, updatedUser.PasswordHash) // Password should be changed
	assert.Nil(t, updatedUser.PasswordResetToken)                   // Token should be cleared
	assert.Nil(t, updatedUser.PasswordResetExpiry)                  // Expiry should be cleared

	// Verify new password works for login
	_, _, err = authService.Login("testuser", newPassword)
	assert.NoError(t, err)

	// Test with expired token (manually manipulate the DB to expire the token)
	expiredTokenCopy := hashedResetToken
	userWithExpiredToken := &models.User{
		Username:            "expireduser",
		Email:               "expired@example.com",
		PasswordHash:        string(hashedPassword),
		PasswordResetToken:  &expiredTokenCopy,
		PasswordResetExpiry: &time.Time{}, // Zero time in the past
		APIKey:              uuid.New(),
	}

	err = db.Create(userWithExpiredToken).Error
	assert.NoError(t, err)

	err = authService.ResetPassword(resetToken, newPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset token has expired")

	// Test with invalid token
	err = authService.ResetPassword("wrong-token", newPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired reset token")
}

func TestAuthService_ResetUserPassword(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

	// Create test user
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// Create a user with existing password reset tokens to test they get cleared
	resetToken := "existing-reset-token"
	hashedResetTokenBytes, err := bcrypt.GenerateFromPassword([]byte(resetToken), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedResetToken := string(hashedResetTokenBytes)

	resetExpiry := time.Now().Add(1 * time.Hour)

	user := &models.User{
		Username:            "testuser",
		Email:               "test@example.com",
		PasswordHash:        string(hashedPassword),
		PasswordResetToken:  &hashedResetToken,
		PasswordResetExpiry: &resetExpiry,
		APIKey:              uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Test valid password reset by admin
	newPassword := "NewValidPass456!"
	err = authService.ResetUserPassword(user.ID, newPassword)
	assert.NoError(t, err)

	// Verify password was updated and reset tokens cleared
	var updatedUser models.User
	err = db.First(&updatedUser, user.ID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, user.PasswordHash, updatedUser.PasswordHash) // Password should be changed
	assert.Nil(t, updatedUser.PasswordResetToken)                   // Token should be cleared
	assert.Nil(t, updatedUser.PasswordResetExpiry)                  // Expiry should be cleared

	// Verify new password works for login
	_, _, err = authService.Login("testuser", newPassword)
	assert.NoError(t, err)

	// Test with invalid user ID
	err = authService.ResetUserPassword(99999, newPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	// Test with invalid password
	err = authService.ResetUserPassword(user.ID, "short1!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password validation failed")
}

func TestAuthService_LockUser(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	authService := NewAuthService(db, "test-secret-key-change-in-production")

	// Create test user
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		APIKey:   uuid.New(),
	}

	err := db.Create(user).Error
	assert.NoError(t, err)

	err = authService.LockUser(user.ID)
	assert.NoError(t, err)

	// Verify the user is locked
	var lockedUser models.User
	err = db.First(&lockedUser, user.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, lockedUser.LockedUntil)
	assert.True(t, lockedUser.LockedUntil.After(time.Now()))
}
