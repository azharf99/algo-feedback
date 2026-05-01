// File: internal/usecase/session_usecase.go
package usecase

import (
	"context"
	"errors"
	"math"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type sessionUsecase struct {
	repo domain.SessionRepository
}

func NewSessionUsecase(repo domain.SessionRepository) domain.SessionUsecase {
	return &sessionUsecase{repo: repo}
}

func (u *sessionUsecase) Create(ctx context.Context, session *domain.Session) error {
	return u.repo.Create(ctx, session)
}

func (u *sessionUsecase) GetByID(ctx context.Context, id uint) (*domain.Session, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *sessionUsecase) GetByGroup(ctx context.Context, groupID uint) ([]domain.Session, error) {
	return u.repo.GetByGroup(ctx, groupID)
}

func (u *sessionUsecase) GetAll(ctx context.Context) ([]domain.Session, error) {
	return u.repo.GetAll(ctx)
}

func (u *sessionUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Session], error) {
	params = pagination.Normalize(params)
	sessions, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &domain.PaginatedResult[domain.Session]{
		Data:       sessions,
		Total:      total,
		TotalPages: totalPages,
		Page:       params.Page,
		Limit:      params.Limit,
	}, nil
}

func (u *sessionUsecase) Update(ctx context.Context, id uint, req *domain.Session) error {
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("sesi tidak ditemukan")
	}

	if req.GroupID != 0 {
		existing.GroupID = req.GroupID
	}
	if req.LessonID != 0 {
		existing.LessonID = req.LessonID
	}
	if !req.DateStart.Time.IsZero() {
		existing.DateStart = req.DateStart
	}
	if !req.TimeStart.Time.IsZero() {
		existing.TimeStart = req.TimeStart
	}
	
	// Untuk boolean, kita asumsikan jika dikirim dalam JSON akan ter-bind.
	// Namun Gin ShouldBindJSON akan selalu set false jika tidak ada.
	// Untuk keamanan, kita hanya update jika ada perubahan nilai dari existing.
	existing.IsDone = req.IsDone

	return u.repo.Update(ctx, existing)
}

func (u *sessionUsecase) Delete(ctx context.Context, id uint) error {
	return u.repo.Delete(ctx, id)
}

func (u *sessionUsecase) UpdateAttendance(ctx context.Context, sessionID uint, studentIDs []uint) error {
	// Menyiapkan struct Session dengan IsDone otomatis True saat absen dikirim
	session := &domain.Session{
		ID:     sessionID,
		IsDone: true,
	}

	// Lempar ke repository untuk melakukan update dasar dan mereplace relasi Many-to-Many
	return u.repo.UpsertAttendance(ctx, session, studentIDs)
}
