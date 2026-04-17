// File: internal/repository/lesson_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
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
	// Mengambil data Lesson beserta relasi Group dan StudentsAttended
	err := r.db.WithContext(ctx).
		Preload("Group").
		Preload("StudentsAttended").
		First(&lesson, id).Error

	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (r *lessonRepository) GetAll(ctx context.Context) ([]domain.Lesson, error) {
	var lessons []domain.Lesson
	err := r.db.WithContext(ctx).
		Preload("Group").
		Preload("StudentsAttended").
		Find(&lessons).Error
	return lessons, err
}

func (r *lessonRepository) Update(ctx context.Context, lesson *domain.Lesson) error {
	return r.db.WithContext(ctx).Save(lesson).Error
}

func (r *lessonRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Lesson{}, id).Error
}

// Upsert menangani pembuatan/pembaruan Lesson dan relasi kehadirannya
func (r *lessonRepository) Upsert(ctx context.Context, lesson *domain.Lesson, studentIDs []uint) (bool, error) {
	var existing domain.Lesson
	var isCreated bool

	// 1. Cek keberadaan Lesson
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
		lesson.ID = existing.ID // Sinkronisasi ID untuk relasi di bawah
	}

	// 2. Tangani Relasi Many-to-Many (Kehadiran Siswa)
	if len(studentIDs) > 0 {
		var students []domain.Student
		r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)

		// Perhatikan nama association di sini harus sama persis dengan nama field di Struct Lesson
		errAssoc := r.db.WithContext(ctx).Model(lesson).Association("StudentsAttended").Replace(&students)
		if errAssoc != nil {
			return isCreated, errAssoc
		}
	} else {
		// Kosongkan kehadiran jika tidak ada ID siswa yang dikirim
		r.db.WithContext(ctx).Model(lesson).Association("StudentsAttended").Clear()
	}

	return isCreated, nil
}
