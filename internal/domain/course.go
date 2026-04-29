// File: internal/domain/course.go
package domain

import (
	"context"
	"io"
	"time"

	"gorm.io/gorm"
)

type Course struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Module      string         `gorm:"type:varchar(100);not null" json:"module"`
	Description *string        `gorm:"type:text" json:"description"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Lessons     []Lesson       `json:"lessons,omitempty"`
	Groups      []Group        `json:"groups,omitempty"`
}

type CourseRepository interface {
	Create(ctx context.Context, course *Course) error
	GetByID(ctx context.Context, id uint) (*Course, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Course, int64, error)
	GetAll(ctx context.Context) ([]Course, error)
	Update(ctx context.Context, course *Course) error
	Delete(ctx context.Context, id uint) error

	Upsert(ctx context.Context, course *Course) (bool, error)
}

type CourseUsecase interface {
	Create(ctx context.Context, course *Course) error
	GetByID(ctx context.Context, id uint) (*Course, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Course], error)
	GetAll(ctx context.Context) ([]Course, error)
	Update(ctx context.Context, id uint, req *Course) error
	Delete(ctx context.Context, id uint) error

	ImportCSV(ctx context.Context, fileReader io.Reader) (*ImportResult, error)
}
