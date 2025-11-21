package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// UserHandler handles user-related requests
type UserHandler struct {
	repo        *services.Repository
	authService *services.AuthService
}

// NewUserHandler creates a new user handler
func NewUserHandler(repo *services.Repository, authService *services.AuthService) *UserHandler {
	return &UserHandler{
		repo:        repo,
		authService: authService,
	}
}

// GetUsers handles retrieving all users (admin only)
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// In a real implementation, we would fetch users with pagination
	// For now, we'll return an empty list
	users := []models.User{}
	total := 0

	return c.JSON(fiber.Map{
		"data": users,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// CreateUser handles creating a new user
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}

	// Validate password
	if err := utils.ValidatePassword(req.Password); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "Password validation failed",
			"details": err.Error(),
		})
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Create user
	newUser := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		IsAdmin:      req.IsAdmin,
	}

	if err := h.repo.CreateUser(newUser); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":       newUser.ID,
		"username": newUser.Username,
		"email":    newUser.Email,
		"is_admin": newUser.IsAdmin,
		"message":  "User created successfully",
	})
}

// GetUser handles retrieving a specific user
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	// Check if user is admin or accessing own profile
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Allow access if user is admin or requesting their own profile
	if !currentUser.IsAdmin && currentUser.ID != int64(userID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	user, err := h.repo.GetUserByID(int64(userID))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"is_admin": user.IsAdmin,
		"created_at": user.CreatedAt,
		"last_login_at": user.LastLoginAt,
	})
}

// UpdateUser handles updating a user
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	// Check if user is admin or updating own profile
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Allow update if user is admin or updating their own profile
	if !currentUser.IsAdmin && currentUser.ID != int64(userID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var req struct {
		Username *string `json:"username,omitempty"`
		Email    *string `json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
		IsAdmin  *bool   `json:"is_admin,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get the user to update
	user, err := h.repo.GetUserByID(int64(userID))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Update fields if provided
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.IsAdmin != nil {
		// Only admin can change admin status
		if !currentUser.IsAdmin {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "Only admins can change admin status",
			})
		}
		user.IsAdmin = *req.IsAdmin
	}
	if req.Password != nil {
		// Validate password if provided
		if err := utils.ValidatePassword(*req.Password); err != nil {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
				"error":   "Password validation failed",
				"details": err.Error(),
			})
		}

		// Hash new password
		passwordHash, err := h.authService.HashPassword(*req.Password)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to hash password",
			})
		}
		user.PasswordHash = passwordHash
	}

	if err := h.repo.UpdateUser(user); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(fiber.Map{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"is_admin": user.IsAdmin,
		"message":  "User updated successfully",
	})
}

// DeleteUser handles deleting a user
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Prevent admin from deleting themselves
	if currentUser.ID == int64(userID) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	// Delete the user
	if err := h.repo.DeleteUser(int64(userID)); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
		"message": "User deleted successfully",
	})
}