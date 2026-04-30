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
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("date: expected format YYYY-MM-DD, got %q", s)
	}
	d.Time = t
	return nil
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Time.Format("2006-01-02") + `"`), nil
}

func (d DateOnly) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time.Format("2006-01-02"), nil
}

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
	case []byte:
		t, err := time.Parse("2006-01-02", string(v))
		if err != nil {
			return err
		}
		d.Time = t
		return nil
	}
	return fmt.Errorf("DateOnly.Scan: cannot scan type %T", value)
}

// TimeOnly wraps time.Time and unmarshals JSON strings in "15:04" or "15:04:05" format.
// Menggunakan year 2000 secara internal untuk menghindari bug historical timezone offset (seperti +07:07 di Jakarta tahun 0001).
type TimeOnly struct{ time.Time }

func (t *TimeOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	var parsed time.Time
	var err error
	// Gunakan UTC agar timezone-agnostic
	parsed, err = time.Parse("15:04", s)
	if err != nil {
		parsed, err = time.Parse("15:04:05", s)
	}
	if err != nil {
		return fmt.Errorf("time: expected format HH:MM or HH:MM:SS, got %q", s)
	}
	// Normalisasi ke year 2000 UTC
	t.Time = time.Date(2000, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC)
	return nil
}

func (t TimeOnly) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.Time.Format("15:04") + `"`), nil
}

func (t TimeOnly) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	// Kirim sebagai string ke DB untuk menghindari konversi timezone oleh driver
	return t.Time.Format("15:04:05"), nil
}

func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	var hour, min, sec int
	switch v := value.(type) {
	case time.Time:
		hour, min, sec = v.Hour(), v.Minute(), v.Second()
	case string:
		parsed, err := time.Parse("15:04:05", v)
		if err != nil {
			parsed, err = time.Parse("15:04", v)
		}
		if err != nil {
			return err
		}
		hour, min, sec = parsed.Hour(), parsed.Minute(), parsed.Second()
	case []byte:
		s := string(v)
		parsed, err := time.Parse("15:04:05", s)
		if err != nil {
			parsed, err = time.Parse("15:04", s)
		}
		if err != nil {
			return err
		}
		hour, min, sec = parsed.Hour(), parsed.Minute(), parsed.Second()
	default:
		return fmt.Errorf("TimeOnly.Scan: cannot scan type %T", value)
	}
	// Selalu simpan sebagai year 2000 UTC agar konsisten
	t.Time = time.Date(2000, 1, 1, hour, min, sec, 0, time.UTC)
	return nil
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
