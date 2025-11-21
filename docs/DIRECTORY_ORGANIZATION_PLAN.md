# Directory Organization Service with Directory Code Generation Plan

## Overview
This document outlines the design for the Directory Organization Service, which is a critical component for handling massive music libraries (300k+ artists) by generating unique directory codes to prevent filesystem performance issues. This service will address the scalability challenges mentioned in the PRD by implementing the directory code feature with collision handling and configurable templates.

## Service Architecture

### Core Components

#### 1. Directory Code Generator
The primary function of this service is to generate unique directory codes for artists to prevent deep directory structures that cause performance issues with massive collections.

```go
type DirectoryCodeGenerator interface {
    Generate(artistName string, options ...CodeOption) (string, error)
    GenerateWithCollisionHandling(artistName string, existingCodes map[string]bool) (string, error)
    Validate(code string) error
}

// Implementation
type directoryCodeGenerator struct {
    config *DirectoryCodeConfig
    db     *gorm.DB // For checking existing codes in database
}

// Default directory code format: First letter + next consonant/vowel pattern, max 8 chars
// Examples: "Led Zeppelin" -> "LZ", "The Beatles" -> "TB", "AC/DC" -> "AC"
```

#### 2. Path Template Resolver
Handles configurable directory templates with placeholders for flexible organization.

```go
type PathTemplateResolver interface {
    Resolve(artist *Artist, album *Album, template string) (string, error)
    ValidateTemplate(template string) error
}

// Template placeholders: {library}, {artist_dir_code}, {artist}, {album}, {year}, etc.
// Example template: "{library}/{artist_dir_code}/{artist}/{year} - {album}"
// Results in: "music/LZ/Led Zeppelin/1971 - Led Zeppelin IV/"
```

### Configuration Options

#### Directory Code Configuration
```go
type DirectoryCodeConfig struct {
    // Format options
    FormatPattern string `mapstructure:"format_pattern"` // e.g., "first_letters", "consonant_vowel", "hash"
    MaxLength     int    `mapstructure:"max_length"`     // Default: 10
    MinLength     int    `mapstructure:"min_length"`     // Default: 2
    
    // Collision handling
    UseSuffixes   bool   `mapstructure:"use_suffixes"`   // Use -2, -3, etc. for collisions
    SuffixPattern string `mapstructure:"suffix_pattern"` // Default: "-%d"
    
    // Character handling
    ValidChars    string `mapstructure:"valid_chars"`    // e.g., "A-Z0-9"
    ReplaceChars  map[string]string `mapstructure:"replace_chars"` // Special character mappings
}
```

#### Path Template Configuration
```go
type PathTemplateConfig struct {
    DefaultTemplate string            `mapstructure:"default_template"`
    AllowedPlaceholders []string      `mapstructure:"allowed_placeholders"`
    MaxDepth          int             `mapstructure:"max_depth"` // Maximum directory depth
    ReservedNames     map[string]bool `mapstructure:"reserved_names"` // Avoid system reserved names
}
```

## Implementation Plan

### 1. Directory Code Generation Algorithm

#### Primary Algorithm
```go
// Generate directory code using consonant/vowel pattern
func (g *directoryCodeGenerator) generatePrimaryCode(artistName string) string {
    // Normalize the name (remove articles, special characters, etc.)
    normalized := g.normalizeName(artistName)
    
    // Extract first letter of each word, focusing on consonants
    words := strings.Fields(normalized)
    var code strings.Builder
    
    for _, word := range words {
        // Skip small articles unless it's the only word
        if len(words) > 1 && g.isArticle(word) {
            continue
        }
        
        // Get first letter that's a consonant, or first letter if no consonant
        firstLetter := g.getFirstLetter(word)
        if firstLetter != 0 {
            code.WriteRune(unicode.ToUpper(firstLetter))
        }
        
        // Limit code length
        if code.Len() >= g.config.MaxLength {
            break
        }
    }
    
    return code.String()
}

// Handle collisions by adding numeric suffixes
func (g *directoryCodeGenerator) generateUniqueCode(artistName string, existingCode string) (string, error) {
    // Check if code already exists in database
    var existingArtistCount int64
    result := g.db.Model(&Artist{}).
        Where("directory_code = ? AND name != ?", existingCode, artistName).
        Count(&existingArtistCount)
    
    if result.Error != nil {
        return "", fmt.Errorf("failed to check existing codes: %w", result.Error)
    }
    
    if existingArtistCount == 0 {
        return existingCode, nil // No collision
    }
    
    // Find next available suffix
    suffix := 2
    for {
        candidateCode := fmt.Sprintf("%s%s", existingCode, 
            strings.Replace(g.config.SuffixPattern, "%d", strconv.Itoa(suffix), 1))
        
        var collisionCount int64
        result := g.db.Model(&Artist{}).
            Where("directory_code = ?", candidateCode).
            Count(&collisionCount)
        
        if result.Error != nil {
            return "", fmt.Errorf("failed to check collision for suffix %d: %w", suffix, result.Error)
        }
        
        if collisionCount == 0 {
            return candidateCode, nil // Found unique code
        }
        
        suffix++
        if suffix > 10000 { // Prevent infinite loops
            return "", fmt.Errorf("too many collisions for artist: %s", artistName)
        }
    }
}
```

### 2. Path Resolution Service

#### Path Resolution Algorithm
```go
type PathResolutionService struct {
    templateResolver PathTemplateResolver
    db               *gorm.DB
}

// Resolve the complete path for a media file
func (s *PathResolutionService) ResolveMediaPath(artist *Artist, album *Album, song *Song) (string, error) {
    // Build the directory path using template
    dirPath, err := s.templateResolver.Resolve(artist, album, s.config.DefaultTemplate)
    if err != nil {
        return "", fmt.Errorf("failed to resolve directory path: %w", err)
    }
    
    // Combine with the actual file name
    fullPath := filepath.Join(s.config.BaseMediaPath, dirPath, song.FileName)
    
    return fullPath, nil
}

// Validate that the path is safe and within allowed boundaries
func (s *PathResolutionService) ValidatePath(path string) error {
    // Check for path traversal attempts
    if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
        return errors.New("path traversal detected")
    }
    
    // Check maximum path length
    if len(path) > s.config.MaxPathLength {
        return fmt.Errorf("path too long: %d characters, max: %d", len(path), s.config.MaxPathLength)
    }
    
    return nil
}
```

### 3. Data Model Integration

#### Artist Model Enhancement
```go
type Artist struct {
    ID               int64            `gorm:"primaryKey;autoIncrement;notNull"`
    APIKey           string           `gorm:"type:uuid;unique;notNull;default:gen_random_uuid()"`
    IsLocked         bool             `gorm:"default:false"`
    Name             string           `gorm:"type:varchar(255);notNull"`
    NameNormalized   string           `gorm:"type:varchar(255);index:idx_artists_name_normalized_gin,gin,option:gin_trgm_ops;notNull"`
    DirectoryCode    string           `gorm:"type:varchar(20);index:idx_artists_directory_code;notNull"`  // NEW FIELD
    SortName         string           `gorm:"type:varchar(255)"`
    AlternateNames   []string         `gorm:"type:text[]"`
    
    // ... other existing fields
    
    // Cache counts for performance
    SongCountCached  int              `gorm:"default:0"`  // NEW FIELD
    AlbumCountCached int              `gorm:"default:0"`  // NEW FIELD
    DurationCached   int64            `gorm:"default:0"`  // NEW FIELD
    
    CreatedAt        time.Time        `gorm:"default:CURRENT_TIMESTAMP"`
    LastScannedAt    *time.Time       // Can be null
    
    // JSONB fields
    Tags             map[string]interface{} `gorm:"type:jsonb"`
    
    // External IDs
    MusicBrainzID    *string          `gorm:"type:uuid"`
    SpotifyID        *string          `gorm:"type:varchar(255)"`
    LastfmID         *string          `gorm:"type:varchar(255)"`
    
    // ... other fields
}
```

### 4. Migration for Existing Data

#### Migration Strategy for Directory Codes
```go
// Migration to add directory codes to existing artists
func (m *MigrationManager) addDirectoryCodesToExistingArtists() error {
    // Fetch artists without directory codes in batches to avoid memory issues
    batchSize := 1000
    offset := 0
    
    for {
        var artists []Artist
        if err := m.db.Offset(offset).Limit(batchSize).Where("directory_code IS NULL OR directory_code = ''").Find(&artists).Error; err != nil {
            return fmt.Errorf("failed to fetch artists for directory code migration: %w", err)
        }
        
        if len(artists) == 0 {
            break // No more artists to migrate
        }
        
        // Process each artist to generate directory code
        for i := range artists {
            code, err := m.generateUniqueDirectoryCode(artists[i].Name)
            if err != nil {
                m.logger.Error().Err(err).Str("artist_name", artists[i].Name).Msg("Failed to generate directory code for artist")
                continue
            }
            
            artists[i].DirectoryCode = code
        }
        
        // Update in batch
        if err := m.db.Select("id", "directory_code").Updates(&artists).Error; err != nil {
            return fmt.Errorf("failed to update artists with directory codes: %w", err)
        }
        
        m.logger.Info().Int("processed", len(artists)).Int("offset", offset).Msg("Updated artists with directory codes")
        
        offset += batchSize
        
        // Add small delay to prevent overwhelming the database
        time.Sleep(100 * time.Millisecond)
    }
    
    return nil
}
```

### 5. Performance Considerations

#### Caching Strategy
```go
type CachedDirectoryCodeService struct {
    service *DirectoryCodeService
    cache   *redis.Client // For caching frequently accessed codes
    logger  *zerolog.Logger
}

// Get directory code with caching
func (c *CachedDirectoryCodeService) GetDirectoryCode(artistID int64) (string, error) {
    cacheKey := fmt.Sprintf("artist:dir_code:%d", artistID)
    
    // Try to get from cache first
    code, err := c.cache.Get(context.Background(), cacheKey).Result()
    if err == nil {
        return code, nil // Cache hit
    }
    
    // Cache miss - get from database
    artist := &Artist{}
    if err := c.service.db.First(artist, artistID).Error; err != nil {
        return "", fmt.Errorf("failed to find artist: %w", err)
    }
    
    // Cache the result
    c.cache.SetEX(context.Background(), cacheKey, artist.DirectoryCode, 24*time.Hour)
    
    return artist.DirectoryCode, nil
}
```

#### Bulk Processing
```go
// Process large numbers of artists efficiently
func (s *DirectoryCodeService) ProcessArtistBatch(artists []Artist) error {
    // Use database transaction for consistency
    return s.db.Transaction(func(tx *gorm.DB) error {
        for i := range artists {
            if artists[i].DirectoryCode == "" {
                code, err := s.generateUniqueCode(artists[i].Name, artists[i].DirectoryCode)
                if err != nil {
                    // Log error but continue processing other artists
                    s.logger.Error().Err(err).Str("artist", artists[i].Name).Msg("Failed to generate directory code")
                    continue
                }
                artists[i].DirectoryCode = code
            }
        }
        
        // Batch update all at once
        if err := tx.Save(&artists).Error; err != nil {
            return fmt.Errorf("failed to update artists in batch: %w", err)
        }
        
        return nil
    })
}
```

### 6. Template System

#### Flexible Template Engine
```go
type TemplateEngine struct {
    config *PathTemplateConfig
}

// Resolve path template with given data
func (t *TemplateEngine) ResolvePath(template string, data TemplateData) (string, error) {
    // Validate template
    if err := t.validateTemplate(template); err != nil {
        return "", fmt.Errorf("invalid template: %w", err)
    }
    
    // Replace placeholders
    result := template
    
    // Replace all known placeholders
    replacements := map[string]string{
        "{library}":         data.Library,
        "{artist_dir_code}": data.ArtistDirCode,
        "{artist}":          data.Artist,
        "{album}":           data.Album,
        "{year}":            data.Year,
        "{genre}":           data.Genre,
        "{type}":            data.Type,
    }
    
    for placeholder, value := range replacements {
        if value != "" { // Only replace if value exists
            result = strings.ReplaceAll(result, placeholder, sanitizePathSegment(value))
        }
    }
    
    // Remove any remaining placeholders (could be optional)
    result = t.removeUnreplacedPlaceholders(result)
    
    return result, nil
}

// Sanitize path segments to prevent issues
func sanitizePathSegment(segment string) string {
    // Remove or replace invalid characters for filesystems
    invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
    segment = invalidChars.ReplaceAllString(segment, "_")
    
    // Replace multiple spaces with single underscore
    multipleSpaces := regexp.MustCompile(`\s+`)
    segment = multipleSpaces.ReplaceAllString(strings.TrimSpace(segment), "_")
    
    return segment
}
```

### 7. Integration with File System Operations

#### File System Path Resolution
```go
type FileSystemPathResolver struct {
    pathResolutionService *PathResolutionService
    basePaths             map[string]string // Map of library types to base paths
}

// Get the actual filesystem path for a song
func (f *FileSystemPathResolver) GetSongPath(song *Song) (string, error) {
    // Get related artist and album records
    var artist Artist
    var album Album
    
    if err := f.pathResolutionService.db.First(&artist, song.ArtistID).Error; err != nil {
        return "", fmt.Errorf("failed to find artist for song %d: %w", song.ID, err)
    }
    
    if err := f.pathResolutionService.db.First(&album, song.AlbumID).Error; err != nil {
        return "", fmt.Errorf("failed to find album for song %d: %w", song.ID, err)
    }
    
    // Resolve the path using the template system
    path, err := f.pathResolutionService.ResolveMediaPath(&artist, &album, song)
    if err != nil {
        return "", fmt.Errorf("failed to resolve path for song %d: %w", song.ID, err)
    }
    
    return path, nil
}

// Verify if the file exists at the resolved path
func (f *FileSystemPathResolver) VerifyFileExists(song *Song) (bool, string, error) {
    path, err := f.GetSongPath(song)
    if err != nil {
        return false, "", err
    }
    
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return false, path, nil
    } else if err != nil {
        return false, path, fmt.Errorf("error checking file existence: %w", err)
    }
    
    return true, path, nil
}
```

## API Endpoints for Directory Code Management

### REST API Endpoints
```go
// API endpoints for directory code management
func RegisterDirectoryCodeRoutes(app *fiber.App, service *DirectoryCodeService) {
    // Get directory code for an artist
    app.Get("/api/artists/:id/directory-code", service.getArtistDirectoryCode)
    
    // Recalculate directory code for an artist (in case of name changes)
    app.Post("/api/artists/:id/recalculate-directory-code", service.recalculateDirectoryCode)
    
    // Get all directory codes with their usage statistics
    app.Get("/api/directory-codes/stats", service.getDirectoryCodeStats)
    
    // Bulk recalculate directory codes (for migration or corrections)
    app.Post("/api/directory-codes/bulk-recalculate", service.bulkRecalculateDirectoryCodes)
}
```

## Error Handling and Validation

### Comprehensive Error Types
```go
// Custom errors for directory code operations
var (
    ErrInvalidArtistName      = errors.New("artist name is empty or invalid")
    ErrDirectoryCodeTooLong   = errors.New("generated directory code exceeds maximum length")
    ErrDirectoryCodeExists    = errors.New("directory code already exists for another artist")
    ErrInvalidPathTemplate    = errors.New("path template contains invalid placeholders")
    ErrPathTraversal          = errors.New("path traversal detected in resolved path")
    ErrTooManyCollisions      = errors.New("too many directory code collisions for artist")
)
```

This Directory Organization Service design addresses the critical performance issue of handling massive artist collections (300k+) by implementing an efficient directory code generation system with proper collision handling, configurable templates, and tight integration with the database schema. The service is designed to scale with the extreme requirements mentioned in the PRD while maintaining data integrity and performance.