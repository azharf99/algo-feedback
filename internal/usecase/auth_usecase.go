// File: internal/usecase/auth_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/golang-jwt/jwt/v5"
)

type authUsecase struct {
	userRepo domain.UserRepository
}

func NewAuthUsecase(userRepo domain.UserRepository) domain.AuthUsecase {
	return &authUsecase{userRepo: userRepo}
}

func (u *authUsecase) Register(ctx context.Context, req *domain.User) error {
	// Hash password sebelum disimpan
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return err
	}
	req.Password = hashedPassword
	return u.userRepo.Create(ctx, req)
}

func (u *authUsecase) Login(ctx context.Context, email, password string) (*domain.LoginResponse, error) {
	// 1. Cari user berdasarkan email
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("kredensial tidak valid")
	}

	// 2. Verifikasi Password
	if !auth.CheckPasswordHash(password, user.Password) {
		return nil, errors.New("kredensial tidak valid")
	}

	// 3. Buat Access & Refresh Token
	accessToken, refreshToken, err := auth.GenerateTokens(user)
	if err != nil {
		return nil, errors.New("gagal membuat token")
	}

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (u *authUsecase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return u.userRepo.GetByEmail(ctx, email)
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshTokenStr string) (*domain.LoginResponse, error) {
	// 1. Validasi Refresh Token
	token, err := auth.ValidateRefreshToken(refreshTokenStr)
	if err != nil || !token.Valid {
		return nil, errors.New("refresh token tidak valid atau kadaluarsa")
	}

	// 2. Ambil User ID dari klaim token
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, errors.New("klaim token tidak valid")
	}

	// Rune ke string ke uint (Sesuai cara kita menyimpan subject tadi)
	userIDInt, _ := strconv.Atoi(claims.Subject)

	// 3. Ambil data user terbaru dari DB
	user, err := u.userRepo.GetByID(ctx, uint(userIDInt))
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	// 4. Buat pasangan token baru
	newAccessToken, newRefreshToken, err := auth.GenerateTokens(user)
	if err != nil {
		return nil, errors.New("gagal membuat token baru")
	}

	return &domain.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         *user,
	}, nil
}

// GoogleLogin mencari user berdasarkan email.
// Jika sudah ada, langsung generate token.
// Jika belum ada, buat akun baru dengan password acak yang aman.
func (u *authUsecase) GoogleLogin(ctx context.Context, email, name string) (*domain.LoginResponse, error) {
	// 1. Cari user berdasarkan email
	user, err := u.userRepo.GetByEmail(ctx, email)

	if err != nil {
		// 2. User belum ada — buat akun baru (auto-register)
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, errors.New("gagal membuat password acak")
		}
		randomPassword := hex.EncodeToString(randomBytes)

		hashedPassword, err := auth.HashPassword(randomPassword)
		if err != nil {
			return nil, errors.New("gagal mengenkripsi password")
		}

		user = &domain.User{
			Name:     name,
			Email:    email,
			Password: hashedPassword,
			Role:     domain.RoleTutor, // Default role untuk Google Sign-In
		}

		if err := u.userRepo.Create(ctx, user); err != nil {
			return nil, errors.New("gagal membuat akun pengguna baru")
		}
	}

	// 3. Generate JWT Access & Refresh Token
	accessToken, refreshToken, err := auth.GenerateTokens(user)
	if err != nil {
		return nil, errors.New("gagal membuat token")
	}

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}
