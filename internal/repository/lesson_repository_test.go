// File: internal/repository/lesson_repository_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestLessonRepository_Upsert_CreateNew(t *testing.T) {
	gormDB, mock := setupGroupMockDB(t) // Gunakan helper yang sudah ada dari test group
	repo := repository.NewLessonRepository(gormDB)

	lesson := &domain.Lesson{ID: 1, Title: "Go Intro", CourseID: 10}

	// 1. Cek Lesson (Kosong -> Trigger Create)
	mock.ExpectQuery(`SELECT \* FROM "lessons"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 2. Create Lesson (Sederhana, tanpa many-to-many transaction)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "lessons"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	isCreated, err := repo.Upsert(context.Background(), lesson)

	assert.NoError(t, err)
	assert.True(t, isCreated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLessonRepository_Upsert_UpdateExisting(t *testing.T) {
	gormDB, mock := setupGroupMockDB(t)
	repo := repository.NewLessonRepository(gormDB)

	lesson := &domain.Lesson{ID: 2, Title: "Go Update", CourseID: 10}

	// 1. Cek Lesson (Ketemu -> Trigger Update)
	mock.ExpectQuery(`SELECT \* FROM "lessons"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

	// 2. Update Lesson
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "lessons"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	isCreated, err := repo.Upsert(context.Background(), lesson)

	assert.NoError(t, err)
	assert.False(t, isCreated) // Karena record sudah ada, status created = false
	assert.NoError(t, mock.ExpectationsWereMet())
}
