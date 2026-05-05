package pagination

import (
	"testing"
	"gorm.io/gorm"
	"gorm.io/driver/postgres"
	"github.com/stretchr/testify/assert"
)

type DummyModel struct {
	ID   uint
	Name string
}

func TestSort(t *testing.T) {
	db, _ := gorm.Open(postgres.Open("host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"), &gorm.Config{DryRun: true})

	tests := []struct {
		name        string
		sortBy      string
		sortDir     string
		defaultSort string
		expectedSQL string
	}{
		{
			name:        "Valid Sort ASC",
			sortBy:      "name",
			sortDir:     "asc",
			defaultSort: "id DESC",
			expectedSQL: `SELECT * FROM "dummy_models" ORDER BY "name"`,
		},
		{
			name:        "Valid Sort DESC",
			sortBy:      "name",
			sortDir:     "desc",
			defaultSort: "id DESC",
			expectedSQL: `SELECT * FROM "dummy_models" ORDER BY "name" DESC`,
		},
		{
			name:        "Invalid Sort, Use Default",
			sortBy:      "name; DROP TABLE students",
			sortDir:     "asc",
			defaultSort: "id DESC",
			expectedSQL: `SELECT * FROM "dummy_models" ORDER BY id DESC`,
		},
		{
			name:        "Empty Sort, Use Default",
			sortBy:      "",
			sortDir:     "",
			defaultSort: "id DESC",
			expectedSQL: `SELECT * FROM "dummy_models" ORDER BY id DESC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := db.Model(&DummyModel{}).Scopes(Sort(tt.sortBy, tt.sortDir, tt.defaultSort)).Find(&[]DummyModel{}).Statement
			assert.Contains(t, stmt.SQL.String(), tt.expectedSQL)
		})
	}
}
