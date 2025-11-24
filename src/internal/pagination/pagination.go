package pagination

import (
	"math"

	"github.com/gofiber/fiber/v2"
)

// Metadata represents the pagination metadata that matches the OpenAPI specification
type Metadata struct {
	TotalCount  int64 `json:"totalCount"`
	PageSize    int   `json:"pageSize"`
	CurrentPage int   `json:"currentPage"`
	TotalPages  int   `json:"totalPages"`
	HasPrevious bool  `json:"hasPrevious"`
	HasNext     bool  `json:"hasNext"`
}

// GetPaginationParams extracts pagination parameters from Fiber context with default values
func GetPaginationParams(c *fiber.Ctx, defaultPage, defaultPageSize int) (page int, pageSize int) {
	page = c.QueryInt("page", defaultPage)
	pageSize = c.QueryInt("pageSize", defaultPageSize)

	// Validate inputs
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > 200 { // Max page size allowed
		pageSize = 200
	}

	return page, pageSize
}

// CalculateOffset calculates the offset for database queries based on page and page size
func CalculateOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return (page - 1) * pageSize
}

// Calculate calculates the complete pagination metadata for a given total count, page, and page size
func Calculate(totalCount int64, page, pageSize int) Metadata {
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	if totalPages < 0 {
		totalPages = 0
	}

	currentPage := page
	if currentPage > totalPages && totalPages > 0 {
		currentPage = totalPages
	}

	hasPrevious := currentPage > 1
	hasNext := currentPage < totalPages

	if totalCount == 0 {
		hasPrevious = false
		hasNext = false
	}

	return Metadata{
		TotalCount:  totalCount,
		PageSize:    pageSize,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HasPrevious: hasPrevious,
		HasNext:     hasNext,
	}
}

// CalculateURLParams calculates page and pageSize from URL parameters using alternative names
func CalculateURLParams(c *fiber.Ctx, defaultPage, defaultPageSize int) (page int, pageSize int) {
	// Try both 'page'/'pageSize' and 'limit'/'offset' style params
	page = c.QueryInt("page", defaultPage)
	if page < 1 {
		page = 1
	}

	pageSize = c.QueryInt("pageSize", defaultPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > 200 {
		pageSize = 200
	}

	return page, pageSize
}

// CalculateWithOffset calculates pagination metadata when using offset/limit style parameters
func CalculateWithOffset(totalCount int64, offset, limit int) Metadata {
	pageSize := limit
	currentPage := (offset/limit) + 1
	
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	
	if totalPages < 0 {
		totalPages = 0
	}

	hasPrevious := offset > 0
	hasNext := int64(offset+limit) < totalCount

	if totalCount == 0 {
		hasPrevious = false
		hasNext = false
		currentPage = 1
	}

	return Metadata{
		TotalCount:  totalCount,
		PageSize:    pageSize,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HasPrevious: hasPrevious,
		HasNext:     hasNext,
	}
}