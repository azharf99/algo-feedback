// File: internal/delivery/http/feedback_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	usecase domain.FeedbackUsecase
}

func NewFeedbackHandler(r *gin.RouterGroup, us domain.FeedbackUsecase) {
	handler := &FeedbackHandler{usecase: us}

	feedbackRoutes := r.Group("/feedbacks")
	{
		// Endpoints canggih!
		feedbackRoutes.POST("/seeder", handler.RunSeeder)
		feedbackRoutes.POST("/generate-pdf", handler.GeneratePDF)
		feedbackRoutes.POST("/send-wa", handler.SendWhatsApp)

		feedbackRoutes.GET("", handler.GetAll)
		feedbackRoutes.GET("/:id", handler.GetByID)
		feedbackRoutes.POST("", handler.Create)
		feedbackRoutes.PUT("/:id", handler.Update)
		feedbackRoutes.DELETE("/:id", handler.Delete)
	}
}

// RunSeeder: POST /feedbacks/seeder?group_id=1&all=true
func (h *FeedbackHandler) RunSeeder(c *gin.Context) {
	allStr := c.Query("all")
	all := allStr == "true"

	var groupIDPtr *uint
	if gIDStr := c.Query("group_id"); gIDStr != "" {
		if id, err := strconv.Atoi(gIDStr); err == nil {
			parsedID := uint(id)
			groupIDPtr = &parsedID
		}
	}

	result, err := h.usecase.GenerateFeedback(c.Request.Context(), groupIDPtr, all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Seeder berhasil dijalankan",
		"data":    result,
	})
}

// GeneratePDF: POST /feedbacks/generate-pdf
func (h *FeedbackHandler) GeneratePDF(c *gin.Context) {
	// Parsing Request Body JSON
	var req struct {
		StudentID *uint   `json:"student_id"`
		Course    *string `json:"course"`
		Number    *uint   `json:"number"`
		All       bool    `json:"all"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid"})
		return
	}

	result, err := h.usecase.GeneratePDFAsync(c.Request.Context(), req.StudentID, req.Course, req.Number, req.All)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menjalankan task PDF"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task PDF berjalan di background",
		"tasks":   result,
	})
}

// SendWhatsApp: POST /feedbacks/send-wa?feedback_id=10
func (h *FeedbackHandler) SendWhatsApp(c *gin.Context) {
	var feedbackIDPtr *uint
	if fIDStr := c.Query("feedback_id"); fIDStr != "" {
		if id, err := strconv.Atoi(fIDStr); err == nil {
			parsedID := uint(id)
			feedbackIDPtr = &parsedID
		}
	}

	result, err := h.usecase.SendFeedbackPDF(c.Request.Context(), feedbackIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pesan WhatsApp berhasil dijadwalkan",
		"data":    result,
	})
}

// GetAll: GET /feedbacks
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *FeedbackHandler) GetAll(c *gin.Context) {
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

	feedbacks, err := h.usecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": feedbacks})
}

// GetByID: GET /feedbacks/:id
func (h *FeedbackHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	feedback, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feedback tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": feedback})
}

// Create: POST /feedbacks
func (h *FeedbackHandler) Create(c *gin.Context) {
	var req domain.Feedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Feedback berhasil dibuat secara manual", "data": req})
}

// Update: PUT /feedbacks/:id
func (h *FeedbackHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req domain.Feedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data feedback berhasil diperbarui"})
}

// Delete: DELETE /feedbacks/:id
func (h *FeedbackHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data feedback berhasil dihapus"})
}
