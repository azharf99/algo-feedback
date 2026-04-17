// File: internal/domain/user_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUserModel_Creation(t *testing.T) {
	// 1. Arrange: Persiapan data
	now := time.Now()
	// Mensimulasikan data yang dihapus (Soft Delete bawaan GORM)
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}

	// 2. Act: Inisialisasi Struct User
	user := domain.User{
		ID:        1,
		Name:      "Admin Algonova",
		Email:     "admin@algonova.com",
		Password:  "hashed_secret_password",
		Role:      domain.RoleAdmin, // Menggunakan Custom Type
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: deletedAt,
	}

	// 3. Assert: Memastikan semua properti terisi dengan tipe data yang benar
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "Admin Algonova", user.Name)
	assert.Equal(t, "admin@algonova.com", user.Email)
	assert.Equal(t, "hashed_secret_password", user.Password)
	assert.Equal(t, domain.RoleAdmin, user.Role)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)

	// Pengecekan khusus untuk field Soft Delete (gorm.DeletedAt)
	assert.True(t, user.DeletedAt.Valid, "DeletedAt seharusnya bernilai valid")
	assert.Equal(t, now, user.DeletedAt.Time)
}

func TestLoginResponse_Creation(t *testing.T) {
	// 1. Arrange
	now := time.Now()

	// 2. Act: Membuat objek LoginResponse yang berisi nested User
	loginRes := domain.LoginResponse{
		AccessToken:  "dummy_access_token_jwt",
		RefreshToken: "dummy_refresh_token_jwt",
		User: domain.User{
			ID:        2,
			Name:      "Tutor Budi",
			Email:     "budi@algonova.com",
			Role:      domain.RoleTutor,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// 3. Assert
	assert.Equal(t, "dummy_access_token_jwt", loginRes.AccessToken)
	assert.Equal(t, "dummy_refresh_token_jwt", loginRes.RefreshToken)

	// Verifikasi data User di dalam response
	assert.Equal(t, uint(2), loginRes.User.ID)
	assert.Equal(t, "Tutor Budi", loginRes.User.Name)
	assert.Equal(t, "budi@algonova.com", loginRes.User.Email)
	assert.Equal(t, domain.RoleTutor, loginRes.User.Role)
	assert.Equal(t, now, loginRes.User.CreatedAt)
	assert.Equal(t, now, loginRes.User.UpdatedAt)
}
