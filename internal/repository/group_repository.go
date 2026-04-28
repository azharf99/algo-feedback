// File: internal/repository/group_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) domain.GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) Create(ctx context.Context, group *domain.Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *groupRepository) GetByID(ctx context.Context, id uint) (*domain.Group, error) {
	var group domain.Group
	// Preload "Course" dan "Students"
	err := r.db.WithContext(ctx).Preload("Course").Preload("Students").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetAll(ctx context.Context) ([]domain.Group, error) {
	var groups []domain.Group
	err := r.db.WithContext(ctx).Preload("Course").Preload("Students").Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Group, int64, error) {
	var groups []domain.Group
	var totalRows int64

	query := r.db.WithContext(ctx).Model(&domain.Group{})
	if params.Search != "" {
		query = query.Where("name ILIKE ?", "%"+params.Search+"%")
	}

	if err := query.Count(&totalRows).Error; err != nil {
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

	err := query.Preload("Course").Preload("Students").Scopes(pagination.Paginate(params)).Find(&groups).Error

	return groups, totalRows, err
}

func (r *groupRepository) Update(ctx context.Context, group *domain.Group) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *groupRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Group{}, id).Error
}

// Upsert (Logikanya tetap sama persis seperti sebelumnya karena masih mengatur relasi Students)
func (r *groupRepository) Upsert(ctx context.Context, group *domain.Group, studentIDs []uint) (bool, error) {
	var existing domain.Group
	var isCreated bool

	err := r.db.WithContext(ctx).First(&existing, group.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if errCreate := r.db.WithContext(ctx).Create(group).Error; errCreate != nil {
				return false, errCreate
			}
			isCreated = true
		} else {
			return false, err
		}
	} else {
		if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(group).Error; errUpdate != nil {
			return false, errUpdate
		}
		group.ID = existing.ID
	}

	if len(studentIDs) > 0 {
		var students []domain.Student
		r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)
		errAssoc := r.db.WithContext(ctx).Model(group).Association("Students").Replace(&students)
		if errAssoc != nil {
			return isCreated, errAssoc
		}
	} else {
		r.db.WithContext(ctx).Model(group).Association("Students").Clear()
	}

	return isCreated, nil
}
