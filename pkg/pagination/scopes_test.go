package pagination

import (
	"regexp"
	"testing"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Dummy struct {
	ID   uint
	Name string
}

func TestSort(t *testing.T) {
	db, _ := gorm.Open(postgres.Open("host=localhost"), &gorm.Config{
		DryRun: true,
	})

	tests := []struct {
		name        string
		params      domain.PaginationParams
		defaultSort string
		wantSQL     string
	}{
		{
			name:        "Valid sort asc",
			params:      domain.PaginationParams{SortBy: "name", SortDir: "asc"},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY "name"`,
		},
		{
			name:        "Valid sort desc",
			params:      domain.PaginationParams{SortBy: "name", SortDir: "desc"},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY "name" DESC`,
		},
		{
			name:        "Valid sort with spaces in SortDir",
			params:      domain.PaginationParams{SortBy: "name", SortDir: " desc "},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY "name" DESC`,
		},
		{
			name:        "Empty SortBy should use defaultSort",
			params:      domain.PaginationParams{SortBy: "", SortDir: "asc"},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY id DESC`,
		},
		{
			name:        "Invalid SortBy (SQL Injection attempt) should use defaultSort",
			params:      domain.PaginationParams{SortBy: "name; DROP TABLE users; --", SortDir: "asc"},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY id DESC`,
		},
		{
			name:        "Invalid SortBy (special chars) should use defaultSort",
			params:      domain.PaginationParams{SortBy: "name,email", SortDir: "asc"},
			defaultSort: "id DESC",
			wantSQL:     `SELECT * FROM "dummies" ORDER BY id DESC`,
		},
		{
			name:        "Empty SortBy and empty defaultSort should not apply order",
			params:      domain.PaginationParams{SortBy: "", SortDir: ""},
			defaultSort: "",
			wantSQL:     `SELECT * FROM "dummies"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := db.Model(&Dummy{}).Scopes(Sort(tt.params, tt.defaultSort)).Find(&[]Dummy{}).Statement
			sql := stmt.SQL.String()

			// Menghapus quote agar test stabil di berbagai dialect
			sql = regexp.MustCompile(`"|'`).ReplaceAllString(sql, "")
			expected := regexp.MustCompile(`"|'`).ReplaceAllString(tt.wantSQL, "")

			assert.Equal(t, expected, sql)
		})
	}
}
