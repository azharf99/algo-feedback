// File: internal/usecase/auth_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/azharf99/algo-feedback/pkg/mail"
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

func (u *authUsecase) ForgotPassword(ctx context.Context, email string) error {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Untuk alasan keamanan, jangan beritahu jika email tidak ditemukan
		return nil
	}

	// 1. Generate Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	// 2. Set Expiry (1 jam)
	expiresAt := time.Now().Add(time.Hour * 1)
	user.ResetPasswordToken = token
	user.ResetPasswordExpiresAt = &expiresAt

	// 3. Simpan ke DB
	if err := u.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 4. Kirim Email
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)
	subject := "Reset Password - Algonova Feedback System"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				.button {
					background-color: #4CAF50;
					border: none;
					color: white;
					padding: 15px 32px;
					text-align: center;
					text-decoration: none;
					display: inline-block;
					font-size: 16px;
					margin: 4px 2px;
					cursor: pointer;
					border-radius: 8px;
				}
				.container {
					font-family: Arial, sans-serif;
					max-width: 600px;
					margin: auto;
					padding: 20px;
					border: 1px solid #ddd;
					border-radius: 10px;
				}
				.footer {
					font-size: 12px;
					color: #888;
					margin-top: 20px;
					border-top: 1px solid #eee;
					padding-top: 10px;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<h2 style="color: #333;">Permintaan Reset Password</h2>
				<p>Halo <strong>%s</strong>,</p>
				<p>Kami menerima permintaan untuk mereset password akun Algonova Feedback Anda. Jika Anda tidak melakukan permintaan ini, silakan abaikan email ini.</p>
				<p>Untuk melanjutkan proses reset password, silakan klik tombol di bawah ini:</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" class="button" style="color: white;">Reset Password Sekarang</a>
				</div>
				<p>Atau salin dan tempel link berikut di browser Anda:</p>
				<p style="word-break: break-all; color: #666; font-size: 14px;">%s</p>
				<p>Link ini akan kadaluarsa dalam <strong>1 jam</strong> demi keamanan akun Anda.</p>
				<div class="footer">
					<p>Email ini dikirim secara otomatis oleh sistem Algonova Feedback.<br>
					© 2026 Algonova Feedback. Semua hak dilindungi.</p>
				</div>
			</div>
		</body>
		</html>
	`, user.Name, resetLink, resetLink)

	return mail.SendMail(user.Email, subject, body)
}

func (u *authUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	// 1. Cari user berdasarkan token
	user, err := u.userRepo.GetByResetToken(ctx, token)
	if err != nil {
		return errors.New("token reset password tidak valid")
	}

	// 2. Cek Expiry
	if user.ResetPasswordExpiresAt == nil || time.Now().After(*user.ResetPasswordExpiresAt) {
		return errors.New("token reset password sudah kadaluarsa")
	}

	// 3. Hash Password Baru
	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 4. Update User & Bersihkan Token
	user.Password = hashedPassword
	user.ResetPasswordToken = ""
	user.ResetPasswordExpiresAt = nil

	return u.userRepo.Update(ctx, user)
}
