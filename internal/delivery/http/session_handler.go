// File: internal/delivery/http/session_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	usecase domain.SessionUsecase
}

func NewSessionHandler(r *gin.RouterGroup, us domain.SessionUsecase) {
	handler := &SessionHandler{usecase: us}

	routes := r.Group("/sessions")
	{
		routes.GET("", handler.GetAll)
		routes.GET("/:id", handler.GetByID)
		routes.GET("/group/:group_id", handler.GetByGroup)
		routes.POST("", handler.Create)
		routes.PUT("/:id", handler.Update)
		routes.DELETE("/:id", handler.Delete)

		// Endpoint Khusus Absensi
		routes.POST("/:id/attendance", handler.UpdateAttendance)
	}
}

func (h *SessionHandler) GetAll(c *gin.Context) {
	var params domain.PaginationParams
	c.ShouldBindQuery(&params)

	result, err := h.usecase.GetPaginated(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *SessionHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	session, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (h *SessionHandler) GetByGroup(c *gin.Context) {
	groupID, _ := strconv.Atoi(c.Param("group_id"))
	sessions, err := h.usecase.GetByGroup(c.Request.Context(), uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

func (h *SessionHandler) Create(c *gin.Context) {
	var session domain.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Create(c.Request.Context(), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": session})
}

func (h *SessionHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var session domain.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid"})
		return
	}
	if err := h.usecase.Update(c.Request.Context(), uint(id), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sesi diperbarui"})
}

func (h *SessionHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sesi dihapus"})
}

// UpdateAttendance: POST /sessions/:id/attendance
// Body: { "student_ids": [101, 102, 105] }
func (h *SessionHandler) UpdateAttendance(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		StudentIDs []uint `json:"student_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format student_ids tidak valid"})
		return
	}

	if err := h.usecase.UpdateAttendance(c.Request.Context(), uint(id), req.StudentIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kehadiran siswa berhasil disimpan"})
}
