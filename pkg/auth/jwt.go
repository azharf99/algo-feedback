// File: pkg/auth/jwt.go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// TODO: Nanti ambil dari .env
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var jwtRefreshSecret = []byte(os.Getenv("JWT_REFRESH_SECRET"))

type JwtCustomClaims struct {
	UserID uint        `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateTokens membuat Access Token (15 menit) & Refresh Token (7 hari)
func GenerateTokens(user *domain.User) (string, string, error) {
	// 1. Access Token
	claims := &JwtCustomClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// 2. Refresh Token (Hanya menyimpan ID untuk keamanan)
	refreshClaims := jwt.RegisteredClaims{
		Subject:   string(rune(user.ID)), // Menyimpan ID user
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(jwtRefreshSecret)

	return accessToken, refreshTokenStr, err
}

// ValidateRefreshToken memverifikasi Refresh Token dan mengembalikan token yang valid
func ValidateRefreshToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode penandatanganan tidak valid")
		}
		return jwtRefreshSecret, nil
	})
}
