// File: internal/middleware/auth_middleware.go
package middleware

import (
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
		secretKey := []byte(os.Getenv("JWT_SECRET"))

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode penandatanganan tidak valid")
			}
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kedaluwarsa"})
			c.Abort()
			return
		}

		// Menyimpan data pengguna ke dalam context (untuk digunakan di handler selanjutnya)
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userIDFloat, _ := claims["user_id"].(float64)
			userID := uint(userIDFloat)

			// Tolak token yang diterbitkan sebelum password terakhir diubah (session
			// revocation). Ini juga otomatis menolak token milik user yang sudah dihapus.
			if user, errUser := userRepo.GetByID(c.Request.Context(), userID); errUser != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kedaluwarsa"})
				c.Abort()
				return
			} else if user.PasswordChangedAt != nil {
				iatFloat, _ := claims["iat"].(float64)
				issuedAt := time.Unix(int64(iatFloat), 0)
				if issuedAt.Before(*user.PasswordChangedAt) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid, silakan login ulang"})
					c.Abort()
					return
				}
			}

			c.Set("user_id", claims["user_id"])
			c.Set("role", claims["role"])

			// Inject ke standard context agar bisa diakses di repository layer
			ctx := ctxutil.WithUserID(c.Request.Context(), userID)
			roleStr, _ := claims["role"].(string)
			ctx = ctxutil.WithRole(ctx, roleStr)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
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
