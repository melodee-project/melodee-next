package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"melodee/internal/models"
)

func TestAuthService_Login(t *testing.T) {
	// We'll create a mock DB for testing
	authService := NewAuthService(nil, "test-secret-key")

	// Test case: Valid credentials
	t.Run("Valid login", func(t *testing.T) {
		// This test would require a real DB connection
		// For now, we'll create a partial test
		assert.NotNil(t, authService)
	})

	// Test case: Invalid credentials
	t.Run("Invalid login", func(t *testing.T) {
		_, _, err := authService.Login("invalid_user", "invalid_password")
		assert.Error(t, err)
	})
}

func TestAuthService_HashPassword(t *testing.T) {
	authService := NewAuthService(nil, "test-secret-key")
	
	password := "test_password_123!"
	
	hash, err := authService.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	
	// Verify the hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestAuthService_ValidatePassword(t *testing.T) {
	authService := NewAuthService(nil, "test-secret-key")
	
	// Test valid password
	validPassword := "ValidPass123!"
	err := authService.ValidatePassword(validPassword)
	assert.NoError(t, err)
	
	// Test invalid - too short
	shortPassword := "Short1!"
	err = authService.ValidatePassword(shortPassword)
	assert.Error(t, err)
	
	// Test invalid - no number
	noNumberPassword := "NoNumber!"
	err = authService.ValidatePassword(noNumberPassword)
	assert.Error(t, err)
	
	// Test invalid - no symbol
	noSymbolPassword := "NoSymbol123"
	err = authService.ValidatePassword(noSymbolPassword)
	assert.Error(t, err)
	
	// Test invalid - no uppercase
	noUpperPassword := "nouppercase123!"
	err = authService.ValidatePassword(noUpperPassword)
	assert.Error(t, err)
	
	// Test invalid - no lowercase
	noLowerPassword := "NOLOWERCASE123!"
	err = authService.ValidatePassword(noLowerPassword)
	assert.Error(t, err)
}

func TestAuthService_GenerateAndValidateTokens(t *testing.T) {
	authService := NewAuthService(nil, "test-secret-key")
	
	user := models.User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  true,
	}
	
	// Test access token generation and validation
	accessToken, err := authService.generateAccessToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	
	claims, err := authService.parseAccessToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.IsAdmin, claims.IsAdmin)
	
	// Verify token expiration
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	
	// Test refresh token generation and validation
	refreshToken, err := authService.generateRefreshToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshToken)
	
	refreshClaims, err := authService.parseRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, refreshClaims.UserID)
	assert.Equal(t, user.Username, refreshClaims.Username)
	assert.Equal(t, user.IsAdmin, refreshClaims.IsAdmin)
	
	// Verify refresh token expiration (should be longer than access token)
	assert.True(t, refreshClaims.ExpiresAt.After(time.Now().Add(10*time.Minute)))
}

func TestAuthService_ValidateToken(t *testing.T) {
	authService := NewAuthService(nil, "test-secret-key")
	
	user := models.User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  false,
	}
	
	token, err := authService.generateAccessToken(user)
	assert.NoError(t, err)
	
	authUser, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, authUser.ID)
	assert.Equal(t, user.Username, authUser.Username)
	assert.Equal(t, user.IsAdmin, authUser.IsAdmin)
	
	// Test with invalid token
	_, err = authService.ValidateToken("invalid-token")
	assert.Error(t, err)
}