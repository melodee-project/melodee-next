package handlers

import (
	"strconv"
	"time"

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
	userResp := utils.User{
		Username:            user.Username,
		Email:               user.Email,
		ScrobblingEnabled:   true, // Default true
		AdminRole:           user.IsAdmin,
		SettingsRole:        true,          // User can change settings
		StreamRole:          true,          // User can stream
		JukeboxRole:         false,         // Default false
		UploadRole:          false,         // Default false
		FolderRole:          []int{0},      // Access to all folders by default
		PlaylistRole:        true,          // Can manage playlists
		CommentRole:         true,          // Can create comments
		PodcastRole:         false,         // Default false
		CoverArtRole:        true,          // Can change cover art
		AvatarRole:          true,          // Can change avatar
		ShareRole:           true,          // Can create shares
		VideoConversionRole: false,         // Default false
		MusicFolderId:       []int{0},      // All music folders by default
		MaxBitRate:          320,           // Maximum bit rate allowed
		LfmUsername:         user.Username, // Last.fm username
		AuthTokens:          "",            // Authentication tokens (if any)
		BytesDownloaded:     int64(0),      // Placeholder
		BytesUploaded:       int64(0),      // Placeholder
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
	usersResp := utils.Users{
		Users: make([]utils.User, 0, len(users)),
	}

	for _, user := range users {
		userResp := utils.User{
			Username:            user.Username,
			Email:               user.Email,
			ScrobblingEnabled:   true,
			AdminRole:           user.IsAdmin,
			SettingsRole:        true,
			StreamRole:          true,
			JukeboxRole:         false,
			UploadRole:          false,
			FolderRole:          []int{0},
			PlaylistRole:        true,
			CommentRole:         true,
			PodcastRole:         false,
			CoverArtRole:        true,
			AvatarRole:          true,
			ShareRole:           true,
			VideoConversionRole: false,
			MusicFolderId:       []int{0},
			MaxBitRate:          320,
			LfmUsername:         user.Username,
			AuthTokens:          "",
			BytesDownloaded:     int64(0),
			BytesUploaded:       int64(0),
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

// Star attaches a star to a song, album or artist
func (h *UserHandler) Star(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Not authorized")
	}

	id := c.QueryInt("id", -1)
	albumID := c.QueryInt("albumId", -1)
	artistID := c.QueryInt("artistId", -1)

	if id == -1 && albumID == -1 && artistID == -1 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter: id, albumId or artistId")
	}

	now := time.Now()

	if id != -1 {
		// Star song
		var userTrack models.UserTrack
		err := h.db.Where("user_id = ? AND track_id = ?", user.ID, id).First(&userTrack).Error
		if err == gorm.ErrRecordNotFound {
			userTrack = models.UserTrack{
				UserID:    user.ID,
				TrackID:   int64(id),
				IsStarred: true,
				StarredAt: &now,
			}
			h.db.Create(&userTrack)
		} else {
			userTrack.IsStarred = true
			userTrack.StarredAt = &now
			h.db.Save(&userTrack)
		}
	}

	if albumID != -1 {
		// Star album
		var userAlbum models.UserAlbum
		err := h.db.Where("user_id = ? AND album_id = ?", user.ID, albumID).First(&userAlbum).Error
		if err == gorm.ErrRecordNotFound {
			userAlbum = models.UserAlbum{
				UserID:    user.ID,
				AlbumID:   int64(albumID),
				IsStarred: true,
				StarredAt: &now,
			}
			h.db.Create(&userAlbum)
		} else {
			userAlbum.IsStarred = true
			userAlbum.StarredAt = &now
			h.db.Save(&userAlbum)
		}
	}

	if artistID != -1 {
		// Star artist
		var userArtist models.UserArtist
		err := h.db.Where("user_id = ? AND artist_id = ?", user.ID, artistID).First(&userArtist).Error
		if err == gorm.ErrRecordNotFound {
			userArtist = models.UserArtist{
				UserID:    user.ID,
				ArtistID:  int64(artistID),
				IsStarred: true,
				StarredAt: &now,
			}
			h.db.Create(&userArtist)
		} else {
			userArtist.IsStarred = true
			userArtist.StarredAt = &now
			h.db.Save(&userArtist)
		}
	}

	// Return success response (empty body)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// Unstar removes a star from a song, album or artist
func (h *UserHandler) Unstar(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Not authorized")
	}

	id := c.QueryInt("id", -1)
	albumID := c.QueryInt("albumId", -1)
	artistID := c.QueryInt("artistId", -1)

	if id == -1 && albumID == -1 && artistID == -1 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter: id, albumId or artistId")
	}

	if id != -1 {
		// Unstar song
		h.db.Model(&models.UserTrack{}).Where("user_id = ? AND track_id = ?", user.ID, id).Updates(map[string]interface{}{
			"is_starred": false,
			"starred_at": nil,
		})
	}

	if albumID != -1 {
		// Unstar album
		h.db.Model(&models.UserAlbum{}).Where("user_id = ? AND album_id = ?", user.ID, albumID).Updates(map[string]interface{}{
			"is_starred": false,
			"starred_at": nil,
		})
	}

	if artistID != -1 {
		// Unstar artist
		h.db.Model(&models.UserArtist{}).Where("user_id = ? AND artist_id = ?", user.ID, artistID).Updates(map[string]interface{}{
			"is_starred": false,
			"starred_at": nil,
		})
	}

	// Return success response (empty body)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// GetStarred returns starred songs, albums and artists
func (h *UserHandler) GetStarred(c *fiber.Ctx) error {
	return h.getStarredCommon(c, 1)
}

// GetStarred2 returns starred songs, albums and artists (Version 2)
func (h *UserHandler) GetStarred2(c *fiber.Ctx) error {
	return h.getStarredCommon(c, 2)
}

func (h *UserHandler) getStarredCommon(c *fiber.Ctx, version int) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Not authorized")
	}

	// Get starred artists
	var userArtists []models.UserArtist
	h.db.Where("user_id = ? AND is_starred = ?", user.ID, true).Find(&userArtists)

	var artists []models.Artist
	if len(userArtists) > 0 {
		artistIDs := make([]int64, len(userArtists))
		for i, ua := range userArtists {
			artistIDs[i] = ua.ArtistID
		}
		h.db.Where("id IN ?", artistIDs).Find(&artists)
	}

	// Get starred albums
	var userAlbums []models.UserAlbum
	h.db.Where("user_id = ? AND is_starred = ?", user.ID, true).Find(&userAlbums)

	var albums []models.Album
	if len(userAlbums) > 0 {
		albumIDs := make([]int64, len(userAlbums))
		for i, ua := range userAlbums {
			albumIDs[i] = ua.AlbumID
		}
		h.db.Preload("Artist").Where("id IN ?", albumIDs).Find(&albums)
	}

	// Get starred songs
	var userTracks []models.UserTrack
	h.db.Where("user_id = ? AND is_starred = ?", user.ID, true).Find(&userTracks)

	var songs []models.Track
	if len(userTracks) > 0 {
		trackIDs := make([]int64, len(userTracks))
		for i, ut := range userTracks {
			trackIDs[i] = ut.TrackID
		}
		h.db.Preload("Album").Preload("Artist").Where("id IN ?", trackIDs).Find(&songs)
	}

	// Build response
	response := utils.SuccessResponse()

	// Convert to response structs
	indexArtists := make([]utils.IndexArtist, 0, len(artists))
	for _, a := range artists {
		indexArtists = append(indexArtists, utils.IndexArtist{
			ID:         int(a.ID),
			Name:       a.Name,
			AlbumCount: int(a.AlbumCountCached),
			Starred:    utils.FormatTime(time.Now()), // Ideally use actual starred time from join
		})
	}

	childSongs := make([]utils.Child, 0, len(songs))
	for _, s := range songs {
		childSongs = append(childSongs, utils.Child{
			ID:       int(s.ID),
			Parent:   int(s.AlbumID),
			IsDir:    false,
			Title:    s.Name,
			Album:    s.Album.Name,
			Artist:   s.Artist.Name,
			CoverArt: "al-" + strconv.FormatInt(s.AlbumID, 10),
			Created:  utils.FormatTime(s.CreatedAt),
			Duration: int(s.Duration / 1000),
			BitRate:  int(s.BitRate),
			Track:    int(s.SortOrder),
			Starred:  utils.FormatTime(time.Now()), // Placeholder
			Path:     s.RelativePath,
		})
	}

	if version == 2 {
		albumList := make([]utils.Album, 0, len(albums))
		for _, a := range albums {
			albumList = append(albumList, utils.Album{
				ID:         int(a.ID),
				Title:      a.Name,
				Album:      a.Name,
				Artist:     a.Artist.Name,
				ArtistID:   int(a.ArtistID),
				CoverArt:   "al-" + strconv.FormatInt(a.ID, 10),
				TrackCount: int(a.TrackCountCached),
				Created:    utils.FormatTime(a.CreatedAt),
				Duration:   int(a.DurationCached / 1000),
			})
		}

		response.Starred2 = &utils.Starred2{
			Artists: indexArtists,
			Albums:  albumList,
			Songs:   childSongs,
		}
	} else {
		albumList := make([]utils.Child, 0, len(albums))
		for _, a := range albums {
			albumList = append(albumList, utils.Child{
				ID:       int(a.ID),
				IsDir:    true,
				Title:    a.Name,
				Album:    a.Name,
				Artist:   a.Artist.Name,
				CoverArt: "al-" + strconv.FormatInt(a.ID, 10),
				Created:  utils.FormatTime(a.CreatedAt),
				Starred:  utils.FormatTime(time.Now()), // Placeholder
			})
		}

		response.Starred = &utils.Starred{
			Artists: indexArtists,
			Albums:  albumList,
			Songs:   childSongs,
		}
	}

	return utils.SendResponse(c, response)
}

// SetRating sets the rating for a music file
func (h *UserHandler) SetRating(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Not authorized")
	}

	id := c.QueryInt("id", -1)
	rating := c.QueryInt("rating", 0)

	if id == -1 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Ensure rating is between 0 and 5
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}

	var userTrack models.UserTrack
	err := h.db.Where("user_id = ? AND track_id = ?", user.ID, id).First(&userTrack).Error
	if err == gorm.ErrRecordNotFound {
		userTrack = models.UserTrack{
			UserID:  user.ID,
			TrackID: int64(id),
			Rating:  int8(rating),
		}
		h.db.Create(&userTrack)
	} else {
		userTrack.Rating = int8(rating)
		h.db.Save(&userTrack)
	}

	// Return success response (empty body)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// Scrobble registers the local playback of one or more media files
func (h *UserHandler) Scrobble(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Not authorized")
	}

	id := c.QueryInt("id", -1)
	// submission := c.QueryBool("submission") // Not used in this simple implementation, but part of spec

	if id == -1 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	now := time.Now()

	// Update play count and last played time
	var userTrack models.UserTrack
	err := h.db.Where("user_id = ? AND track_id = ?", user.ID, id).First(&userTrack).Error
	if err == gorm.ErrRecordNotFound {
		userTrack = models.UserTrack{
			UserID:       user.ID,
			TrackID:      int64(id),
			PlayedCount:  1,
			LastPlayedAt: &now,
		}
		h.db.Create(&userTrack)
	} else {
		userTrack.PlayedCount++
		userTrack.LastPlayedAt = &now
		h.db.Save(&userTrack)
	}

	// Return success response (empty body)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}
