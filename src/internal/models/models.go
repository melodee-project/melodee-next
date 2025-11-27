package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents the users table
type User struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey              uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Username            string     `gorm:"size:255;uniqueIndex;not null" json:"username"`
	Email               string     `gorm:"size:255" json:"email"`
	PasswordHash        string     `gorm:"size:255;not null" json:"-"` // Don't expose password hash in JSON
	IsAdmin             bool       `gorm:"default:false" json:"is_admin"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`     // Number of consecutive failed login attempts
	LockedUntil         *time.Time `json:"locked_until,omitempty"` // Time until which the account is locked
	PasswordResetToken  *string    `gorm:"size:255" json:"-"`      // Hash of the password reset token
	PasswordResetExpiry *time.Time `json:"-"`                      // When the password reset token expires
	CreatedAt           time.Time  `json:"created_at"`
	LastLoginAt         *time.Time `json:"last_login_at"`
}

func (User) TableName() string {
	return "users"
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
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"size:255;not null" json:"name"`
	Path       string    `gorm:"not null" json:"path"`
	Type       string    `gorm:"size:50;not null;check:type IN ('inbound', 'staging', 'production')" json:"type"`
	IsLocked   bool      `gorm:"default:false" json:"is_locked"`
	CreatedAt  time.Time `json:"created_at"`
	TrackCount int32     `gorm:"default:0" json:"track_count"`
	AlbumCount int32     `gorm:"default:0" json:"album_count"`
	Duration   int64     `gorm:"default:0" json:"duration"` // duration in milliseconds
}

func (Library) TableName() string {
	return "libraries"
}

// Artist represents the artists table
type Artist struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey           uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	IsLocked         bool       `gorm:"default:false" json:"is_locked"`
	Name             string     `gorm:"size:255;not null" json:"name"`
	NameNormalized   string     `gorm:"size:255;not null;index:idx_artists_name_normalized_gin,gin" json:"name_normalized"` // For efficient searching
	DirectoryCode    string     `gorm:"size:20;index" json:"directory_code"`                                                // Directory code for filesystem performance
	SortName         string     `gorm:"size:255" json:"sort_name"`
	AlternateNames   []string   `gorm:"type:text[]" json:"alternate_names"`
	TrackCountCached int32      `gorm:"default:0" json:"track_count_cached"` // Pre-calculated for performance
	AlbumCountCached int32      `gorm:"default:0" json:"album_count_cached"` // Pre-calculated for performance
	DurationCached   int64      `gorm:"default:0" json:"duration_cached"`    // Pre-calculated for performance
	CreatedAt        time.Time  `json:"created_at"`
	LastScannedAt    *time.Time `json:"last_scanned_at"`
	Tags             []byte     `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	MusicBrainzID    *uuid.UUID `gorm:"type:uuid;index" json:"musicbrainz_id"`
	SpotifyID        string     `gorm:"size:255" json:"spotify_id"`
	LastFmID         string     `gorm:"size:255" json:"lastfm_id"`
	DiscogsID        string     `gorm:"size:255" json:"discogs_id"`
	ITunesID         string     `gorm:"size:255" json:"itunes_id"`
	AMGID            string     `gorm:"size:255" json:"amg_id"`
	WikidataID       string     `gorm:"size:255" json:"wikidata_id"`
	SortOrder        int32      `gorm:"default:0" json:"sort_order"`

	// Relationships
	Albums []Album `gorm:"foreignKey:ArtistID" json:"-"`
}

func (Artist) TableName() string {
	return "artists"
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
	ArtistID            int64      `gorm:"index:idx_albums_artist_id_covering;not null" json:"artist_id"`
	TrackCountCached    int32      `gorm:"default:0" json:"track_count_cached"` // Pre-calculated for performance
	DurationCached      int64      `gorm:"default:0" json:"duration_cached"`    // duration in milliseconds
	CreatedAt           time.Time  `json:"created_at"`
	Tags                []byte     `gorm:"type:jsonb;index:idx_albums_tags_gin" json:"tags"` // Stored as JSONB
	ReleaseDate         *time.Time `json:"release_date"`
	OriginalReleaseDate *time.Time `json:"original_release_date"`
	AlbumType           string     `gorm:"size:50;default:'NotSet';check:album_type IN ('NotSet', 'Album', 'EP', 'Single', 'Compilation', 'Live', 'Remix', 'Soundtrack', 'SpokenWord', 'Interview', 'Audiobook');index:idx_albums_type" json:"album_type"`
	Directory           string     `gorm:"size:512;not null;index:idx_albums_directory" json:"directory"` // Relative path from library base
	SortName            string     `gorm:"size:255;index:idx_albums_sort_name" json:"sort_name"`
	SortOrder           int32      `gorm:"default:0;index:idx_albums_sort_order" json:"sort_order"`
	ImageCount          int32      `gorm:"default:0" json:"image_count"`
	Comment             string     `json:"comment"`
	Description         string     `json:"description"`
	Genres              []string   `gorm:"type:text[];index:idx_albums_genres_gin" json:"genres"`
	Moods               []string   `gorm:"type:text[];index:idx_albums_moods_gin" json:"moods"`
	Notes               string     `json:"notes"`
	DeezerID            string     `gorm:"size:255;index:idx_albums_deezer_id" json:"deezer_id"`
	MusicBrainzID       *uuid.UUID `gorm:"type:uuid;index" json:"musicbrainz_id"`
	SpotifyID           string     `gorm:"size:255;index:idx_albums_spotify_id" json:"spotify_id"`
	LastFmID            string     `gorm:"size:255;index:idx_albums_lastfm_id" json:"lastfm_id"`
	DiscogsID           string     `gorm:"size:255;index:idx_albums_discogs_id" json:"discogs_id"`
	ITunesID            string     `gorm:"size:255;index:idx_albums_itunes_id" json:"itunes_id"`
	AMGID               string     `gorm:"size:255;index:idx_albums_amg_id" json:"amg_id"`
	WikidataID          string     `gorm:"size:255;index:idx_albums_wikidata_id" json:"wikidata_id"`
	IsCompilation       bool       `gorm:"default:false;index:idx_albums_compilation" json:"is_compilation"`

	// Relationships
	Artist *Artist `gorm:"foreignKey:ArtistID" json:"artist"`
	Tracks []Track `gorm:"foreignKey:AlbumID" json:"tracks"`
}

func (Album) TableName() string {
	return "albums"
}

// BeforeCreate sets the API key before creating an album
func (a *Album) BeforeCreate(tx *gorm.DB) error {
	if a.APIKey == uuid.Nil {
		a.APIKey = uuid.New()
	}
	return nil
}

// Track represents the tracks table
type Track struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey         uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	NameNormalized string    `gorm:"size:255;not null;index:idx_tracks_name_normalized_gin,gin" json:"name_normalized"`
	SortName       string    `gorm:"size:255" json:"sort_name"`
	AlbumID        int64     `gorm:"index:idx_tracks_album_id_hash,hash;index:idx_tracks_album_id_sort_order;not null" json:"album_id"`
	ArtistID       int64     `gorm:"index:idx_tracks_artist_id_hash,hash;index:idx_tracks_artist_id_album_id;not null" json:"artist_id"` // Denormalized for performance
	Duration       int64     `json:"duration"`                                                                                            // duration in milliseconds
	BitRate        int32     `json:"bit_rate"`                                                                                            // in kbps
	BitDepth       int32     `json:"bit_depth"`
	SampleRate     int32     `json:"sample_rate"` // in Hz
	Channels       int32     `json:"channels"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []byte    `gorm:"type:jsonb;index:idx_tracks_tags_gin" json:"tags"`            // Stored as JSONB
	Directory      string    `gorm:"size:512;not null" json:"directory"`                          // Relative path from library base
	FileName       string    `gorm:"not null" json:"file_name"`                                   // Just the filename for optimized storage
	RelativePath   string    `gorm:"not null;index:idx_tracks_relative_path" json:"relative_path"` // directory + file_name
	CRCHash        string    `gorm:"size:255;not null" json:"crc_hash"`
	SortOrder      int32     `gorm:"default:0;index:idx_tracks_sort_order" json:"sort_order"`

	// Relationships
	Album  *Album  `gorm:"foreignKey:AlbumID" json:"album"`
	Artist *Artist `gorm:"foreignKey:ArtistID" json:"artist"`
}

func (Track) TableName() string {
	return "tracks"
}

// BeforeCreate sets the API key before creating a track
func (t *Track) BeforeCreate(tx *gorm.DB) error {
	if t.APIKey == uuid.Nil {
		t.APIKey = uuid.New()
	}
	return nil
}

// Playlist represents the playlists table
type Playlist struct {
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey     uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	UserID     int64     `gorm:"index;not null" json:"user_id"`
	Name       string    `gorm:"size:255;not null" json:"name"`
	Comment    string    `json:"comment"`
	Public     bool      `gorm:"default:false;index:idx_playlists_public_where" json:"public"`
	CreatedAt  time.Time `json:"created_at"`
	ChangedAt  time.Time `json:"changed_at"`
	Duration   int64     `json:"duration"` // duration in milliseconds
	TrackCount int32     `json:"track_count"`
	CoverArtID *int32    `json:"cover_art_id"` // foreign key to images table

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user"`
}

func (Playlist) TableName() string {
	return "playlists"
}

// BeforeCreate sets the API key before creating a playlist
func (p *Playlist) BeforeCreate(tx *gorm.DB) error {
	if p.APIKey == uuid.Nil {
		p.APIKey = uuid.New()
	}
	return nil
}

// PlaylistTrack represents the playlist_tracks junction table
type PlaylistTrack struct {
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlaylistID int32     `gorm:"index:idx_playlist_tracks_playlist_pos;not null" json:"playlist_id"`
	TrackID    int64     `gorm:"index;not null" json:"track_id"`
	Position   int32     `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	Playlist *Playlist `gorm:"foreignKey:PlaylistID" json:"playlist"`
	Track    *Track    `gorm:"foreignKey:TrackID" json:"track"`

	// Constraints: UNIQUE(playlist_id, position)
}

func (PlaylistTrack) TableName() string {
	return "playlist_tracks"
}

// UserTrack represents user interactions with tracks
type UserTrack struct {
	ID           int32      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64      `gorm:"index:idx_user_tracks_user_id;not null" json:"user_id"`
	TrackID      int64      `gorm:"index:idx_user_tracks_track_id;not null" json:"track_id"`
	PlayedCount  int32      `gorm:"default:0;index:idx_user_tracks_played_count" json:"played_count"`
	LastPlayedAt *time.Time `gorm:"index:idx_user_tracks_last_played" json:"last_played_at"`
	IsStarred    bool       `gorm:"default:false;index:idx_user_tracks_starred" json:"is_starred"`
	IsHated      bool       `gorm:"default:false;index:idx_user_tracks_hated" json:"is_hated"` // When true, don't include in randomization
	StarredAt    *time.Time `json:"starred_at"`
	Rating       int8       `gorm:"check:rating >= 0 AND rating <= 5;default:0;index:idx_user_tracks_rating" json:"rating"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `gorm:"index:idx_user_tracks_updated_at" json:"updated_at"`

	// Constraints: UNIQUE(user_id, track_id)
}

// UserAlbum represents user interactions with albums
type UserAlbum struct {
	ID           int32      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64      `gorm:"index:idx_user_albums_user_id;not null" json:"user_id"`
	AlbumID      int64      `gorm:"index:idx_user_albums_album_id;not null" json:"album_id"`
	PlayedCount  int32      `gorm:"default:0;index:idx_user_albums_played_count" json:"played_count"`
	LastPlayedAt *time.Time `gorm:"index:idx_user_albums_last_played" json:"last_played_at"`
	IsStarred    bool       `gorm:"default:false;index:idx_user_albums_starred" json:"is_starred"`
	IsHated      bool       `gorm:"default:false;index:idx_user_albums_hated" json:"is_hated"` // When true, don't include in randomization
	StarredAt    *time.Time `json:"starred_at"`
	Rating       int8       `gorm:"check:rating >= 0 AND rating <= 5;default:0;index:idx_user_albums_rating" json:"rating"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `gorm:"index:idx_user_albums_updated_at" json:"updated_at"`

	// Constraints: UNIQUE(user_id, album_id)
}

// UserArtist represents user interactions with artists
type UserArtist struct {
	ID        int32      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64      `gorm:"index:idx_user_artists_user_id;not null" json:"user_id"`
	ArtistID  int64      `gorm:"index:idx_user_artists_artist_id;not null" json:"artist_id"`
	IsStarred bool       `gorm:"default:false;index:idx_user_artists_starred" json:"is_starred"`
	IsHated   bool       `gorm:"default:false;index:idx_user_artists_hated" json:"is_hated"` // When true, don't include in randomization
	StarredAt *time.Time `json:"starred_at"`
	Rating    int8       `gorm:"check:rating >= 0 AND rating <= 5;default:0;index:idx_user_artists_rating" json:"rating"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `gorm:"index:idx_user_artists_updated_at" json:"updated_at"`

	// Constraints: UNIQUE(user_id, artist_id)
}

// UserPin represents pinned content
type UserPin struct {
	ID       int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   int64     `gorm:"not null" json:"user_id"`
	TrackID  *int64    `json:"track_id"`
	AlbumID  *int64    `json:"album_id"`
	ArtistID *int64    `json:"artist_id"`
	PinnedAt time.Time `json:"pinned_at"`

	// Only one of TrackID, AlbumID, or ArtistID should be set
}

// Bookmark represents user bookmarks
type Bookmark struct {
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	TrackID   int64     `gorm:"not null" json:"track_id"`
	Comment   string    `json:"comment"`
	Position  int32     `gorm:"not null" json:"position"` // Position in milliseconds
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Constraints: UNIQUE(user_id, track_id)
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
	ID             int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64     `gorm:"not null" json:"user_id"`
	TrackID        int64     `gorm:"not null" json:"track_id"`
	TrackAPIKey    uuid.UUID `gorm:"type:uuid;not null" json:"track_api_key"` // To not expose internal track IDs to API consumers
	IsCurrentTrack bool      `gorm:"default:false" json:"is_current_track"`
	ChangedBy      string    `gorm:"size:255;not null" json:"changed_by"`
	Position       float64   `gorm:"default:0" json:"position"`
	PlayQueueID    int32     `gorm:"not null" json:"play_queue_id"` // To manage order in the queue
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SearchHistory represents user search history
type SearchHistory struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       *int64    `gorm:"index:idx_search_histories_user_id" json:"user_id"`
	SearchTerm   string    `gorm:"size:500;not null;index:idx_search_histories_search_term_gin,gin" json:"search_term"`
	SearchType   string    `gorm:"size:50;not null;check:search_type IN ('artist', 'album', 'track', 'any');index:idx_search_histories_search_type" json:"search_type"`
	ResultsCount int32     `gorm:"default:0;index:idx_search_histories_results_count" json:"results_count"`
	CreatedAt    time.Time `gorm:"index:idx_search_histories_created_at" json:"created_at"`
}

// Share represents shared content
type Share struct {
	ID                  int32      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID              int64      `gorm:"not null" json:"user_id"`
	Name                string     `gorm:"size:255" json:"name"`
	Description         string     `json:"description"`
	ExpiresAt           *time.Time `json:"expires_at"`
	MaxStreamingMinutes int32      `json:"max_streaming_minutes"`
	MaxStreamingCount   int32      `json:"max_streaming_count"`
	AllowStreaming      bool       `gorm:"default:true" json:"allow_streaming"`
	AllowDownload       bool       `gorm:"default:false" json:"allow_download"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// ShareActivity represents share usage tracking
type ShareActivity struct {
	ID         int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID    int32     `gorm:"not null" json:"share_id"`
	UserID     *int64    `json:"user_id"` // User who accessed (null if anonymous)
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent  string    `json:"user_agent"`
}

// LibraryScanHistory represents library scanning history
type LibraryScanHistory struct {
	ID             int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	LibraryID      int32     `gorm:"not null" json:"library_id"`
	Status         string    `gorm:"size:50;not null;check:status IN ('started', 'in_progress', 'completed', 'failed')" json:"status"`
	Message        string    `json:"message"`
	TotalFiles     int32     `json:"total_files"`
	ProcessedFiles int32     `json:"processed_files"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	ID            int32      `gorm:"primaryKey;autoIncrement" json:"id"`
	FromArtistID  int64      `gorm:"not null" json:"from_artist_id"`
	ToArtistID    int64      `gorm:"not null" json:"to_artist_id"`
	RelationType  string     `gorm:"size:100;not null" json:"relation_type"` // e.g., 'member', 'collaborator', 'influenced_by'
	RelationStart *time.Time `json:"relation_start"`                         // When the relationship started
	RelationEnd   *time.Time `json:"relation_end"`                           // When the relationship ended
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	IsLocked      bool       `gorm:"default:false" json:"is_locked"`
	SortOrder     int32      `gorm:"default:0" json:"sort_order"`
	APIKey        uuid.UUID  `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Tags          []byte     `gorm:"type:jsonb" json:"tags"` // Stored as JSONB
	Notes         string     `json:"notes"`
	Description   string     `json:"description"`

	// Constraints: UNIQUE(from_artist_id, to_artist_id, relation_type)
}

// RadioStation represents radio stations
type RadioStation struct {
	ID              int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKey          uuid.UUID `gorm:"type:uuid;uniqueIndex;default:gen_random_uuid()" json:"api_key"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	StreamURL       string    `gorm:"not null" json:"stream_url"`
	HomePageURL     string    `json:"home_page_url"`
	CreatedByUserID *int64    `json:"created_by_user_id"`
	TrackCount       int32     `json:"song_count"`
	IsEnabled       bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt       time.Time `json:"created_at"`
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
	ID        int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Type      string    `gorm:"size:100;not null" json:"type"` // e.g., 'performer', 'composer', 'producer'
	SortName  string    `gorm:"size:255" json:"sort_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CapacityStatus represents the capacity status of a library
type CapacityStatus struct {
	ID           int32     `gorm:"primaryKey;autoIncrement" json:"id"`
	LibraryID    int32     `gorm:"not null;index" json:"library_id"`
	Path         string    `gorm:"not null" json:"path"`
	UsedPercent  float64   `json:"used_percent"`
	Status       string    `json:"status"` // "ok", "warning", "alert", "unknown"
	LatestReadAt time.Time `json:"latest_read_at"`
	ErrorCount   int       `json:"error_count"`
	LastError    string    `json:"last_error"`
	NextCheckAt  time.Time `json:"next_check_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CapacityStatus) TableName() string {
	return "capacity_statuses"
}

// StagingItem represents items in the file-based staging workflow
type StagingItem struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID       string     `gorm:"not null;index" json:"scan_id"`
	StagingPath  string     `gorm:"not null;unique" json:"staging_path"`
	MetadataFile string     `gorm:"not null" json:"metadata_file"`
	ArtistName   string     `gorm:"not null;index:idx_staging_artist_album" json:"artist_name"`
	AlbumName    string     `gorm:"not null;index:idx_staging_artist_album" json:"album_name"`
	TrackCount   int32      `gorm:"default:0" json:"track_count"`
	TotalSize    int64      `gorm:"default:0" json:"total_size"`
	ProcessedAt  time.Time  `gorm:"not null" json:"processed_at"`
	Status       string     `gorm:"size:50;not null;check:status IN ('pending_review', 'approved', 'rejected');index" json:"status"`
	ReviewedBy   *int64     `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	Notes        string     `json:"notes"`
	Checksum     string     `gorm:"not null" json:"checksum"`
	CreatedAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (StagingItem) TableName() string {
	return "staging_items"
}
