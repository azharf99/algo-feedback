// File: internal/repository/group_repository_test.go
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupGroupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	// Kita mematikan logger GORM di test agar terminal tidak kotor
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestGroupRepository_Upsert_CreateNew(t *testing.T) {
	// 1. Arrange
	gormDB, mock := setupGroupMockDB(t)
	repo := repository.NewGroupRepository(gormDB)

	now := time.Now()
	group := &domain.Group{ID: 1, Name: "Golang Backend 1", CreatedAt: now, UpdatedAt: now}
	studentIDs := []uint{10, 11}

	// --- FASE 1: Cek & Buat Grup ---
	mock.ExpectQuery(`SELECT \* FROM "groups"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "groups"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// --- FASE 2: Cari Siswa ---
	studentRows := sqlmock.NewRows([]string{"id", "fullname"}).AddRow(10, "Budi").AddRow(11, "Siti")
	mock.ExpectQuery(`SELECT \* FROM "students"`).WillReturnRows(studentRows)

	// --- FASE 3: Replace Association (TERNYATA DIPECAH 2 TRANSAKSI OLEH GORM!) ---

	// Transaksi 3A: Append (Update parent, Save child, Insert Pivot)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "groups"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`INSERT INTO "students"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10).AddRow(11))
	mock.ExpectExec(`INSERT INTO "group_students"`).WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectCommit()

	// Transaksi 3B: Clear (Menghapus data lama di pivot)
	mock.ExpectBegin() // <--- INI DIA PENYEBAB ERROR TERAKHIRMU!
	mock.ExpectExec(`DELETE FROM "group_students"`).WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectCommit()

	// 2. Act
	isCreated, err := repo.Upsert(context.Background(), group, studentIDs)

	// 3. Assert
	assert.NoError(t, err)
	assert.True(t, isCreated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupRepository_Upsert_UpdateExisting_EmptyStudents(t *testing.T) {
	// 1. Arrange
	gormDB, mock := setupGroupMockDB(t)
	repo := repository.NewGroupRepository(gormDB)

	group := &domain.Group{ID: 2, Name: "Golang Backend 2"}
	studentIDs := []uint{}

	// Langkah 1: Cek Grup
	mock.ExpectQuery(`SELECT \* FROM "groups"`).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "Golang Lamo"))

	// Langkah 2: Update Grup
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "groups" SET`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Langkah 3: Clear Association
	mock.ExpectBegin()

	// [FIXED] Ekspektasi UPDATE "groups" dihapus karena GORM tidak memicunya di sini.
	// GORM langsung mengeksekusi DELETE pada tabel pivot.
	mock.ExpectExec(`DELETE FROM "group_students"`).WillReturnResult(sqlmock.NewResult(1, 5))

	mock.ExpectCommit()

	// 2. Act
	isCreated, err := repo.Upsert(context.Background(), group, studentIDs)

	// 3. Assert
	assert.NoError(t, err)
	assert.False(t, isCreated)
	assert.NoError(t, mock.ExpectationsWereMet())
}
