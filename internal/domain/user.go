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

// UpdateUserRequest adalah payload untuk Create/Update User
type UpdateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"` // Kosong = tidak diubah
	Role     Role   `json:"role"`
}

// Kontrak untuk User Repository
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	GetAll(ctx context.Context) ([]User, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]User, int64, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}

// Kontrak untuk Auth Usecase (Login/Register)
type AuthUsecase interface {
	Register(ctx context.Context, req *User) error
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	GoogleLogin(ctx context.Context, email, name string) (*LoginResponse, error)
}

// Kontrak untuk User Management Usecase (CRUD)
type UserUsecase interface {
	GetAll(ctx context.Context) ([]User, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[User], error)
	GetByID(ctx context.Context, id uint) (*User, error)
	Create(ctx context.Context, req *UpdateUserRequest) (*User, error)
	Update(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id uint) error
}
