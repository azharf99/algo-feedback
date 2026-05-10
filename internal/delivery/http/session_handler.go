// File: internal/delivery/http/session_handler.go
package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	usecase domain.SessionUsecase
}

func (h *SessionHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

func NewSessionHandler(r *gin.RouterGroup, us domain.SessionUsecase) {
	handler := &SessionHandler{usecase: us}

	routes := r.Group("/sessions")
	{
		routes.GET("", handler.GetAll)
		routes.GET("/summary", handler.GetWeeklySummary)
		routes.GET("/:id", handler.GetByID)
		routes.GET("/group/:group_id", handler.GetByGroup)
		routes.POST("", handler.Create)
		routes.PUT("/:id", handler.Update)
		routes.DELETE("/:id", handler.Delete)
		routes.DELETE("/bulk", handler.BulkDelete)

		// Endpoint Khusus Absensi
		routes.POST("/:id/attendance", handler.UpdateAttendance)
		routes.POST("/mark-done", handler.MarkDoneUpToDate)
		routes.POST("/mark-cancelled", handler.MarkCancelled)
		routes.POST("/auto-fill-attendance", handler.AutoFillAttendance)
	}
}

func (h *SessionHandler) GetWeeklySummary(c *gin.Context) {
	result, err := h.usecase.GetWeeklySummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
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
	lang := h.getLang(c)
	session, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_session_not_found")})
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
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Create(c.Request.Context(), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": session})
}

func (h *SessionHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var session domain.Session
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}
	if err := h.usecase.Update(c.Request.Context(), uint(id), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

func (h *SessionHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

func (h *SessionHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	if err := h.usecase.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

// UpdateAttendance: POST /sessions/:id/attendance
// Body: { "student_ids": [101, 102, 105] }
func (h *SessionHandler) UpdateAttendance(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	var req struct {
		StudentIDs []uint `json:"student_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	if err := h.usecase.UpdateAttendance(c.Request.Context(), uint(id), req.StudentIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_attendance_success")})
}

func (h *SessionHandler) MarkDoneUpToDate(c *gin.Context) {
	lang := h.getLang(c)
	var req struct {
		GroupID   uint   `json:"group_id" binding:"required"`
		UntilDate string `json:"until_date" binding:"required"` // YYYY-MM-DD
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.UntilDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_date_format")})
		return
	}

	if err := h.usecase.MarkDoneUpToDate(c.Request.Context(), req.GroupID, date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_mark_done_success")})
}

func (h *SessionHandler) AutoFillAttendance(c *gin.Context) {
	lang := h.getLang(c)
	var req struct {
		GroupID   uint   `json:"group_id" binding:"required"`
		UntilDate string `json:"until_date" binding:"required"` // YYYY-MM-DD
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.UntilDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_date_format")})
		return
	}

	if err := h.usecase.AutoFillAttendance(c.Request.Context(), req.GroupID, date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_autofill_success")})
}

func (h *SessionHandler) MarkCancelled(c *gin.Context) {
	lang := h.getLang(c)
	var req struct {
		GroupID    uint   `json:"group_id" binding:"required"`
		FromDate   string `json:"from_date" binding:"required"`   // YYYY-MM-DD
		BeforeDate string `json:"before_date" binding:"required"` // YYYY-MM-DD
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fromDate, err := time.Parse("2006-01-02", req.FromDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_date_format")})
		return
	}

	beforeDate, err := time.Parse("2006-01-02", req.BeforeDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_date_format")})
		return
	}

	if err := h.usecase.MarkCancelled(c.Request.Context(), req.GroupID, fromDate, beforeDate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_cancel_success")})
}
