// File: pkg/pagination/scopes_test.go
package pagination

import (
	"testing"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestModel struct {
	ID   uint
	Name string
}

func TestSort(t *testing.T) {
	// Menggunakan DryRun jauh lebih ideal untuk testing Scope GORM
	// karena kita hanya memvalidasi SQL yang di-generate, bukan koneksi DB-nya.
	db, _ := gorm.Open(postgres.Open("host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"), &gorm.Config{DryRun: true})

	tests := []struct {
		name        string
		params      domain.PaginationParams
		defaultSort string
		expectedSQL string
	}{
		{
			name: "Valid SortBy and SortDir DESC",
			params: domain.PaginationParams{
				SortBy:  "name",
				SortDir: "DESC",
			},
			defaultSort: "id DESC",
			expectedSQL: `ORDER BY "name" DESC`, // GORM clause otomatis menambahkan kutip
		},
		{
			name: "Valid SortBy, Default SortDir to ASC",
			params: domain.PaginationParams{
				SortBy:  "name",
				SortDir: "",
			},
			defaultSort: "id DESC",
			expectedSQL: `ORDER BY "name"`, // GORM tidak merender kata 'ASC', default SQL adalah ASC
		},
		{
			name: "Valid SortBy, lowercase sortDir DESC",
			params: domain.PaginationParams{
				SortBy:  "updated_at",
				SortDir: "desc",
			},
			defaultSort: "id DESC",
			expectedSQL: `ORDER BY "updated_at" DESC`,
		},
		{
			name: "Invalid SortBy (SQL Injection Attempt)",
			params: domain.PaginationParams{
				SortBy:  "name; DROP TABLE users;",
				SortDir: "ASC",
			},
			defaultSort: "id DESC",
			expectedSQL: `ORDER BY id DESC`, // Menggunakan defaultSort (tanpa kutip karena raw string)
		},
		{
			name: "Empty SortBy, Fallback to Default Sort ASC",
			params: domain.PaginationParams{
				SortBy:  "",
				SortDir: "ASC",
			},
			defaultSort: "id ASC",
			expectedSQL: `ORDER BY id ASC`,
		},
		{
			name: "Empty SortBy, No Default Sort",
			params: domain.PaginationParams{
				SortBy:  "",
				SortDir: "",
			},
			defaultSort: "",
			expectedSQL: ``, // Seharusnya tidak ada klausa ORDER BY
		},
		{
			name: "Invalid SortDir, defaults to ASC",
			params: domain.PaginationParams{
				SortBy:  "id",
				SortDir: "INVALID",
			},
			defaultSort: "id DESC",
			expectedSQL: `ORDER BY "id"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := db.Model(&TestModel{}).Scopes(Sort(tt.params, tt.defaultSort)).Find(&[]TestModel{}).Statement
			
			if tt.expectedSQL == "" {
				// Jika tidak diekspektasikan ada sorting, pastikan ORDER BY tidak ada di kueri
				assert.NotContains(t, stmt.SQL.String(), "ORDER BY")
			} else {
				// Pastikan string SQL yang di-generate memuat ekspektasi kita
				assert.Contains(t, stmt.SQL.String(), tt.expectedSQL)
			}
		})
	}
}