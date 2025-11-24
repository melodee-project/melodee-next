package database

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"melodee/internal/models"
)

// MigrationManager manages database migrations
type MigrationManager struct {
	db     *gorm.DB
	logger *zerolog.Logger
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB, logger *zerolog.Logger) *MigrationManager {
	return &MigrationManager{
		db:     db,
		logger: logger,
	}
}

// Migrate runs database migrations with partition-aware approach
func (m *MigrationManager) Migrate() error {
	// Migrate static tables first
	if err := m.migrateStaticTables(); err != nil {
		return fmt.Errorf("failed to migrate static tables: %w", err)
	}

	// Migrate partitioned tables (special handling needed)
	if err := m.migratePartitionedTables(); err != nil {
		return fmt.Errorf("failed to migrate partitioned tables: %w", err)
	}

	// Create indexes and constraints
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Create materialized views
	if err := m.createMaterializedViews(); err != nil {
		return fmt.Errorf("failed to create materialized views: %w", err)
	}

	m.logger.Info().Msg("Database migrations completed successfully")
	return nil
}

// migrateStaticTables handles migration of non-partitioned tables
func (m *MigrationManager) migrateStaticTables() error {
	// Order: (1) extensions (`uuid-ossp`, `pg_trgm`), (2) enum types, (3) static tables
	if err := m.createExtensions(); err != nil {
		return fmt.Errorf("failed to create extensions: %w", err)
	}

	// Auto-migrate static tables
	if err := m.db.AutoMigrate(
		&models.User{}, &models.Library{}, &models.Playlist{}, &models.PlaylistSong{},
		&models.UserSong{}, &models.UserAlbum{}, &models.UserArtist{}, &models.UserPin{}, &models.Bookmark{},
		&models.Player{}, &models.PlayQueue{}, &models.SearchHistory{}, &models.Share{}, &models.ShareActivity{},
		&models.LibraryScanHistory{}, &models.Setting{}, &models.ArtistRelation{}, &models.RadioStation{}, &models.Contributor{},
	); err != nil {
		return fmt.Errorf("failed to migrate static tables: %w", err)
	}

	return nil
}

// migratePartitionedTables handles migration of partitioned tables
func (m *MigrationManager) migratePartitionedTables() error {
	// For PostgreSQL partitioning, we need to create the main table first with partitioning
	// and then create the partitions separately

	// For Artists table, we'll use hash partitioning
	// Note: We'll handle the partitioning manually for now with GORM's raw SQL capabilities
	// because GORM doesn't have direct support for PostgreSQL partitioning

	// Create the main artists table with partitioning (this needs to be done via raw SQL)
	// For simplicity in this initial implementation, we'll handle it separately
	if err := m.createArtistsPartitions(); err != nil {
		return fmt.Errorf("failed to create artists partitions: %w", err)
	}

	// Create the main albums table with partitioning
	if err := m.createAlbumsTable(); err != nil {
		return fmt.Errorf("failed to create albums table: %w", err)
	}

	// Create initial album partitions
	if err := m.createAlbumPartitions(); err != nil {
		return fmt.Errorf("failed to create album partitions: %w", err)
	}

	// Create the main songs table with partitioning
	if err := m.createSongsTable(); err != nil {
		return fmt.Errorf("failed to create songs table: %w", err)
	}

	// Create initial songs partitions
	if err := m.createSongsPartitions(); err != nil {
		return fmt.Errorf("failed to create songs partitions: %w", err)
	}

	return nil
}

// createExtensions creates PostgreSQL extensions needed by the application
func (m *MigrationManager) createExtensions() error {
	extensions := []string{"uuid-ossp", "pg_trgm", "btree_gin"}

	for _, ext := range extensions {
		if err := m.db.Exec(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", ext)).Error; err != nil {
			return fmt.Errorf("failed to create extension %s: %w", ext, err)
		}
	}

	return nil
}

// createArtistsPartitions creates the artists table with manual partitioning
func (m *MigrationManager) createArtistsPartitions() error {
	// For artists, we use hash partitioning
	// Since GORM doesn't directly support PostgreSQL partitioning yet,
	// we'll create the base table and partitions using raw SQL
	queries := []string{
		// Create the main partitioned table
		`CREATE TABLE IF NOT EXISTS melodee_artists (
			id BIGSERIAL PRIMARY KEY,
			api_key UUID UNIQUE DEFAULT gen_random_uuid(),
			is_locked BOOLEAN DEFAULT FALSE,
			name VARCHAR(255) NOT NULL,
			name_normalized VARCHAR(255) NOT NULL,
			directory_code VARCHAR(20),
			sort_name VARCHAR(255),
			alternate_names TEXT[],
			song_count_cached INTEGER DEFAULT 0,
			album_count_cached INTEGER DEFAULT 0,
			duration_cached BIGINT DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_scanned_at TIMESTAMP WITH TIME ZONE,
			tags JSONB,
			musicbrainz_id UUID,
			spotify_id VARCHAR(255),
			lastfm_id VARCHAR(255),
			discogs_id VARCHAR(255),
			itunes_id VARCHAR(255),
			amg_id VARCHAR(255),
			wikidata_id VARCHAR(255),
			sort_order INTEGER DEFAULT 0
		) PARTITION BY HASH (id);`,

		// Create partitions (example with 4 partitions, can be scaled)
		`CREATE TABLE IF NOT EXISTS melodee_artists_0 PARTITION OF melodee_artists FOR VALUES WITH (MODULUS 4, REMAINDER 0);`,
		`CREATE TABLE IF NOT EXISTS melodee_artists_1 PARTITION OF melodee_artists FOR VALUES WITH (MODULUS 4, REMAINDER 1);`,
		`CREATE TABLE IF NOT EXISTS melodee_artists_2 PARTITION OF melodee_artists FOR VALUES WITH (MODULUS 4, REMAINDER 2);`,
		`CREATE TABLE IF NOT EXISTS melodee_artists_3 PARTITION OF melodee_artists FOR VALUES WITH (MODULUS 4, REMAINDER 3);`,
	}

	for _, query := range queries {
		if err := m.db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to execute partition query: %w", err)
		}
	}

	return nil
}

// createAlbumsTable creates the main albums partitioned table
func (m *MigrationManager) createAlbumsTable() error {
	query := `CREATE TABLE IF NOT EXISTS melodee_albums (
		id BIGSERIAL PRIMARY KEY,
		api_key UUID UNIQUE DEFAULT gen_random_uuid(),
		is_locked BOOLEAN DEFAULT FALSE,
		name VARCHAR(255) NOT NULL,
		name_normalized VARCHAR(255) NOT NULL,
		alternate_names TEXT[],
		artist_id BIGINT,
		song_count_cached INTEGER DEFAULT 0,
		duration_cached BIGINT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		tags JSONB,
		release_date DATE,
		original_release_date DATE,
		album_status VARCHAR(50) DEFAULT 'New' CHECK (album_status IN ('New', 'Ok', 'Invalid')),
		album_type VARCHAR(50) DEFAULT 'NotSet' CHECK (album_type IN ('NotSet', 'Album', 'EP', 'Single', 'Compilation', 'Live', 'Remix', 'Soundtrack', 'SpokenWord', 'Interview', 'Audiobook')),
		directory VARCHAR(512) NOT NULL,
		sort_name VARCHAR(255),
		sort_order INTEGER DEFAULT 0,
		image_count INTEGER DEFAULT 0,
		comment TEXT,
		description TEXT,
		genres TEXT[],
		moods TEXT[],
		notes TEXT,
		deezer_id VARCHAR(255),
		musicbrainz_id UUID,
		spotify_id VARCHAR(255),
		lastfm_id VARCHAR(255),
		discogs_id VARCHAR(255),
		itunes_id VARCHAR(255),
		amg_id VARCHAR(255),
		wikidata_id VARCHAR(255),
		is_compilation BOOLEAN DEFAULT FALSE
	) PARTITION BY RANGE (created_at);`

	return m.db.Exec(query).Error
}

// createAlbumPartitions creates monthly partitions for the albums table
func (m *MigrationManager) createAlbumPartitions() error {
	now := time.Now()
	for i := 0; i < 6; i++ { // Create partitions for 6 months
		partitionDate := now.AddDate(0, i, 0)
		partitionName := fmt.Sprintf("melodee_albums_%d_%02d", partitionDate.Year(), partitionDate.Month())
		startDate := time.Date(partitionDate.Year(), partitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(partitionDate.Year(), partitionDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)

		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s
			PARTITION OF melodee_albums
			FOR VALUES FROM ('%s') TO ('%s');
		`, partitionName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		if err := m.db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
		}
	}

	return nil
}

// createSongsTable creates the main songs partitioned table
func (m *MigrationManager) createSongsTable() error {
	query := `CREATE TABLE IF NOT EXISTS melodee_songs (
		id BIGSERIAL PRIMARY KEY,
		api_key UUID UNIQUE DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		name_normalized VARCHAR(255) NOT NULL,
		sort_name VARCHAR(255),
		album_id BIGINT,
		artist_id BIGINT,
		duration BIGINT,
		bit_rate INTEGER,
		bit_depth INTEGER,
		sample_rate INTEGER,
		channels INTEGER,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		tags JSONB,
		directory VARCHAR(512) NOT NULL,
		file_name TEXT NOT NULL,
		relative_path TEXT NOT NULL,
		crc_hash VARCHAR(255) NOT NULL,
		sort_order INTEGER DEFAULT 0
	) PARTITION BY RANGE (created_at);`

	return m.db.Exec(query).Error
}

// createSongsPartitions creates monthly partitions for the songs table
func (m *MigrationManager) createSongsPartitions() error {
	now := time.Now()
	for i := 0; i < 6; i++ { // Create partitions for 6 months
		partitionDate := now.AddDate(0, i, 0)
		partitionName := fmt.Sprintf("melodee_songs_%d_%02d", partitionDate.Year(), partitionDate.Month())
		startDate := time.Date(partitionDate.Year(), partitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(partitionDate.Year(), partitionDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)

		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s
			PARTITION OF melodee_songs
			FOR VALUES FROM ('%s') TO ('%s');
		`, partitionName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		if err := m.db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
		}
	}

	return nil
}

// createIndexes creates performance indexes
func (m *MigrationManager) createIndexes() error {
	// For partitioned tables, we need to create indexes on each partition individually
	// For simplicity, we'll focus on creating indexes on tables GORM can handle directly

	// Artists performance indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_artists_name_normalized_gin ON melodee_artists USING gin(name_normalized gin_trgm_ops);`).Error; err != nil {
		return fmt.Errorf("failed to create artists name_normalized gin index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_artists_directory_code ON melodee_artists(directory_code);`).Error; err != nil {
		return fmt.Errorf("failed to create artists directory_code index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_artists_api_key ON melodee_artists(api_key);`).Error; err != nil {
		return fmt.Errorf("failed to create artists api_key index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_artists_musicbrainz_id ON melodee_artists(musicbrainz_id);`).Error; err != nil {
		return fmt.Errorf("failed to create artists musicbrainz_id index: %w", err)
	}

	// Albums performance indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_albums_artist_id ON melodee_albums(artist_id);`).Error; err != nil {
		return fmt.Errorf("failed to create albums artist_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_albums_name_normalized_gin ON melodee_albums USING gin(name_normalized gin_trgm_ops);`).Error; err != nil {
		return fmt.Errorf("failed to create albums name_normalized gin index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_albums_api_key ON melodee_albums(api_key);`).Error; err != nil {
		return fmt.Errorf("failed to create albums api_key index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_albums_musicbrainz_id ON melodee_albums(musicbrainz_id);`).Error; err != nil {
		return fmt.Errorf("failed to create albums musicbrainz_id index: %w", err)
	}

	// Songs performance indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_songs_album_id_hash ON melodee_songs USING hash (album_id);`).Error; err != nil {
		return fmt.Errorf("failed to create songs album_id hash index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_songs_artist_id_hash ON melodee_songs USING hash (artist_id);`).Error; err != nil {
		return fmt.Errorf("failed to create songs artist_id hash index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_songs_name_normalized_gin ON melodee_songs USING gin(name_normalized gin_trgm_ops);`).Error; err != nil {
		return fmt.Errorf("failed to create songs name_normalized gin index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_songs_api_key ON melodee_songs(api_key);`).Error; err != nil {
		return fmt.Errorf("failed to create songs api_key index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_songs_relative_path ON melodee_songs(relative_path);`).Error; err != nil {
		return fmt.Errorf("failed to create songs relative_path index: %w", err)
	}

	// User performance indexes (for user management)
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`).Error; err != nil {
		return fmt.Errorf("failed to create users username index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key);`).Error; err != nil {
		return fmt.Errorf("failed to create users api_key index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`).Error; err != nil {
		return fmt.Errorf("failed to create users email index: %w", err)
	}

	// Playlist performance indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create playlists user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_playlists_api_key ON playlists(api_key);`).Error; err != nil {
		return fmt.Errorf("failed to create playlists api_key index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_playlists_public_where ON playlists(public) WHERE public = true;`).Error; err != nil {
		return fmt.Errorf("failed to create playlists public index: %w", err)
	}

	// Playlist songs indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_pos ON playlist_songs(playlist_id);`).Error; err != nil {
		return fmt.Errorf("failed to create playlist_songs playlist_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_playlist_songs_song_id ON playlist_songs(song_id);`).Error; err != nil {
		return fmt.Errorf("failed to create playlist_songs song_id index: %w", err)
	}

	// Library performance indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_libraries_path ON libraries(path);`).Error; err != nil {
		return fmt.Errorf("failed to create libraries path index: %w", err)
	}

	// Search history indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_search_histories_user_id ON search_histories(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create search_histories user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_search_histories_search_term_gin ON search_histories USING gin(search_term gin_trgm_ops);`).Error; err != nil {
		return fmt.Errorf("failed to create search_histories search_term gin index: %w", err)
	}

	// Shares indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_shares_user_id ON shares(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create shares user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_shares_expires_at ON shares(expires_at);`).Error; err != nil {
		return fmt.Errorf("failed to create shares expires_at index: %w", err)
	}

	// User song/album/artist interaction indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_songs_user_id ON user_songs(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_songs user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_songs_song_id ON user_songs(song_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_songs song_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_albums_user_id ON user_albums(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_albums user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_albums_album_id ON user_albums(album_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_albums album_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_artists_user_id ON user_artists(user_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_artists user_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_artists_artist_id ON user_artists(artist_id);`).Error; err != nil {
		return fmt.Errorf("failed to create user_artists artist_id index: %w", err)
	}

	// Capacity status indexes
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_capacity_status_library_id ON capacity_status(library_id);`).Error; err != nil {
		return fmt.Errorf("failed to create capacity_status library_id index: %w", err)
	}
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_capacity_status_path ON capacity_status(path);`).Error; err != nil {
		return fmt.Errorf("failed to create capacity_status path index: %w", err)
	}

	// Artist indexes (for partitioned table, these need to be on each partition)
	// These are handled in the partitioned table creation section above

	return nil
}

// createMaterializedViews creates materialized views for performance
func (m *MigrationManager) createMaterializedViews() error {
	query := `
		CREATE MATERIALIZED VIEW IF NOT EXISTS melodee_library_stats AS
		SELECT
			l.id as library_id,
			l.name as library_name,
			l.type,
			COUNT(s.id) as total_songs,
			COUNT(DISTINCT a.id) as total_artists,
			COUNT(DISTINCT al.id) as total_albums,
			SUM(s.duration) as total_duration,
			SUM(s.bit_rate * s.duration / 8 / 1000) as approx_size_mb -- Approximate size
		FROM melodee_libraries l
		LEFT JOIN melodee_albums al ON al.directory LIKE l.path || '%'
		LEFT JOIN melodee_songs s ON s.album_id = al.id
		LEFT JOIN melodee_artists a ON a.id = al.artist_id
		GROUP BY l.id, l.name, l.type;
	`

	if err := m.db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to create materialized view: %w", err)
	}

	// Create index on materialized view
	if err := m.db.Exec(`CREATE INDEX IF NOT EXISTS idx_library_stats_library_id ON melodee_library_stats(library_id);`).Error; err != nil {
		return fmt.Errorf("failed to create library_stats index: %w", err)
	}

	return nil
}