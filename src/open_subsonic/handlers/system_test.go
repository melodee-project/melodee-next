package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestSystemHandler_Ping(t *testing.T) {
	handler := NewSystemHandler(nil)
	app := fiber.New()
	app.Get("/rest/ping", handler.Ping)

	req := httptest.NewRequest("GET", "/rest/ping", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_GetLicense(t *testing.T) {
	handler := NewSystemHandler(nil)
	app := fiber.New()
	app.Get("/rest/getLicense", handler.GetLicense)

	req := httptest.NewRequest("GET", "/rest/getLicense", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_GetOpenSubsonicExtensions(t *testing.T) {
	handler := NewSystemHandler(nil)
	app := fiber.New()
	app.Get("/rest/getOpenSubsonicExtensions", handler.GetOpenSubsonicExtensions)

	req := httptest.NewRequest("GET", "/rest/getOpenSubsonicExtensions", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
