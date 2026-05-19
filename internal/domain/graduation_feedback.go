// File: internal/domain/graduation_feedback.go
package domain

import (
	"context"
	"time"
)

type GraduationFeedback struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"not null;index"` // Tutor/Admin creator
	StudentID     uint      `json:"student_id" gorm:"not null;index"`
	Course        string    `json:"course" gorm:"type:varchar(100);not null"`
	Grade         string    `json:"grade" gorm:"type:varchar(10);not null"`
	TutorFeedback string    `json:"tutor_feedback" gorm:"type:text"`
	URLPDF        *string   `json:"url_pdf" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relasi
	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type GraduationFeedbackRepository interface {
	Create(ctx context.Context, gf *GraduationFeedback) error
	GetPaginated(ctx context.Context, params PaginationParams) ([]GraduationFeedback, int64, error)
	GetByID(ctx context.Context, id uint) (*GraduationFeedback, error)
	Update(ctx context.Context, gf *GraduationFeedback) error
	Delete(ctx context.Context, id uint) error
}
