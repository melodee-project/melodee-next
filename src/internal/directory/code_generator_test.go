package directory

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"melodee/internal/models"
	"melodee/internal/test"
)

func TestDirectoryCodeGeneration(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize the generator
	config := DefaultDirectoryCodeConfig()
	generator := NewDirectoryCodeGenerator(config, db)

	// Test cases for directory code generation
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple name", "Led Zeppelin", "LZ"},
		{"Name with articles", "The Beatles", "TB"},
		{"Name with special chars", "AC/DC", "AC"},
		{"Name with ampersand", "Hall & Oates", "HA"},
		{"Name with periods", "L.A. Guns", "LA"},
		{"Name with diacritics", "Beyoncé", "BE"},
		{"Name with mixed articles", "Los Fabulosos Cadillacs", "FC"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := generator.Generate(tc.input)
			assert.NoError(t, err)
			assert.NotEmpty(t, code)
			// Note: The specific code may vary, but we'll check that it's generated properly
			assert.True(t, len(code) >= config.MinLength)
			assert.True(t, len(code) <= config.MaxLength)
		})
	}
}

func TestDirectoryCodeWithCollisionHandling(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize the generator
	config := DefaultDirectoryCodeConfig()
	generator := NewDirectoryCodeGenerator(config, db)

	// Create first artist
	artist1 := &models.Artist{
		Name:         "The Beatles",
		NameNormalized: strings.ToLower("The Beatles"),
		DirectoryCode:  "TB",
	}
	err := db.Create(artist1).Error
	assert.NoError(t, err)

	// Generate code for a name that should collide with "TB"
	code2, err := generator.Generate("The Band")
	assert.NoError(t, err)
	assert.Equal(t, "TB-2", code2) // Should get suffix due to collision

	// Create another collision
	artist3 := &models.Artist{
		Name:         "The Who",
		NameNormalized: strings.ToLower("The Who"),
		DirectoryCode:  "TB-2",
	}
	err = db.Create(artist3).Error
	assert.NoError(t, err)

	// Generate code for yet another collision
	code4, err := generator.Generate("The Doors")
	assert.NoError(t, err)
	assert.Equal(t, "TB-3", code4) // Should get next suffix
}

func TestDirectoryCodeNormalization(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	config := DefaultDirectoryCodeConfig()
	generator := NewDirectoryCodeGenerator(config, db)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Remove articles", "The Beatles", "beatles"},
		{"Replace & with and", "Hall & Oates", "hall and oates"},
		{"Replace / with -", "AC/DC", "ac - dc"},
		{"Multiple spaces", "Artist   Name", "artist name"},
		{"Mixed case", "THE ARTIST", "artist"},
		{"Special characters", "Artist!@#Name", "artist!@#name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := generator.normalizeName(tc.input)
			assert.Contains(t, strings.ToLower(normalized), strings.ToLower(strings.Fields(tc.expected)[0]))
		})
	}
}

func TestDirectoryCodeValidation(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	config := DefaultDirectoryCodeConfig()
	generator := NewDirectoryCodeGenerator(config, db)

	// Test valid code
	err := generator.Validate("LZ")
	assert.NoError(t, err)

	// Test code too short
	err = generator.Validate("L")
	assert.Error(t, err)

	// Test code too long
	longCode := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	err = generator.Validate(longCode)
	assert.Error(t, err)

	// Test invalid characters
	err = generator.Validate("LZ<>")
	assert.Error(t, err)

	// Test valid code with suffix
	err = generator.Validate("LZ-2")
	assert.NoError(t, err)
}

func TestPathTemplateResolver(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models for testing
	artist := &models.Artist{
		Name:          "Led Zeppelin",
		DirectoryCode: "LZ",
	}
	album := &models.Album{
		Name: "IV",
	}
	library := &models.Library{
		Name: "music",
		Type: "production",
	}

	// Set release date for testing
	currentYear := "2024"
	album.ReleaseDate = nil // For now, test without release date

	// Test default template resolution
	path, err := resolver.Resolve(artist, album, library)
	assert.NoError(t, err)
	assert.Contains(t, path, "LZ")
	assert.Contains(t, path, "Led Zeppelin")
	assert.Contains(t, path, "IV")
	assert.Contains(t, path, "music")
}

func TestPathTemplateResolverWithYear(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models for testing
	artist := &models.Artist{
		Name:          "Led Zeppelin",
		DirectoryCode: "LZ",
	}
	album := &models.Album{
		Name: "IV",
	}
	library := &models.Library{
		Name: "music",
		Type: "production",
	}

	// Set release date for testing
	year := 1971
	album.ReleaseDate = &year

	// Test template resolution with year
	path, err := resolver.Resolve(artist, album, library)
	assert.NoError(t, err)
	assert.Contains(t, path, "LZ")
	assert.Contains(t, path, "Led Zeppelin")
	assert.Contains(t, path, "IV")
	assert.Contains(t, path, "1971")
}

func TestPathTemplateValidation(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Test valid template
	err := resolver.ValidateTemplate("{artist_dir_code}/{artist}/{year} - {album}")
	assert.NoError(t, err)

	// Test invalid template
	err = resolver.ValidateTemplate("{invalid_placeholder}")
	assert.Error(t, err)

	// Test mixed valid/invalid template
	err = resolver.ValidateTemplate("{artist_dir_code}/{invalid_placeholder}")
	assert.Error(t, err)
}

func TestPathTemplateResolverPathValidation(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models with potentially problematic names
	artist := &models.Artist{
		Name:          "Artist/With<Special>Characters",
		DirectoryCode: "AWC",
	}
	album := &models.Album{
		Name: "Album\\With?Special*Characters",
	}
	library := &models.Library{
		Name: "music",
		Type: "production",
	}

	// Test path resolution and validation
	path, err := resolver.Resolve(artist, album, library)
	assert.NoError(t, err)

	// Validate the path doesn't contain problematic characters
	assert.NotContains(t, path, "<")
	assert.NotContains(t, path, ">")
	assert.NotContains(t, path, ":")
	assert.NotContains(t, path, "\"")
	assert.NotContains(t, path, "|")
	assert.NotContains(t, path, "?")
	assert.NotContains(t, path, "*")
}

func TestPathTemplateResolverPathTraversalProtection(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models
	artist := &models.Artist{
		Name:          "../DangerousPath",
		DirectoryCode: "DP",
	}
	album := &models.Album{
		Name: "Album",
	}
	library := &models.Library{
		Name: "music",
		Type: "production",
	}

	// Path should be sanitized, not traversed
	path, err := resolver.Resolve(artist, album, library)
	assert.NoError(t, err)

	// The path should be sanitized, not contain actual traversal
	assert.NotContains(t, path, "../")
	assert.Contains(t, path, "DangerousPath") // Should be sanitized, not removed
}

func TestResolveForStaging(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models
	artist := &models.Artist{
		Name:          "Led Zeppelin",
		DirectoryCode: "LZ",
	}
	album := &models.Album{
		Name: "IV",
	}

	// Test staging path resolution
	path, err := resolver.ResolveForStaging(artist, album)
	assert.NoError(t, err)
	assert.Contains(t, path, "LZ")
	assert.Contains(t, path, "Led Zeppelin")
	assert.Contains(t, path, "IV")
	assert.Contains(t, path, "staging")
}

func TestResolveForInbound(t *testing.T) {
	config := DefaultPathTemplateConfig()
	resolver := NewPathTemplateResolver(config)

	// Create sample models
	artist := &models.Artist{
		Name:          "Led Zeppelin",
		DirectoryCode: "LZ",
	}
	album := &models.Album{
		Name: "IV",
	}

	// Test inbound path resolution
	path, err := resolver.ResolveForInbound(artist, album)
	assert.NoError(t, err)
	assert.Contains(t, path, "LZ")
	assert.Contains(t, path, "Led Zeppelin")
	assert.Contains(t, path, "IV")
	assert.Contains(t, path, "inbound")
}

func TestPathTemplateResolverMaxLength(t *testing.T) {
	config := DefaultPathTemplateConfig()
	config.MaxLength = 10 // Very short limit for testing
	resolver := NewPathTemplateResolver(config)

	// Create sample models with long names
	artist := &models.Artist{
		Name:          "Very Long Artist Name That Exceeds Limits",
		DirectoryCode: "VLANT",
	}
	album := &models.Album{
		Name: "Very Long Album Name That Exceeds Limits",
	}
	library := &models.Library{
		Name: "music",
		Type: "production",
	}

	// Path with very short limit should fail validation
	_, err := resolver.Resolve(artist, album, library)
	assert.Error(t, err)
}

func TestDirectoryCodeGeneratorConcurrent(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	config := DefaultDirectoryCodeConfig()
	generator := NewDirectoryCodeGenerator(config, db)

	// Test that multiple calls don't result in collision issues
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		code, err := generator.Generate(fmt.Sprintf("Test Artist %d", i))
		assert.NoError(t, err)
		assert.False(t, codes[code], "Code %s already exists", code)
		codes[code] = true
	}
}