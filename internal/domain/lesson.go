// File: internal/domain/lesson.go
package domain

import (
	"context"
	"io"
	"time"

	"gorm.io/gorm"
)

type Lesson struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CourseID    uint           `json:"course_id"`
	Course      *Course        `json:"course,omitempty"` // Relasi ke Course
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Category    *string        `gorm:"type:varchar(100)" json:"category"`
	Module      string         `gorm:"type:varchar(100)" json:"module"`
	Level       string         `gorm:"type:varchar(50)" json:"level"`
	Number      uint           `json:"number"`
	Description *string        `gorm:"type:text" json:"description"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type LessonRepository interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id uint) (*Lesson, error)
	GetAll(ctx context.Context) ([]Lesson, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Lesson, int64, error)
	Update(ctx context.Context, lesson *Lesson) error
	Delete(ctx context.Context, id uint) error

	Upsert(ctx context.Context, lesson *Lesson) (bool, error)
}

type LessonUsecase interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id uint) (*Lesson, error)
	GetAll(ctx context.Context) ([]Lesson, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Lesson], error)
	Update(ctx context.Context, id uint, req *Lesson) error
	Delete(ctx context.Context, id uint) error

	ImportCSV(ctx context.Context, fileReader io.Reader) (*ImportResult, error)
}
