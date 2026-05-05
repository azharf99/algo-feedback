// File: internal/domain/group.go
package domain

import (
	"context"
	"io"
	"time"
)

// Group merepresentasikan kelas/kohort yang mengambil suatu Course
type Group struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	CourseID        uint      `json:"course_id"` // Relasi ke Course (Blueprint)
	Course          *Course   `json:"course,omitempty"`
	Name            string    `json:"name" gorm:"type:varchar(50);not null"`
	Description     *string   `json:"description" gorm:"type:text"`
	Type            string    `json:"type" gorm:"type:varchar(10);default:'Group'"`
	GroupPhone      *string   `json:"group_phone" gorm:"type:varchar(50)"`
	MeetingLink     *string   `json:"meeting_link" gorm:"type:text"`
	RecordingsLink  *string   `json:"recordings_link" gorm:"type:text"`
	FirstLessonDate *DateOnly `json:"first_lesson_date" gorm:"type:date"`
	FirstLessonTime *TimeOnly `json:"first_lesson_time" gorm:"type:time"`
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Students        []Student `json:"students" gorm:"many2many:group_students;"`
}

type GroupRepository interface {
	Create(ctx context.Context, group *Group, studentIDs []uint) error
	GetByID(ctx context.Context, id uint) (*Group, error)
	GetAll(ctx context.Context) ([]Group, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Group, int64, error)
	Update(ctx context.Context, group *Group, studentIDs []uint) error
	Delete(ctx context.Context, id uint) error
	Upsert(ctx context.Context, group *Group, studentIDs []uint) (bool, error)
}

type GroupUsecase interface {
	Create(ctx context.Context, group *Group, studentIDs []uint) error
	GetByID(ctx context.Context, id uint) (*Group, error)
	GetAll(ctx context.Context) ([]Group, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Group], error)
	Update(ctx context.Context, id uint, req *Group, studentIDs []uint) error
	Delete(ctx context.Context, id uint) error
	ImportCSV(ctx context.Context, fileReader io.Reader) (*ImportResult, error)
}
