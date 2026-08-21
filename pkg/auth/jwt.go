// File: pkg/auth/jwt.go
package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// GetJWTSecret mengambil secret key dari environment variable
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set")
	}
	return []byte(secret)
}

// GetJWTRefreshSecret mengambil refresh secret key dari environment variable
func GetJWTRefreshSecret() []byte {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		panic("JWT_REFRESH_SECRET environment variable is not set")
	}
	return []byte(secret)
}

type JwtCustomClaims struct {
	UserID uint        `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateTokens membuat Access Token (7 hari) & Refresh Token (14 hari)
func GenerateTokens(user *domain.User) (string, string, error) {
	now := time.Now()

	// 1. Access Token (Diperpanjang ke 7 hari sesuai permintaan user)
	claims := &JwtCustomClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(GetJWTSecret())
	if err != nil {
		return "", "", err
	}

	// 2. Refresh Token (Diperpanjang ke 14 hari)
	refreshClaims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID), // Simpan ID sebagai string
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(14 * 24 * time.Hour)),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(GetJWTRefreshSecret())

	return accessToken, refreshTokenStr, err
}

// ValidateRefreshToken memverifikasi Refresh Token dan mengembalikan token yang valid
func ValidateRefreshToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return GetJWTRefreshSecret(), nil
	})
}
