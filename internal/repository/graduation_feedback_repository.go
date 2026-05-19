// File: internal/repository/graduation_feedback_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

type graduationFeedbackRepository struct {
	db *gorm.DB
}

func NewGraduationFeedbackRepository(db *gorm.DB) domain.GraduationFeedbackRepository {
	return &graduationFeedbackRepository{db: db}
}

func (r *graduationFeedbackRepository) Create(ctx context.Context, gf *domain.GraduationFeedback) error {
	userID, _ := ctxutil.GetUserID(ctx)
	gf.UserID = userID
	return r.db.WithContext(ctx).Create(gf).Error
}

func (r *graduationFeedbackRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.GraduationFeedback, int64, error) {
	var gfs []domain.GraduationFeedback
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.GraduationFeedback{}).Scopes(scopeByUser(ctx))
	if params.Search != "" {
		query = query.Where("course ILIKE ?", "%"+params.Search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Student").Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&gfs).Error
	if err != nil {
		return nil, 0, err
	}
	return gfs, total, nil
}

func (r *graduationFeedbackRepository) GetByID(ctx context.Context, id uint) (*domain.GraduationFeedback, error) {
	var gf domain.GraduationFeedback
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Student").First(&gf, id).Error
	return &gf, err
}

func (r *graduationFeedbackRepository) Update(ctx context.Context, gf *domain.GraduationFeedback) error {
	userID, _ := ctxutil.GetUserID(ctx)
	gf.UserID = userID
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Model(gf).Updates(gf).Error
}

func (r *graduationFeedbackRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Delete(&domain.GraduationFeedback{}, id).Error
}
