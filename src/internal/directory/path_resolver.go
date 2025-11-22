package directory

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"melodee/internal/models"
)

// PathTemplateConfig holds configuration for path templates
type PathTemplateConfig struct {
	DefaultTemplate     string            `mapstructure:"default_template"`
	AllowedPlaceholders []string          `mapstructure:"allowed_placeholders"`
	MaxDepth            int               `mapstructure:"max_depth"`             // Maximum directory depth
	ReservedNames       map[string]bool   `mapstructure:"reserved_names"`      // Avoid system reserved names
	MaxLength           int               `mapstructure:"max_path_length"`       // Maximum path length
}

// DefaultPathTemplateConfig returns the default configuration
func DefaultPathTemplateConfig() *PathTemplateConfig {
	return &PathTemplateConfig{
		DefaultTemplate:     "{artist_dir_code}/{artist}/{year} - {album}",
		AllowedPlaceholders: []string{"{library}", "{artist_dir_code}", "{artist}", "{album}", "{year}", "{genre}", "{type}"},
		MaxDepth:            8,
		ReservedNames: map[string]bool{
			"CON": true, "PRN": true, "AUX": true, "NUL": true,
			"COM1": true, "COM2": true, "COM3": true, "COM4": true,
			"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
			"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
			"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
		},
		MaxLength: 4096, // Default max path length
	}
}

// PathTemplateResolver resolves path templates with placeholder replacement
type PathTemplateResolver struct {
	config *PathTemplateConfig
}

// NewPathTemplateResolver creates a new path template resolver
func NewPathTemplateResolver(config *PathTemplateConfig) *PathTemplateResolver {
	if config == nil {
		config = DefaultPathTemplateConfig()
	}
	
	return &PathTemplateResolver{
		config: config,
	}
}

// TemplateData holds data for template resolution
type TemplateData struct {
	Library      string
	ArtistDirCode string
	Artist       string
	Album        string
	Year         string
	Genre        string
	Type         string // e.g., "inbound", "staging", "production"
}

// Resolve resolves a path template with the given data
func (r *PathTemplateResolver) Resolve(artist *models.Artist, album *models.Album, library *models.Library) (string, error) {
	template := r.config.DefaultTemplate
	if library != nil && library.PathTemplate != "" {
		template = library.PathTemplate
	}

	// Create template data
	data := TemplateData{
		Library:      library.Name,
		ArtistDirCode: artist.DirectoryCode,
		Artist:       artist.Name,
		Album:        album.Name,
		Type:         library.Type,
	}

	// Set year if available
	if album.ReleaseDate != nil {
		data.Year = fmt.Sprintf("%d", album.ReleaseDate.Year())
	}

	// Replace placeholders
	path := template
	path = strings.ReplaceAll(path, "{library}", sanitizePathSegment(data.Library))
	path = strings.ReplaceAll(path, "{artist_dir_code}", sanitizePathSegment(data.ArtistDirCode))
	path = strings.ReplaceAll(path, "{artist}", sanitizePathSegment(data.Artist))
	path = strings.ReplaceAll(path, "{album}", sanitizePathSegment(data.Album))
	path = strings.ReplaceAll(path, "{year}", sanitizePathSegment(data.Year))
	path = strings.ReplaceAll(path, "{genre}", sanitizePathSegment(data.Genre))
	path = strings.ReplaceAll(path, "{type}", sanitizePathSegment(data.Type))

	// Remove any remaining placeholders (could be optional)
	path = r.removeUnreplacedPlaceholders(path)

	// Validate the path
	if err := r.validatePath(path); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	return path, nil
}

// sanitizePathSegment removes or replaces invalid characters for filesystems
func sanitizePathSegment(segment string) string {
	if segment == "" {
		return ""
	}

	// Remove or replace invalid characters for filesystems
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := invalidChars.ReplaceAllString(segment, "_")

	// Replace multiple spaces with single underscore
	multipleSpaces := regexp.MustCompile(`\s+`)
	sanitized = multipleSpaces.ReplaceAllString(strings.TrimSpace(sanitized), "_")

	// Check if it's a reserved name and modify if needed
	if strings.ToUpper(sanitized) == sanitized && r.config.ReservedNames[strings.ToUpper(sanitized)] {
		sanitized = sanitized + "_"
	}

	return sanitized
}

// removeUnreplacedPlaceholders removes any placeholders that weren't replaced
func (r *PathTemplateResolver) removeUnreplacedPlaceholders(path string) string {
	// Replace any remaining placeholder-like patterns with empty strings
	// This regex looks for patterns like {unknown_placeholder}
	remainingPlaceholder := regexp.MustCompile(`\{[^{}]*\}`)
	return remainingPlaceholder.ReplaceAllString(path, "")
}

// validatePath validates the resolved path according to the configuration
func (r *PathTemplateResolver) validatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return fmt.Errorf("path traversal detected")
	}

	// Check maximum path length
	if len(path) > r.config.MaxLength {
		return fmt.Errorf("path too long: %d characters, max: %d", len(path), r.config.MaxLength)
	}

	// Check directory depth
	depth := strings.Count(path, string(filepath.Separator))
	if r.config.MaxDepth > 0 && depth > r.config.MaxDepth {
		return fmt.Errorf("path too deep: %d levels, max: %d", depth, r.config.MaxDepth)
	}

	// Check for reserved names
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.ToUpper(part) == part && r.config.ReservedNames[strings.ToUpper(part)] {
			return fmt.Errorf("path contains reserved name: %s", part)
		}
	}

	return nil
}

// ValidateTemplate validates that a template contains only allowed placeholders
func (r *PathTemplateResolver) ValidateTemplate(template string) error {
	// Find all placeholders in the template
	placeholderRegex := regexp.MustCompile(`\{[^{}]*\}`)
	placeholders := placeholderRegex.FindAllString(template, -1)

	for _, placeholder := range placeholders {
		allowed := false
		for _, allowedPlaceholder := range r.config.AllowedPlaceholders {
			if placeholder == allowedPlaceholder {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("template contains invalid placeholder: %s", placeholder)
		}
	}

	return nil
}

// ResolveForArtistAlbum resolves the path for a specific artist and album in a production library
func (r *PathTemplateResolver) ResolveForArtistAlbum(artist *models.Artist, album *models.Album, libraryPath string) (string, error) {
	// This function assumes we're working with a production library
	// Create a dummy library model for template resolution if needed
	library := &models.Library{
		Name: "production",
		Type: "production",
	}

	path, err := r.Resolve(artist, album, library)
	if err != nil {
		return "", err
	}

	// Combine with library base path
	fullPath := filepath.Join(libraryPath, path)

	// Validate the full path
	if err := r.validatePath(fullPath); err != nil {
		return "", fmt.Errorf("full path validation failed: %w", err)
	}

	return fullPath, nil
}

// ResolveForStaging resolves the path for staging area
func (r *PathTemplateResolver) ResolveForStaging(artist *models.Artist, album *models.Album) (string, error) {
	// Use a staging-specific template or default one
	stagingTemplate := "{artist_dir_code}/{artist}/{year} - {album}"
	
	// Create template data
	data := TemplateData{
		ArtistDirCode: artist.DirectoryCode,
		Artist:        artist.Name,
		Album:         album.Name,
		Type:          "staging",
	}

	// Set year if available
	if album.ReleaseDate != nil {
		data.Year = fmt.Sprintf("%d", album.ReleaseDate.Year())
	}

	// Replace placeholders
	path := stagingTemplate
	path = strings.ReplaceAll(path, "{artist_dir_code}", sanitizePathSegment(data.ArtistDirCode))
	path = strings.ReplaceAll(path, "{artist}", sanitizePathSegment(data.Artist))
	path = strings.ReplaceAll(path, "{album}", sanitizePathSegment(data.Album))
	path = strings.ReplaceAll(path, "{year}", sanitizePathSegment(data.Year))
	path = strings.ReplaceAll(path, "{type}", sanitizePathSegment(data.Type))

	// Remove any remaining placeholders
	path = r.removeUnreplacedPlaceholders(path)

	// Validate the path
	if err := r.validatePath(path); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	return path, nil
}

// ResolveForInbound resolves the path for inbound area
func (r *PathTemplateResolver) ResolveForInbound(artist *models.Artist, album *models.Album) (string, error) {
	// Use an inbound-specific template
	inboundTemplate := "{artist_dir_code}/{artist}/{year} - {album}"
	
	// Create template data
	data := TemplateData{
		ArtistDirCode: artist.DirectoryCode,
		Artist:        artist.Name,
		Album:         album.Name,
		Type:          "inbound",
	}

	// Set year if available
	if album.ReleaseDate != nil {
		data.Year = fmt.Sprintf("%d", album.ReleaseDate.Year())
	}

	// Replace placeholders
	path := inboundTemplate
	path = strings.ReplaceAll(path, "{artist_dir_code}", sanitizePathSegment(data.ArtistDirCode))
	path = strings.ReplaceAll(path, "{artist}", sanitizePathSegment(data.Artist))
	path = strings.ReplaceAll(path, "{album}", sanitizePathSegment(data.Album))
	path = strings.ReplaceAll(path, "{year}", sanitizePathSegment(data.Year))
	path = strings.ReplaceAll(path, "{type}", sanitizePathSegment(data.Type))

	// Remove any remaining placeholders
	path = r.removeUnreplacedPlaceholders(path)

	// Validate the path
	if err := r.validatePath(path); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	return path, nil
}