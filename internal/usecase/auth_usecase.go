// File: internal/usecase/auth_usecase.go
package usecase

import (
	"context"
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
