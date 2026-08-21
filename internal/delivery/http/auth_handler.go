// File: internal/delivery/http/auth_handler.go
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/middleware"
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/oauth"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type AuthHandler struct {
	usecase domain.AuthUsecase
}

func (h *AuthHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
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
		authRoutes.POST("/google/login", handler.GoogleOneTap)
		authRoutes.POST("/forgot-password", handler.ForgotPassword)
		authRoutes.POST("/reset-password", handler.ResetPassword)
	}

	// Callback harus di-register di level /api (bukan /api/auth)
	// agar sesuai dengan redirect URI yang didaftarkan di GCP: /api/callback
	r.GET("/callback", handler.GoogleCallback)
}

// Request Body Structs
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	// Registrasi publik hanya boleh membuat akun Tutor/Siswa. "Admin" sengaja tidak masuk
	// daftar yang diizinkan di sini — akun Admin hanya dibuat lewat seeder atau oleh Admin
	// lain via /api/users (RoleAdmin only).
	Role         domain.Role `json:"role" binding:"required,oneof=Tutor Siswa"`
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

type GoogleOneTapRequest struct {
	Credential string `json:"credential" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
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
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	valid, err := auth.VerifyRecaptcha(req.CaptchaToken)
	if err != nil || !valid {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_unauthorized")})
		return
	}

	// Validasi Email apakah sudah ada? GetUserByEmail mengembalikan gorm.ErrRecordNotFound
	// (bukan registeredUser == nil) untuk email yang belum terdaftar — itu kondisi normal,
	// bukan error. Hanya anggap sebagai konflik jika user-nya benar-benar ditemukan.
	registeredUser, err := h.usecase.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil && registeredUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_user_not_found")})
		return
	}

	hashPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}

	user := domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashPassword,
		Role:     req.Role,
	}

	if err := h.usecase.Register(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registration successful"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	// Verifikasi Captcha
	valid, err := auth.VerifyRecaptcha(req.CaptchaToken)
	if err != nil || !valid {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_unauthorized")})
		return
	}

	res, err := h.usecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "data": res})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	res, err := h.usecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_token_refreshed"), "data": res})
}

// GoogleLogin mengarahkan pengguna ke halaman consent Google OAuth2.
// State parameter digunakan untuk mencegah serangan CSRF.
// State disimpan di HttpOnly cookie agar tidak bisa diakses oleh JavaScript.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	oauthConfig := oauth.GoogleOAuthConfig()
	lang := h.getLang(c)

	// Generate state acak yang aman secara kriptografis (32 byte = 64 hex chars)
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_state_gen_failed")})
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
	lang := h.getLang(c)

	// 1. Validasi CSRF — bandingkan state dari query dengan state dari cookie
	queryState := c.Query("state")
	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_csrf_detected")})
		return
	}

	if queryState != cookieState {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_csrf_detected")})
		return
	}

	// Hapus cookie state setelah digunakan (one-time use)
	isSecure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("oauth_state", "", -1, "/", "", isSecure, true)

	// 2. Tukar authorization code dengan access token
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_no_auth_code")})
		return
	}

	token, err := oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_token_exchange_failed")})
		return
	}

	// 3. Ambil profil user dari Google API menggunakan access token
	client := oauthConfig.Client(c.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_fetch_google_profile_failed")})
		return
	}
	defer resp.Body.Close()

	// Batasi pembacaan body hingga 1MB untuk keamanan
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_read_google_resp_failed")})
		return
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_process_google_data_failed")})
		return
	}

	// 4. Validasi: pastikan email terverifikasi
	if !googleUser.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_email_unverified")})
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

	// Ubah struct User menjadi JSON byte
	userBytes, err := json.Marshal(loginRes.User)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_process_user_data_failed")})
		return
	}

	userEncoded := url.QueryEscape(string(userBytes))

	redirectURL := fmt.Sprintf(
		"%s/auth/success#access_token=%s&refresh_token=%s&user=%s",
		frontendURL,
		loginRes.AccessToken,
		loginRes.RefreshToken,
		userEncoded,
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// GoogleOneTap menangani login via Google One Tap (ID Token verification)
func (h *AuthHandler) GoogleOneTap(c *gin.Context) {
	var req GoogleOneTapRequest
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	// 1. Verifikasi ID Token
	googleUser, err := oauth.VerifyIDToken(c.Request.Context(), req.Credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.Tf(lang, "error_google_verify_failed", err.Error())})
		return
	}

	// 2. Pastikan email terverifikasi
	if !googleUser.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(lang, "error_email_unverified")})
		return
	}

	// 3. Login atau auto-register via AuthUsecase
	loginRes, err := h.usecase.GoogleLogin(c.Request.Context(), googleUser.Email, googleUser.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_login_success"),
		"data":    loginRes,
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	if err := h.usecase.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_send_email_failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_forgot_password_sent")})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}
