package middleware

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"melodee/internal/models"
)

// OpenSubsonicAuthMiddleware handles OpenSubsonic authentication
type OpenSubsonicAuthMiddleware struct {
	db        *gorm.DB
	jwtSecret string
}

// NewOpenSubsonicAuthMiddleware creates a new OpenSubsonic auth middleware
func NewOpenSubsonicAuthMiddleware(db *gorm.DB, jwtSecret string) *OpenSubsonicAuthMiddleware {
	return &OpenSubsonicAuthMiddleware{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// Authenticate authenticates requests using OpenSubsonic auth methods
func (m *OpenSubsonicAuthMiddleware) Authenticate(c *fiber.Ctx) error {
	// Try different authentication methods in order:
	// 1. Username/password parameters
	// 2. Authorization header
	// 3. Username/token parameters

	username := c.Query("u", "")
	password := c.Query("p", "")
	token := c.Query("t", "")
	salt := c.Query("s", "")

	var user *models.User
	var err error

	// Check for token-based authentication (Subsonic API method)
	if username != "" && token != "" && salt != "" {
		user, err = m.authenticateWithToken(username, password, token, salt)
	} else if username != "" && password != "" {
		// Standard username/password authentication
		user, err = m.authenticateWithPassword(username, password)
	} else {
		// Check for Authorization header
		authHeader := c.Get("Authorization", "")
		if authHeader != "" {
			user, err = m.authenticateWithHeader(authHeader)
		}
	}

	if err != nil || user == nil {
		return m.sendOpenSubsonicError(c, 50, "not authorized")
	}

	// Store user in context
	c.Locals("user", user)

	// Continue to the next handler
	return c.Next()
}

// authenticateWithPassword handles username/password authentication
func (m *OpenSubsonicAuthMiddleware) authenticateWithPassword(username, password string) (*models.User, error) {
	// Find user by username
	var user models.User
	if err := m.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return &user, nil
}

// authenticateWithToken handles token-based authentication (Subsonic-style)
func (m *OpenSubsonicAuthMiddleware) authenticateWithToken(username, password, token, salt string) (*models.User, error) {
	// Find user by username
	var user models.User
	if err := m.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// In Subsonic-style auth, the token is the MD5 hash of (password + salt)
	// For this implementation, we'll verify the token matches what we'd expect
	// from the stored password hash
	expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(user.PasswordHash+salt)))

	if expectedToken != token {
		return nil, fmt.Errorf("invalid token")
	}

	return &user, nil
}

// authenticateWithHeader handles HTTP Basic Authentication
func (m *OpenSubsonicAuthMiddleware) authenticateWithHeader(authHeader string) (*models.User, error) {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return nil, fmt.Errorf("invalid authorization header")
	}

	// For this implementation, we'll handle Basic auth
	// In a real implementation, we'd decode the base64 credentials
	// This is a simplified version
	return nil, fmt.Errorf("basic auth not implemented in this example")
}

// sendOpenSubsonicError sends an OpenSubsonic formatted error response
func (m *OpenSubsonicAuthMiddleware) sendOpenSubsonicError(c *fiber.Ctx, code int, message string) error {
	// Set the X-Status-Code header for observability
	c.Set("X-Status-Code", "401")
	
	// Send as XML response (OpenSubsonic format)
	xmlResponse := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><subsonic-response status="failed" version="1.16.1"><error code="%d" message="%s"/></subsonic-response>`,
		code,
		message,
	)
	
	c.Set("Content-Type", "text/xml; charset=utf-8")
	return c.Status(200).SendString(xmlResponse)
}

// GetUserFromContext retrieves the authenticated user from the context
func GetUserFromContext(c *fiber.Ctx) (*models.User, bool) {
	user, ok := c.Locals("user").(*models.User)
	return user, ok
}