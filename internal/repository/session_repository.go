// File: internal/repository/session_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
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
	userID, _ := ctxutil.GetUserID(ctx)
	session.UserID = userID
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) GetByID(ctx context.Context, id uint) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Preload("Group").Preload("Lesson").Preload("StudentsAttended").
		First(&session, id).Error
	return &session, err
}

func (r *sessionRepository) GetByGroup(ctx context.Context, groupID uint) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Where("group_id = ?", groupID).Preload("Lesson").Preload("StudentsAttended").
		Order("date_start ASC").Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetByLesson(ctx context.Context, lessonID uint) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Where("lesson_id = ?", lessonID).Preload("Group").Preload("Lesson").Preload("StudentsAttended").
		Order("date_start ASC").Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetAll(ctx context.Context) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Preload("Group").Preload("Lesson").Preload("StudentsAttended").
		Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Session, int64, error) {
	var sessions []domain.Session
	var totalRows int64
	query := r.db.WithContext(ctx).Model(&domain.Session{}).Scopes(scopeByUser(ctx))
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Group").Preload("Lesson").Preload("StudentsAttended").
		Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).Find(&sessions).Error
	return sessions, totalRows, err
}

func (r *sessionRepository) Update(ctx context.Context, session *domain.Session) error {
	userID, _ := ctxutil.GetUserID(ctx)
	session.UserID = userID
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Where("id = ?", session.ID).Updates(session).Error
}

func (r *sessionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Delete(&domain.Session{}, id).Error
}

func (r *sessionRepository) Upsert(ctx context.Context, session *domain.Session) (bool, error) {
	var existing domain.Session
	userID, _ := ctxutil.GetUserID(ctx)
	session.UserID = userID

	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).
		Where("group_id = ? AND lesson_id = ?", session.GroupID, session.LessonID).
		First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			err = r.db.WithContext(ctx).Create(session).Error
			return true, err
		}
		return false, err
	}
	if existing.IsDone {
		return false, nil
	}
	err = r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"date_start": session.DateStart,
		"time_start": session.TimeStart,
	}).Error
	return false, err
}

func (r *sessionRepository) UpsertAttendance(ctx context.Context, session *domain.Session, studentIDs []uint) error {
	var existing domain.Session
	if err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).First(&existing, session.ID).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&existing).Updates(session).Error; err != nil {
		return err
	}
	if len(studentIDs) > 0 {
		var students []domain.Student
		r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Where("id IN ?", studentIDs).Find(&students)
		errAssoc := r.db.WithContext(ctx).Model(&existing).Association("StudentsAttended").Replace(&students)
		if errAssoc != nil {
			return errAssoc
		}
	} else {
		r.db.WithContext(ctx).Model(&existing).Association("StudentsAttended").Clear()
	}
	return nil
}
