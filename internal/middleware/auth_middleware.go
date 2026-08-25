// File: internal/middleware/auth_middleware.go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware memvalidasi JWT Token. userRepo dipakai untuk mengecek PasswordChangedAt,
// sehingga access token yang diterbitkan (iat) sebelum password terakhir diubah otomatis
// ditolak — tanpa ini, token yang bocor tetap valid sampai masa berlakunya habis (7 hari)
// walau korban sudah mereset password lewat "Lupa Password".
func AuthMiddleware(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
			c.Abort()
			return
		}

		// Token biasanya dikirim dalam format: "Bearer <token>"
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		userID, role, err := ValidateAccessToken(c.Request.Context(), userRepo, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kedaluwarsa"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)

		// Inject ke standard context agar bisa diakses di repository layer
		ctx := ctxutil.WithUserID(c.Request.Context(), userID)
		ctx = ctxutil.WithRole(ctx, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// WSAuthMiddleware memvalidasi JWT yang dikirim lewat query param ?token=, karena Browser
// WebSocket API (`new WebSocket(url)`) tidak bisa mengirim header Authorization kustom.
// Validasi (termasuk revocation berbasis PasswordChangedAt) identik dengan AuthMiddleware.
func WSAuthMiddleware(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Query("token")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
			c.Abort()
			return
		}

		userID, role, err := ValidateAccessToken(c.Request.Context(), userRepo, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kedaluwarsa"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)

		ctx := ctxutil.WithUserID(c.Request.Context(), userID)
		ctx = ctxutil.WithRole(ctx, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// ValidateAccessToken memvalidasi signature, expiry, dan revocation (PasswordChangedAt)
// sebuah access token JWT, lalu mengembalikan user_id dan role di dalamnya.
// Dipakai bersama oleh AuthMiddleware (header) dan WSAuthMiddleware (query param) agar
// kedua jalur autentikasi selalu punya aturan validasi yang sama persis.
func ValidateAccessToken(ctx context.Context, userRepo domain.UserRepository, tokenString string) (uint, string, error) {
	secretKey := []byte(os.Getenv("JWT_SECRET"))

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode penandatanganan tidak valid")
		}
		return secretKey, nil
	})
	if err != nil || !token.Valid {
		return 0, "", fmt.Errorf("token tidak valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", fmt.Errorf("token tidak valid")
	}

	userIDFloat, _ := claims["user_id"].(float64)
	userID := uint(userIDFloat)

	// Tolak token yang diterbitkan sebelum password terakhir diubah (session revocation).
	// Ini juga otomatis menolak token milik user yang sudah dihapus.
	user, errUser := userRepo.GetByID(ctx, userID)
	if errUser != nil {
		return 0, "", fmt.Errorf("token tidak valid")
	}
	if user.PasswordChangedAt != nil {
		iatFloat, _ := claims["iat"].(float64)
		issuedAt := time.Unix(int64(iatFloat), 0)
		if issuedAt.Before(*user.PasswordChangedAt) {
			return 0, "", fmt.Errorf("sesi tidak valid, silakan login ulang")
		}
	}

	roleStr, _ := claims["role"].(string)
	return userID, roleStr, nil
}

// RoleMiddleware membatasi akses berdasarkan Role (RBAC)
func RoleMiddleware(allowedRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			c.Abort()
			return
		}

		isAllowed := false
		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Format role tidak valid"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if domain.Role(roleStr) == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk akses ini"})
			c.Abort()
			return
		}
		c.Next()
	}
}
