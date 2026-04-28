// File: internal/repository/session_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) domain.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) GetByID(ctx context.Context, id uint) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).
		Preload("Group").
		Preload("Lesson").
		Preload("StudentsAttended").
		First(&session, id).Error
	return &session, err
}

func (r *sessionRepository) GetByGroup(ctx context.Context, groupID uint) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Preload("Lesson").
		Preload("StudentsAttended").
		Order("date_start ASC").
		Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetAll(ctx context.Context) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.WithContext(ctx).
		Preload("Group").
		Preload("Lesson").
		Preload("StudentsAttended").
		Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Session, int64, error) {
	var sessions []domain.Session
	var totalRows int64

	query := r.db.WithContext(ctx).Model(&domain.Session{})

	// Hitung total baris
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	// Eksekusi pencarian dengan Pagination dan Preload lengkap
	err := query.
		Preload("Group").
		Preload("Lesson").
		Preload("StudentsAttended").
		Scopes(pagination.Paginate(params)).
		Order(params.SortBy + " " + params.SortDir).
		Find(&sessions).Error

	return sessions, totalRows, err
}

func (r *sessionRepository) Update(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *sessionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Session{}, id).Error
}

func (r *sessionRepository) UpsertAttendance(ctx context.Context, session *domain.Session, studentIDs []uint) error {
	var existing domain.Session

	// Pastikan sesi dengan ID tersebut ada di database
	if err := r.db.WithContext(ctx).First(&existing, session.ID).Error; err != nil {
		return err
	}

	// Update atribut dasar sesi (Misal: jika ada perubahan is_done, waktu, dsb)
	if err := r.db.WithContext(ctx).Model(&existing).Updates(session).Error; err != nil {
		return err
	}

	// Tangani relasi Many-to-Many Siswa yang Hadir
	if len(studentIDs) > 0 {
		var students []domain.Student
		r.db.WithContext(ctx).Where("id IN ?", studentIDs).Find(&students)

		// Ganti (Replace) data lama dengan data kehadiran yang baru
		errAssoc := r.db.WithContext(ctx).Model(&existing).Association("StudentsAttended").Replace(&students)
		if errAssoc != nil {
			return errAssoc
		}
	} else {
		// Kosongkan tabel relasi jika tidak ada siswa yang hadir
		r.db.WithContext(ctx).Model(&existing).Association("StudentsAttended").Clear()
	}

	return nil
}
