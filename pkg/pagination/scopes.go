package pagination

import (
	"regexp"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var validColumnNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

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

// Sort adalah GORM Scope yang mengaplikasikan ORDER BY ke query secara aman, mencegah SQL Injection.
func Sort(params domain.PaginationParams, defaultSort string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		sortBy := strings.TrimSpace(params.SortBy)
		if sortBy != "" && validColumnNameRegex.MatchString(sortBy) {
			desc := false
			if strings.EqualFold(strings.TrimSpace(params.SortDir), "desc") {
				desc = true
			}
			return db.Order(clause.OrderByColumn{Column: clause.Column{Name: sortBy}, Desc: desc})
		}

		// Fallback default
		if defaultSort != "" {
			return db.Order(defaultSort)
		}
		return db
	}
}