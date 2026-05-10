// File: internal/delivery/http/student_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type StudentHandler struct {
	usecase domain.StudentUsecase
}

func (h *StudentHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

// NewStudentHandler membuat instance handler dan mendaftarkan rute API-nya
func NewStudentHandler(r *gin.RouterGroup, us domain.StudentUsecase) {
	handler := &StudentHandler{
		usecase: us,
	}

	// Mendaftarkan Endpoint (seperti urls.py di Django)
	studentRoutes := r.Group("/students")
	{
		studentRoutes.GET("", handler.GetAll)
		studentRoutes.GET("/:id", handler.GetByID)
		studentRoutes.POST("", handler.Create)
		studentRoutes.PUT("/:id", handler.Update)
		studentRoutes.DELETE("/:id", handler.Delete)
		studentRoutes.DELETE("/bulk", handler.BulkDelete)
		studentRoutes.POST("/import", handler.ImportCSV)
	}
}

func (h *StudentHandler) BulkDelete(c *gin.Context) {
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

// GetAll: GET /students
// Mendukung pagination opsional via query params: ?page=1&limit=10
// Jika tidak ada query params, mengembalikan seluruh data (backward-compatible).
func (h *StudentHandler) GetAll(c *gin.Context) {
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
func (h *StudentHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	lang := h.getLang(c)
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	student, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_student_not_found")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": student})
}

// Create: POST /students
func (h *StudentHandler) Create(c *gin.Context) {
	lang := h.getLang(c)
	var req domain.UpdateStudentRequest
	// Mem-parsing body JSON ke dalam struct request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": req})
}

// Update: PUT /students/:id
func (h *StudentHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	lang := h.getLang(c)
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var req domain.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

// Delete: DELETE /students/:id
func (h *StudentHandler) Delete(c *gin.Context) {
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

// ImportCSV: POST /students/import
func (h *StudentHandler) ImportCSV(c *gin.Context) {
	lang := h.getLang(c)
	// Mengambil file dari request form-data dengan key "file"
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}

	// Membuka file yang diupload
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_file_open")})
		return
	}
	defer openedFile.Close()

	// Memproses file ke Usecase
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
