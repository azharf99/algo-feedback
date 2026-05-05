// File: internal/repository/course_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type courseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) domain.CourseRepository {
	return &courseRepository{db: db}
}

func (r *courseRepository) Create(ctx context.Context, course *domain.Course) error {
	return r.db.WithContext(ctx).Create(course).Error
}

func (r *courseRepository) GetByID(ctx context.Context, id uint) (*domain.Course, error) {
	var course domain.Course
	// Menarik detail Course sekalian dengan daftar Lesson dan Group-nya
	err := r.db.WithContext(ctx).Preload("Lessons").Preload("Groups").First(&course, id).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetAll(ctx context.Context) ([]domain.Course, error) {
	var courses []domain.Course
	err := r.db.WithContext(ctx).Preload("Lessons").Preload("Groups").Find(&courses).Error
	return courses, err
}

func (r *courseRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var totalRows int64

	query := r.db.WithContext(ctx).Model(&domain.Course{})

	// Fitur Pencarian berdasarkan Judul atau Modul
	if params.Search != "" {
		query = query.Where("title ILIKE ? OR module ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	if params.SortBy != "" {
		sortDir := "ASC" // Default arah sort
		if params.SortDir != "" {
			sortDir = strings.ToUpper(params.SortDir)
		}

		desc := sortDir == "DESC"
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: params.SortBy}, Desc: desc})
	} else {
		// Fallback default: urutkan dari data terbaru
		query = query.Order("id DESC")
	}

	// Ambil data dengan Pagination
	err := query.Preload("Lessons").Preload("Groups").Scopes(pagination.Paginate(params)).Find(&courses).Error

	return courses, totalRows, err
}

func (r *courseRepository) Update(ctx context.Context, course *domain.Course) error {
	return r.db.WithContext(ctx).Save(course).Error
}

func (r *courseRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Course{}, id).Error
}

func (r *courseRepository) Upsert(ctx context.Context, course *domain.Course) (bool, error) {
	var existing domain.Course
	var isCreated bool

	// Cek apakah Course sudah ada berdasarkan ID
	err := r.db.WithContext(ctx).First(&existing, course.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jika tidak ada, Create baru
			if errCreate := r.db.WithContext(ctx).Create(course).Error; errCreate != nil {
				return false, errCreate
			}
			isCreated = true
		} else {
			return false, err
		}
	} else {
		// Jika ada, Update data yang lama
		if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(course).Error; errUpdate != nil {
			return false, errUpdate
		}
		course.ID = existing.ID
	}

	return isCreated, nil
}
