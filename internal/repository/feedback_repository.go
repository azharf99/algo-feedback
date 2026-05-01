// File: internal/repository/feedback_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
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

// GetPaginated: Mengambil data feedback dengan pagination
func (r *feedbackRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feedback, int64, error) {
	var feedbacks []domain.Feedback
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Feedback{})
	if params.Search != "" {
		query = query.Where("course ILIKE ? OR group_name ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if params.SortBy != "" {
		sortDir := "ASC" // Default arah sort
		if params.SortDir != "" {
			sortDir = params.SortDir
		}
		query = query.Order(params.SortBy + " " + sortDir)
	} else {
		// Fallback default: urutkan dari data terbaru
		query = query.Order("id DESC")
	}
	err := query.Preload("Student").Scopes(pagination.Paginate(params)).Find(&feedbacks).Error
	if err != nil {
		return nil, 0, err
	}
	return feedbacks, total, nil
}

func (r *feedbackRepository) Update(ctx context.Context, feedback *domain.Feedback) error {
	// Gunakan Updates(struct) agar GORM hanya memperbarui field yang tidak kosong (non-zero).
	// Ini mencegah field lain seperti student_id tertimpa menjadi null jika tidak dikirim.
	return r.db.WithContext(ctx).Model(feedback).Updates(feedback).Error
}

func (r *feedbackRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Feedback{}, id).Error
}

// UpsertSeeder menggantikan update_or_create milik Django
func (r *feedbackRepository) UpsertSeeder(ctx context.Context, f *domain.Feedback) (bool, error) {
	var existing domain.Feedback

	// Menggunakan Find().Limit(1) alih-alih First() untuk menghindari log "record not found" yang berisik
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND number = ? AND course = ?", f.StudentID, f.Number, f.Course).
		Limit(1).
		Find(&existing).Error

	if err != nil {
		return false, err
	}

	// Jika ID masih 0, berarti data tidak ditemukan -> Buat Baru
	if existing.ID == 0 {
		if errCreate := r.db.WithContext(ctx).Create(f).Error; errCreate != nil {
			return false, errCreate
		}
		return true, nil // True = Created
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
