package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login handles user login requests
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}

	// Authenticate user
	authToken, user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	return c.JSON(fiber.Map{
		"access_token":  authToken.AccessToken,
		"refresh_token": authToken.RefreshToken,
		"expires_in":    authToken.ExpiresIn,
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
	})
}

// Refresh handles token refresh requests
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	authToken, user, err := h.authService.RefreshTokens(req.RefreshToken)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"access_token":  authToken.AccessToken,
		"refresh_token": authToken.RefreshToken,
		"expires_in":    authToken.ExpiresIn,
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
	})
}

// RequestReset handles password reset requests
func (h *AuthHandler) RequestReset(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// This would normally send a reset email
	// For now, we'll just return success to avoid user enumeration
	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		// Log the error but don't reveal it to the user
		// In a real implementation, we'd still return 202 to avoid enumeration
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"message": "If an account with this email exists, a reset link will be sent",
	})
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Token == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token and password are required",
		})
	}

	// Validate password
	if err := utils.ValidatePassword(req.Password); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "Password validation failed",
			"details": err.Error(),
		})
	}

	if err := h.authService.ResetPassword(req.Token, req.Password); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Password reset failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
	})
}