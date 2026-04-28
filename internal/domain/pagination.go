// File: internal/domain/pagination.go
package domain

// PaginationParams adalah struct yang di-bind dari query string request.
// Contoh: GET /students?page=1&limit=10&search=budi
type PaginationParams struct {
	Page    int    `form:"page"`
	Limit   int    `form:"limit"`
	Search  string `form:"search"`
	SortBy  string `form:"sort_by"`
	SortDir string `form:"sort_dir"` // "asc" atau "desc"
}

// PaginatedResult adalah wrapper response standar untuk semua endpoint yang mendukung pagination.
// Menggunakan Go Generics (Go 1.18+) agar bisa dipakai oleh semua domain.
type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
