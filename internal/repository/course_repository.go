// File: internal/repository/course_repository.go
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

type courseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) domain.CourseRepository {
	return &courseRepository{db: db}
}

func (r *courseRepository) Create(ctx context.Context, course *domain.Course) error {
	userID, _ := ctxutil.GetUserID(ctx)
	course.UserID = userID
	err := r.db.WithContext(ctx).Create(course).Error
	if err != nil {
		if strings.Contains(err.Error(), "SQLSTATE 23505") && strings.Contains(err.Error(), "courses_pkey") {
			r.db.Exec("SELECT setval(pg_get_serial_sequence('courses', 'id'), COALESCE((SELECT MAX(id) FROM courses), 1), true)")
			return r.db.WithContext(ctx).Create(course).Error
		}
	}
	return err
}

func (r *courseRepository) GetByID(ctx context.Context, id uint) (*domain.Course, error) {
	var course domain.Course
	// Menarik detail Course sekalian dengan daftar Lesson dan Group-nya
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Lessons").Preload("Groups").First(&course, id).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetAll(ctx context.Context) ([]domain.Course, error) {
	var courses []domain.Course
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("Lessons").Preload("Groups").Find(&courses).Error
	return courses, err
}

func (r *courseRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var totalRows int64

	query := r.db.WithContext(ctx).Model(&domain.Course{}).Scopes(scopeByUser(ctx), pagination.StatusFilter(params, "is_active"))

	// Fitur Pencarian berdasarkan Judul atau Modul
	if params.Search != "" {
		query = query.Where("title ILIKE ? OR module ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	// Ambil data dengan Pagination
	err := query.Preload("Lessons").Preload("Groups").Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&courses).Error

	return courses, totalRows, err
}

func (r *courseRepository) Update(ctx context.Context, course *domain.Course) error {
	userID, _ := ctxutil.GetUserID(ctx)
	course.UserID = userID
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Where("id = ?", course.ID).Select("*").Updates(course).Error
}

func (r *courseRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Delete(&domain.Course{}, id).Error
}

func (r *courseRepository) BulkDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Delete(&domain.Course{}, ids).Error
}

func (r *courseRepository) Upsert(ctx context.Context, course *domain.Course) (bool, error) {
	var existing domain.Course
	var isCreated bool

	userID, _ := ctxutil.GetUserID(ctx)
	course.UserID = userID

	// Cek apakah Course sudah ada berdasarkan ID
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).First(&existing, course.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jika tidak ada, Create baru
			if errCreate := r.db.WithContext(ctx).Create(course).Error; errCreate != nil {
				return false, errCreate
			}
			r.db.Exec("SELECT setval(pg_get_serial_sequence('courses', 'id'), COALESCE((SELECT MAX(id) FROM courses), 1), true)")
			isCreated = true
		} else {
			return false, err
		}
	} else {
		// Jika ada, Update data yang lama
		if errUpdate := r.db.WithContext(ctx).Model(&existing).Select("*").Updates(course).Error; errUpdate != nil {
			return false, errUpdate
		}
		course.ID = existing.ID
	}

	return isCreated, nil
}
