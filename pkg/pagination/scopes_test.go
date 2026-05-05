package pagination

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock db: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize gorm with mock db: %v", err)
	}

	return gormDB, mock
}

type TestModel struct {
	ID   uint
	Name string
}

func TestSort(t *testing.T) {
	db, mock := setupTestDB(t)

	tests := []struct {
		name          string
		params        domain.PaginationParams
		defaultSort   string
		expectedQuery string
	}{
		{
			name: "Valid SortBy and SortDir",
			params: domain.PaginationParams{
				SortBy:  "name",
				SortDir: "DESC",
			},
			defaultSort:   "id DESC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY name DESC`,
		},
		{
			name: "Valid SortBy, Default SortDir to ASC",
			params: domain.PaginationParams{
				SortBy:  "created_at",
				SortDir: "",
			},
			defaultSort:   "id DESC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY created_at ASC`,
		},
		{
			name: "Valid SortBy, lowercase sortDir DESC",
			params: domain.PaginationParams{
				SortBy:  "updated_at",
				SortDir: "desc",
			},
			defaultSort:   "id DESC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY updated_at DESC`,
		},
		{
			name: "Invalid SortBy (SQL Injection Attempt)",
			params: domain.PaginationParams{
				SortBy:  "name; DROP TABLE users;",
				SortDir: "ASC",
			},
			defaultSort:   "id DESC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY id DESC`,
		},
		{
			name: "Empty SortBy, Fallback to Default Sort",
			params: domain.PaginationParams{
				SortBy:  "",
				SortDir: "ASC",
			},
			defaultSort:   "id ASC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY id ASC`,
		},
		{
			name: "Empty SortBy, No Default Sort",
			params: domain.PaginationParams{
				SortBy:  "",
				SortDir: "",
			},
			defaultSort:   "",
			expectedQuery: `SELECT \* FROM "test_models"`,
		},
		{
			name: "Invalid SortDir, defaults to ASC",
			params: domain.PaginationParams{
				SortBy:  "id",
				SortDir: "INVALID",
			},
			defaultSort:   "id DESC",
			expectedQuery: `SELECT \* FROM "test_models" ORDER BY id ASC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(tt.expectedQuery).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

			var results []TestModel
			err := db.Scopes(Sort(tt.params, tt.defaultSort)).Find(&results).Error

			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
