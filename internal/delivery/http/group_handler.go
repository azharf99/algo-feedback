// File: internal/delivery/http/group_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	usecase domain.GroupUsecase
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
		groupRoutes.POST("/import", handler.ImportCSV)
	}
}

// GetAll: GET /groups
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *GroupHandler) GetAll(c *gin.Context) {
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
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	student, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grup Siswa tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": student})
}

// Create: POST /students
func (h *GroupHandler) Create(c *gin.Context) {
	var req domain.Group
	// Mem-parsing body JSON ke dalam struct Student (seperti Serializer di Django)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Grup Siswa berhasil dibuat", "data": req})
}

// Update: PUT /students/:id
func (h *GroupHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req domain.Group
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data Grup siswa berhasil diperbarui"})
}

// Delete: DELETE /students/:id
func (h *GroupHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data Grup siswa berhasil dihapus"})
}

// ImportCSV: POST /groups/import
func (h *GroupHandler) ImportCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File CSV tidak ditemukan"})
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
		return
	}
	defer openedFile.Close()

	result, err := h.usecase.ImportCSV(c.Request.Context(), openedFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Proses import selesai",
		"created": result.Created,
		"updated": result.Updated,
		"errors":  result.Errors,
	})
}
