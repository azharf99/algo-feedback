// File: internal/delivery/http/group_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	usecase domain.GroupUsecase
}

func (h *GroupHandler) getLang(c *gin.Context) string {
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		return "Indonesia"
	}
	return lang
}

func NewGroupHandler(r *gin.RouterGroup, us domain.GroupUsecase) {
	handler := &GroupHandler{usecase: us}

	groupRoutes := r.Group("/groups")
	{
		groupRoutes.GET("", handler.GetAll)
		groupRoutes.GET("/:id", handler.GetByID)
		groupRoutes.POST("", handler.Create)
		groupRoutes.PUT("/:id", handler.Update)
		groupRoutes.DELETE("/:id", handler.Delete)
		groupRoutes.DELETE("/bulk", handler.BulkDelete)
		groupRoutes.POST("/import", handler.ImportCSV)
	}
}

func (h *GroupHandler) BulkDelete(c *gin.Context) {
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

// GetAll: GET /groups
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *GroupHandler) GetAll(c *gin.Context) {
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

	students, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": students})
}

// GetByID: GET /students/:id
func (h *GroupHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	lang := h.getLang(c)
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	student, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_group_not_found")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": student})
}

// Create: POST /groups
func (h *GroupHandler) Create(c *gin.Context) {
	lang := h.getLang(c)
	var payload struct {
		domain.Group
		StudentIDs []uint `json:"student_ids"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &payload.Group, payload.StudentIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": payload.Group})
}

// Update: PUT /groups/:id
func (h *GroupHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	lang := h.getLang(c)
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var payload struct {
		domain.Group
		StudentIDs []uint `json:"student_ids"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &payload.Group, payload.StudentIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

// Delete: DELETE /students/:id
func (h *GroupHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	lang := h.getLang(c)
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

// ImportCSV: POST /groups/import
func (h *GroupHandler) ImportCSV(c *gin.Context) {
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
