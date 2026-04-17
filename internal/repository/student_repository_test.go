// File: internal/repository/student_repository_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupMockDB adalah fungsi bantuan untuk menyiapkan database palsu (mock)
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	// 1. Membuat koneksi database "palsu" (mock)
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// 2. Menyambungkan GORM dengan database palsu kita menggunakan driver postgres
	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	return gormDB, mock
}

// Test fungsi GetByID
func TestStudentRepository_GetByID(t *testing.T) {
	// Arrange: Persiapan
	gormDB, mock := setupMockDB(t)
	repo := repository.NewStudentRepository(gormDB)

	// Kita membuat baris data palsu (seolah-olah ini kembalian dari tabel database)
	rows := sqlmock.NewRows([]string{"id", "fullname", "username"}).
		AddRow(1, "Budi Santoso", "budisan")

	// Kita beri tahu mock: "Kalau GORM menjalankan query SELECT ke tabel students dengan ID 1,
	// tolong kembalikan data 'rows' yang sudah kita buat di atas"
	// Catatan: GORM menggunakan \. dan \* dalam query, jadi kita pakai penulisan regex
	mock.ExpectQuery(`SELECT \* FROM "students" WHERE "students"\."id" = \$1`).
		WithArgs(1, 1). // Argumen $1 adalah ID=1, Argumen $2 adalah LIMIT=1 (bawaan dari fungsi First() GORM)
		WillReturnRows(rows)

	// Act: Eksekusi fungsi yang akan diuji
	student, err := repo.GetByID(context.Background(), 1)

	// Assert: Verifikasi bahwa hasilnya sesuai dengan ekspektasi kita
	assert.NoError(t, err, "Seharusnya tidak ada error saat mengambil data")
	assert.NotNil(t, student, "Objek student tidak boleh kosong")
	assert.Equal(t, "Budi Santoso", student.Fullname)
	assert.Equal(t, "budisan", student.Username)

	// Memastikan semua query yang kita ekspektasikan benar-benar dieksekusi oleh GORM
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err, "Ada query database yang tidak terpanggil atau tidak sesuai")
}

// Test fungsi Create
func TestStudentRepository_Create(t *testing.T) {
	// Arrange: Persiapan
	gormDB, mock := setupMockDB(t)
	repo := repository.NewStudentRepository(gormDB)

	// Kita siapkan ekspektasi untuk proses INSERT data
	// Saat GORM melakukan Create, ia akan memulai Transaction (BEGIN)
	mock.ExpectBegin()

	// GORM menjalankan query INSERT dengan kolom-kolom yang ada
	mock.ExpectQuery(`INSERT INTO "students"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2)) // Simulasi database memberikan ID 2

	// GORM mengakhiri Transaction (COMMIT)
	mock.ExpectCommit()

	// Act: Menjalankan fungsi Create
	// Note: Di test nyata kita isi data properti student secara lengkap, tapi di mock kita pakai data minimal
	newStudent := &domain.Student{
		Fullname: "Siti Aminah",
		Username: "sitiaminah",
	}
	err := repo.Create(context.Background(), newStudent)

	// Assert: Verifikasi
	assert.NoError(t, err)
	assert.Equal(t, uint(2), newStudent.ID, "ID seharusnya otomatis terisi 2 dari kembalian database")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
