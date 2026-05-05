// File: internal/repository/lesson_repository.go
package repository

import (
	"context"
	"errors"


	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

type lessonRepository struct {
	db *gorm.DB
}

func NewLessonRepository(db *gorm.DB) domain.LessonRepository {
	return &lessonRepository{db: db}
}

func (r *lessonRepository) Create(ctx context.Context, lesson *domain.Lesson) error {
	return r.db.WithContext(ctx).Create(lesson).Error
}

func (r *lessonRepository) GetByID(ctx context.Context, id uint) (*domain.Lesson, error) {
	var lesson domain.Lesson
	// Preload diganti ke "Course" karena Lesson sekarang bergantung pada Course
	err := r.db.WithContext(ctx).Preload("Course").First(&lesson, id).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (r *lessonRepository) GetAll(ctx context.Context) ([]domain.Lesson, error) {
	var lessons []domain.Lesson
	err := r.db.WithContext(ctx).Preload("Course").Find(&lessons).Error
	return lessons, err
}

func (r *lessonRepository) GetByCourse(ctx context.Context, courseID uint) ([]domain.Lesson, error) {
	var lessons []domain.Lesson
	err := r.db.WithContext(ctx).Preload("Course").Where("course_id = ?", courseID).Order("number ASC").Find(&lessons).Error
	return lessons, err
}

func (r *lessonRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Lesson, int64, error) {
	var lessons []domain.Lesson
	var totalRows int64

	query := r.db.WithContext(ctx).Model(&domain.Lesson{})
	if params.Search != "" {
		query = query.Where("title ILIKE ? OR module ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Course").Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&lessons).Error

	return lessons, totalRows, err
}

func (r *lessonRepository) Update(ctx context.Context, lesson *domain.Lesson) error {
	return r.db.WithContext(ctx).Save(lesson).Error
}

func (r *lessonRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Lesson{}, id).Error
}

// Upsert HANYA memperbarui atau membuat record Lesson
func (r *lessonRepository) Upsert(ctx context.Context, lesson *domain.Lesson) (bool, error) {
	var existing domain.Lesson
	var isCreated bool

	err := r.db.WithContext(ctx).First(&existing, lesson.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat Baru
			if errCreate := r.db.WithContext(ctx).Create(lesson).Error; errCreate != nil {
				return false, errCreate
			}
			isCreated = true
		} else {
			return false, err
		}
	} else {
		// Perbarui Data yang ada
		if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(lesson).Error; errUpdate != nil {
			return false, errUpdate
		}
		lesson.ID = existing.ID
	}

	return isCreated, nil
}
