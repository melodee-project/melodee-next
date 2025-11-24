package pagination

import (
	"testing"
)

func BenchmarkCalculate(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = Calculate(1000000, 10, 50) // Large total, page 10, page size 50
	}
}

func BenchmarkCalculateLarge(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = Calculate(10000000, 100, 200) // Very large total, later page, max page size
	}
}

func TestCalculateWithLargeDataset(t *testing.T) {
	tests := []struct {
		name        string
		totalCount  int64
		page        int
		pageSize    int
		expectedTotalPages int
	}{
		{"Large dataset, first page", 1000000, 1, 50, 20000},
		{"Large dataset, middle page", 1000000, 5000, 50, 20000},
		{"Large dataset, last page", 1000000, 20000, 50, 20000},
		{"Max page size", 1000000, 1, 200, 5000},
		{"Very large dataset", 100000000, 1, 200, 500000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := Calculate(tt.totalCount, tt.page, tt.pageSize)
			
			if meta.TotalCount != tt.totalCount {
				t.Errorf("TotalCount = %d, want %d", meta.TotalCount, tt.totalCount)
			}
			if meta.CurrentPage != tt.page {
				t.Errorf("CurrentPage = %d, want %d", meta.CurrentPage, tt.page)
			}
			if meta.TotalPages != tt.expectedTotalPages {
				t.Errorf("TotalPages = %d, want %d", meta.TotalPages, tt.expectedTotalPages)
			}
			if meta.PageSize != tt.pageSize {
				t.Errorf("PageSize = %d, want %d", meta.PageSize, tt.pageSize)
			}
		})
	}
}

func TestCalculateEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		totalCount  int64
		page        int
		pageSize    int
	}{
		{"Zero total", 0, 1, 50},
		{"Negative page", 1000, -1, 50},
		{"Zero page", 1000, 0, 50},
		{"Zero page size", 1000, 1, 0},
		{"Very large page", 1000, 1000000, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := Calculate(tt.totalCount, tt.page, tt.pageSize)
			
			if meta.TotalCount != tt.totalCount {
				t.Errorf("TotalCount = %d, want %d", meta.TotalCount, tt.totalCount)
			}
		})
	}
}