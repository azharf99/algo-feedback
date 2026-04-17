// File: internal/domain/lesson.go
package domain

import (
	"context"
	"io"
	"time"
)

// Lesson merepresentasikan tabel lessons di database.
type Lesson struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"type:varchar(100);not null"`
	Category    *string   `json:"category" gorm:"type:varchar(100)"`
	Module      string    `json:"module" gorm:"type:varchar(50);not null"`
	Level       string    `json:"level" gorm:"type:varchar(50);not null"`
	Number      uint      `json:"number" gorm:"not null"`
	Description *string   `json:"description" gorm:"type:text"`
	DateStart   time.Time `json:"date_start" gorm:"type:date"`
	TimeStart   time.Time `json:"time_start" gorm:"type:time"`
	MeetingLink *string   `json:"meeting_link" gorm:"type:text"`
	Feedback    *string   `json:"feedback" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relasi Foreign Key (One-to-Many) ke Group
	// OnDelete:CASCADE memastikan jika Group dihapus, Lesson ini juga ikut terhapus
	GroupID uint   `json:"group_id" gorm:"not null"`
	Group   *Group `json:"group,omitempty" gorm:"foreignKey:GroupID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Relasi Many-to-Many ke Student (Siswa yang hadir)
	StudentsAttended []Student `json:"students_attended" gorm:"many2many:lesson_students_attended;"`
}

// LessonRepository mendefinisikan operasi ke database
type LessonRepository interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id uint) (*Lesson, error)
	GetAll(ctx context.Context) ([]Lesson, error)
	Update(ctx context.Context, lesson *Lesson) error
	Delete(ctx context.Context, id uint) error

	// Upsert untuk keperluan Import CSV (menyimpan relasi Many-to-Many Siswa)
	Upsert(ctx context.Context, lesson *Lesson, studentIDs []uint) (bool, error)
}

// LessonUsecase mendefinisikan logika bisnis
type LessonUsecase interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id uint) (*Lesson, error)
	GetAll(ctx context.Context) ([]Lesson, error)
	Update(ctx context.Context, id uint, req *Lesson) error
	Delete(ctx context.Context, id uint) error

	// Kita pinjam struct ImportResult yang ada di package usecase
	ImportCSV(ctx context.Context, fileReader io.Reader) (interface{}, error)
}
