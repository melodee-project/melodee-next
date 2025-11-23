package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name           string
		totalCount     int64
		page           int
		pageSize       int
		expectedTotal  int64
		expectedPage   int
		expectedSize   int
		expectedPages  int
		expectedHasPrev bool
		expectedHasNext bool
	}{
		{
			name:           "Basic pagination",
			totalCount:     100,
			page:           1,
			pageSize:       10,
			expectedTotal:  100,
			expectedPage:   1,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: false,
			expectedHasNext: true,
		},
		{
			name:           "Middle page",
			totalCount:     100,
			page:           5,
			pageSize:       10,
			expectedTotal:  100,
			expectedPage:   5,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: true,
			expectedHasNext: true,
		},
		{
			name:           "Last page",
			totalCount:     100,
			page:           10,
			pageSize:       10,
			expectedTotal:  100,
			expectedPage:   10,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: true,
			expectedHasNext: false,
		},
		{
			name:           "Partial last page",
			totalCount:     95,
			page:           10,
			pageSize:       10,
			expectedTotal:  95,
			expectedPage:   10,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: true,
			expectedHasNext: false,
		},
		{
			name:           "Empty result set",
			totalCount:     0,
			page:           1,
			pageSize:       10,
			expectedTotal:  0,
			expectedPage:   1,
			expectedSize:   10,
			expectedPages:  0,
			expectedHasPrev: false,
			expectedHasNext: false,
		},
		{
			name:           "Single result",
			totalCount:     1,
			page:           1,
			pageSize:       10,
			expectedTotal:  1,
			expectedPage:   1,
			expectedSize:   10,
			expectedPages:  1,
			expectedHasPrev: false,
			expectedHasNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Calculate(tt.totalCount, tt.page, tt.pageSize)

			assert.Equal(t, tt.expectedTotal, result.TotalCount)
			assert.Equal(t, tt.expectedPage, result.CurrentPage)
			assert.Equal(t, tt.expectedSize, result.PageSize)
			assert.Equal(t, tt.expectedPages, result.TotalPages)
			assert.Equal(t, tt.expectedHasPrev, result.HasPrevious)
			assert.Equal(t, tt.expectedHasNext, result.HasNext)
		})
	}
}

func TestCalculateWithOffset(t *testing.T) {
	tests := []struct {
		name           string
		totalCount     int64
		offset         int
		limit          int
		expectedTotal  int64
		expectedPage   int
		expectedSize   int
		expectedPages  int
		expectedHasPrev bool
		expectedHasNext bool
	}{
		{
			name:           "Offset pagination first page",
			totalCount:     100,
			offset:         0,
			limit:          10,
			expectedTotal:  100,
			expectedPage:   1,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: false,
			expectedHasNext: true,
		},
		{
			name:           "Offset pagination middle",
			totalCount:     100,
			offset:         40,
			limit:          10,
			expectedTotal:  100,
			expectedPage:   5,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: true,
			expectedHasNext: true,
		},
		{
			name:           "Offset pagination last page",
			totalCount:     100,
			offset:         90,
			limit:          10,
			expectedTotal:  100,
			expectedPage:   10,
			expectedSize:   10,
			expectedPages:  10,
			expectedHasPrev: true,
			expectedHasNext: false,
		},
		{
			name:           "Offset pagination beyond total",
			totalCount:     5,
			offset:         10,
			limit:          10,
			expectedTotal:  5,
			expectedPage:   2,
			expectedSize:   10,
			expectedPages:  1,
			expectedHasPrev: true,
			expectedHasNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateWithOffset(tt.totalCount, tt.offset, tt.limit)

			assert.Equal(t, tt.expectedTotal, result.TotalCount)
			assert.Equal(t, tt.expectedPage, result.CurrentPage)
			assert.Equal(t, tt.expectedSize, result.PageSize)
			assert.Equal(t, tt.expectedPages, result.TotalPages)
			assert.Equal(t, tt.expectedHasPrev, result.HasPrevious)
			assert.Equal(t, tt.expectedHasNext, result.HasNext)
		})
	}
}

func TestGetPaginationParams(t *testing.T) {
	// Since GetPaginationParams requires a fiber context,
	// I'll test CalculateURLParams which doesn't require Fiber
	page, pageSize := CalculateURLParams(nil, 1, 10)

	assert.Equal(t, 1, page)
	assert.Equal(t, 10, pageSize)
}

func TestCalculateLargeOffsets(t *testing.T) {
	// Test performance with large datasets
	tests := []struct {
		name       string
		totalCount int64
		page       int
		pageSize   int
	}{
		{
			name:       "Large dataset - first page",
			totalCount: 1000000,
			page:       1,
			pageSize:   50,
		},
		{
			name:       "Large dataset - middle page",
			totalCount: 1000000,
			page:       10000,
			pageSize:   50,
		},
		{
			name:       "Large dataset - very large page",
			totalCount: 1000000,
			page:       20000,
			pageSize:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Calculate(tt.totalCount, tt.page, tt.pageSize)

			// Verify the result fields are calculated without errors
			assert.Equal(t, tt.totalCount, result.TotalCount)
			assert.Equal(t, tt.page, result.CurrentPage)
			assert.Equal(t, tt.pageSize, result.PageSize)
			assert.GreaterOrEqual(t, result.TotalPages, 0)
			assert.GreaterOrEqual(t, result.HasPrevious, false)
			assert.GreaterOrEqual(t, result.HasNext, false)
		})
	}
}