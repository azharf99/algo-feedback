// File: internal/domain/session.go
package domain

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DateOnly wraps time.Time and unmarshals JSON strings in "2006-01-02" format.
type DateOnly struct{ time.Time }

func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	// strip surrounding quotes
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("date_start: expected format YYYY-MM-DD, got %q", s)
	}
	d.Time = t
	return nil
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Time.Format("2006-01-02") + `"`), nil
}

// TimeOnly wraps time.Time and unmarshals JSON strings in "15:04" or "15:04:05" format.
type TimeOnly struct{ time.Time }

func (t *TimeOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	// strip surrounding quotes
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	var parsed time.Time
	var err error
	parsed, err = time.Parse("15:04", s)
	if err != nil {
		parsed, err = time.Parse("15:04:05", s)
	}
	if err != nil {
		return fmt.Errorf("time_start: expected format HH:MM or HH:MM:SS, got %q", s)
	}
	t.Time = parsed
	return nil
}

func (t TimeOnly) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format("15:04") + `"`), nil
}

type Session struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	GroupID          uint           `json:"group_id" gorm:"not null"`
	Group            *Group         `json:"group,omitempty"`
	LessonID         uint           `json:"lesson_id" gorm:"not null"`
	Lesson           *Lesson        `json:"lesson,omitempty"`
	DateStart        DateOnly       `json:"date_start" gorm:"type:date"`
	TimeStart        TimeOnly       `json:"time_start" gorm:"type:time"`
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
