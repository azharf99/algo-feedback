// File: internal/repository/user_repository.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/internal/domain"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByResetToken(ctx context.Context, token string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("reset_password_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	err := r.db.WithContext(ctx).Order("id asc").Find(&users).Error
	return users, err
}

func (r *userRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.User{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.Limit
	err := query.Order("id asc").Offset(offset).Limit(params.Limit).Find(&users).Error
	return users, total, err
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}
