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
	XMLName       xml.Name `xml:"subsonic-response"`
	Status        string   `xml:"status,attr"`
	Version       string   `xml:"version,attr"`
	Type          string   `xml:"type,attr"`
	ServerVersion string   `xml:"serverVersion,attr"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr,omitempty"`
	Error         *ErrorResponse `xml:"error,omitempty"`

	// Response data fields
	MusicFolders *MusicFolders `xml:"musicFolders,omitempty"`
	Indexes      *Indexes      `xml:"indexes,omitempty"`
	Artists      *Artists      `xml:"artists,omitempty"`
	Artist       *Artist       `xml:"artist,omitempty"`
	AlbumInfo    *AlbumInfo    `xml:"albumInfo,omitempty"`
	Directory    *Directory    `xml:"directory,omitempty"`
	Album        *Album        `xml:"album,omitempty"`
	Song         *Child        `xml:"song,omitempty"`
	Genres       *Genres       `xml:"genres,omitempty"`

	// Search results
	SearchResult3 *SearchResult3 `xml:"searchResult3,omitempty"`

	// Playlists
	Playlists *Playlists `xml:"playlists,omitempty"`
	Playlist  *Playlist  `xml:"playlist,omitempty"`

	// System
	License *License `xml:"license,omitempty"`

	// Users
	User *User `xml:"user,omitempty"`
	Users *Users `xml:"users,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `xml:"code,attr"`
	Message string `xml:"message,attr"`
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
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
		},
	}
}

// SendResponse sends an OpenSubsonic response as XML
func SendResponse(c *fiber.Ctx, response interface{}) error {
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
	
	// Default size is 50 per spec, max is 500
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
	XMLName xml.Name     `xml:"musicFolders"`
	Folders []MusicFolder `xml:"musicFolder"`
}

type MusicFolder struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type Indexes struct {
	XMLName         xml.Name `xml:"indexes"`
	LastModified    string   `xml:"lastModified,attr"`
	IgnoredArticles string   `xml:"ignoredArticles,attr"`
	Indexes         []Index  `xml:"index"`
}

type Index struct {
	Name    string        `xml:"name,attr"`
	Artists []IndexArtist `xml:"artist"`
}

type IndexArtist struct {
	ID         int    `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	AlbumCount int    `xml:"albumCount,attr"`
	CoverArt   string `xml:"coverArt,attr,omitempty"`
	Created    string `xml:"created,attr,omitempty"`
	LastScanned string `xml:"lastScanned,attr,omitempty"`  // OpenSubsonic uses lastScanned field
	Starred    string `xml:"starred,attr,omitempty"`
}

type Artists struct {
	XMLName xml.Name    `xml:"artists"`
	Artists []IndexArtist `xml:"artist"`
}

type Artist struct {
	ID         int            `xml:"id,attr"`
	Name       string         `xml:"name,attr"`
	AlbumCount int            `xml:"albumCount,attr"`
	Albums     []ArtistAlbum  `xml:"album,omitempty"`
	CoverArt   string         `xml:"coverArt,attr,omitempty"`
	Starred    string         `xml:"starred,attr,omitempty"`
}

type ArtistAlbum struct {
	ID        int    `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Artist    string `xml:"artist,attr"`
	ArtistID  int    `xml:"artistId,attr"`
	CoverArt  string `xml:"coverArt,attr,omitempty"`
	SongCount int    `xml:"songCount,attr"`
	Duration  int64  `xml:"duration,attr,omitempty"`
	PlayCount int    `xml:"playCount,attr,omitempty"`
	Created   string `xml:"created,attr,omitempty"`
	Year      int    `xml:"year,attr,omitempty"`
	Genre     string `xml:"genre,attr,omitempty"`
}

type AlbumInfo struct {
	ID int `xml:"id,attr"`
}

type Directory struct {
	ID       int      `xml:"id,attr"`
	Parent   int      `xml:"parent,attr,omitempty"`
	Name     string   `xml:"name,attr,omitempty"`
	Artist   string   `xml:"artist,attr,omitempty"`
	CoverArt string   `xml:"coverArt,attr,omitempty"`
	Created  string   `xml:"created,attr,omitempty"`
	Children []Child  `xml:"child,omitempty"`
}

type Child struct {
	ID         int    `xml:"id,attr"`
	Parent     int    `xml:"parent,attr,omitempty"`
	IsDir      bool   `xml:"isDir,attr"`
	Title      string `xml:"title,attr"`
	Album      string `xml:"album,attr,omitempty"`
	Artist     string `xml:"artist,attr,omitempty"`
	CoverArt   string `xml:"coverArt,attr,omitempty"`
	Created    string `xml:"created,attr,omitempty"`
	Starred    string `xml:"starred,attr,omitempty"`
	Duration   int    `xml:"duration,attr,omitempty"`
	BitRate    int    `xml:"bitRate,attr,omitempty"`
	Track      int    `xml:"track,attr,omitempty"`
	DiscNumber int    `xml:"discNumber,attr,omitempty"`
	Year       int    `xml:"year,attr,omitempty"`
	Genre      string `xml:"genre,attr,omitempty"`
	Size       int64  `xml:"size,attr,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty"`
	Path        string `xml:"path,attr,omitempty"`
}

type Album struct {
	ID        int     `xml:"id,attr"`
	Title     string  `xml:"title,attr"`
	Album     string  `xml:"name,attr"`
	Artist    string  `xml:"artist,attr"`
	ArtistID  int     `xml:"artistId,attr"`
	CoverArt  string  `xml:"coverArt,attr,omitempty"`
	SongCount int     `xml:"songCount,attr"`
	Created   string  `xml:"created,attr"`
	Duration  int     `xml:"duration,attr"`
	Year      int     `xml:"year,attr,omitempty"`
	Songs     []Child `xml:"song,omitempty"`
}

type Genres struct {
	XMLName xml.Name `xml:"genres"`
	Genres  []Genre  `xml:"genre"`
}

type Genre struct {
	Name  string `xml:"name"`
	Count int    `xml:"songCount,attr"`
}

type SearchResult3 struct {
	XMLName   xml.Name        `xml:"searchResult3"`
	Offset    int             `xml:"offset,attr"`
	Size      int             `xml:"size,attr"`
	TotalHits int             `xml:"totalHits,attr,omitempty"`
	Artists   []IndexArtist   `xml:"artist,omitempty"`
	Albums    []SearchAlbum   `xml:"album,omitempty"`
	Songs     []Child         `xml:"song,omitempty"`
}

type SearchResult2 struct {
	XMLName   xml.Name        `xml:"searchResult2"`
	Offset    int             `xml:"offset,attr"`
	Size      int             `xml:"size,attr"`
	TotalHits int             `xml:"totalHits,attr,omitempty"`
	Artists   []IndexArtist   `xml:"artist,omitempty"`
	Albums    []SearchAlbum   `xml:"album,omitempty"`
	Songs     []Child         `xml:"song,omitempty"`
}

type SearchAlbum struct {
	ID        int    `xml:"id,attr"`
	Name      string `xml:"title,attr"` // In search results, album name is called 'title'
	Artist    string `xml:"artist,attr"`
	ArtistID  int    `xml:"artistId,attr"`
	CoverArt  string `xml:"coverArt,attr,omitempty"`
	SongCount int    `xml:"songCount,attr"`
	Duration  int    `xml:"duration,attr,omitempty"`
	PlayCount int    `xml:"playCount,attr,omitempty"`
	Created   string `xml:"created,attr,omitempty"`
	Year      int    `xml:"year,attr,omitempty"`
	Genre     string `xml:"genre,attr,omitempty"`
}

type Playlists struct {
	XMLName  xml.Name `xml:"playlists"`
	Playlist []Playlist `xml:"playlist"`
}

type Playlist struct {
	XMLName      xml.Name `xml:"playlist"`
	ID           int      `xml:"id,attr"`
	Name         string   `xml:"title,attr"`
	Comment      string   `xml:"comment,attr,omitempty"`
	Public       bool     `xml:"public,attr"`
	Owner        string   `xml:"owner,attr"`
	SongCount    int      `xml:"songCount,attr"`
	Created      string   `xml:"created,attr"`
	Changed      string   `xml:"changed,attr"`
	Duration     int      `xml:"duration,attr"`
	CoverArtID   int      `xml:"coverArt,attr,omitempty"`
	Entries      []Child  `xml:"entry,omitempty"`
}

type License struct {
	XMLName xml.Name `xml:"license"`
	ID      string   `xml:"id,attr"`
	Email   string   `xml:"email,attr"`
	License string   `xml:"license,attr"` // The type of license
	Version string   `xml:"version,attr"`
	Created string   `xml:"created,attr"`
	Expiry  string   `xml:"expires,attr"` // When the license expires
	Valid   bool     `xml:"valid,attr"`
}

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

type Users struct {
	XMLName xml.Name `xml:"users"`
	Users   []User   `xml:"user"`
}