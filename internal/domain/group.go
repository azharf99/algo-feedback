// File: internal/domain/group.go
package domain

import (
	"context"
	"io"
	"time"
)

// Group merepresentasikan tabel groups di database.
type Group struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"type:varchar(50);not null"`
	Description     *string    `json:"description" gorm:"type:text"`
	Type            string     `json:"type" gorm:"type:varchar(10);default:'Group'"` // Pengganti choices GROUP_TYPES
	GroupPhone      *string    `json:"group_phone" gorm:"type:varchar(50)"`
	MeetingLink     *string    `json:"meeting_link" gorm:"type:text"` // Di Golang, URLField cukup disimpan sebagai text/varchar
	RecordingsLink  *string    `json:"recordings_link" gorm:"type:text"`
	FirstLessonDate *time.Time `json:"first_lesson_date" gorm:"type:date"` // Hanya menyimpan tanggal
	FirstLessonTime *time.Time `json:"first_lesson_time" gorm:"type:time"` // Hanya menyimpan waktu
	IsActive        bool       `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	Students        []Student  `json:"students" gorm:"many2many:group_students;"`
	// Relasi Many-to-Many dengan model Student
	// GORM otomatis akan membuat tabel perantara bernama 'group_students'
}

type GroupRepository interface {
	Create(ctx context.Context, group *Group) error
	GetByID(ctx context.Context, id uint) (*Group, error)
	GetAll(ctx context.Context) ([]Group, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Group, int64, error)
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id uint) error
	// Upsert dengan tambahan array ID siswa untuk Many-to-Many
	Upsert(ctx context.Context, group *Group, studentIDs []uint) (bool, error)
}

type GroupUsecase interface {
	Create(ctx context.Context, group *Group) error
	GetByID(ctx context.Context, id uint) (*Group, error)
	GetAll(ctx context.Context) ([]Group, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Group], error)
	Update(ctx context.Context, id uint, req *Group) error
	Delete(ctx context.Context, id uint) error
	// Kita akan menggunakan kembali struct ImportResult yang sudah ada di package usecase
	ImportCSV(ctx context.Context, fileReader io.Reader) (*ImportResult, error)
}
