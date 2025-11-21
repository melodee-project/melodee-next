package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents the users table
type User struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey      uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Username    string     `gorm:"size:255;uniqueIndex;not null" json:"username"`
	Email       string     `gorm:"size:255" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"password_hash"`
	IsAdmin     bool       `gorm:"default:false" json:"is_admin"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// BeforeCreate sets the API key before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.APIKey == uuid.Nil {
		u.APIKey = uuid.New()
	}
	return nil
}

// Library represents the libraries table
type Library struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	Path         string    `gorm:"not null" json:"path"`
	Type         string    `gorm:"size:50;not null;check:type IN ('inbound', 'staging', 'production')" json:"type"`
	IsLocked     bool      `gorm:"default:false" json:"is_locked"`
	CreatedAt    time.Time `json:"created_at"`
	SongCount    int32     `gorm:"default:0" json:"song_count"`
	AlbumCount   int32     `gorm:"default:0" json:"album_count"`
	Duration     int64     `gorm:"default:0" json:"duration"` // duration in milliseconds
	BasePath     string    `gorm:"size:512;not null" json:"base_path"`
}

// Artist represents the artists table
type Artist struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey          uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	IsLocked        bool       `gorm:"default:false" json:"is_locked"`
	Name            string     `gorm:"size:255;not null" json:"name"`
	NameNormalized  string     `gorm:"size:255;not null;index:idx_artists_name_normalized_gin,gin" json:"name_normalized"` // For efficient searching
	DirectoryCode   string     `gorm:"size:20;index" json:"directory_code"` // Directory code for filesystem performance
	SortName        string     `gorm:"size:255" json:"sort_name"`
	AlternateNames  []string   `gorm:"type:text[]" json:"alternate_names"`
	SongCountCached int32      `gorm:"default:0" json:"song_count_cached"` // Pre-calculated for performance
	AlbumCountCached int32     `gorm:"default:0" json:"album_count_cached"` // Pre-calculated for performance
	DurationCached  int64      `gorm:"default:0" json:"duration_cached"` // Pre-calculated for performance
	CreatedAt       time.Time  `json:"created_at"`
	LastScannedAt   *time.Time `json:"last_scanned_at"`
	Tags            []byte     `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	MusicBrainzID   *uuid.UUID `gorm:"type:uuid;index" json:"musicbrainz_id"`
	SpotifyID       string     `gorm:"size:255" json:"spotify_id"`
	LastFmID        string     `gorm:"size:255" json:"lastfm_id"`
	DiscogsID       string     `gorm:"size:255" json:"discogs_id"`
	ITunesID        string     `gorm:"size:255" json:"itunes_id"`
	AMGID           string     `gorm:"size:255" json:"amg_id"`
	WikidataID      string     `gorm:"size:255" json:"wikidata_id"`
	SortOrder       int32      `gorm:"default:0" json:"sort_order"`

	// Relationships
	Albums []Album `gorm:"foreignKey:ArtistID" json:"-"`
}

// BeforeCreate sets the API key before creating an artist
func (a *Artist) BeforeCreate(tx *gorm.DB) error {
	if a.APIKey == uuid.Nil {
		a.APIKey = uuid.New()
	}
	return nil
}

// Album represents the albums table
type Album struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey              uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	IsLocked            bool       `gorm:"default:false" json:"is_locked"`
	Name                string     `gorm:"size:255;not null" json:"name"`
	NameNormalized      string     `gorm:"size:255;not null;index:idx_albums_name_normalized_gin,gin" json:"name_normalized"`
	AlternateNames      []string   `gorm:"type:text[]" json:"alternate_names"`
	ArtistID            int64      `gorm:"index;not null" json:"artist_id"`
	SongCountCached     int32      `gorm:"default:0" json:"song_count_cached"` // Pre-calculated for performance
	DurationCached      int64      `gorm:"default:0" json:"duration_cached"` // duration in milliseconds
	CreatedAt           time.Time  `json:"created_at"`
	Tags                []byte     `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	ReleaseDate         *time.Time `json:"release_date"`
	OriginalReleaseDate *time.Time `json:"original_release_date"`
	AlbumStatus         string     `gorm:"size:50;default:'New';check:album_status IN ('New', 'Ok', 'Invalid')" json:"album_status"`
	AlbumType           string     `gorm:"size:50;default:'NotSet';check:album_type IN ('NotSet', 'Album', 'EP', 'Single', 'Compilation', 'Live', 'Remix', 'Soundtrack', 'SpokenWord', 'Interview', 'Audiobook')" json:"album_type"`
	Directory           string     `gorm:"size:512;not null" json:"directory"` // Relative path from library base
	SortName            string     `gorm:"size:255" json:"sort_name"`
	SortOrder           int32      `gorm:"default:0" json:"sort_order"`
	ImageCount          int32      `gorm:"default:0" json:"image_count"`
	Comment             string     `json:"comment"`
	Description         string     `json:"description"`
	Genres              []string   `gorm:"type:text[]" json:"genres"`
	Moods               []string   `gorm:"type:text[]" json:"moods"`
	Notes               string     `json:"notes"`
	DeezerID            string     `gorm:"size:255" json:"deezer_id"`
	MusicBrainzID       *uuid.UUID `gorm:"type:uuid;index" json:"musicbrainz_id"`
	SpotifyID           string     `gorm:"size:255" json:"spotify_id"`
	LastFmID            string     `gorm:"size:255" json:"lastfm_id"`
	DiscogsID           string     `gorm:"size:255" json:"discogs_id"`
	ITunesID            string     `gorm:"size:255" json:"itunes_id"`
	AMGID               string     `gorm:"size:255" json:"amg_id"`
	WikidataID          string     `gorm:"size:255" json:"wikidata_id"`
	IsCompilation       bool       `gorm:"default:false" json:"is_compilation"`

	// Relationships
	Artist *Artist `gorm:"foreignKey:ArtistID" json:"artist"`
	Songs  []Song  `gorm:"foreignKey:AlbumID" json:"songs"`
}

// BeforeCreate sets the API key before creating an album
func (a *Album) BeforeCreate(tx *gorm.DB) error {
	if a.APIKey == uuid.Nil {
		a.APIKey = uuid.New()
	}
	return nil
}

// Song represents the songs table
type Song struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey      uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	NameNormalized string  `gorm:"size:255;not null" json:"name_normalized"`
	SortName    string     `gorm:"size:255" json:"sort_name"`
	AlbumID     int64      `gorm:"index:idx_songs_album_id_hash,hash;not null" json:"album_id"`
	ArtistID    int64      `gorm:"index:idx_songs_artist_id_hash,hash;not null" json:"artist_id"` // Denormalized for performance
	Duration    int64      `json:"duration"` // duration in milliseconds
	BitRate     int32      `json:"bit_rate"` // in kbps
	BitDepth    int32      `json:"bit_depth"`
	SampleRate  int32      `json:"sample_rate"` // in Hz
	Channels    int32      `json:"channels"`
	CreatedAt   time.Time  `json:"created_at"`
	Tags        []byte     `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	Directory   string     `gorm:"size:512;not null" json:"directory"` // Relative path from library base
	FileName    string     `gorm:"not null" json:"file_name"` // Just the filename for optimized storage
	RelativePath string    `gorm:"not null" json:"relative_path"` // directory + file_name
	CRCHash     string     `gorm:"size:255;not null" json:"crc_hash"`
	SortOrder   int32      `gorm:"default:0" json:"sort_order"`

	// Relationships
	Album  *Album  `gorm:"foreignKey:AlbumID" json:"album"`
	Artist *Artist `gorm:"foreignKey:ArtistID" json:"artist"`
}

// BeforeCreate sets the API key before creating a song
func (s *Song) BeforeCreate(tx *gorm.DB) error {
	if s.APIKey == uuid.Nil {
		s.APIKey = uuid.New()
	}
	return nil
}

// Playlist represents the playlists table
type Playlist struct {
	ID            int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey        uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	UserID        int64     `gorm:"not null" json:"user_id"`
	Name          string    `gorm:"size:255;not null" json:"name"`
	Comment       string    `json:"comment"`
	Public        bool      `gorm:"default:false" json:"public"`
	CreatedAt     time.Time `json:"created_at"`
	ChangedAt     time.Time `json:"changed_at"`
	Duration      int64     `json:"duration"` // duration in milliseconds
	SongCount     int32     `json:"song_count"`
	CoverArtID    *int32    `json:"cover_art_id"` // foreign key to images table

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user"`
}

// BeforeCreate sets the API key before creating a playlist
func (p *Playlist) BeforeCreate(tx *gorm.DB) error {
	if p.APIKey == uuid.Nil {
		p.APIKey = uuid.New()
	}
	return nil
}

// PlaylistSong represents the playlist_songs junction table
type PlaylistSong struct {
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlaylistID int32     `gorm:"not null" json:"playlist_id"`
	SongID     int64     `gorm:"not null" json:"song_id"`
	Position   int32     `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	Playlist *Playlist `gorm:"foreignKey:PlaylistID" json:"playlist"`
	Song     *Song     `gorm:"foreignKey:SongID" json:"song"`

	// Constraints: UNIQUE(playlist_id, position)
}

// UserSong represents user interactions with songs
type UserSong struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null" json:"user_id"`
	SongID       int64     `gorm:"not null" json:"song_id"`
	PlayedCount  int32     `gorm:"default:0" json:"played_count"`
	LastPlayedAt *time.Time `json:"last_played_at"`
	IsStarred    bool      `gorm:"default:false" json:"is_starred"`
	IsHated      bool      `gorm:"default:false" json:"is_hated"` // When true, don't include in randomization
	StarredAt    *time.Time `json:"starred_at"`
	Rating       int8      `gorm:"check:rating >= 0 AND rating <= 5;default:0" json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Constraints: UNIQUE(user_id, song_id)
}

// UserAlbum represents user interactions with albums
type UserAlbum struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null" json:"user_id"`
	AlbumID      int64     `gorm:"not null" json:"album_id"`
	PlayedCount  int32     `gorm:"default:0" json:"played_count"`
	LastPlayedAt *time.Time `json:"last_played_at"`
	IsStarred    bool      `gorm:"default:false" json:"is_starred"`
	IsHated      bool      `gorm:"default:false" json:"is_hated"` // When true, don't include in randomization
	StarredAt    *time.Time `json:"starred_at"`
	Rating       int8      `gorm:"check:rating >= 0 AND rating <= 5;default:0" json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Constraints: UNIQUE(user_id, album_id)
}

// UserArtist represents user interactions with artists
type UserArtist struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	ArtistID  int64     `gorm:"not null" json:"artist_id"`
	IsStarred bool      `gorm:"default:false" json:"is_starred"`
	IsHated   bool      `gorm:"default:false" json:"is_hated"` // When true, don't include in randomization
	StarredAt *time.Time `json:"starred_at"`
	Rating    int8      `gorm:"check:rating >= 0 AND rating <= 5;default:0" json:"rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Constraints: UNIQUE(user_id, artist_id)
}

// UserPin represents pinned content
type UserPin struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	SongID    *int64    `json:"song_id"`
	AlbumID   *int64    `json:"album_id"`
	ArtistID  *int64    `json:"artist_id"`
	PinnedAt  time.Time `json:"pinned_at"`

	// Only one of SongID, AlbumID, or ArtistID should be set
}

// Bookmark represents user bookmarks
type Bookmark struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	SongID    int64     `gorm:"not null" json:"song_id"`
	Comment   string    `json:"comment"`
	Position  int32     `gorm:"not null" json:"position"` // Position in milliseconds
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Constraints: UNIQUE(user_id, song_id)
}

// Player represents user players/devices
type Player struct {
	ID              int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	UserAgent       string    `json:"user_agent"`
	UserID          int64     `gorm:"not null" json:"user_id"`
	Client          string    `gorm:"size:500;not null" json:"client"`
	IPAddress       string    `gorm:"size:45" json:"ip_address"` // Support for IPv6 addresses
	LastSeenAt      time.Time `gorm:"not null" json:"last_seen_at"`
	MaxBitrate      int32     `json:"max_bitrate"` // Maximum bitrate for this player
	ScrobbleEnabled bool      `gorm:"default:true" json:"scrobble_enabled"`
	TranscodingID   string    `gorm:"size:255" json:"transcoding_id"`
	Hostname        string    `gorm:"size:500" json:"hostname"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PlayQueue represents play queues
type PlayQueue struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null" json:"user_id"`
	SongID       int64     `gorm:"not null" json:"song_id"`
	SongAPIKey   uuid.UUID `gorm:"type:uuid;not null" json:"song_api_key"` // To not expose internal song IDs to API consumers
	IsCurrentSong bool     `gorm:"default:false" json:"is_current_song"`
	ChangedBy    string    `gorm:"size:255;not null" json:"changed_by"`
	Position     float64   `gorm:"default:0" json:"position"`
	PlayQueueID  int32     `gorm:"not null" json:"play_queue_id"` // To manage order in the queue
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SearchHistory represents user search history
type SearchHistory struct {
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     *int64    `json:"user_id"`
	SearchTerm string    `gorm:"size:500;not null" json:"search_term"`
	SearchType string    `gorm:"size:50;not null;check:search_type IN ('artist', 'album', 'song', 'any')" json:"search_type"`
	ResultsCount int32   `gorm:"default:0" json:"results_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Share represents shared content
type Share struct {
	ID                    int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                int64     `gorm:"not null" json:"user_id"`
	Name                  string    `gorm:"size:255" json:"name"`
	Description           string    `json:"description"`
	ExpiresAt             *time.Time `json:"expires_at"`
	MaxStreamingMinutes   int32     `json:"max_streaming_minutes"`
	MaxStreamingCount     int32     `json:"max_streaming_count"`
	AllowStreaming        bool      `gorm:"default:true" json:"allow_streaming"`
	AllowDownload         bool      `gorm:"default:false" json:"allow_download"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ShareActivity represents share usage tracking
type ShareActivity struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID   int32     `gorm:"not null" json:"share_id"`
	UserID    *int64    `json:"user_id"` // User who accessed (null if anonymous)
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent string    `json:"user_agent"`
}

// LibraryScanHistory represents library scanning history
type LibraryScanHistory struct {
	ID            int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	LibraryID     int32     `gorm:"not null" json:"library_id"`
	Status        string    `gorm:"size:50;not null;check:status IN ('started', 'in_progress', 'completed', 'failed')" json:"status"`
	Message       string    `json:"message"`
	TotalFiles    int32     `json:"total_files"`
	ProcessedFiles int32    `json:"processed_files"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Setting represents application configuration
type Setting struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       string    `gorm:"size:500;unique;not null" json:"key"`
	Value     string    `json:"value"`
	Category  *int32    `json:"category"` // Enum for setting category
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ArtistRelation represents artist relationships
type ArtistRelation struct {
	ID             int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	FromArtistID   int64     `gorm:"not null" json:"from_artist_id"`
	ToArtistID     int64     `gorm:"not null" json:"to_artist_id"`
	RelationType   string    `gorm:"size:100;not null" json:"relation_type"` // e.g., 'member', 'collaborator', 'influenced_by'
	RelationStart  *time.Time `json:"relation_start"` // When the relationship started
	RelationEnd    *time.Time `json:"relation_end"` // When the relationship ended
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	IsLocked       bool      `gorm:"default:false" json:"is_locked"`
	SortOrder      int32     `gorm:"default:0" json:"sort_order"`
	APIKey         uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Tags           []byte    `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	Notes          string    `json:"notes"`
	Description    string    `json:"description"`

	// Constraints: UNIQUE(from_artist_id, to_artist_id, relation_type)
}

// RadioStation represents radio stations
type RadioStation struct {
	ID               int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey           uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Name             string    `gorm:"size:255;not null" json:"name"`
	StreamURL        string    `gorm:"not null" json:"stream_url"`
	HomePageURL      string    `json:"home_page_url"`
	CreatedByUserID  *int64    `json:"created_by_user_id"`
	SongCount        int32     `json:"song_count"`
	IsEnabled        bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

// BeforeCreate sets the API key before creating a radio station
func (r *RadioStation) BeforeCreate(tx *gorm.DB) error {
	if r.APIKey == uuid.Nil {
		r.APIKey = uuid.New()
	}
	return nil
}

// Contributor represents song contributors like composers, performers, etc.
type Contributor struct {
	ID       int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string    `gorm:"size:255;not null" json:"name"`
	Type     string    `gorm:"size:100;not null" json:"type"` // e.g., 'performer', 'composer', 'producer'
	SortName string    `gorm:"size:255" json:"sort_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}