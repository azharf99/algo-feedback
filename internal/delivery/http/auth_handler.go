// File: internal/delivery/http/auth_handler.go
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/middleware"
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/azharf99/algo-feedback/pkg/oauth"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type AuthHandler struct {
	usecase domain.AuthUsecase
}

func NewAuthHandler(r *gin.RouterGroup, us domain.AuthUsecase) {
	handler := &AuthHandler{usecase: us}

	authRoutes := r.Group("/auth")
	// Rate limit: 5 request per menit per IP untuk Login/Register
	authRoutes.Use(middleware.RateLimitMiddleware(rate.Limit(5.0/60.0), 10))
	{
		authRoutes.POST("/register", handler.Register)
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/refresh", handler.RefreshToken)
		authRoutes.GET("/google/login", handler.GoogleLogin)
	}

	// Callback harus di-register di level /api (bukan /api/auth)
	// agar sesuai dengan redirect URI yang didaftarkan di GCP: /api/callback
	r.GET("/callback", handler.GoogleCallback)
}

// Request Body Structs
type RegisterRequest struct {
	Name         string      `json:"name" binding:"required"`
	Email        string      `json:"email" binding:"required,email"`
	Password     string      `json:"password" binding:"required,min=6"`
	Role         domain.Role `json:"role" binding:"required"`
	CaptchaToken string      `json:"captcha_token" binding:"required"`
}

type LoginRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	CaptchaToken string `json:"captcha_token" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// GoogleUserInfo menyimpan data profil user dari Google API
type GoogleUserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifikasi Captcha
	valid, err := auth.VerifyRecaptcha(req.CaptchaToken)
	if err != nil || !valid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Verifikasi captcha gagal"})
		return
	}

	hashPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
		return
	}

	user := domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashPassword,
		Role:     req.Role,
	}

	if err := h.usecase.Register(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan pengguna"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registrasi berhasil"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format email atau password salah"})
		return
	}

	// Verifikasi Captcha
	valid, err := auth.VerifyRecaptcha(req.CaptchaToken)
	if err != nil || !valid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Verifikasi captcha gagal"})
		return
	}

	res, err := h.usecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login berhasil", "data": res})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token diperlukan"})
		return
	}

	res, err := h.usecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token berhasil diperbarui", "data": res})
}

// GoogleLogin mengarahkan pengguna ke halaman consent Google OAuth2.
// State parameter digunakan untuk mencegah serangan CSRF.
// State disimpan di HttpOnly cookie agar tidak bisa diakses oleh JavaScript.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	oauthConfig := oauth.GoogleOAuthConfig()

	// Generate state acak yang aman secara kriptografis (32 byte = 64 hex chars)
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat state keamanan"})
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Tentukan apakah cookie harus Secure (HTTPS-only)
	isSecure := os.Getenv("GIN_MODE") == "release"

	// Simpan state di HttpOnly Cookie — tidak bisa diakses via JavaScript (XSS-safe)
	c.SetCookie(
		"oauth_state", // nama cookie
		state,         // nilai state
		600,           // max age: 10 menit (cukup untuk proses login)
		"/",           // path
		"",            // domain (otomatis)
		isSecure,      // secure: true hanya di HTTPS (production)
		true,          // httpOnly: true — mencegah akses dari JavaScript
	)

	// Redirect ke halaman consent Google
	url := oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback menangani callback dari Google setelah user login.
// Alur keamanan:
// 1. Validasi state parameter (anti-CSRF)
// 2. Tukar authorization code dengan access token
// 3. Ambil profil user dari Google API
// 4. Login atau auto-register user
// 5. Redirect ke frontend dengan JWT token
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	oauthConfig := oauth.GoogleOAuthConfig()

	// 1. Validasi CSRF — bandingkan state dari query dengan state dari cookie
	queryState := c.Query("state")
	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "State cookie tidak ditemukan. Kemungkinan serangan CSRF."})
		return
	}

	if queryState != cookieState {
		c.JSON(http.StatusForbidden, gin.H{"error": "State tidak cocok. Kemungkinan serangan CSRF."})
		return
	}

	// Hapus cookie state setelah digunakan (one-time use)
	isSecure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("oauth_state", "", -1, "/", "", isSecure, true)

	// 2. Tukar authorization code dengan access token
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code tidak ditemukan"})
		return
	}

	token, err := oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menukar kode otorisasi dengan token"})
		return
	}

	// 3. Ambil profil user dari Google API menggunakan access token
	client := oauthConfig.Client(c.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data profil Google"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca respons profil Google"})
		return
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses data profil Google"})
		return
	}

	// 4. Validasi: pastikan email terverifikasi
	if !googleUser.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email Google Anda belum terverifikasi"})
		return
	}

	// 5. Login atau auto-register via AuthUsecase
	loginRes, err := h.usecase.GoogleLogin(c.Request.Context(), googleUser.Email, googleUser.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 6. Redirect ke frontend dengan token di URL parameter
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Fallback aman untuk development
	}

	redirectURL := fmt.Sprintf(
		"%s/auth/success?access_token=%s&refresh_token=%s",
		frontendURL,
		loginRes.AccessToken,
		loginRes.RefreshToken,
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
