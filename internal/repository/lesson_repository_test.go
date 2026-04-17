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
	gormDB, mock := setupGroupMockDB(t) // Kita gunakan helper yang sama
	repo := repository.NewLessonRepository(gormDB)

	lesson := &domain.Lesson{ID: 1, Title: "Go Intro", GroupID: 1}
	studentIDs := []uint{101}

	// 1. Cek Lesson
	mock.ExpectQuery(`SELECT \* FROM "lessons"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 2. Create Lesson
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "lessons"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// 3. Cari Siswa
	mock.ExpectQuery(`SELECT \* FROM "students"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(101))

	// 4. Replace Association (Double Transaction Pattern)
	// Transaksi A: Append
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "lessons"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`INSERT INTO "students"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(101))
	mock.ExpectExec(`INSERT INTO "lesson_students_attended"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Transaksi B: Clear
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "lesson_students_attended"`).WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectCommit()

	isCreated, err := repo.Upsert(context.Background(), lesson, studentIDs)

	assert.NoError(t, err)
	assert.True(t, isCreated)
	assert.NoError(t, mock.ExpectationsWereMet())
}
