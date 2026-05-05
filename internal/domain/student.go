// File: internal/domain/student.go
package domain

import (
	"context"
	"io"
	"time"
)

// Student merepresentasikan tabel students di database.
type Student struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"not null;index"`
	Fullname      string    `json:"fullname" gorm:"type:varchar(150);not null"`
	Surname       string    `json:"surname" gorm:"type:varchar(50);not null"`
	Username      string    `json:"username" gorm:"type:varchar(50);unique;not null"`
	Password      string    `json:"-" gorm:"type:varchar(128);not null"`  // json:"-" agar password tidak ikut terkirim saat response API
	PhoneNumber   *string   `json:"phone_number" gorm:"type:varchar(15)"` // Pointer (*) digunakan agar bisa bernilai NULL di database
	ParentName    *string   `json:"parent_name" gorm:"type:varchar(60)"`
	ParentContact *string   `json:"parent_contact" gorm:"type:varchar(15)"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"` // Pengganti auto_now_add=True
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"` // Pengganti auto_now=True
}

// UpdateStudentRequest adalah payload untuk Create/Update Siswa
type UpdateStudentRequest struct {
	Fullname      string  `json:"fullname"`
	Surname       string  `json:"surname"`
	Username      string  `json:"username"`
	Password      string  `json:"password"` // Opsional
	PhoneNumber   *string `json:"phone_number"`
	ParentName    *string `json:"parent_name"`
	ParentContact *string `json:"parent_contact"`
	IsActive      bool    `json:"is_active"`
}


// Tambahkan baris ini di dalam interface StudentRepository di internal/domain/student.go
type StudentRepository interface {
	Create(ctx context.Context, student *Student) error
	GetByID(ctx context.Context, id uint) (*Student, error)
	GetAll(ctx context.Context) ([]Student, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Student, int64, error)
	Update(ctx context.Context, student *Student) error
	Delete(ctx context.Context, id uint) error

	// Fitur baru untuk keperluan Import CSV
	Upsert(ctx context.Context, student *Student) (bool, error)
}

// ImportResult menyimpan hasil dari proses import CSV
type ImportResult struct {
	Created int
	Updated int
	Errors  []map[string]interface{}
}

// StudentUsecase mendefinisikan kontrak logika bisnis untuk Student
type StudentUsecase interface {
	Create(ctx context.Context, req *UpdateStudentRequest) error
	GetByID(ctx context.Context, id uint) (*Student, error)
	GetAll(ctx context.Context) ([]Student, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Student], error)
	Update(ctx context.Context, id uint, req *UpdateStudentRequest) error
	Delete(ctx context.Context, id uint) error
	ImportCSV(ctx context.Context, fileReader io.Reader) (*ImportResult, error)
}
