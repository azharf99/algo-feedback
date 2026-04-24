// File: internal/repository/student_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

// studentRepository adalah implementasi nyata dari domain.StudentRepository
type studentRepository struct {
	db *gorm.DB
}

// NewStudentRepository membuat instance/objek baru dari repository
func NewStudentRepository(db *gorm.DB) domain.StudentRepository {
	return &studentRepository{
		db: db,
	}
}

// Create: Menyimpan data siswa baru ke database
func (r *studentRepository) Create(ctx context.Context, student *domain.Student) error {
	return r.db.WithContext(ctx).Create(student).Error
}

// GetByID: Mencari siswa berdasarkan ID
func (r *studentRepository) GetByID(ctx context.Context, id uint) (*domain.Student, error) {
	var student domain.Student
	err := r.db.WithContext(ctx).First(&student, id).Error
	if err != nil {
		return nil, err
	}
	return &student, nil
}

// GetAll: Mengambil semua data siswa
func (r *studentRepository) GetAll(ctx context.Context) ([]domain.Student, error) {
	var students []domain.Student
	err := r.db.WithContext(ctx).Find(&students).Error
	if err != nil {
		return nil, err
	}
	return students, nil
}

// GetPaginated: Mengambil data siswa dengan pagination
func (r *studentRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Student, int64, error) {
	var students []domain.Student
	var total int64

	r.db.WithContext(ctx).Model(&domain.Student{}).Count(&total)
	err := r.db.WithContext(ctx).Scopes(pagination.Paginate(params)).Find(&students).Error
	if err != nil {
		return nil, 0, err
	}
	return students, total, nil
}

// Update: Memperbarui data siswa yang sudah ada
func (r *studentRepository) Update(ctx context.Context, student *domain.Student) error {
	return r.db.WithContext(ctx).Save(student).Error
}

// Delete: Menghapus data siswa (Bisa Hard Delete atau Soft Delete)
func (r *studentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Student{}, id).Error
}

// Upsert: Memperbarui jika ada, membuat baru jika tidak ada (mirip update_or_create di Django)
// Mengembalikan boolean (true jika data baru dibuat, false jika data lama diperbarui)
func (r *studentRepository) Upsert(ctx context.Context, student *domain.Student) (bool, error) {
	var existing domain.Student

	// Cek apakah data dengan ID tersebut sudah ada
	err := r.db.WithContext(ctx).First(&existing, student.ID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jika tidak ada, buat data baru
			if errCreate := r.db.WithContext(ctx).Create(student).Error; errCreate != nil {
				return false, errCreate
			}
			return true, nil // True = Created
		}
		// Error lain saat query database
		return false, err
	}

	// Jika ada, perbarui data (Update)
	if errUpdate := r.db.WithContext(ctx).Model(&existing).Updates(student).Error; errUpdate != nil {
		return false, errUpdate
	}

	return false, nil // False = Updated
}
