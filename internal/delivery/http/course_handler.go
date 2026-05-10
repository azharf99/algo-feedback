// File: internal/delivery/http/course_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	usecase domain.CourseUsecase
}

func (h *CourseHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

func NewCourseHandler(r *gin.RouterGroup, us domain.CourseUsecase) {
	handler := &CourseHandler{usecase: us}

	routes := r.Group("/courses")
	{
		routes.GET("", handler.GetAll)
		routes.GET("/:id", handler.GetByID)
		routes.POST("", handler.Create)
		routes.PUT("/:id", handler.Update)
		routes.DELETE("/:id", handler.Delete)
		routes.DELETE("/bulk", handler.BulkDelete)
		routes.POST("/import", handler.ImportCSV)
	}
}

func (h *CourseHandler) BulkDelete(c *gin.Context) {
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

func (h *CourseHandler) GetAll(c *gin.Context) {
	lang := h.getLang(c)
	if c.Query("page") != "" {
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

	courses, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": courses})
}

func (h *CourseHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	course, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_course_not_found")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": course})
}

func (h *CourseHandler) Create(c *gin.Context) {
	lang := h.getLang(c)
	var course domain.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Create(c.Request.Context(), &course); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": course})
}

func (h *CourseHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	var course domain.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}
	if err := h.usecase.Update(c.Request.Context(), uint(id), &course); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

func (h *CourseHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lang := h.getLang(c)
	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

func (h *CourseHandler) ImportCSV(c *gin.Context) {
	lang := h.getLang(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_file_open")})
		return
	}
	defer openedFile.Close()

	result, err := h.usecase.ImportCSV(c.Request.Context(), openedFile)
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
