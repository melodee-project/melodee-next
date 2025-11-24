package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
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
		return utils.SendForbiddenError(c, "Admin access required")
	}

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 10)

	// In a real implementation, we would fetch users with pagination
	// For now, we'll return an empty list
	users := []models.User{}
	total := int64(0)

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       users,
		"pagination": paginationMeta,
	})
}

// CreateUser handles creating a new user
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return utils.SendForbiddenError(c, "Admin access required")
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		return utils.SendError(c, http.StatusBadRequest, "Username and password are required")
	}

	// Validate password
	if err := utils.ValidatePassword(req.Password); err != nil {
		return utils.SendValidationError(c, "password", err.Error())
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to hash password")
	}

	// Create user
	newUser := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		IsAdmin:      req.IsAdmin,
	}

	if err := h.repo.CreateUser(newUser); err != nil {
		return utils.SendInternalServerError(c, "Failed to create user")
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
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
	}

	// Allow access if user is admin or requesting their own profile
	if !currentUser.IsAdmin && currentUser.ID != int64(userID) {
		return utils.SendForbiddenError(c, "Access denied")
	}

	user, err := h.repo.GetUserByID(int64(userID))
	if err != nil {
		return utils.SendNotFoundError(c, "User")
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
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
	}

	// Allow update if user is admin or updating their own profile
	if !currentUser.IsAdmin && currentUser.ID != int64(userID) {
		return utils.SendForbiddenError(c, "Access denied")
	}

	var req struct {
		Username *string `json:"username,omitempty"`
		Email    *string `json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
		IsAdmin  *bool   `json:"is_admin,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Get the user to update
	user, err := h.repo.GetUserByID(int64(userID))
	if err != nil {
		return utils.SendNotFoundError(c, "User")
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
			return utils.SendForbiddenError(c, "Only admins can change admin status")
		}
		user.IsAdmin = *req.IsAdmin
	}
	if req.Password != nil {
		// Validate password if provided
		if err := utils.ValidatePassword(*req.Password); err != nil {
			return utils.SendValidationError(c, "password", err.Error())
		}

		// Hash new password
		passwordHash, err := h.authService.HashPassword(*req.Password)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to hash password")
		}
		user.PasswordHash = passwordHash
	}

	if err := h.repo.UpdateUser(user); err != nil {
		return utils.SendInternalServerError(c, "Failed to update user")
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
		return utils.SendForbiddenError(c, "Admin access required")
	}

	// Get user ID from route params
	userID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
	}

	// Prevent admin from deleting themselves
	if currentUser.ID == int64(userID) {
		return utils.SendError(c, http.StatusBadRequest, "Cannot delete your own account")
	}

	// Delete the user
	if err := h.repo.DeleteUser(int64(userID)); err != nil {
		return utils.SendInternalServerError(c, "Failed to delete user")
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
		"message": "User deleted successfully",
	})
}