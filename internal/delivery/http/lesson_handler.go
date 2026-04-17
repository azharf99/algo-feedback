// File: internal/delivery/http/lesson_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type LessonHandler struct {
	usecase domain.LessonUsecase
}

func NewLessonHandler(r *gin.RouterGroup, us domain.LessonUsecase) {
	handler := &LessonHandler{usecase: us}

	routes := r.Group("/lessons")
	{
		routes.GET("", handler.GetAll)
		routes.GET("/:id", handler.GetByID)
		routes.POST("", handler.Create)
		routes.PUT("/:id", handler.Update)
		routes.DELETE("/:id", handler.Delete)
		routes.POST("/import", handler.ImportCSV)
	}
}

func (h *LessonHandler) GetAll(c *gin.Context) {
	lessons, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": lessons})
}

func (h *LessonHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lesson, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelajaran tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": lesson})
}

func (h *LessonHandler) Create(c *gin.Context) {
	var lesson domain.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Create(c.Request.Context(), &lesson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": lesson})
}

func (h *LessonHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var lesson domain.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.usecase.Update(c.Request.Context(), uint(id), &lesson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pelajaran diperbarui"})
}

func (h *LessonHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pelajaran dihapus"})
}

func (h *LessonHandler) ImportCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File CSV diperlukan"})
		return
	}
	opened, _ := file.Open()
	defer opened.Close()

	result, err := h.usecase.ImportCSV(c.Request.Context(), opened)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
