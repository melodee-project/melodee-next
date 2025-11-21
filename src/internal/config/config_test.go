package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoader_Load(t *testing.T) {
	// Create a temporary config file
	tmpfile := "test_config.yaml"
	configContent := `
server:
  host: "localhost"
  port: 9090
database:
  host: "test-db"
  user: "testuser"
  dbname: "testdb"
  password: "testpass"
redis:
  addr: "localhost:6379"
  password: "redispass"
`

	err := os.WriteFile(tmpfile, []byte(configContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(tmpfile)

	// Set config file name
	loader := NewConfigLoader()
	loader.viper.SetConfigFile(tmpfile)

	config, err := loader.Load()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Test server config
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, 9090, config.Server.Port)

	// Test database config
	assert.Equal(t, "test-db", config.Database.Host)
	assert.Equal(t, "testuser", config.Database.User)
	assert.Equal(t, "testdb", config.Database.DBName)
	assert.Equal(t, "testpass", config.Database.Password)

	// Test redis config
	assert.Equal(t, "localhost:6379", config.Redis.Addr)
	assert.Equal(t, "redispass", config.Redis.Password)
}

func TestConfigLoader_EnvironmentOverride(t *testing.T) {
	// Create a temporary config file that overrides some values
	tmpfile := "test_config_with_env_override.yaml"
	configContent := `
server:
  host: "localhost"
  port: 8888
database:
  host: "env-db"
  user: "testuser"
  dbname: "testdb"
  password: "testpass"
redis:
  addr: "localhost:6379"
paths:
  data_dir: "/tmp/data"
  storage_dir: "/tmp/storage"
directory:
  template: "test-template"
`

	err := os.WriteFile(tmpfile, []byte(configContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(tmpfile)

	// Set environment variables to override specific values
	os.Setenv("MELODEE_SERVER_HOST", "env-host")
	os.Setenv("MELODEE_DATABASE_HOST", "override-db")  // This should override the YAML
	defer os.Unsetenv("MELODEE_SERVER_HOST")
	defer os.Unsetenv("MELODEE_DATABASE_HOST")

	loader := NewConfigLoader()
	loader.viper.SetConfigFile(tmpfile)

	config, err := loader.Load()
	assert.NoError(t, err, "Config loading should succeed with required values")
	assert.NotNil(t, config)

	if config != nil {
		// Environment variable should override YAML config
		assert.Equal(t, "env-host", config.Server.Host)
		assert.Equal(t, 8888, config.Server.Port)  // From YAML file
		assert.Equal(t, "override-db", config.Database.Host)  // From environment override
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &AppConfig{
		Server: ServerConfig{Port: 8080},
		Database: DatabaseConfig{
			Host:   "localhost",
			User:   "testuser",
			DBName: "testdb",
		},
		Redis: RedisConfig{Addr: "localhost:6379"},
		Paths: PathConfig{
			DataDir:    "/data",
			StorageDir: "/storage",
		},
		Directory: DirectoryConfig{
			Template: "template",
			CodeConfig: DirectoryCodeConfig{
				MaxLength: 10,
			},
		},
	}

	err := validateConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid port
	invalidPortConfig := *validConfig
	invalidPortConfig.Server.Port = 70000 // Invalid port
	err = validateConfig(&invalidPortConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server.port must be between 1 and 65535")

	// Test missing database fields
	missingDBConfig := *validConfig
	missingDBConfig.Database.Host = ""
	err = validateConfig(&missingDBConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database.host cannot be empty")

	// Test missing required paths
	missingPathConfig := *validConfig
	missingPathConfig.Paths.DataDir = ""
	err = validateConfig(&missingPathConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "paths.data_dir cannot be empty")

	// Test invalid directory code config
	invalidDirConfig := *validConfig
	invalidDirConfig.Directory.CodeConfig.MaxLength = 1 // Less than minimum
	err = validateConfig(&invalidDirConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory.code_config.max_length must be at least 2")
}