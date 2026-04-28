// File: internal/domain/session.go
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Session struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	GroupID          uint           `json:"group_id" gorm:"not null"`
	Group            *Group         `json:"group,omitempty"`
	LessonID         uint           `json:"lesson_id" gorm:"not null"`
	Lesson           *Lesson        `json:"lesson,omitempty"`
	DateStart        time.Time      `json:"date_start" gorm:"type:date"`
	TimeStart        time.Time      `json:"time_start" gorm:"type:time"`
	IsDone           bool           `json:"is_done" gorm:"default:false"`
	StudentsAttended []Student      `json:"students_attended" gorm:"many2many:session_students;"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id uint) (*Session, error)
	GetByGroup(ctx context.Context, groupID uint) ([]Session, error)
	GetAll(ctx context.Context) ([]Session, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Session, int64, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id uint) error

	UpsertAttendance(ctx context.Context, session *Session, studentIDs []uint) error
}

type SessionUsecase interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id uint) (*Session, error)
	GetByGroup(ctx context.Context, groupID uint) ([]Session, error)
	GetAll(ctx context.Context) ([]Session, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Session], error)
	Update(ctx context.Context, id uint, req *Session) error
	Delete(ctx context.Context, id uint) error

	UpdateAttendance(ctx context.Context, sessionID uint, studentIDs []uint) error
}
