// File: internal/domain/user.go
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin Role = "Admin"
	RoleTutor Role = "Tutor"
	RoleSiswa Role = "Siswa"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Email     string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"` // Password disembunyikan dari JSON
	Role      Role           `gorm:"type:varchar(20);not null;default:'Siswa'" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Struktur untuk Response Login
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// Kontrak untuk User Repository
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
}

// Kontrak untuk Auth/User Usecase
type AuthUsecase interface {
	Register(ctx context.Context, req *User) error
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
}
