// File: internal/repository/group_repository.go
package repository

import (
	"context"
	"errors"

	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) domain.GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) Create(ctx context.Context, group *domain.Group, studentIDs []uint) error {
	err := r.db.WithContext(ctx).Omit("Students").Create(group).Error
	if err != nil {
		return err
	}
	if studentIDs != nil {
		var students []domain.Student
		if len(studentIDs) > 0 {
			r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)
		}
		return r.db.WithContext(ctx).Model(group).Association("Students").Replace(&students)
	}
	return nil
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

	err := query.Preload("Course").Preload("Students").Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&groups).Error

	return groups, totalRows, err
}

func (r *groupRepository) Update(ctx context.Context, group *domain.Group, studentIDs []uint) error {
	// Gunakan Updates dengan map agar field bernilai zero (seperti is_active: false) tetap terupdate,
	// dan Omit("CreatedAt") agar timestamp pembuatannya tidak tertimpa nilai zero.
	updateData := map[string]interface{}{
		"course_id":         group.CourseID,
		"name":              group.Name,
		"description":       group.Description,
		"type":              group.Type,
		"group_phone":       group.GroupPhone,
		"meeting_link":      group.MeetingLink,
		"recordings_link":   group.RecordingsLink,
		"first_lesson_date": group.FirstLessonDate,
		"first_lesson_time": group.FirstLessonTime,
		"is_active":         group.IsActive,
	}

	err := r.db.WithContext(ctx).Model(group).Omit("Students").Updates(updateData).Error
	if err != nil {
		return err
	}
	if studentIDs != nil {
		var students []domain.Student
		if len(studentIDs) > 0 {
			r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)
		}
		return r.db.WithContext(ctx).Model(group).Association("Students").Replace(&students)
	}
	return nil
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
