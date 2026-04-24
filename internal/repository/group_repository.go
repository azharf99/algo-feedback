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
	// Gunakan Preload("Students") agar data siswa di dalam grup ikut terpanggil!
	err := r.db.WithContext(ctx).Preload("Students").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetAll(ctx context.Context) ([]domain.Group, error) {
	var groups []domain.Group
	err := r.db.WithContext(ctx).Preload("Students").Find(&groups).Error
	return groups, err
}

// GetPaginated: Mengambil data grup dengan pagination
func (r *groupRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Group, int64, error) {
	var groups []domain.Group
	var total int64

	r.db.WithContext(ctx).Model(&domain.Group{}).Count(&total)
	err := r.db.WithContext(ctx).Preload("Students").Scopes(pagination.Paginate(params)).Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

func (r *groupRepository) Update(ctx context.Context, group *domain.Group) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *groupRepository) Delete(ctx context.Context, id uint) error {
	// GORM otomatis akan menghapus data di tabel pivot group_students (CASCADE)
	return r.db.WithContext(ctx).Delete(&domain.Group{}, id).Error
}

// Upsert menangani pembuatan/pembaruan Grup SEKALIGUS mengisi relasi Siswa
func (r *groupRepository) Upsert(ctx context.Context, group *domain.Group, studentIDs []uint) (bool, error) {
	var existing domain.Group
	var isCreated bool

	// 1. Cek apakah Grup sudah ada
	err := r.db.WithContext(ctx).First(&existing, group.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat Baru
			if errCreate := r.db.WithContext(ctx).Create(group).Error; errCreate != nil {
				return false, errCreate
			}
			isCreated = true
		} else {
			return false, err
		}
	} else {
		// Perbarui Data yang ada
		if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(group).Error; errUpdate != nil {
			return false, errUpdate
		}
		group.ID = existing.ID // Pastikan ID grup terbaru digunakan untuk relasi
	}

	// 2. Tangani Relasi Many-to-Many (Siswa)
	if len(studentIDs) > 0 {
		var students []domain.Student
		r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)

		// .Replace() akan menghapus relasi lama dan memasukkan relasi baru yang sesuai
		errAssoc := r.db.WithContext(ctx).Model(group).Association("Students").Replace(&students)
		if errAssoc != nil {
			return isCreated, errAssoc
		}
	} else {
		// Jika CSV kosong, bersihkan semua relasi siswa di grup ini
		r.db.WithContext(ctx).Model(group).Association("Students").Clear()
	}

	return isCreated, nil
}
