// File: internal/domain/student_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestStudentModel_Creation(t *testing.T) {
	// Persiapan data (Arrange)
	phoneNumber := "08123456789"
	now := time.Now()

	// Tindakan: Membuat objek Student (Act)
	student := domain.Student{
		ID:          1,
		Fullname:    "Budi Santoso",
		Surname:     "Santoso",
		Username:    "budisan",
		Password:    "hashedpassword123",
		PhoneNumber: &phoneNumber, // Menggunakan referensi memori (pointer)
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Verifikasi (Assert)
	// 1. Memastikan data terisi dengan benar
	assert.Equal(t, uint(1), student.ID)
	assert.Equal(t, "Budi Santoso", student.Fullname)
	assert.Equal(t, "budisan", student.Username)
	assert.Equal(t, "Santoso", student.Surname)
	assert.Equal(t, "hashedpassword123", student.Password)
	assert.Equal(t, now, student.CreatedAt)
	assert.Equal(t, now, student.UpdatedAt)
	assert.True(t, student.IsActive)

	// 2. Memastikan field pointer terisi dengan benar (tidak nil)
	assert.NotNil(t, student.PhoneNumber)
	assert.Equal(t, "08123456789", *student.PhoneNumber) // Tambahkan '*' untuk membaca nilai asli pointer

	// 3. Memastikan field opsional yang tidak diisi akan bernilai nil (NULL)
	assert.Nil(t, student.ParentName)
	assert.Nil(t, student.ParentContact)
}
