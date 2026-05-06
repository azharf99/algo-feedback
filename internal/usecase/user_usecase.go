// File: internal/usecase/user_usecase.go
package usecase

import (
	"context"
	"errors"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/auth"
)

type userUsecase struct {
	userRepo domain.UserRepository
}

// NewUserUsecase membuat instance UserUsecase
func NewUserUsecase(userRepo domain.UserRepository) domain.UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

// GetAll mengembalikan semua user tanpa pagination
func (u *userUsecase) GetAll(ctx context.Context) ([]domain.User, error) {
	return u.userRepo.GetAll(ctx)
}

// GetPaginated mengembalikan data user dengan pagination
func (u *userUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.User], error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	users, total, err := u.userRepo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.Limit
	if int(total)%params.Limit != 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.User]{
		Data:       users,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetByID mengembalikan satu user berdasarkan ID
func (u *userUsecase) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}
	return user, nil
}

// Create membuat user baru dengan password yang sudah di-hash
func (u *userUsecase) Create(ctx context.Context, req *domain.UpdateUserRequest) (*domain.User, error) {
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("name, email, dan password wajib diisi")
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("gagal memproses password")
	}

	role := req.Role
	if role == "" {
		role = domain.RoleTutor // Default role untuk user baru yang dibuat admin
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password:         hashedPassword,
		Role:             role,
		WhatsappAPIKey:   req.WhatsappAPIKey,
		WhatsappDeviceID: req.WhatsappDeviceID,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, errors.New("gagal membuat pengguna, email mungkin sudah terdaftar")
	}
	return user, nil
}

// Update memperbarui data user. Password hanya diubah jika dikirim (tidak kosong).
func (u *userUsecase) Update(ctx context.Context, id uint, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	// Update field yang dikirim
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	// Password hanya diperbarui jika dikirim
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			return nil, errors.New("gagal memproses password baru")
		}
		user.Password = hashedPassword
	}

	if req.WhatsappAPIKey != "" {
		user.WhatsappAPIKey = req.WhatsappAPIKey
	}
	if req.WhatsappDeviceID != "" {
		user.WhatsappDeviceID = req.WhatsappDeviceID
	}

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, errors.New("gagal memperbarui data pengguna")
	}
	return user, nil
}

// UpdateProfile memperbarui data profil user sendiri (Nama, Password, WA Credentials)
func (u *userUsecase) UpdateProfile(ctx context.Context, id uint, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	// Update field yang diizinkan (Nama, Password, WA API, WA Device)
	if req.Name != "" {
		user.Name = req.Name
	}

	// Password hanya diperbarui jika dikirim
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			return nil, errors.New("gagal memproses password baru")
		}
		user.Password = hashedPassword
	}

	if req.WhatsappAPIKey != "" {
		user.WhatsappAPIKey = req.WhatsappAPIKey
	}
	if req.WhatsappDeviceID != "" {
		user.WhatsappDeviceID = req.WhatsappDeviceID
	}

	// Email dan Role sengaja tidak diupdate di sini untuk keamanan

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, errors.New("gagal memperbarui profil")
	}
	return user, nil
}

// Delete menghapus user secara soft-delete berdasarkan ID
func (u *userUsecase) Delete(ctx context.Context, id uint) error {
	if _, err := u.userRepo.GetByID(ctx, id); err != nil {
		return errors.New("pengguna tidak ditemukan")
	}
	return u.userRepo.Delete(ctx, id)
}
