// File: pkg/pagination/scopes.go
package pagination

import (
	"github.com/azharf99/algo-feedback/internal/domain"
	"gorm.io/gorm"
)

// Paginate adalah GORM Scope yang mengaplikasikan OFFSET dan LIMIT ke query.
// Digunakan di semua repository dengan r.db.Scopes(pagination.Paginate(params)).
// Secara otomatis menangani nilai default dan batas maksimum.
func Paginate(params domain.PaginationParams) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		page := params.Page
		if page <= 0 {
			page = 1
		}

		limit := params.Limit
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		offset := (page - 1) * limit
		return db.Offset(offset).Limit(limit)
	}
}

// Normalize mengembalikan PaginationParams yang sudah di-clamp ke nilai valid.
// Berguna untuk dipakai di usecase saat menghitung TotalPages.
func Normalize(params domain.PaginationParams) domain.PaginationParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	return params
}
