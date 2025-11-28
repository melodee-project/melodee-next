package utils

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/models"
)

// OpenSubsonicResponse represents the common OpenSubsonic response wrapper
type OpenSubsonicResponse struct {
	XMLName       xml.Name `xml:"subsonic-response" json:"-"`
	Status        string   `xml:"status,attr" json:"status"`
	Version       string   `xml:"version,attr" json:"version"`
	Type          string   `xml:"type,attr" json:"type"`
	ServerVersion string   `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr,omitempty" json:"openSubsonic,omitempty"`
	Error         *ErrorDetail `xml:"error,omitempty" json:"error,omitempty"`

	// Response data fields
	MusicFolders *MusicFolders `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes      *Indexes      `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Artists      *Artists      `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist       *Artist       `xml:"artist,omitempty" json:"artist,omitempty"`
	AlbumInfo    *AlbumInfo    `xml:"albumInfo,omitempty" json:"albumInfo,omitempty"`
	Directory    *Directory    `xml:"directory,omitempty" json:"directory,omitempty"`
	Album        *Album        `xml:"album,omitempty" json:"album,omitempty"`
	Song         *Child        `xml:"song,omitempty" json:"song,omitempty"`
	Genres       *Genres       `xml:"genres,omitempty" json:"genres,omitempty"`

	// Search results
	SearchResult2 *SearchResult2 `xml:"searchResult2,omitempty" json:"searchResult2,omitempty"`
	SearchResult3 *SearchResult3 `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`

	// Playlists
	Playlists *Playlists `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist  *Playlist  `xml:"playlist,omitempty" json:"playlist,omitempty"`

	// System
	License *License `xml:"license,omitempty" json:"license,omitempty"`
	OpenSubsonicExtensions *OpenSubsonicExtensions `xml:"openSubsonicExtensions,omitempty" json:"openSubsonicExtensions,omitempty"`

	// Users
	User *User `xml:"user,omitempty" json:"user,omitempty"`
	Users *Users `xml:"users,omitempty" json:"users,omitempty"`
}

// ErrorDetail represents an error response detail
type ErrorDetail struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

// SuccessResponse creates a success response
func SuccessResponse() *OpenSubsonicResponse {
	return &OpenSubsonicResponse{
		Status:        "ok",
		Version:       "1.16.1",
		Type:          "Melodee",
		ServerVersion: "1.0.0",
		OpenSubsonic:  true,
	}
}

// ErrorResponse creates an error response
func ErrorResponse(code int, message string) *OpenSubsonicResponse {
	return &OpenSubsonicResponse{
		Status:        "failed",
		Version:       "1.16.1",
		Type:          "Melodee",
		ServerVersion: "1.0.0",
		OpenSubsonic:  true,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// SendResponse sends an OpenSubsonic response as XML or JSON
func SendResponse(c *fiber.Ctx, response interface{}) error {
	format := c.Query("f")
	if format == "json" {
		return sendJSONResponse(c, response)
	}

	// Set headers
	c.Set("Content-Type", "text/xml; charset=utf-8")
	
	// Marshal to XML
	xmlData, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}

	// Add XML declaration
	xmlResponse := xml.Header + string(xmlData)
	
	return c.Status(200).SendString(xmlResponse)
}

func sendJSONResponse(c *fiber.Ctx, response interface{}) error {
	c.Set("Content-Type", "application/json; charset=utf-8")
	
	// Wrap in subsonic-response object
	wrapper := map[string]interface{}{
		"subsonic-response": response,
	}
	
	return c.Status(200).JSON(wrapper)
}

// SendOpenSubsonicError sends an OpenSubsonic formatted error response
func SendOpenSubsonicError(c *fiber.Ctx, code int, message string) error {
	// Set the X-Status-Code header for observability
	c.Set("X-Status-Code", fmt.Sprintf("%d", getHTTPStatusForErrorCode(code)))
	
	response := ErrorResponse(code, message)
	return SendResponse(c, response)
}

// getHTTPStatusForErrorCode maps OpenSubsonic error codes to HTTP status codes
func getHTTPStatusForErrorCode(code int) int {
	switch code {
	case 10, 40: // Missing parameter, incompatible version
		return 400
	case 50: // Not authorized
		return 401
	case 70: // Data not found
		return 404
	default:
		return 500
	}
}

// ParsePaginationParams parses OpenSubsonic pagination parameters
func ParsePaginationParams(c *fiber.Ctx) (offset int, size int) {
	offset = c.QueryInt("offset", 0)

	// Default size is 50 per spec, max is 500 for most operations
	// But some operations have stricter limits
	defaultSize := 50
	maxSize := 500
	size = c.QueryInt("size", defaultSize)
	if size <= 0 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}

	return offset, size
}

// ParseSearchPaginationParams applies stricter limits for search operations
func ParseSearchPaginationParams(c *fiber.Ctx) (offset int, size int) {
	offset = c.QueryInt("offset", 0)

	// Search operations have stricter limits to prevent resource exhaustion
	defaultSize := 20
	maxSize := 100  // More restrictive limit for search operations
	size = c.QueryInt("size", defaultSize)
	if size <= 0 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}

	return offset, size
}

// ParseMaxOffset ensures that offset values don't get too large to prevent performance issues
func ParseMaxOffset(c *fiber.Ctx, maxOffset int) (offset int) {
	offset = c.QueryInt("offset", 0)

	// Ensure offset doesn't exceed maximum allowed value
	if offset > maxOffset {
		return maxOffset
	}
	if offset < 0 {
		return 0
	}

	return offset
}

// ParseMaxLimit parses a limit parameter with a maximum ceiling
func ParseMaxLimit(c *fiber.Ctx, defaultLimit, maxLimit int) int {
	limit := c.QueryInt("limit", defaultLimit)
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// FormatTime formats time as ISO8601 UTC string for OpenSubsonic responses
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ParseTime parses ISO8601 time string
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// NormalizeString normalizes strings according to OpenSubsonic specs
func NormalizeString(s string) string {
	// For now, this is a placeholder - implement based on spec requirements
	return s
}

// GetUserFromContext retrieves the authenticated user from the context
func GetUserFromContext(c *fiber.Ctx) (*models.User, bool) {
	user, ok := c.Locals("user").(*models.User)
	return user, ok
}
type MusicFolders struct {
	XMLName xml.Name     `xml:"musicFolders" json:"-"`
	Folders []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

type MusicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type Indexes struct {
	XMLName         xml.Name `xml:"indexes" json:"-"`
	LastModified    string   `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string   `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Indexes         []Index  `xml:"index" json:"index"`
}

type Index struct {
	Name    string        `xml:"name,attr" json:"name"`
	Artists []IndexArtist `xml:"artist" json:"artist"`
}

type IndexArtist struct {
	ID         int    `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Created    string `xml:"created,attr,omitempty" json:"created,omitempty"`
	LastScanned string `xml:"lastScanned,attr,omitempty" json:"lastScanned,omitempty"`  // OpenSubsonic uses lastScanned field
	Starred    string `xml:"starred,attr,omitempty" json:"starred,omitempty"`
}

type Artists struct {
	XMLName xml.Name    `xml:"artists" json:"-"`
	Artists []IndexArtist `xml:"artist" json:"artist"`
}

type Artist struct {
	ID         int            `xml:"id,attr" json:"id"`
	Name       string         `xml:"name,attr" json:"name"`
	AlbumCount int            `xml:"albumCount,attr" json:"albumCount"`
	Albums     []ArtistAlbum  `xml:"album,omitempty" json:"album,omitempty"`
	CoverArt   string         `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Starred    string         `xml:"starred,attr,omitempty" json:"starred,omitempty"`
}

type ArtistAlbum struct {
	ID        int    `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	Artist    string `xml:"artist,attr" json:"artist"`
	ArtistID  int    `xml:"artistId,attr" json:"artistId"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	TrackCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int64  `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	PlayCount int    `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Created   string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Year      int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre     string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
}

type AlbumInfo struct {
	ID int `xml:"id,attr" json:"id"`
}

type Directory struct {
	ID       int      `xml:"id,attr" json:"id"`
	Parent   int      `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Name     string   `xml:"name,attr,omitempty" json:"name,omitempty"`
	Artist   string   `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	CoverArt string   `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Created  string   `xml:"created,attr,omitempty" json:"created,omitempty"`
	Children []Child  `xml:"child,omitempty" json:"child,omitempty"`
}

type Child struct {
	ID         int    `xml:"id,attr" json:"id"`
	Parent     int    `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir      bool   `xml:"isDir,attr" json:"isDir"`
	Title      string `xml:"title,attr" json:"title"`
	Album      string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist     string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Created    string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Starred    string `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	Duration   int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate    int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Track      int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	DiscNumber int    `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	Year       int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre      string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	Size       int64  `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"`
}

type Album struct {
	ID        int     `xml:"id,attr" json:"id"`
	Title     string  `xml:"title,attr" json:"title"`
	Album     string  `xml:"name,attr" json:"name"`
	Artist    string  `xml:"artist,attr" json:"artist"`
	ArtistID  int     `xml:"artistId,attr" json:"artistId"`
	CoverArt  string  `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	TrackCount int     `xml:"songCount,attr" json:"songCount"`
	Created   string  `xml:"created,attr" json:"created"`
	Duration  int     `xml:"duration,attr" json:"duration"`
	Year      int     `xml:"year,attr,omitempty" json:"year,omitempty"`
	Songs     []Child `xml:"song,omitempty" json:"song,omitempty"`
}

type Genres struct {
	XMLName xml.Name `xml:"genres" json:"-"`
	Genres  []Genre  `xml:"genre" json:"genre"`
}

type Genre struct {
	Name  string `xml:"name" json:"name"`
	Count int    `xml:"songCount,attr" json:"songCount"`
}

type SearchResult3 struct {
	XMLName   xml.Name        `xml:"searchResult3" json:"-"`
	Offset    int             `xml:"offset,attr" json:"offset"`
	Size      int             `xml:"size,attr" json:"size"`
	TotalHits int             `xml:"totalHits,attr,omitempty" json:"totalHits,omitempty"`
	Artists   []IndexArtist   `xml:"artist,omitempty" json:"artist,omitempty"`
	Albums    []SearchAlbum   `xml:"album,omitempty" json:"album,omitempty"`
	Songs     []Child         `xml:"song,omitempty" json:"song,omitempty"`
}

type SearchResult2 struct {
	XMLName   xml.Name        `xml:"searchResult2" json:"-"`
	Offset    int             `xml:"offset,attr" json:"offset"`
	Size      int             `xml:"size,attr" json:"size"`
	TotalHits int             `xml:"totalHits,attr,omitempty" json:"totalHits,omitempty"`
	Artists   []IndexArtist   `xml:"artist,omitempty" json:"artist,omitempty"`
	Albums    []SearchAlbum   `xml:"album,omitempty" json:"album,omitempty"`
	Songs     []Child         `xml:"song,omitempty" json:"song,omitempty"`
}

type SearchAlbum struct {
	ID        int    `xml:"id,attr" json:"id"`
	Name      string `xml:"title,attr" json:"title"` // In search results, album name is called 'title'
	Artist    string `xml:"artist,attr" json:"artist"`
	ArtistID  int    `xml:"artistId,attr" json:"artistId"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	TrackCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	PlayCount int    `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Created   string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Year      int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre     string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
}

type Playlists struct {
	XMLName  xml.Name `xml:"playlists" json:"-"`
	Playlist []Playlist `xml:"playlist" json:"playlist"`
}

type Playlist struct {
	XMLName      xml.Name `xml:"playlist" json:"-"`
	ID           int      `xml:"id,attr" json:"id"`
	Name         string   `xml:"title,attr" json:"title"`
	Comment      string   `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Public       bool     `xml:"public,attr" json:"public"`
	Owner        string   `xml:"owner,attr" json:"owner"`
	TrackCount    int      `xml:"songCount,attr" json:"songCount"`
	Created      string   `xml:"created,attr" json:"created"`
	Changed      string   `xml:"changed,attr" json:"changed"`
	Duration     int      `xml:"duration,attr" json:"duration"`
	CoverArtID   int      `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Entries      []Child  `xml:"entry,omitempty" json:"entry,omitempty"`
}

type License struct {
	XMLName xml.Name `xml:"license" json:"-"`
	ID      string   `xml:"id,attr" json:"id"`
	Email   string   `xml:"email,attr" json:"email"`
	License string   `xml:"license,attr" json:"license"` // The type of license
	Version string   `xml:"version,attr" json:"version"`
	Created string   `xml:"created,attr" json:"created"`
	Expiry  string   `xml:"expires,attr" json:"expires"` // When the license expires
	Valid   bool     `xml:"valid,attr" json:"valid"`
}

type OpenSubsonicExtensions struct {
	XMLName    xml.Name    `xml:"openSubsonicExtensions" json:"-"`
	Extensions []Extension `xml:"extension" json:"extension"`
}

type Extension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"-" json:"versions"` // Use custom marshaling or just handle it in the handler
	VersionsXML string `xml:"versions,attr" json:"-"`
}

type User struct {
	XMLName xml.Name `xml:"user" json:"-"`
	Username string `xml:"username,attr" json:"username"`
	Email string `xml:"email,attr,omitempty" json:"email,omitempty"`
	ScrobblingEnabled bool `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	AdminRole bool `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole bool `xml:"settingsRole,attr" json:"settingsRole"`
	StreamRole bool `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole bool `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	UploadRole bool `xml:"uploadRole,attr" json:"uploadRole"`
	FolderRole []int `xml:"folderRole,attr,omitempty" json:"folderRole,omitempty"`
	PlaylistRole bool `xml:"playlistRole,attr" json:"playlistRole"`
	CommentRole bool `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole bool `xml:"podcastRole,attr" json:"podcastRole"`
	CoverArtRole bool `xml:"coverArtRole,attr" json:"coverArtRole"`
	AvatarRole bool `xml:"avatarRole,attr" json:"avatarRole"`
	ShareRole bool `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	MusicFolderId []int `xml:"musicFolderId,attr,omitempty" json:"musicFolderId,omitempty"`
	MaxBitRate int `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	LfmUsername string `xml:"lfmUsername,attr,omitempty" json:"lfmUsername,omitempty"`
	AuthTokens string `xml:"authTokens,attr,omitempty" json:"authTokens,omitempty"`
	BytesDownloaded int64 `xml:"bytesDownloaded,attr,omitempty" json:"bytesDownloaded,omitempty"`
	BytesUploaded int64 `xml:"bytesUploaded,attr,omitempty" json:"bytesUploaded,omitempty"`
}

type Users struct {
	XMLName xml.Name `xml:"users" json:"-"`
	Users   []User   `xml:"user" json:"user"`
}