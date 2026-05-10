// File: internal/delivery/http/lesson_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type LessonHandler struct {
	usecase domain.LessonUsecase
}

func (h *LessonHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

func NewLessonHandler(r *gin.RouterGroup, us domain.LessonUsecase) {
	handler := &LessonHandler{usecase: us}

	routes := r.Group("/lessons")
	{
		routes.GET("", handler.GetAll)
		routes.GET("/course/:course_id", handler.GetByCourse)
		routes.GET("/:id", handler.GetByID)
		routes.POST("", handler.Create)
		routes.PUT("/:id", handler.Update)
		routes.DELETE("/:id", handler.Delete)
		routes.DELETE("/bulk", handler.BulkDelete)
		routes.POST("/import", handler.ImportCSV)
		routes.POST("/import-competencies", handler.ImportCompetenciesCSV)
	}
}

func (h *LessonHandler) BulkDelete(c *gin.Context) {
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

// GetAll: GET /lessons
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *LessonHandler) GetAll(c *gin.Context) {
	lang := h.getLang(c)
	if c.Query("page") != "" || c.Query("limit") != "" {
		var params domain.PaginationParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
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

	lessons, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": lessons})
}

// GetByCourse: GET /lessons/course/:course_id
func (h *LessonHandler) GetByCourse(c *gin.Context) {
	courseID, _ := strconv.Atoi(c.Param("course_id"))
	lang := h.getLang(c)
	if courseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}
	lessons, err := h.usecase.GetByCourse(c.Request.Context(), uint(courseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": lessons})
}

func (h *LessonHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	lesson, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_lesson_not_found")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": lesson})
}

func (h *LessonHandler) Create(c *gin.Context) {
	lang := h.getLang(c)
	var lesson domain.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Create(c.Request.Context(), &lesson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": lesson})
}

func (h *LessonHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	var lesson domain.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Update(c.Request.Context(), uint(id), &lesson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

func (h *LessonHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

func (h *LessonHandler) ImportCSV(c *gin.Context) {
	lang := h.getLang(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}
	opened, _ := file.Open()
	defer opened.Close()

	result, err := h.usecase.ImportCSV(c.Request.Context(), opened)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_import_success"),
		"created": result.Created,
		"updated": result.Updated,
		"errors":  result.Errors,
	})
}

func (h *LessonHandler) ImportCompetenciesCSV(c *gin.Context) {
	lang := h.getLang(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}
	opened, _ := file.Open()
	defer opened.Close()

	result, err := h.usecase.ImportCompetenciesCSV(c.Request.Context(), opened)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_import_success"),
		"created": result.Created,
		"updated": result.Updated,
		"errors":  result.Errors,
	})
}
