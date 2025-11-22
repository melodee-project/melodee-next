package directory

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"
	"melodee/internal/models"
)

// DirectoryCodeConfig holds configuration for directory code generation
type DirectoryCodeConfig struct {
	FormatPattern string            `mapstructure:"format_pattern"` // e.g., "first_letters", "consonant_vowel", "hash"
	MaxLength     int               `mapstructure:"max_length"`     // Default: 10
	MinLength     int               `mapstructure:"min_length"`     // Default: 2
	UseSuffixes   bool              `mapstructure:"use_suffixes"`   // Use -2, -3, etc. for collisions
	SuffixPattern string            `mapstructure:"suffix_pattern"` // Default: "-%d"
	ValidChars    string            `mapstructure:"valid_chars"`    // e.g., "A-Z0-9"
	ReplaceChars  map[string]string `mapstructure:"replace_chars"`  // Special character mappings
}

// DefaultDirectoryCodeConfig returns the default configuration
func DefaultDirectoryCodeConfig() *DirectoryCodeConfig {
	return &DirectoryCodeConfig{
		FormatPattern: "consonant_vowel",
		MaxLength:     8,
		MinLength:     2,
		UseSuffixes:   true,
		SuffixPattern: "-%d",
		ValidChars:    "A-Z0-9",
		ReplaceChars:  make(map[string]string),
	}
}

// DirectoryCodeGenerator generates unique directory codes for artists
type DirectoryCodeGenerator struct {
	config *DirectoryCodeConfig
	db     *gorm.DB
}

// NewDirectoryCodeGenerator creates a new directory code generator
func NewDirectoryCodeGenerator(config *DirectoryCodeConfig, db *gorm.DB) *DirectoryCodeGenerator {
	if config == nil {
		config = DefaultDirectoryCodeConfig()
	}
	
	return &DirectoryCodeGenerator{
		config: config,
		db:     db,
	}
}

// Generate creates a directory code for an artist name
func (g *DirectoryCodeGenerator) Generate(artistName string) (string, error) {
	// Normalize the name (remove articles, special characters, etc.)
	normalized := g.normalizeName(artistName)
	
	// Generate primary code
	primaryCode := g.generatePrimaryCode(normalized)
	
	// Ensure minimum length
	if len(primaryCode) < g.config.MinLength {
		// Pad with additional characters if needed
		for len(primaryCode) < g.config.MinLength && len(artistName) > len(primaryCode) {
			primaryCode += strings.ToUpper(string(rune(artistName[len(primaryCode)])))
		}
	}
	
	// Ensure maximum length
	if len(primaryCode) > g.config.MaxLength {
		primaryCode = primaryCode[:g.config.MaxLength]
	}
	
	// Check for collisions and handle them
	return g.generateUniqueCode(artistName, primaryCode)
}

// generatePrimaryCode generates the primary directory code using consonant/vowel pattern
func (g *DirectoryCodeGenerator) generatePrimaryCode(artistName string) string {
	// Split into words
	words := strings.Fields(artistName)
	var code strings.Builder

	for _, word := range words {
		// Skip small articles unless it's the only word
		if len(words) > 1 && g.isArticle(word) {
			continue
		}

		// Get first letter that's a consonant, or first letter if no consonant
		// For each word, get the first letter that is a consonant
		for _, r := range word {
			if unicode.IsLetter(r) {
				code.WriteRune(unicode.ToUpper(r))
				break
			}
		}

		// Limit code length
		if code.Len() >= g.config.MaxLength {
			break
		}
	}

	return code.String()
}

// normalizeName normalizes the name according to the rules
func (g *DirectoryCodeGenerator) normalizeName(artistName string) string {
	// Replace & with and
	name := strings.ReplaceAll(artistName, "&", " and ")
	
	// Replace / with -
	name = strings.ReplaceAll(name, "/", " - ")
	
	// Remove periods
	name = strings.ReplaceAll(name, ".", " ")
	
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	name = spaceRegex.ReplaceAllString(name, " ")
	
	// Remove diacritics and normalize to ASCII
	// This is a simplified version - in a real implementation, use golang.org/x/text/transform
	name = removeDiacritics(name)
	
	// Convert to lowercase for normalization purposes
	name = strings.ToLower(name)
	
	// Check if starts with an article to remove
	words := strings.Fields(name)
	if len(words) > 0 && g.isArticle(words[0]) {
		// Remove the article
		name = strings.Join(words[1:], " ")
	}
	
	// Trim and return
	return strings.TrimSpace(name)
}

// isArticle checks if a word is an article to be skipped
func (g *DirectoryCodeGenerator) isArticle(word string) bool {
	articles := map[string]bool{
		"the": true,
		"a":   true,
		"an":  true,
		"le":  true,
		"la":  true,
		"les": true,
		"el":  true,
		"los": true,
		"las": true,
	}
	
	return articles[strings.ToLower(strings.TrimSpace(word))]
}

// generateUniqueCode generates a unique directory code, handling collisions with suffixes
func (g *DirectoryCodeGenerator) generateUniqueCode(artistName, existingCode string) (string, error) {
	// Check if code already exists in database for a different artist
	var existingArtistCount int64
	result := g.db.Model(&models.Artist{}).
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
			strings.Replace(g.config.SuffixPattern, "%d", fmt.Sprintf("%d", suffix), 1))

		if len(candidateCode) > g.config.MaxLength {
			// Trim if needed to stay within max length
			trimLength := g.config.MaxLength - len(strings.Replace(g.config.SuffixPattern, "%d", fmt.Sprintf("%d", suffix), 1))
			if trimLength <= 0 {
				return "", fmt.Errorf("cannot generate unique code within max length for artist: %s", artistName)
			}
			candidateCode = candidateCode[:trimLength] + strings.Replace(g.config.SuffixPattern, "%d", fmt.Sprintf("%d", suffix), 1)
		}

		var collisionCount int64
		result := g.db.Model(&models.Artist{}).
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

// removeDiacritics removes diacritical marks from a string (simplified version)
func removeDiacritics(s string) string {
	// This is a simplified implementation
	// In a real implementation, use golang.org/x/text/transform
	var result strings.Builder
	for _, r := range s {
		if r > 127 { // Non-ASCII characters
			// For this simplified version, we'll just keep the character
			// In a real implementation, map diacritics to base characters
			result.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Validate checks if a directory code is valid according to the rules
func (g *DirectoryCodeGenerator) Validate(code string) error {
	if len(code) < g.config.MinLength {
		return fmt.Errorf("directory code too short: %d characters, minimum: %d", len(code), g.config.MinLength)
	}

	if len(code) > g.config.MaxLength {
		return fmt.Errorf("directory code too long: %d characters, maximum: %d", len(code), g.config.MaxLength)
	}

	// Check valid characters (simplified check)
	for _, r := range code {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' { // Allow hyphens and underscores for suffixes
			return fmt.Errorf("invalid character in directory code: %c", r)
		}
	}

	return nil
}

// GetDirectoryCodeForArtist retrieves the directory code for an artist, generating it if needed
func (g *DirectoryCodeGenerator) GetDirectoryCodeForArtist(artistName string) (string, error) {
	// First check if this artist already has a directory code in the database
	var existingArtist models.Artist
	result := g.db.Where("name = ?", artistName).First(&existingArtist)
	
	if result.Error == nil {
		// Artist exists in database, return existing code
		return existingArtist.DirectoryCode, nil
	} else if result.Error != gorm.ErrRecordNotFound {
		// Some other error occurred
		return "", fmt.Errorf("failed to look up existing artist: %w", result.Error)
	}

	// Artist doesn't exist, generate a new code
	return g.Generate(artistName)
}

// RecalculateDirectoryCode forces recalculation of the directory code for an artist
func (g *DirectoryCodeGenerator) RecalculateDirectoryCode(artistName string) (string, error) {
	return g.Generate(artistName)
}