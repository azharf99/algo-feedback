// File: internal/repository/feedback_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
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
	userID, _ := ctxutil.GetUserID(ctx)
	feedback.UserID = userID
	return r.db.WithContext(ctx).Create(feedback).Error
}

func (r *feedbackRepository) GetByID(ctx context.Context, id uint) (*domain.Feedback, error) {
	var feedback domain.Feedback
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Student").First(&feedback, id).Error
	return &feedback, err
}

func (r *feedbackRepository) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	var feedbacks []domain.Feedback
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Student").Find(&feedbacks).Error
	return feedbacks, err
}

// GetPaginated: Mengambil data feedback dengan pagination
func (r *feedbackRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feedback, int64, error) {
	var feedbacks []domain.Feedback
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Feedback{}).Scopes(scopeByUser(ctx))
	if params.Search != "" {
		query = query.Where("course ILIKE ? OR group_name ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Student").Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&feedbacks).Error
	if err != nil {
		return nil, 0, err
	}
	return feedbacks, total, nil
}

func (r *feedbackRepository) Update(ctx context.Context, feedback *domain.Feedback) error {
	userID, _ := ctxutil.GetUserID(ctx)
	feedback.UserID = userID
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Where("id = ?", feedback.ID).Updates(feedback).Error
}

func (r *feedbackRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Delete(&domain.Feedback{}, id).Error
}

// UpsertSeeder menggantikan update_or_create milik Django
func (r *feedbackRepository) UpsertSeeder(ctx context.Context, f *domain.Feedback) (bool, error) {
	var existing domain.Feedback

	userID, _ := ctxutil.GetUserID(ctx)
	f.UserID = userID

	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Where("student_id = ? AND number = ? AND course = ?", f.StudentID, f.Number, f.Course).
		Limit(1).Find(&existing).Error
	if err != nil {
		return false, err
	}

	if existing.ID == 0 {
		if errCreate := r.db.WithContext(ctx).Create(f).Error; errCreate != nil {
			return false, errCreate
		}
		return true, nil
	}

	f.ID = existing.ID
	if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(f).Error; errUpdate != nil {
		return false, errUpdate
	}
	return false, nil
}

// GetUnsentFeedbacks mengambil feedback yang belum terkirim (is_sent = false)
func (r *feedbackRepository) GetUnsentFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint) ([]domain.Feedback, error) {
	return r.GetFeedbacks(ctx, studentID, course, number, true)
}

// GetFeedbacks mengambil data feedback dengan filter fleksibel
func (r *feedbackRepository) GetFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint, onlyUnsent bool) ([]domain.Feedback, error) {
	query := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Student")

	if onlyUnsent {
		query = query.Where("is_sent = ?", false)
	}
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
