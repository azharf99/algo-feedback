// File: internal/delivery/http/student_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type StudentHandler struct {
	usecase domain.StudentUsecase
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
		studentRoutes.POST("/import", handler.ImportCSV)
	}
}

// GetAll: GET /students
func (h *StudentHandler) GetAll(c *gin.Context) {
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
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	student, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Siswa tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": student})
}

// Create: POST /students
func (h *StudentHandler) Create(c *gin.Context) {
	var req domain.Student
	// Mem-parsing body JSON ke dalam struct Student (seperti Serializer di Django)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Siswa berhasil dibuat", "data": req})
}

// Update: PUT /students/:id
func (h *StudentHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req domain.Student
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data siswa berhasil diperbarui"})
}

// Delete: DELETE /students/:id
func (h *StudentHandler) Delete(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"message": "Data siswa berhasil dihapus"})
}

// ImportCSV: POST /students/import
func (h *StudentHandler) ImportCSV(c *gin.Context) {
	// Mengambil file dari request form-data dengan key "file"
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File CSV tidak ditemukan pada form-data key 'file'"})
		return
	}

	// Membuka file yang diupload
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
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
		"message": "Proses import selesai",
		"created": result.Created,
		"updated": result.Updated,
		"errors":  result.Errors,
	})
}
