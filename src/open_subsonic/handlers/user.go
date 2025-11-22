package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// UserHandler handles OpenSubsonic user management endpoints
type UserHandler struct {
	db *gorm.DB
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		db: db,
	}
}

// GetUser returns information about a user
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	username := c.Query("username", "")
	if username == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter username")
	}

	// Get the user
	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "User not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve user")
	}

	// Create response
	response := utils.SuccessResponse()
	userResp := User{
		Username:    user.Username,
		Email:       user.Email,
		ScrobblingEnabled: true, // Default true
		AdminRole:   user.IsAdmin,
		SettingsRole: true,     // User can change settings
		StreamRole:  true,     // User can stream
		JukeboxRole: false,    // Default false
		UploadRole:  false,    // Default false
		FolderRole:  []int{0}, // Access to all folders by default
		PlaylistRole: true,   // Can manage playlists
		CommentRole: true,    // Can create comments
		PodcastRole: false,   // Default false
		CoverArtRole: true,   // Can change cover art
		AvatarRole: true,     // Can change avatar
		ShareRole: true,      // Can create shares
		VideoConversionRole: false,  // Default false
		MusicFolderId: []int{0},     // All music folders by default
		MaxBitRate: 320,             // Maximum bit rate allowed
		LfmUsername: user.Username,  // Last.fm username
		AuthTokens: "", // Authentication tokens (if any)
		BytesDownloaded: int64(0),   // Placeholder
		BytesUploaded: int64(0),     // Placeholder
	}

	response.User = &userResp
	return utils.SendResponse(c, response)
}

// GetUsers returns all users (admin only)
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	// Check if the requesting user is an admin
	// For this implementation, we'll skip the auth check and return all users
	// In a real implementation, this would check the authenticated user's permissions

	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve users")
	}

	// Create response
	response := utils.SuccessResponse()
	usersResp := Users{
		Users: make([]User, 0, len(users)),
	}

	for _, user := range users {
		userResp := User{
			Username:    user.Username,
			Email:       user.Email,
			ScrobblingEnabled: true,
			AdminRole:   user.IsAdmin,
			SettingsRole: true,
			StreamRole:  true,
			JukeboxRole: false,
			UploadRole:  false,
			FolderRole:  []int{0},
			PlaylistRole: true,
			CommentRole: true,
			PodcastRole: false,
			CoverArtRole: true,
			AvatarRole: true,
			ShareRole: true,
			VideoConversionRole: false,
			MusicFolderId: []int{0},
			MaxBitRate: 320,
			LfmUsername: user.Username,
			AuthTokens: "",
			BytesDownloaded: int64(0),
			BytesUploaded: int64(0),
		}
		usersResp.Users = append(usersResp.Users, userResp)
	}

	response.Users = &usersResp
	return utils.SendResponse(c, response)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	username := c.Query("username", "")
	password := c.Query("password", "")
	email := c.Query("email", "")

	if username == "" || password == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter username or password")
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return utils.SendOpenSubsonicError(c, 0, "User already exists")
	} else if err != gorm.ErrRecordNotFound {
		return utils.SendOpenSubsonicError(c, 0, "Failed to check for existing user")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to hash password")
	}

	// Create the user
	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := h.db.Create(&user).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to create user")
	}

	// Return success response
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	username := c.Query("username", "")
	if username == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter username")
	}

	// Get the user to update
	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "User not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve user")
	}

	// Update fields that are provided
	newEmail := c.Query("email", "")
	if newEmail != "" {
		user.Email = newEmail
	}

	newPassword := c.Query("password", "")
	if newPassword != "" {
		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to hash password")
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Update admin role if provided (admin only)
	adminRoleStr := c.Query("adminRole", "")
	if adminRoleStr != "" {
		adminRole, err := strconv.ParseBool(adminRoleStr)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 10, "Invalid adminRole value")
		}
		user.IsAdmin = adminRole
	}

	// Save the updated user
	if err := h.db.Save(&user).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to update user")
	}

	// Return success response
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	username := c.Query("username", "")
	if username == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter username")
	}

	// Don't allow deletion of the current user (this would require checking auth context)
	// For this demo, we'll proceed with deletion
	// In a real implementation, we'd check if the authenticated user can delete this user

	// Delete the user
	if err := h.db.Where("username = ?", username).Delete(&models.User{}).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to delete user")
	}

	// Return success response (empty body for delete operations)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// User represents a user in OpenSubsonic API responses
type User struct {
	XMLName xml.Name `xml:"user"`
	Username string `xml:"username,attr"`
	Email string `xml:"email,attr,omitempty"`
	ScrobblingEnabled bool `xml:"scrobblingEnabled,attr"`
	AdminRole bool `xml:"adminRole,attr"`
	SettingsRole bool `xml:"settingsRole,attr"`
	StreamRole bool `xml:"streamRole,attr"`
	JukeboxRole bool `xml:"jukeboxRole,attr"`
	UploadRole bool `xml:"uploadRole,attr"`
	FolderRole []int `xml:"folderRole,attr,omitempty"`
	PlaylistRole bool `xml:"playlistRole,attr"`
	CommentRole bool `xml:"commentRole,attr"`
	PodcastRole bool `xml:"podcastRole,attr"`
	CoverArtRole bool `xml:"coverArtRole,attr"`
	AvatarRole bool `xml:"avatarRole,attr"`
	ShareRole bool `xml:"shareRole,attr"`
	VideoConversionRole bool `xml:"videoConversionRole,attr"`
	MusicFolderId []int `xml:"musicFolderId,attr,omitempty"`
	MaxBitRate int `xml:"maxBitRate,attr,omitempty"`
	LfmUsername string `xml:"lfmUsername,attr,omitempty"`
	AuthTokens string `xml:"authTokens,attr,omitempty"`
	BytesDownloaded int64 `xml:"bytesDownloaded,attr,omitempty"`
	BytesUploaded int64 `xml:"bytesUploaded,attr,omitempty"`
}

// Users represents a list of users
type Users struct {
	XMLName xml.Name `xml:"users"`
	Users   []User   `xml:"user"`
}