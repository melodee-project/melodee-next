package services_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"melodee/internal/handlers"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ContractTest defines the structure for API contract testing
type ContractTest struct {
	Name           string                 `json:"name"`
	Method         string                 `json:"method"`
	Endpoint       string                 `json:"endpoint"`
	Request        map[string]interface{} `json:"request"`
	ExpectedStatus int                    `json:"expected_status"`
	ExpectedBody   map[string]interface{} `json:"expected_body"`
	ExpectedSchema map[string]interface{} `json:"expected_schema"`
}

// APIContractTester handles API contract testing
type APIContractTester struct {
	app *fiber.App
	db  *gorm.DB
}

// NewAPIContractTester creates a new contract tester
func NewAPIContractTester(app *fiber.App, db *gorm.DB) *APIContractTester {
	return &APIContractTester{
		app: app,
		db:  db,
	}
}

// TestAuthContract tests authentication endpoint contracts
func TestAuthContract(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create Fiber app for testing
	app := fiber.New()

	// Initialize services
	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create test user
	password := "ValidPass123!"
	hashedPassword, err := authService.HashPassword(password)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "contract-test-user",
		Email:        "contract@test.com",
		APIKey:       uuid.New(),
		PasswordHash: hashedPassword,
	}

	err = repo.CreateUser(user)
	assert.NoError(t, err)

	// Set up routes for testing
	authHandler := handlers.NewAuthHandler(authService)
	app.Post("/api/auth/login", authHandler.Login)

	// Define contract test cases for auth endpoints
	contractTests := []ContractTest{
		{
			Name:     "Successful login returns correct structure",
			Method:   "POST",
			Endpoint: "/api/auth/login",
			Request: map[string]interface{}{
				"username": "contract-test-user",
				"password": "ValidPass123!",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"access_token":  "string",
				"refresh_token": "string",
				"expires_in":    "number", // in seconds
				"user": map[string]interface{}{
					"id":       "number", // int64
					"username": "string",
					"is_admin": "boolean",
				},
			},
		},
		{
			Name:     "Invalid credentials return 401",
			Method:   "POST",
			Endpoint: "/api/auth/login",
			Request: map[string]interface{}{
				"username": "contract-test-user",
				"password": "wrongpassword",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedSchema: map[string]interface{}{
				"error": "string",
			},
		},
		{
			Name:     "Missing password returns 400",
			Method:   "POST",
			Endpoint: "/api/auth/login",
			Request: map[string]interface{}{
				"username": "testuser",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedSchema: map[string]interface{}{
				"error": "string",
			},
		},
	}

	for _, testCase := range contractTests {
		t.Run(testCase.Name, func(t *testing.T) {
			// Convert request to JSON
			jsonData, err := json.Marshal(testCase.Request)
			assert.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(testCase.Method, testCase.Endpoint, io.NopCloser(bytes.NewBuffer(jsonData)))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

			// Read response body
			responseBody := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(responseBody)
			assert.NoError(t, err)

			// If expecting a specific schema, validate it
			if testCase.ExpectedSchema != nil {
				var responseJson map[string]interface{}
				err := json.Unmarshal(responseBody, &responseJson)
				assert.NoError(t, err)

				validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
			}
		})
	}
}

// validateResponseSchema validates that the response matches the expected schema
func validateResponseSchema(t *testing.T, response, schema interface{}) {
	switch s := schema.(type) {
	case map[string]interface{}:
		respMap, ok := response.(map[string]interface{})
		assert.True(t, ok, "expected response to be a map, got %T", response)

		for key, expectedType := range s {
			value, exists := respMap[key]
			assert.True(t, exists, "expected field '%s' to exist in response", key)

			if expectedTypeStr, isString := expectedType.(string); isString {
				// Validate type
				switch expectedTypeStr {
				case "string":
					assert.IsType(t, "", value, "field '%s' should be a string", key)
				case "number":
					assert.Condition(t, func() bool {
						_, isFloat64 := value.(float64)
						_, isInt := value.(int)
						_, isInt64 := value.(int64)
						return isFloat64 || isInt || isInt64
					}, "field '%s' should be a number", key)
				case "boolean":
					assert.IsType(t, true, value, "field '%s' should be a boolean", key)
				case "array":
					assert.IsType(t, []interface{}{}, value, "field '%s' should be an array", key)
				default:
					// Nested object
					if nestedSchema, isMap := expectedType.(map[string]interface{}); isMap {
						validateResponseSchema(t, value, nestedSchema)
					}
				}
			}
		}
	case string:
		// Primitive type validation
		switch s {
		case "string":
			assert.IsType(t, "", response, "expected a string, got %T", response)
		case "number":
			assert.Condition(t, func() bool {
				_, isFloat64 := response.(float64)
				_, isInt := response.(int)
				_, isInt64 := response.(int64)
				return isFloat64 || isInt || isInt64
			}, "expected a number, got %T", response)
		case "boolean":
			assert.IsType(t, true, response, "expected a boolean, got %T", response)
		}
	}
}

// TestUserManagementContract tests user management endpoint contracts
func TestUserManagementContract(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create Fiber app for testing
	app := fiber.New()

	// Initialize services
	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create admin user for testing
	adminPassword := "ValidPass123!"
	hashedAdminPassword, err := authService.HashPassword(adminPassword)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin-contract-test",
		Email:        "admin-contract@test.com",
		IsAdmin:      true,
		APIKey:       uuid.New(),
		PasswordHash: hashedAdminPassword,
	}

	err = repo.CreateUser(adminUser)
	assert.NoError(t, err)

	// Set up routes for testing
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(repo, authService)

	// We need auth middleware for protected routes
	app.Post("/api/auth/login", authHandler.Login)
	app.Get("/api/users", userHandler.GetUsers)    // This would need auth middleware in real implementation
	app.Post("/api/users", userHandler.CreateUser) // This would need auth middleware in real implementation

	// Define contract test cases for user management endpoints
	contractTests := []ContractTest{
		{
			Name:     "Create user returns correct structure",
			Method:   "POST",
			Endpoint: "/api/users",
			Request: map[string]interface{}{
				"username": "new-user-contract",
				"email":    "new@contract.com",
				"password": "ValidPass123!",
				"is_admin": false,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": map[string]interface{}{
					"id":         "number",
					"username":   "string",
					"email":      "string",
					"is_admin":   "boolean",
					"created_at": "string", // ISO date format
				},
			},
		},
		{
			Name:           "Get users returns correct structure",
			Method:         "GET",
			Endpoint:       "/api/users",
			Request:        nil, // No request body for GET
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": "array",
				"pagination": map[string]interface{}{
					"offset": "number",
					"limit":  "number",
					"total":  "number",
				},
			},
		},
	}

	for _, testCase := range contractTests {
		t.Run(testCase.Name, func(t *testing.T) {
			// Prepare request body if needed
			var req *http.Request
			if testCase.Request != nil {
				jsonData, err := json.Marshal(testCase.Request)
				assert.NoError(t, err)
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, io.NopCloser(bytes.NewBuffer(jsonData)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, nil)
			}

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

			// Read response body
			responseBody := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(responseBody)
			if err != nil && err.Error() != "EOF" {
				assert.NoError(t, err)
			}

			// If expecting a specific schema, validate it
			if testCase.ExpectedSchema != nil && len(responseBody) > 0 {
				var responseJson map[string]interface{}
				err := json.Unmarshal(responseBody, &responseJson)
				assert.NoError(t, err)

				validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
			}
		})
	}
}

// TestLibraryManagementContract tests library management endpoint contracts
func TestLibraryManagementContract(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create Fiber app for testing
	app := fiber.New()

	// Initialize services
	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create admin user for testing
	adminPassword := "ValidPass123!"
	hashedAdminPassword, err := authService.HashPassword(adminPassword)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin-lib-test",
		Email:        "admin-lib@test.com",
		IsAdmin:      true,
		APIKey:       uuid.New(),
		PasswordHash: hashedAdminPassword,
	}

	err = repo.CreateUser(adminUser)
	assert.NoError(t, err)

	// Set up routes for testing
	libraryHandler := handlers.NewLibraryHandler(repo, nil, nil, nil)

	// For testing purposes, let's directly test the handler functions
	app.Get("/api/libraries", libraryHandler.GetLibraries)
	app.Post("/api/libraries", libraryHandler.CreateLibrary)

	// Define contract test cases for library management endpoints
	contractTests := []ContractTest{
		{
			Name:     "Create library returns correct structure",
			Method:   "POST",
			Endpoint: "/api/libraries",
			Request: map[string]interface{}{
				"name": "Test Library",
				"path": "/test/path",
				"type": "production",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": map[string]interface{}{
					"id":          "number",
					"name":        "string",
					"path":        "string",
					"type":        "string", // "inbound", "staging", "production"
					"is_locked":   "boolean",
					"created_at":  "string",
					"track_count":  "number",
					"album_count": "number",
					"duration":    "number",
				},
			},
		},
		{
			Name:           "Get libraries returns correct structure",
			Method:         "GET",
			Endpoint:       "/api/libraries",
			Request:        nil,
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": "array",
				"pagination": map[string]interface{}{
					"offset": "number",
					"limit":  "number",
					"total":  "number",
				},
			},
		},
	}

	for _, testCase := range contractTests {
		t.Run(testCase.Name, func(t *testing.T) {
			// Prepare request body if needed
			var req *http.Request
			if testCase.Request != nil {
				jsonData, err := json.Marshal(testCase.Request)
				assert.NoError(t, err)
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, io.NopCloser(bytes.NewBuffer(jsonData)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, nil)
			}

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

			// Read response body
			responseBody := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(responseBody)
			if err != nil && err.Error() != "EOF" {
				assert.NoError(t, err)
			}

			// If expecting a specific schema, validate it
			if testCase.ExpectedSchema != nil && len(responseBody) > 0 {
				var responseJson map[string]interface{}
				err := json.Unmarshal(responseBody, &responseJson)
				assert.NoError(t, err)

				validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
			}
		})
	}
}

// TestPlaylistContract tests playlist endpoint contracts
func TestPlaylistContract(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create Fiber app for testing
	app := fiber.New()

	// Initialize services
	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create test user
	userPassword := "ValidPass123!"
	hashedUserPassword, err := authService.HashPassword(userPassword)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "playlist-contract-test",
		Email:        "playlist-contract@test.com",
		APIKey:       uuid.New(),
		PasswordHash: hashedUserPassword,
	}

	err = repo.CreateUser(user)
	assert.NoError(t, err)

	// Set up routes for testing
	playlistHandler := handlers.NewPlaylistHandler(repo)

	app.Get("/api/playlists", playlistHandler.GetPlaylists)
	app.Post("/api/playlists", playlistHandler.CreatePlaylist)

	// Define contract test cases for playlist endpoints
	contractTests := []ContractTest{
		{
			Name:     "Create playlist returns correct structure",
			Method:   "POST",
			Endpoint: "/api/playlists",
			Request: map[string]interface{}{
				"name":        "Test Playlist",
				"description": "A test playlist",
				"owner_id":    user.ID,
				"is_public":   false,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": map[string]interface{}{
					"id":          "number",
					"api_key":     "string",
					"name":        "string",
					"description": "string",
					"owner_id":    "number",
					"is_public":   "boolean",
					"track_count":  "number",
					"duration":    "number",
					"created_at":  "string",
					"changed_at":  "string",
				},
			},
		},
		{
			Name:           "Get playlists returns correct structure",
			Method:         "GET",
			Endpoint:       "/api/playlists",
			Request:        nil,
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": "array",
				"pagination": map[string]interface{}{
					"offset": "number",
					"limit":  "number",
					"total":  "number",
				},
			},
		},
	}

	for _, testCase := range contractTests {
		t.Run(testCase.Name, func(t *testing.T) {
			// Prepare request body if needed
			var req *http.Request
			if testCase.Request != nil {
				jsonData, err := json.Marshal(testCase.Request)
				assert.NoError(t, err)
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, io.NopCloser(bytes.NewBuffer(jsonData)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(testCase.Method, testCase.Endpoint, nil)
			}

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

			// Read response body
			responseBody := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(responseBody)
			if err != nil && err.Error() != "EOF" {
				assert.NoError(t, err)
			}

			// If expecting a specific schema, validate it
			if testCase.ExpectedSchema != nil && len(responseBody) > 0 {
				var responseJson map[string]interface{}
				err := json.Unmarshal(responseBody, &responseJson)
				assert.NoError(t, err)

				validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
			}
		})
	}
}

// TestSearchContract tests search endpoint contracts
func TestSearchContract(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create Fiber app for testing
	app := fiber.New()

	// Initialize services
	repo := services.NewRepository(db)

	// Set up routes for testing
	searchHandler := handlers.NewSearchHandler(repo)

	app.Get("/api/search", searchHandler.Search)

	// Define contract test cases for search endpoint
	contractTests := []ContractTest{
		{
			Name:           "Search with query returns correct structure",
			Method:         "GET",
			Endpoint:       "/api/search?q=test&type=any&limit=10&offset=0",
			Request:        nil,
			ExpectedStatus: http.StatusOK,
			ExpectedSchema: map[string]interface{}{
				"data": map[string]interface{}{ // data contains results grouped by type
					"artists": "array",
					"albums":  "array",
					"songs":   "array",
				},
				"pagination": map[string]interface{}{
					"offset": "number",
					"limit":  "number",
					"total":  "number",
				},
			},
		},
		{
			Name:           "Search without query returns error",
			Method:         "GET",
			Endpoint:       "/api/search",
			Request:        nil,
			ExpectedStatus: http.StatusBadRequest,
			ExpectedSchema: map[string]interface{}{
				"error": "string",
			},
		},
	}

	for _, testCase := range contractTests {
		t.Run(testCase.Name, func(t *testing.T) {
			// Prepare request
			req := httptest.NewRequest(testCase.Method, testCase.Endpoint, nil)
			if testCase.Request != nil {
				jsonData, err := json.Marshal(testCase.Request)
				assert.NoError(t, err)
				req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
			}

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Check status code
			assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

			// Read response body
			responseBody := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(responseBody)
			if err != nil && err.Error() != "EOF" {
				assert.NoError(t, err)
			}

			// If expecting a specific schema, validate it
			if testCase.ExpectedSchema != nil && len(responseBody) > 0 {
				var responseJson map[string]interface{}
				err := json.Unmarshal(responseBody, &responseJson)
				assert.NoError(t, err)

				validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
			}
		})
	}
}

// TestHealthCheckContract tests health check endpoint contracts
func TestHealthCheckContract(t *testing.T) {
	// Create Fiber app for testing
	app := fiber.New()

	// Initialize health service
	healthHandler := handlers.NewHealthHandler(test.GetTestDBManager(t))

	// Set up route for testing
	app.Get("/healthz", healthHandler.HealthCheck)

	// Define contract test case for health check endpoint
	testCase := ContractTest{
		Name:           "Health check returns correct structure",
		Method:         "GET",
		Endpoint:       "/healthz",
		Request:        nil,
		ExpectedStatus: http.StatusOK,
		ExpectedSchema: map[string]interface{}{
			"status": "string", // "ok", "degraded", "down"
			"db": map[string]interface{}{
				"status":     "string", // "ok", "degraded", "down"
				"latency_ms": "number",
			},
			"redis": map[string]interface{}{
				"status":     "string", // "ok", "degraded", "down"
				"latency_ms": "number",
			},
		},
	}

	t.Run(testCase.Name, func(t *testing.T) {
		// Prepare request
		req := httptest.NewRequest(testCase.Method, testCase.Endpoint, nil)

		// Execute request
		resp, err := app.Test(req)
		assert.NoError(t, err)

		// Check status code
		assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode)

		// Read response body
		responseBody := make([]byte, resp.ContentLength)
		_, err = resp.Body.Read(responseBody)
		if err != nil && err.Error() != "EOF" {
			assert.NoError(t, err)
		}

		// If expecting a specific schema, validate it
		if testCase.ExpectedSchema != nil && len(responseBody) > 0 {
			var responseJson map[string]interface{}
			err := json.Unmarshal(responseBody, &responseJson)
			assert.NoError(t, err)

			validateResponseSchema(t, responseJson, testCase.ExpectedSchema)
		}
	})
}
