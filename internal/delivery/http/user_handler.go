// File: internal/delivery/http/user_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	usecase domain.UserUsecase
}

// NewUserHandler membuat instance handler dan mendaftarkan rute API User Management
func NewUserHandler(r *gin.RouterGroup, us domain.UserUsecase) {
	handler := &UserHandler{
		usecase: us,
	}

	userRoutes := r.Group("/users")
	{
		userRoutes.GET("", handler.GetAll)
		userRoutes.GET("/:id", handler.GetByID)
		userRoutes.POST("", handler.Create)
		userRoutes.PUT("/:id", handler.Update)
		userRoutes.DELETE("/:id", handler.Delete)
	}
}

// NewProfileHandler mendaftarkan rute API untuk profil user sendiri
func NewProfileHandler(r *gin.RouterGroup, us domain.UserUsecase) {
	handler := &UserHandler{
		usecase: us,
	}

	r.PUT("/profile", handler.UpdateProfile)
}

// GetAll: GET /users
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *UserHandler) GetAll(c *gin.Context) {
	if c.Query("page") != "" || c.Query("limit") != "" {
		var params domain.PaginationParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter pagination tidak valid"})
			return
		}
		result, err := h.usecase.GetPaginated(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	users, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// GetByID: GET /users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	user, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// Create: POST /users
// Membuat akun pengguna baru (oleh Admin)
func (h *UserHandler) Create(c *gin.Context) {
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.usecase.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pengguna berhasil dibuat",
		"data":    user,
	})
}

// Update: PUT /users/:id
// Memperbarui data pengguna. Password opsional — jika kosong, tidak diubah.
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.usecase.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data pengguna berhasil diperbarui",
		"data":    user,
	})
}

// Delete: DELETE /users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengguna berhasil dihapus"})
}

// UpdateProfile: PUT /profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// Mengambil user_id dari context (di-set oleh AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak ditemukan"})
		return
	}

	// JWT biasanya menyimpan numeric sebagai float64 jika menggunakan jwt-go/v5 secara default
	var uid uint
	if idFloat, ok := userID.(float64); ok {
		uid = uint(idFloat)
	} else if idUint, ok := userID.(uint); ok {
		uid = idUint
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Format User ID tidak valid"})
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.usecase.UpdateProfile(c.Request.Context(), uid, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil berhasil diperbarui",
		"data":    user,
	})
}
