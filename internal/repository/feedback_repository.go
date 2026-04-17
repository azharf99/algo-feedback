// File: internal/repository/feedback_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
	"gorm.io/gorm"
)

type feedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) domain.FeedbackRepository {
	return &feedbackRepository{db: db}
}

func (r *feedbackRepository) Create(ctx context.Context, feedback *domain.Feedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

func (r *feedbackRepository) GetByID(ctx context.Context, id uint) (*domain.Feedback, error) {
	var feedback domain.Feedback
	err := r.db.WithContext(ctx).Preload("Student").First(&feedback, id).Error
	return &feedback, err
}

func (r *feedbackRepository) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	var feedbacks []domain.Feedback
	err := r.db.WithContext(ctx).Preload("Student").Find(&feedbacks).Error
	return feedbacks, err
}

func (r *feedbackRepository) Update(ctx context.Context, feedback *domain.Feedback) error {
	return r.db.WithContext(ctx).Save(feedback).Error
}

func (r *feedbackRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Feedback{}, id).Error
}

// UpsertSeeder menggantikan update_or_create milik Django
func (r *feedbackRepository) UpsertSeeder(ctx context.Context, f *domain.Feedback) (bool, error) {
	var existing domain.Feedback

	// Mencari berdasarkan relasi StudentID, Number, dan Course (Kunci unik di seeder Python-mu)
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND number = ? AND course = ?", f.StudentID, f.Number, f.Course).
		First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat Baru
			if errCreate := r.db.WithContext(ctx).Create(f).Error; errCreate != nil {
				return false, errCreate
			}
			return true, nil // True = Created
		}
		return false, err
	}

	// Jika ada, Update data yang lama dengan data baru
	f.ID = existing.ID
	if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(f).Error; errUpdate != nil {
		return false, errUpdate
	}
	return false, nil // False = Updated
}

// GetUnsentFeedbacks mengambil feedback yang belum terkirim (is_sent = false)
// Digunakan di Generator PDF dan Pengirim WA
func (r *feedbackRepository) GetUnsentFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint) ([]domain.Feedback, error) {
	query := r.db.WithContext(ctx).Preload("Student").Where("is_sent = ?", false)

	// Filter dinamis mirip di generator_pdf.py Python-mu
	if studentID != nil {
		query = query.Where("student_id = ?", *studentID)
	}
	if course != nil {
		query = query.Where("course = ?", *course)
	}
	if number != nil {
		query = query.Where("number = ?", *number)
	}

	var feedbacks []domain.Feedback
	err := query.Find(&feedbacks).Error
	return feedbacks, err
}
