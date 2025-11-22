package utils

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse represents a standardized error response format
type ErrorResponse struct {
	Error   string      `json:"error"`
	Details string      `json:"details,omitempty"`
	Code    int         `json:"code,omitempty"`
	Status  string      `json:"status,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// SendErrorResponse sends a standardized error response
func SendErrorResponse(c *fiber.Ctx, httpCode int, message string, details string) error {
	return c.Status(httpCode).JSON(ErrorResponse{
		Error:   message,
		Details: details,
		Code:    httpCode,
	})
}

// SendError sends a structured error response based on HTTP status code and error details
func SendError(c *fiber.Ctx, httpCode int, message string) error {
	return c.Status(httpCode).JSON(ErrorResponse{
		Error:  message,
		Status: http.StatusText(httpCode),
		Code:   httpCode,
	})
}

// SendValidationError sends a validation error response
func SendValidationError(c *fiber.Ctx, field string, message string) error {
	return c.Status(http.StatusUnprocessableEntity).JSON(ErrorResponse{
		Error:   "Validation failed",
		Details: field + ": " + message,
		Code:    http.StatusUnprocessableEntity,
	})
}

// SendNotFoundError sends a not found error response
func SendNotFoundError(c *fiber.Ctx, resource string) error {
	return c.Status(http.StatusNotFound).JSON(ErrorResponse{
		Error:   "Resource not found",
		Details: resource + " does not exist",
		Code:    http.StatusNotFound,
	})
}

// SendUnauthorizedError sends an unauthorized error response
func SendUnauthorizedError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Error:   "Unauthorized",
		Details: message,
		Code:    http.StatusUnauthorized,
	})
}

// SendForbiddenError sends a forbidden error response
func SendForbiddenError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusForbidden).JSON(ErrorResponse{
		Error:   "Forbidden",
		Details: message,
		Code:    http.StatusForbidden,
	})
}

// SendInternalServerError sends an internal server error response
func SendInternalServerError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Error:   "Internal server error",
		Details: message,
		Code:    http.StatusInternalServerError,
	})
}

// ErrorWithCode represents an error with an associated HTTP status code
type ErrorWithCode struct {
	Err  error
	Code int
}

// NewErrorWithCode creates a new ErrorWithCode
func NewErrorWithCode(err error, code int) *ErrorWithCode {
	return &ErrorWithCode{
		Err:  err,
		Code: code,
	}
}

// Error returns the error message
func (e *ErrorWithCode) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *ErrorWithCode) Unwrap() error {
	return e.Err
}

// GetCode returns the associated HTTP status code
func (e *ErrorWithCode) GetCode() int {
	return e.Code
}