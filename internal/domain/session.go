// File: internal/domain/session.go
package domain

import (
	"context"
	"database/sql/driver"
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

// Value implements driver.Valuer — GORM/DB menggunakan ini saat INSERT/UPDATE.
func (d DateOnly) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time, nil
}

// Scan implements sql.Scanner — GORM menggunakan ini saat SELECT.
func (d *DateOnly) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		d.Time = v
		return nil
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
		d.Time = t
		return nil
	}
	return fmt.Errorf("DateOnly.Scan: cannot scan type %T", value)
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

// Value implements driver.Valuer — GORM/DB menggunakan ini saat INSERT/UPDATE.
func (t TimeOnly) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time, nil
}

// Scan implements sql.Scanner — GORM menggunakan ini saat SELECT.
func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		parsed, err := time.Parse("15:04:05", v)
		if err != nil {
			parsed, err = time.Parse("15:04", v)
		}
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	}
	return fmt.Errorf("TimeOnly.Scan: cannot scan type %T", value)
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
