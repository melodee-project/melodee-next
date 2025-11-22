package admin

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// UserAdminHandler manages admin user operations
type UserAdminHandler struct {
	db          *gorm.DB
	repo        *services.Repository
	authService *services.AuthService
}

// NewUserAdminHandler creates a new user admin handler
func NewUserAdminHandler(
	db *gorm.DB,
	repo *services.Repository,
	authService *services.AuthService,
) *UserAdminHandler {
	return &UserAdminHandler{
		db:          db,
		repo:        repo,
		authService: authService,
	}
}

// User represents a user in the response
type User struct {
	ID          int64     `json:"id"`
	APIKey      string    `json:"api_key"`
	Username    string    `json:"username"`
	Email       *string   `json:"email,omitempty"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// CreateUserRequest is the request structure for creating a user
type CreateUserRequest struct {
	Username   string `json:"username"`
	Email      *string `json:"email,omitempty"`
	Password   string `json:"password"`
	IsAdmin    bool   `json:"is_admin"`
}

// UpdateUserRequest is the request structure for updating a user
type UpdateUserRequest struct {
	Username   *string `json:"username,omitempty"`
	Email      *string `json:"email,omitempty"`
	Password   *string `json:"password,omitempty"`
	IsAdmin    *bool   `json:"is_admin,omitempty"`
}

// GetUsersResponse is the response structure for getting users
type GetUsersResponse struct {
	Data       []User     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// GetUsers retrieves all users (admin only)
func (h *UserAdminHandler) GetUsers(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse query parameters
	page := c.QueryInt("page", 1)
	size := c.QueryInt("size", 50)

	// Set limits
	if size > 100 {
		size = 100 // Max page size
	}

	// Get total count
	var total int64
	if err := h.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count users",
		})
	}

	// Get users with pagination
	var dbUsers []models.User
	offset := (page - 1) * size
	if err := h.db.Offset(offset).Limit(size).Find(&dbUsers).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve users",
		})
	}

	// Convert to response format
	users := make([]User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = User{
			ID:          u.ID,
			APIKey:      u.APIKey.String(),
			Username:    u.Username,
			Email:       u.Email,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   u.CreatedAt,
			LastLoginAt: u.LastLoginAt,
		}
	}

	response := GetUsersResponse{
		Data: users,
		Pagination: Pagination{
			Page:  page,
			Size:  len(users),
			Total: int(total),
		},
	}

	return c.JSON(response)
}

// GetUser retrieves a specific user
func (h *UserAdminHandler) GetUser(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var dbUser models.User
	if err := h.db.First(&dbUser, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user",
		})
	}

	user := User{
		ID:          dbUser.ID,
		APIKey:      dbUser.APIKey.String(),
		Username:    dbUser.Username,
		Email:       dbUser.Email,
		IsAdmin:     dbUser.IsAdmin,
		CreatedAt:   dbUser.CreatedAt,
		LastLoginAt: dbUser.LastLoginAt,
	}

	return c.JSON(user)
}

// CreateUser creates a new user
func (h *UserAdminHandler) CreateUser(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req CreateUserRequest
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

	// Validate password strength
	if err := h.authService.ValidatePassword(req.Password); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "Password validation failed",
			"details": err.Error(),
		})
	}

	// Check if username already exists
	var existingUser models.User
	if err := h.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": "Username already exists",
		})
	} else if err != gorm.ErrRecordNotFound {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing user",
		})
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Create new user
	dbUser := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		IsAdmin:      req.IsAdmin,
		CreatedAt:    time.Now(),
	}

	if err := h.db.Create(&dbUser).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Return created user
	user := User{
		ID:          dbUser.ID,
		APIKey:      dbUser.APIKey.String(),
		Username:    dbUser.Username,
		Email:       dbUser.Email,
		IsAdmin:     dbUser.IsAdmin,
		CreatedAt:   dbUser.CreatedAt,
		LastLoginAt: dbUser.LastLoginAt,
	}

	return c.Status(http.StatusCreated).JSON(user)
}

// UpdateUser updates an existing user
func (h *UserAdminHandler) UpdateUser(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var dbUser models.User
	if err := h.db.First(&dbUser, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update fields if provided
	if req.Username != nil {
		dbUser.Username = *req.Username
	}
	if req.Email != nil {
		dbUser.Email = req.Email
	}
	if req.IsAdmin != nil {
		dbUser.IsAdmin = *req.IsAdmin
	}
	if req.Password != nil {
		// Validate password strength
		if err := h.authService.ValidatePassword(*req.Password); err != nil {
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
		dbUser.PasswordHash = passwordHash
	}

	if err := h.db.Save(&dbUser).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	// Return updated user
	user := User{
		ID:          dbUser.ID,
		APIKey:      dbUser.APIKey.String(),
		Username:    dbUser.Username,
		Email:       dbUser.Email,
		IsAdmin:     dbUser.IsAdmin,
		CreatedAt:   dbUser.CreatedAt,
		LastLoginAt: dbUser.LastLoginAt,
	}

	return c.JSON(user)
}

// DeleteUser deletes a user
func (h *UserAdminHandler) DeleteUser(c *fiber.Ctx) error {
	// Check if user is admin
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok || !currentUser.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Prevent admin from deleting themselves
	if int64(userID) == currentUser.ID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot delete yourself",
		})
	}

	var existingUser models.User
	if err := h.db.First(&existingUser, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify user exists",
		})
	}

	// Delete the user
	if err := h.db.Delete(&existingUser).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}